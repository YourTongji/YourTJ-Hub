import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import publish_android_latest as publisher


def release(version='1.0.11', number=12):
    tag = f'mobile-v{version}'
    assets = []
    for abi, offset in [('arm64-v8a', 2000), ('armeabi-v7a', 1000), ('x86_64', 4000)]:
        data = f'signed fixture {version} {abi}'.encode()
        assets.append({'name': f'YourTJ-{version}+{offset + number}-{abi}.apk',
                       'size': len(data), 'digest': 'sha256:' + hashlib.sha256(data).hexdigest()})
    return {'id': 1, 'tag_name': tag, 'draft': False, 'prerelease': False, 'assets': assets}


class LatestDownloadTest(unittest.TestCase):
    def test_latest_is_mobile_semver_not_server_or_publish_order(self):
        older = release('1.0.9', 10)
        newer = release()
        draft = dict(release('9.0.0'), draft=True)
        prerelease = dict(release('8.0.0'), prerelease=True)
        server = dict(release(), tag_name='v99.0.0')
        alias = dict(release(), tag_name='mobile-latest', prerelease=True)
        self.assertEqual(publisher.select_source([older, newer, draft, prerelease, server, alias]), newer)

    def test_missing_abi_digest_or_inconsistent_build_fails_closed(self):
        for defect in ['abi', 'digest', 'code', 'duplicate']:
            source = release()
            if defect == 'abi': source['assets'].pop()
            if defect == 'digest': source['assets'][0]['digest'] = None
            if defect == 'code': source['assets'][0]['name'] = 'YourTJ-1.0.11+2013-arm64-v8a.apk'
            if defect == 'duplicate': source['assets'].append(source['assets'][0])
            with self.subTest(defect=defect), self.assertRaises(ValueError):
                publisher.source_assets(source)

    def test_stages_verified_bytes_with_stable_names_before_publication(self):
        source = release()
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            def download(*args):
                name = args[args.index('--pattern') + 1]
                abi = next(abi for abi in publisher.ABIS if name.endswith(f'-{abi}.apk'))
                (directory / name).write_bytes(f'signed fixture 1.0.11 {abi}'.encode())
                return ''
            with patch.object(publisher, 'gh', side_effect=download):
                assets = publisher.stage_source(source, directory)
            self.assertEqual([path.name for path, _ in assets],
                             [f'YourTJ-{abi}.apk' for abi in publisher.ABIS]
                             + [publisher.PROBE_NAME, 'SHA256SUMS.txt'])
            staged_probe = directory / publisher.PROBE_NAME
            self.assertEqual(staged_probe.read_bytes(), publisher.PROBE_SOURCE.read_bytes())
            self.assertTrue(128 * 1024 <= staged_probe.stat().st_size <= 256 * 1024)
            self.assertTrue(staged_probe.read_bytes().startswith(publisher.PNG_SIGNATURE))
            checksums = (directory / 'SHA256SUMS.txt').read_text()
            self.assertIn('YourTJ-arm64-v8a.apk', checksums)
            self.assertIn(f'{publisher.file_digest(staged_probe)}  {publisher.PROBE_NAME}', checksums)

    def test_missing_probe_fails_staging(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            with patch.object(publisher, 'source_assets', return_value=[]), \
                 patch.object(publisher, 'PROBE_SOURCE', root / 'missing.png'), \
                 self.assertRaisesRegex(ValueError, 'probe'):
                publisher.stage_source(release(), root / 'stage')
            self.assertFalse((root / 'stage' / 'SHA256SUMS.txt').exists())

    def test_probe_must_be_a_png_within_the_size_range(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            source = root / 'source.png'
            stage = root / 'stage'
            invalid = [publisher.PNG_SIGNATURE, b'not a png' + bytes(128 * 1024)]
            with patch.object(publisher, 'source_assets', return_value=[]), \
                 patch.object(publisher, 'PROBE_SOURCE', source):
                for data in invalid:
                    with self.subTest(size=len(data), signature=data[:8]):
                        source.write_bytes(data)
                        with self.assertRaisesRegex(ValueError, 'probe'):
                            publisher.stage_source(release(), stage)

    def test_failed_staging_never_publishes_alias(self):
        with patch.object(publisher, 'gh', return_value=json.dumps([release()])), \
             patch.object(publisher, 'stage_source', side_effect=ValueError('probe staging failed')), \
             patch.object(publisher, 'publish_alias') as publish, \
             patch.object(sys, 'argv', ['publish_android_latest.py']):
            with self.assertRaisesRegex(ValueError, 'staging failed'):
                publisher.main()
        publish.assert_not_called()

    def test_verify_only_stages_assets_without_publishing_alias(self):
        source = release()
        with patch.object(publisher, 'gh', return_value=json.dumps([source])), \
             patch.object(publisher, 'stage_source') as stage, \
             patch.object(publisher, 'publish_alias') as publish, \
             patch.object(sys, 'argv', ['publish_android_latest.py', '--verify-only']):
            publisher.main()
        stage.assert_called_once()
        publish.assert_not_called()

    def test_corrupt_download_never_mutates_the_alias(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            def download(*args):
                (directory / args[args.index('--pattern') + 1]).write_bytes(b'corrupt')
                return ''
            with patch.object(publisher, 'gh', side_effect=download) as gh:
                with self.assertRaises(ValueError):
                    publisher.stage_source(release(), directory)
            self.assertTrue(all(call.args[:2] == ('release', 'download') for call in gh.call_args_list))

    def test_alias_upload_is_retryable_and_never_changes_global_latest_or_source(self):
        source = release()
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / 'YourTJ-arm64-v8a.apk'
            path.write_bytes(b'new signed bytes')
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
            alias = {'id': 77, 'tag_name': 'mobile-latest', 'prerelease': True, 'assets': [],
                     'body': publisher.source_marker(source)}
            for existing in [[], [{'name': path.name, 'digest': f'sha256:{digest}'}]]:
                alias['assets'] = existing
                verified = {'assets': [{'name': path.name, 'digest': f'sha256:{digest}'}]}
                with patch.object(publisher, 'find_release', return_value=alias), \
                     patch.object(publisher, 'gh', return_value=json.dumps(verified)) as gh:
                    publisher.publish_alias(source, [(path, digest)], Path(temp))
                uploads = [call for call in gh.call_args_list if call.args[:2] == ('release', 'upload')]
                self.assertEqual(len(uploads), 0 if existing else 1)
                for call in uploads:
                    self.assertEqual(call.args[2], 'mobile-latest')
                    self.assertIn('--clobber', call.args)
                edits = [call for call in gh.call_args_list if call.args[:2] == ('release', 'edit')]
                self.assertIn('--latest=false', edits[0].args)
                self.assertIn('--prerelease', edits[0].args)

    def test_older_recovery_cannot_replace_newer_alias(self):
        alias = {'tag_name': 'mobile-latest', 'prerelease': True, 'assets': [],
                 'body': publisher.source_marker(release('1.1.0'))}
        with patch.object(publisher, 'find_release', return_value=alias), \
             patch.object(publisher, 'gh') as gh:
            publisher.publish_alias(release(), [], Path('.'))
        gh.assert_not_called()

    def test_new_alias_starts_as_a_draft_at_the_source_commit(self):
        source = release()
        alias = {'id': 77, 'assets': []}
        with tempfile.TemporaryDirectory() as temp, \
             patch.object(publisher, 'find_release', side_effect=[None, alias]), \
             patch.object(publisher, 'gh', side_effect=['a' * 40, '', '{"assets":[]}', '']) as gh:
            publisher.publish_alias(source, [], Path(temp))
        create = gh.call_args_list[1].args
        self.assertEqual(create[:3], ('release', 'create', 'mobile-latest'))
        for flag in ['--draft', '--prerelease', '--latest=false']:
            self.assertIn(flag, create)
        self.assertEqual(create[create.index('--target') + 1], 'a' * 40)

    def test_alias_digest_failure_never_publishes_or_changes_source_marker(self):
        source = release()
        alias = {'id': 77, 'tag_name': 'mobile-latest', 'prerelease': True,
                 'assets': [], 'body': publisher.source_marker(source)}
        with tempfile.TemporaryDirectory() as temp, \
             patch.object(publisher, 'find_release', return_value=alias), \
             patch.object(publisher, 'gh', return_value='{"assets":[]}') as gh, \
             patch.object(publisher.time, 'sleep'):
            with self.assertRaises(ValueError):
                publisher.publish_alias(source, [(Path(temp) / publisher.PROBE_NAME, 'a' * 64)], Path(temp))
        self.assertFalse(any(call.args[:2] == ('release', 'edit') for call in gh.call_args_list))

    def test_matching_probe_digest_is_not_uploaded_again(self):
        source = release()
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / publisher.PROBE_NAME
            path.write_bytes(b'fixed probe')
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
            existing = [{'name': publisher.PROBE_NAME, 'digest': f'sha256:{digest}'}]
            alias = {'id': 77, 'tag_name': 'mobile-latest', 'prerelease': True,
                     'assets': existing, 'body': publisher.source_marker(source)}
            with patch.object(publisher, 'find_release', return_value=alias), \
                 patch.object(publisher, 'gh', return_value=json.dumps({'assets': existing})) as gh:
                publisher.publish_alias(source, [(path, digest)], Path(temp))
            uploads = [call for call in gh.call_args_list if call.args[:2] == ('release', 'upload')]
            self.assertEqual(uploads, [])

    def test_unknown_alias_owner_marker_is_rejected(self):
        alias = {'tag_name': 'mobile-latest', 'prerelease': True, 'assets': [], 'body': 'unrelated'}
        with patch.object(publisher, 'find_release', return_value=alias), \
             patch.object(publisher, 'gh') as gh, self.assertRaises(ValueError):
            publisher.publish_alias(release(), [], Path('.'))
        gh.assert_not_called()

    def test_public_stable_or_immutable_alias_is_not_overwritten(self):
        for fields in [{'prerelease': False}, {'prerelease': True, 'immutable': True}]:
            alias = {'tag_name': 'mobile-latest', 'draft': False, **fields}
            with patch.object(publisher, 'find_release', return_value=alias), \
                 patch.object(publisher, 'gh') as gh, self.assertRaises(ValueError):
                publisher.publish_alias(release(), [], Path('.'))
            gh.assert_not_called()


if __name__ == '__main__':
    unittest.main()
