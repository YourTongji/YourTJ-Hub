import json
import unittest
from prepare_mobile import next_release, parse_pubspec, parse_annotation


class MobileVersionTest(unittest.TestCase):
    def test_patch_minor_major_and_first_release(self):
        for bump, expected in [('patch', '1.2.4'), ('minor', '1.3.0'), ('major', '2.0.0')]:
            with self.subTest(bump=bump):
                self.assertEqual(next_release(('1.0.0', 1), [('1.2.3', 8)], bump), (expected, 9))
        self.assertEqual(next_release(('1.0.0', 1), [], 'patch'), ('1.0.1', 2))

    def test_numeric_version_order_and_global_build_counter(self):
        self.assertEqual(next_release(('1.0.0', 1), [('1.9.0', 30), ('1.10.0', 20)], 'patch'), ('1.10.1', 31))
        self.assertEqual(next_release(('2.0.0', 50), [('1.10.0', 20)], 'minor'), ('2.1.0', 51))

    def test_invalid_bump_and_overflow_are_rejected(self):
        with self.assertRaises(ValueError): next_release(('1.0.0', 1), [], 'invalid')
        with self.assertRaises(ValueError): next_release(('1.0.0', 2099996000), [], 'patch')

    def test_pubspec_floor_is_strict(self):
        self.assertEqual(parse_pubspec('name: forum_app\nversion: 1.0.0+1\n'), ('1.0.0', 1))
        for value in ['1.0', '1.0.0+0', '1.0.0+abc']:
            with self.subTest(value=value), self.assertRaises(ValueError): parse_pubspec('version: '+value+'\n')

    def test_tag_annotation_preserves_release_identity(self):
        message = json.dumps({'schema': 1, 'version': '1.2.3', 'buildNumber': 19})
        self.assertEqual(parse_annotation('mobile-v1.2.3', message), ('1.2.3', 19))
        for data in [{}, {'schema': 1, 'version': '1.2.4', 'buildNumber': 19},
                     {'schema': 1, 'version': '1.2.3', 'buildNumber': True},
                     {'schema': 1, 'version': '1.2.3', 'buildNumber': -1}]:
            with self.subTest(data=data), self.assertRaises(ValueError):
                parse_annotation('mobile-v1.2.3', json.dumps(data))

class MobilePreparationTest(unittest.TestCase):
    def test_ios_recovery_requires_tag_before_any_release_work(self):
        import os
        from unittest.mock import patch
        import prepare_mobile as mobile
        with patch.dict(os.environ, RECOVER_IOS='true', RESUME_TAG='', GITHUB_REF='refs/heads/main'), patch.object(mobile.subprocess, 'run') as run:
            with self.assertRaisesRegex(ValueError, 'recovery.*tag'):
                mobile.main()
            run.assert_not_called()

    def test_dev_forwards_recovery_and_testflight_target_to_main(self):
        import os
        from unittest.mock import patch
        import prepare_mobile as mobile
        with patch.dict(os.environ, RECOVER_IOS='true', TESTFLIGHT_ONLY='true', BUMP='patch', RESUME_TAG='mobile-v1.0.2', GITHUB_REF='refs/heads/dev'), patch.object(mobile.subprocess, 'run'), patch.object(mobile, 'tag_identity'), patch.object(mobile, 'read') as read:
            mobile.main()
        args = read.call_args.args
        self.assertIn('recover_ios=true', args)
        self.assertIn('testflight_only=true', args)
        self.assertIn('tag=mobile-v1.0.2', args)

    def test_resume_keeps_tag_build_and_skips_new_release_pr(self):
        import os
        from pathlib import Path
        from tempfile import TemporaryDirectory
        from unittest.mock import patch
        import prepare_mobile as mobile
        with TemporaryDirectory() as folder:
            output = Path(folder) / 'output'
            with patch.dict(os.environ, BUMP='major', RESUME_TAG='mobile-v1.2.3', GITHUB_REF='refs/heads/main', GITHUB_OUTPUT=str(output)), \
                 patch.object(mobile.subprocess, 'run'), \
                 patch.object(mobile, 'tag_identity', return_value=('abc', '1.2.3', 19)), \
                 patch.object(mobile, 'ensure_promoted') as promote, \
                 patch.object(mobile, 'read') as read:
                mobile.main()
            promote.assert_not_called()
            read.assert_not_called()
            self.assertIn('number=19\n', output.read_text())
            self.assertIn('tag=mobile-v1.2.3\n', output.read_text())

    def test_unmerged_source_only_prepares_pr(self):
        import os
        from unittest.mock import patch
        import prepare_mobile as mobile
        with patch.dict(os.environ, BUMP='patch', RESUME_TAG='', GITHUB_REF='refs/heads/dev'), \
             patch.object(mobile.subprocess, 'run'), \
             patch.object(mobile, 'ensure_promoted', return_value=False), \
             patch.object(mobile, 'read') as read:
            mobile.main()
        read.assert_not_called()

    def test_dev_dispatches_to_main_without_reserving_a_version(self):
        import os
        from unittest.mock import patch
        import prepare_mobile as mobile
        with patch.dict(os.environ, BUMP='minor', RESUME_TAG='', GITHUB_REF='refs/heads/dev'), \
             patch.object(mobile.subprocess, 'run'), \
             patch.object(mobile, 'ensure_promoted', return_value=True), \
             patch.object(mobile, 'read') as read:
            mobile.main()
        read.assert_called_once_with('gh', 'workflow', 'run', 'release-mobile.yml', '--ref', 'main', '-f', 'bump=minor')

    def test_main_reserves_annotation_and_passes_computed_identity_to_builds(self):
        import os
        from pathlib import Path
        from tempfile import TemporaryDirectory
        from unittest.mock import patch
        import prepare_mobile as mobile
        calls = []
        def read(*args):
            calls.append(args)
            if args[:2] == ('git', 'rev-parse'): return 'approved-sha'
            if args[:2] == ('git', 'tag'): return 'v99.0.0\nmobile-v1.2.3\nmobile-v2.0.0-beta'
            if args[:2] == ('git', 'show'): return 'version: 1.0.0+1\n'
            if args[:2] == ('gh', 'api'):
                if args[4].endswith('/git/tags'):
                    return json.dumps({'sha': 'annotation-sha'})
                return '{}'
            self.fail(f'Unexpected call {args}')
        with TemporaryDirectory() as folder:
            output = Path(folder) / 'output'
            with patch.dict(os.environ, BUMP='minor', RESUME_TAG='', GITHUB_REF='refs/heads/main', GITHUB_OUTPUT=str(output)), \
                 patch.object(mobile.subprocess, 'run'), \
                 patch.object(mobile, 'tag_identity', return_value=('old-sha', '1.2.3', 8)), \
                 patch.object(mobile, 'ensure_promoted', return_value=True), \
                 patch.object(mobile, 'read', side_effect=read):
                mobile.main()
            self.assertIn('version=1.3.0\nnumber=9\n', output.read_text())
        annotation = next(call for call in calls if '/git/tags' in ' '.join(call))
        self.assertIn('message={"schema": 1, "version": "1.3.0", "buildNumber": 9}', annotation)
        ref = next(call for call in calls if '/git/refs' in ' '.join(call))
        self.assertIn('sha=annotation-sha', ref)

    def test_lightweight_legacy_tag_must_match_its_pubspec(self):
        from unittest.mock import patch
        import prepare_mobile as mobile
        with patch.object(mobile.subprocess, 'run'), patch.object(mobile, 'read', side_effect=['sha', 'commit', 'version: 1.2.3+19\n']):
            self.assertEqual(mobile.tag_identity('mobile-v1.2.3'), ('sha', '1.2.3', 19))
        with patch.object(mobile.subprocess, 'run'), patch.object(mobile, 'read', side_effect=['sha', 'commit', 'version: 1.0.0+1\n']), self.assertRaises(ValueError):
            mobile.tag_identity('mobile-v1.2.3')

    def test_same_commit_requires_recovery_instead_of_second_reservation(self):
        import os
        from unittest.mock import patch
        import prepare_mobile as mobile
        with patch.dict(os.environ, BUMP='patch', RESUME_TAG='', GITHUB_REF='refs/heads/main'), \
             patch.object(mobile.subprocess, 'run'), \
             patch.object(mobile, 'ensure_promoted', return_value=True), \
             patch.object(mobile, 'read', side_effect=['same-sha', 'mobile-v1.2.3']) as read, \
             patch.object(mobile, 'tag_identity', return_value=('same-sha', '1.2.3', 19)), \
             self.assertRaisesRegex(ValueError, 'resume'):
            mobile.main()
        self.assertEqual(read.call_count, 2)
