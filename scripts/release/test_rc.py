"""RC uses production bytes and promotes immutable build records, not a second build."""
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
import zipfile
import base64

from model import REPOSITORY, ReleaseError
from rc import artifact_name, read_record, reusable, validate_identity
import rc
import workflow
from cli import execute, parser
from test_model import candidate


def identity():
    return {'candidateId': 'mobile-1.0.19-42', 'sourceSha': 'a' * 40,
            'product': 'mobile', 'version': '1.0.19', 'buildNumber': 42,
            'environment': 'production', 'preview': False,
            'run': '100', 'controllerSha': 'b' * 40, 'platforms': ['android', 'ios']}


class RCTests(unittest.TestCase):
    def test_rc_retry_with_deleted_original_never_rebuilds_a_recorded_identity(self):
        original = identity()
        remote = Mock()
        remote.pages.return_value = [{'name': 'rc-result-android-1'}]
        previous = {'identity': original, 'success': True, 'details': {'assets': {'a.apk': 'd' * 64}}}
        with tempfile.TemporaryDirectory() as temp, \
             patch.dict(os.environ, CHANNEL='android', GITHUB_OUTPUT=str(Path(temp) / 'output')), \
             patch('rc.prepare_build', return_value=({}, original, {})), \
             patch('rc.read_json_artifact', return_value=previous):
            with self.assertRaisesRegex(ReleaseError, 'Original RC artifact'):
                rc.handle('start', remote, original['candidateId'], Path(temp))

    def test_deployment_accepts_bound_rc_digest_but_rejects_unversioned_images(self):
        import subprocess
        root = Path(__file__).resolve().parents[2]
        text = (root / '.github/workflows/deploy-main.yml').read_text()
        program = text.split("python3 - <<'PY'\n", 1)[1].split('\n          PY', 1)[0]
        import textwrap
        program = textwrap.dedent(program)
        for tag in ['main-v0.0.54', 'main-rc-web-0.0.54-100-200']:
            result = subprocess.run(['python3', '-c', program], env={**os.environ, 'IMAGE_REF': tag + '@sha256:' + 'a' * 64},
                                    capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
        for image in ['main-v0.0.54', 'main-rc-web-0.0.54-100', 'dev@sha256:' + 'a' * 64, 'main-v0.0.54@sha256:no']:
            result = subprocess.run(['python3', '-c', program], env={**os.environ, 'IMAGE_REF': image}, capture_output=True)
            self.assertNotEqual(result.returncode, 0)

    def test_prepare_reserves_resolved_main_sha_instead_of_symbolic_main(self):
        manifest = candidate() | {'schemaVersion': 3, 'rcRun': None}
        with tempfile.TemporaryDirectory() as temp, \
             patch.dict(os.environ, SCOPE='mobile', BUMP='patch', SOURCE='main', EXISTING_RELEASE='',
                        GITHUB_OUTPUT=str(Path(temp) / 'output')), \
             patch('sys.argv', ['workflow.py', 'prepare', '--folder', temp]), patch('workflow.GitHub'), \
             patch('workflow.plan', return_value=manifest), patch('workflow.prepare', return_value={}), \
             patch('workflow.write_json'), patch('workflow.reserve_number') as reserve:
            workflow.main()
        self.assertEqual(reserve.call_args.args[1], manifest['sourceSha'])

    def test_resolve_same_selected_run_can_resume_but_different_run_cannot(self):
        remote = Mock(repository=REPOSITORY)
        pr = {'number': 1, 'state': 'open', 'head': {'repo': {'full_name': REPOSITORY},
              'ref': 'codex/release/mobile-1.0.19-42', 'sha': 'd' * 40}, 'base': {'ref': 'main'}}
        remote.pages.return_value = [pr]
        remote.api.side_effect = lambda path, **kw: pr if path == 'pulls/1' else None
        manifest = identity() | {'schemaVersion': 3, 'operation': 'release', 'rcRun': '100',
                                 'tag': 'mobile-v1.0.19', 'channels': ['android', 'ios-app-store']}
        with tempfile.TemporaryDirectory() as temp, patch('rc.git'), \
             patch('controller.load_candidate', return_value=(manifest, Path(temp))), \
             patch('build_identity.verify_production_number'), patch('state.latest_receipt', return_value=None):
            self.assertEqual(rc.resolve(remote, manifest['candidateId'], Path(temp), linked_run='100')[0], manifest)
            with self.assertRaises(ReleaseError):
                rc.resolve(remote, manifest['candidateId'], Path(temp), linked_run='101')
            with patch('state.latest_receipt', return_value={'started': True}), self.assertRaises(ReleaseError):
                rc.resolve(remote, manifest['candidateId'], Path(temp), linked_run='100')

    def test_existing_ios_promotion_never_rebuilds(self):
        manifest = candidate() | {'schemaVersion': 3, 'rcRun': None, 'operation': 'promote-ios',
                                  'channels': ['ios-app-store'],
                                  'existingRelease': {'candidateId': 'original', 'buildId': 'apple-15'}}
        with tempfile.TemporaryDirectory() as temp:
            output = Path(temp) / 'output'
            with patch.dict(os.environ, CANDIDATE=manifest['candidateId'], CHANNEL='ios-app-store',
                            GITHUB_RUN_ID='200', GITHUB_OUTPUT=str(output), RECOVER='false'), \
                 patch('sys.argv', ['workflow.py', 'start']), patch('workflow.GitHub'), \
                 patch('workflow.authorize', return_value=(manifest, {'contentDigest': 'notes'})), \
                 patch('build_identity.verify_production_number'), patch('workflow.latest_receipt', return_value=None), \
                 patch('workflow.record'):
                workflow.main()
            self.assertIn('build_required=false\n', output.read_text())
            self.assertIn('existing_build_id=apple-15\n', output.read_text())

    def test_collect_requires_complete_success_and_uses_latest_platform_attempt(self):
        original = identity()
        with tempfile.TemporaryDirectory() as temp, patch('rc.Path', side_effect=lambda p: Path(temp) / p), \
             patch('rc.GitHub'), patch('rc.read_json_artifact', return_value=original), \
             patch('sys.argv', ['rc.py', 'collect']), \
             patch.dict(os.environ, GITHUB_RUN_ID='100', GITHUB_STEP_SUMMARY=str(Path(temp) / 'summary')):
            folder = Path(temp) / 'rc-results'
            folder.mkdir()
            android = {'identity': original, 'platform': 'android', 'attempt': 1, 'success': True,
                       'details': {'buildRunId': '100', 'artifact': 'android-1.0.19-42', 'assets': {'a.apk': 'd' * 64}}}
            (folder / 'android-1.json').write_text(json.dumps(android))
            with self.assertRaises(ReleaseError):
                rc.main()
            ios = {'identity': original, 'platform': 'ios', 'attempt': 1, 'success': False,
                   'details': {'buildRunId': '100', 'artifact': 'ios-1.0.19-42', 'buildId': 'apple-42'}}
            (folder / 'ios-1.json').write_text(json.dumps(ios))
            with self.assertRaises(ReleaseError):
                rc.main()
            (folder / 'ios-2.json').write_text(json.dumps(ios | {'attempt': 2, 'success': True}))
            rc.main()
            record = json.loads((Path(temp) / 'rc-record.json').read_text())
            self.assertEqual(record['platforms']['ios']['buildId'], 'apple-42')
            (folder / 'ios-3.json').write_text(json.dumps(ios | {'attempt': 3, 'success': False}))
            with self.assertRaises(ReleaseError):
                rc.main()

    def test_attach_preserves_notes_manifest_and_uses_blob_compare_and_swap(self):
        original = identity()
        manifest = {k: original[k] for k in rc.IDENTITY_KEYS} | {'rcRun': None, 'notes': {'android': 'android.zh-CN.md'}}
        remote = Mock()
        remote.api.return_value = {'sha': 'blob-sha', 'content': base64.b64encode(json.dumps(manifest).encode()).decode()}
        with tempfile.TemporaryDirectory() as temp, patch('rc.Path', side_effect=lambda p: Path(temp) / p), \
             patch('rc.GitHub', return_value=remote), patch('rc.trusted_run'), \
             patch('rc.resolve', return_value=(manifest, Path(temp))), \
             patch('sys.argv', ['rc.py', 'attach']), patch.dict(os.environ, GITHUB_RUN_ID='100'):
            (Path(temp) / 'rc-record.json').write_text(json.dumps({'identity': original}))
            rc.main()
            payload = remote.api.call_args.kwargs['data']
            self.assertEqual(payload['sha'], 'blob-sha')
            self.assertEqual(json.loads(base64.b64decode(payload['content'])), manifest | {'rcRun': '100'})
            remote.api.reset_mock()
            remote.api.return_value = {'sha': 'other', 'content': base64.b64encode(json.dumps(manifest | {'rcRun': '99'}).encode()).decode()}
            with self.assertRaises(ReleaseError):
                rc.main()
            self.assertEqual(remote.api.call_count, 1)

    def test_replacement_rc_can_keep_unpublished_version_with_a_new_build_number(self):
        from controller import plan
        remote = Mock()
        remote.pages.return_value = []
        tags = [{'tag': 'mobile-v1.0.18', 'buildNumber': 40},
                {'tag': 'mobile-build-42', 'buildNumber': 42}]
        with patch('controller.open_request', return_value=None), patch('controller.source_sha', return_value='a' * 40), \
             patch('controller.reservations', return_value=tags), patch('controller.git', return_value='version: 1.0.18+40'), \
             patch('controller.baselines', return_value={}):
            result = plan('mobile', 'patch', 'main', 'testflight', remote, version='1.0.19')
            self.assertEqual((result['version'], result['buildNumber']), ('1.0.19', 43))
            self.assertEqual(result['schemaVersion'], 4)
            self.assertIsNone(result['rcRun'])
            for version in ['1.0.18', '1.0.17']:
                with self.assertRaises(ReleaseError):
                    plan('mobile', 'patch', 'main', 'testflight', remote, version=version)
        with patch('controller.open_request', return_value=None), patch('controller.source_sha', return_value='a' * 40), \
             patch('controller.reservations', return_value=[{'tag': 'v0.0.53', 'buildNumber': None}]), \
             patch('controller.baselines', return_value={}), patch.dict(os.environ, GITHUB_RUN_ID='100'):
            result = plan('web', 'patch', 'main', 'testflight', remote, version='0.0.54')
            self.assertEqual(result['candidateId'], 'web-0.0.54-100')

    def test_cli_rc_is_explicit_and_dry_by_default(self):
        remote = Mock()
        args = parser().parse_args(['rc', '--candidate', 'mobile-1.0.19-42'])
        result = execute(args, remote)
        self.assertEqual(result['workflow'], 'release-rc.yml')
        self.assertEqual(result['inputs'], {'candidate': 'mobile-1.0.19-42'})
        remote.dispatch.assert_not_called()
        args.apply = True
        execute(args, remote)
        remote.dispatch.assert_called_once_with('release-rc.yml', result['inputs'])

    def test_new_release_start_skips_cycle_qa_and_reuses_the_rc_build_run(self):
        base = candidate() | {'schemaVersion': 3, 'rcRun': '100'}
        record = {'identity': identity(), 'platforms': {
            'android': {'buildRunId': '100', 'artifact': 'android-1.0.19-42', 'assets': {'a.apk': 'd' * 64}},
            'ios': {'buildRunId': '100', 'artifact': 'ios-1.0.19-42', 'buildId': 'apple-42'},
            'web': {'buildRunId': '100', 'artifact': 'web-v1.0.19', 'image': 'main-rc@sha256:' + 'd' * 64,
                    'binarySha256': 'e' * 64}}}
        for channel in ['android', 'ios-testflight', 'web']:
            manifest = base | {'channels': [channel]}
            with self.subTest(channel=channel), tempfile.TemporaryDirectory() as temp:
                output = Path(temp) / 'output'
                env = {'CANDIDATE': base['candidateId'], 'CHANNEL': channel, 'GITHUB_RUN_ID': '200',
                       'GITHUB_OUTPUT': str(output), 'RECOVER': 'false'}
                with patch.dict(os.environ, env), patch('sys.argv', ['workflow.py', 'start']), \
                     patch('workflow.GitHub'), patch('workflow.authorize', return_value=(manifest, {'contentDigest': 'notes'})), \
                     patch('workflow.production_gate', side_effect=AssertionError('No dev/production QA gate')), \
                     patch('build_identity.verify_production_number'), patch('workflow.latest_receipt', return_value=None), \
                     patch('workflow.baselines', return_value={channel: manifest['baselines'].get(channel, {})}), \
                     patch('workflow.reservations', return_value=[]), patch('rc.read_record', return_value=record), \
                     patch('workflow.record') as receipt:
                    if channel == 'web': manifest['baselines']['web'] = {}
                    workflow.main()
                    self.assertIn('build_required=false\n', output.read_text())
                    self.assertIn('build_run_id=100\n', output.read_text())
                    self.assertIn('reuse_rc=true\n', output.read_text())
                    self.assertEqual(receipt.call_args.args[-1]['buildRunId'], '100')
                    if channel == 'ios-testflight':
                        self.assertIn('existing_build_id=apple-42', output.read_text())

    def test_direct_schema_three_release_builds_without_rc_or_cycle(self):
        manifest = candidate() | {'schemaVersion': 3, 'rcRun': None}
        remote = Mock()
        remote.pages.return_value = []
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / 'output'
            with patch.dict(os.environ, CANDIDATE=manifest['candidateId'], CHANNEL='android',
                            GITHUB_RUN_ID='200', GITHUB_OUTPUT=str(path), RECOVER='false'), \
                 patch('sys.argv', ['workflow.py', 'start']), patch('workflow.GitHub', return_value=remote), \
                 patch('workflow.authorize', return_value=(manifest, {'contentDigest': 'notes'})), \
                 patch('workflow.production_gate', side_effect=AssertionError('No cycle')), \
                 patch('build_identity.verify_production_number'), patch('workflow.latest_receipt', return_value=None), \
                 patch('workflow.baselines', return_value=manifest['baselines']), \
                 patch('workflow.reservations', return_value=[]), patch('workflow.record'), \
                 patch('rc.read_record', side_effect=AssertionError('No RC lookup for direct release')):
                workflow.main()
                self.assertIn('build_required=true\n', path.read_text())
                self.assertIn('build_run_id=200\n', path.read_text())
                remote.pages.return_value = [{'ref': 'refs/tags/rc-' + manifest['candidateId'] + '-100'}]
                with self.assertRaisesRegex(ReleaseError, 'RC already owns this build identity'):
                    workflow.main()

    def test_identity_is_production_and_cannot_adopt_dev_or_other_builds(self):
        original = identity()
        validate_identity(original, original)
        for change in [{'sourceSha': 'c' * 40}, {'version': '1.0.20'}, {'buildNumber': 43},
                       {'environment': 'dev'}, {'preview': True}, {'candidateId': 'mobile-1.0.19-43'}]:
            with self.subTest(change=change), self.assertRaises(ReleaseError):
                validate_identity(original | change, original)

    def test_promotion_selects_only_explicit_platform_artifact(self):
        row = {'identity': identity(), 'platforms': {
            'android': {'buildRunId': '100', 'artifact': 'android-1.0.19-42', 'assets': {'a.apk': 'd' * 64}},
            'ios': {'buildRunId': '100', 'artifact': 'ios-1.0.19-42', 'buildId': 'apple-42'}}}
        self.assertEqual(reusable(row, 'ios-app-store')['buildId'], 'apple-42')
        self.assertEqual(reusable(row, 'android')['buildRunId'], '100')
        self.assertEqual(artifact_name(identity(), 'android'), 'android-1.0.19-42')
        with self.assertRaises(ReleaseError):
            reusable(row, 'web')

    def test_record_requires_successful_trusted_main_run_and_immutable_artifact(self):
        remote = Mock(repository=REPOSITORY)
        run = {'path': '.github/workflows/release-rc.yml', 'event': 'workflow_dispatch',
               'head_branch': 'main', 'head_repository': {'full_name': REPOSITORY},
               'head_sha': 'b' * 40, 'status': 'completed', 'conclusion': 'success', 'run_attempt': 1}
        remote.api.return_value = run
        artifact = {'id': 1, 'name': 'rc-record-1', 'expired': False, 'size_in_bytes': 1000}
        remote.pages.return_value = [artifact]
        record = {'identity': identity() | {'platforms': ['ios']}, 'platforms': {'ios': {'buildRunId': '100', 'buildId': 'apple-42', 'artifact': 'ios-1.0.19-42'}}}
        def archive(value):
            stream = io.BytesIO()
            with zipfile.ZipFile(stream, 'w') as zipped:
                zipped.writestr('rc-record.json', json.dumps(value))
            return stream.getvalue()
        with patch('rc.subprocess.check_output', return_value=archive(record)):
            self.assertEqual(read_record(remote, '100', identity()), record)
            for change in [{'head_branch': 'dev'}, {'event': 'pull_request'},
                           {'path': '.github/workflows/other.yml'}, {'status': 'in_progress'}]:
                with self.subTest(change=change), patch.dict(run, change), self.assertRaises(ReleaseError):
                    read_record(remote, '100', identity())
            # Metadata attachment can fail after a complete immutable record was sealed.
            with patch.dict(run, conclusion='failure'):
                self.assertEqual(read_record(remote, '100', identity()), record)
            artifact['expired'] = True
            with self.assertRaises(ReleaseError):
                read_record(remote, '100', identity())
        artifact['expired'] = False
        with patch('rc.subprocess.check_output', return_value=archive(record | {'identity': identity() | {'sourceSha': 'c' * 40}})):
            with self.assertRaises(ReleaseError):
                read_record(remote, '100', identity())
        with patch('rc.subprocess.check_output', return_value=archive(record | {'identity': identity()})):
            with self.assertRaises(ReleaseError):
                read_record(remote, '100', identity())

    def test_release_workflows_do_not_run_source_or_native_test_suites(self):
        root = Path(__file__).resolve().parents[2]
        for name in ['release-prerelease.yml', 'release-publish.yml', 'release-rc.yml']:
            value = (root / '.github/workflows' / name).read_text()
            self.assertNotIn('uses: ./.github/workflows/ci-', value, name)
            self.assertNotIn('unittest discover', value, name)
        preview = (root / '.github/workflows/release-preview-ios.yml').read_text()
        self.assertNotIn('unittest discover', preview)
        ready = (root / '.github/workflows/release-rc.yml').read_text().split('  ready:', 1)[1]
        self.assertIn('fetch-depth: 0', ready, 'RC attachment verifies main ancestry and needs full checkout history')


if __name__ == '__main__':
    unittest.main()
