"""Harmony keeps shared identity, isolated groups and truthful approval facts."""
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
from zipfile import ZipFile

from harmony import app_identity, configuration, publish_original, APP_ID, BUNDLE, main
from harmony_reconcile import availability, reconcile, import_initial_release
from state import baselines
from model import ReleaseError
from controller import candidate_path
from catalog import build_catalog, candidate_content_digest
from rc import reusable


class HarmonyTests(unittest.TestCase):
    def test_history_backfill_does_not_become_a_new_publishing_baseline(self):
        remote = Mock()
        archive = {'id': 1, 'sha': 'a' * 40, 'payload': {
            'candidateId': 'mobile-1.0.18-36', 'tag': 'mobile-v1.0.18'}}
        remote.pages.side_effect = lambda endpoint: [archive] if endpoint.startswith('deployments?') else [{'state': 'success'}]
        self.assertEqual(baselines(remote, ['harmony-apptest'], []), {
            'harmony-apptest': {'tag': None, 'sourceSha': None}})
        formal = {'id': 2, 'sha': 'b' * 40, 'payload': {
            'candidateId': 'mobile-1.0.19-108', 'tag': 'mobile-v1.0.19'}}
        remote.pages.side_effect = lambda endpoint: [archive, formal] if endpoint.startswith('deployments?') else [{'state': 'success'}]
        self.assertEqual(baselines(remote, ['harmony-apptest'], [])['harmony-apptest']['sourceSha'], 'b' * 40)

    def test_newer_dev_build_blocks_old_rc_before_draft_or_upload(self):
        client = Mock()
        client.versions.return_value = [{'packageList': [{'versionCode': 109}]}]
        env = {'HARMONY_APP_DIRECTORY': 'unused', 'MOBILE_VERSION': '1.0.19', 'MOBILE_BUILD_NUMBER': '108',
               'HARMONY_TEST_GROUP_ID': 'a' * 32, 'HARMONY_SERVICE_ACCOUNT_JSON': '{}', 'AUTOMATIC_DEV': 'false'}
        with patch.dict(os.environ, env), patch('sys.argv', ['harmony.py', 'publish']), patch('harmony.Client', return_value=client), patch('harmony.app_identity', return_value=(Path('original.app'), 'b' * 64)), patch('harmony.publish_original') as submit:
            with self.assertRaisesRegex(ReleaseError, 'fresh RC'):
                main()
            submit.assert_not_called()
            client.upload.assert_not_called()
            client.request.assert_not_called()

    def test_original_digest_version_and_bundle_are_required(self):
        with tempfile.TemporaryDirectory() as temp:
            app = Path(temp) / 'original.app'
            with ZipFile(app, 'w') as archive:
                archive.writestr('pack.info', json.dumps({'summary': {'app': {
                    'bundleName': BUNDLE, 'version': {'name': '1.0.18', 'code': 36}}}}))
            sha = hashlib.sha256(app.read_bytes()).hexdigest()
            app.with_suffix('.sha256').write_text(f'{sha}  {app.name}\n')
            self.assertEqual(app_identity(temp, '1.0.18', '36', sha), (app, sha))
            for version, number, expected in [('1.0.19', '36', sha), ('1.0.18', '37', sha), ('1.0.18', '36', '0' * 64)]:
                with self.assertRaises(ReleaseError):
                    app_identity(temp, version, number, expected)

    def test_configuration_preserves_description_and_binds_exact_group(self):
        detail = {'languages': [{'lang': 'zh-CN', 'appDesc': 'YourTJ 校园社区', 'appName': 'YourTJ'}],
                  'appInfo': {'testUserPassword': 'must never enter a public receipt'}}
        config = configuration(detail, 'a' * 32, '改进桌面小组件。', 100)
        self.assertEqual(config['languagesList'], [{'language': 'zh-CN', 'appDesc': 'YourTJ 校园社区', 'newFeatures': '改进桌面小组件。'}])
        self.assertEqual(config['openTestInfo']['startTime'], 100000)
        self.assertEqual(config['openTestInfo']['testTaskInfo']['groupInfos'], [{'groupId': 'a' * 32, 'bind': 1}])
        self.assertNotIn('appInfo', config)
        with self.assertRaises(ReleaseError):
            configuration(detail, 'a' * 32, '字' * 1025, 100)
        with self.assertRaises(ReleaseError):
            configuration({'languages': [{'lang': 'zh-CN', 'appDesc': '同济大学'}]}, 'a' * 32, '更新', 100)

    def test_dev_group_share_link_stays_disabled(self):
        config = configuration({'languages': [{'lang': 'zh-CN', 'appDesc': 'YourTJ'}]}, 'a' * 32, '更新', 100, share=False)
        self.assertEqual(config['openTestInfo']['testTaskInfo']['needShareLink'], 0)

    def test_archive_cannot_trigger_automatic_publication(self):
        self.assertEqual(candidate_path('mobile-1.0.18-36'), 'releases/history/mobile-1.0.18-36')
        self.assertEqual(candidate_path('mobile-1.0.19-108'), 'releases/requests/mobile-1.0.19-108')

    def test_initial_archive_never_imports_an_unknown_or_mismatched_package(self):
        remote = Mock()
        folder = Path(__file__).resolve().parents[2] / 'releases/history/mobile-1.0.18-36'
        manifest = json.loads((folder / 'manifest.json').read_text())
        details = json.loads((folder / 'evidence.json').read_text())['evidence'][0]['receipt']
        version = {'versionId': details['versionId'], 'testType': 3, 'releaseType': 6, 'state': 4,
                   'packageList': [{'packageId': details['packageId'], 'versionCode': 36, 'versionName': '1.0.18'}]}
        with patch('harmony_reconcile.latest_receipt', return_value=None), patch('harmony_reconcile.load_candidate', return_value=(manifest, folder)), patch('harmony_reconcile.record') as record:
            import_initial_release(remote, {details['versionId']: version})
            record.assert_not_called()
            version['state'] = 0
            import_initial_release(remote, {details['versionId']: version})
            self.assertEqual(record.call_args.args[4], 'success')
            self.assertEqual(record.call_args.args[5]['appSha256'], details['appSha256'])
            version['packageList'][0]['packageId'] = 'other'
            with self.assertRaises(ReleaseError):
                import_initial_release(remote, {details['versionId']: version})

    def test_verified_initial_archive_renders_existing_history_schema(self):
        folder = Path(__file__).resolve().parents[2] / 'releases/history/mobile-1.0.18-36'
        manifest = json.loads((folder / 'manifest.json').read_text())
        details = json.loads((folder / 'evidence.json').read_text())['evidence'][0]['receipt']
        payload = {'candidateId': manifest['candidateId'], 'channel': 'harmony-apptest', 'tag': manifest['tag'],
                   'binding': {'sourceSha': manifest['sourceSha'], 'contentDigest': candidate_content_digest(folder)},
                   'details': details | {'availability': 'APPROVED'}}
        remote = Mock()
        remote.pages.side_effect = lambda endpoint: [{'id': 1, 'sha': manifest['sourceSha'], 'payload': payload}] if endpoint == 'deployments?environment=release-harmony-apptest' else [{'state': 'success'}] if endpoint == 'deployments/1/statuses' else []
        with patch('catalog.candidate_paths', return_value={manifest['candidateId']: str(folder)}), patch('catalog.load_published_candidate', return_value=(manifest, folder)), patch('catalog.public_app_store_version', return_value=None):
            result = build_catalog(remote, channels={'harmony-apptest'})
        self.assertEqual(result['releases'][0]['buildNumber'], 36)
        self.assertEqual(result['releases'][0]['channels'], ['harmony-apptest'])
        self.assertEqual(result['releases'][0]['highlights'][0]['platforms'], ['harmony-apptest'])
        self.assertEqual(result['historyCoverage']['byChannel']['harmony-apptest']['completeFromBuild'], 36)
        with patch('catalog.candidate_paths', return_value={}), patch('catalog.public_app_store_version', return_value=None):
            self.assertEqual(build_catalog(remote, channels={'android'})['releases'], [])

    def test_ambiguous_creation_never_repeats_on_retry(self):
        client = Mock()
        client.request.side_effect = ValueError('lost response')
        with tempfile.TemporaryDirectory() as temp:
            receipt = Path(temp) / 'receipt.json'
            with self.assertRaises(ValueError):
                publish_original(client, Path('unused'), receipt, 'b' * 40, 'c' * 64, 'a' * 32, '更新')
            self.assertEqual(json.loads(receipt.read_text())['phase'], 'creating')
            with self.assertRaises(ReleaseError):
                publish_original(client, Path('unused'), receipt, 'b' * 40, 'c' * 64, 'a' * 32, '更新')
            client.request.assert_called_once()

    def test_rc_requires_signed_original_digest(self):
        with self.assertRaises(ReleaseError):
            reusable({'platforms': {'harmonyos': {}}}, 'harmony-apptest')
        result = {'appSha256': 'a' * 64}
        self.assertEqual(reusable({'platforms': {'harmonyos': result}}, 'harmony-apptest'), result)

    def test_reviewing_is_not_approved_and_package_must_match(self):
        row = {'versionId': '123', 'testType': 3, 'releaseType': 6, 'state': 4,
               'packageList': [{'packageId': '456'}]}
        details = {'versionId': '123', 'packageId': '456'}
        self.assertEqual(availability(row, details), 'in_progress')
        self.assertEqual(availability(row | {'state': 0}, details), 'success')
        for state in [1, 2, 10, 11, 13]:
            self.assertEqual(availability(row | {'state': state}, details), 'failure')
        with self.assertRaises(ReleaseError):
            availability(row, details | {'packageId': '999'})

    def test_approval_keeps_original_review_binding_and_source(self):
        remote, client = Mock(), Mock()
        details = {'appId': APP_ID, 'versionId': '123', 'packageId': '456', 'appSha256': 'c' * 64,
                   'sourceSha': 'b' * 40, 'availability': 'SUBMITTED'}
        payload = {'candidateId': 'mobile-1.0.18-36', 'binding': {'contentDigest': 'unchanged'}, 'details': details}
        deployment = {'id': 1, 'sha': 'b' * 40, 'payload': payload}
        remote.pages.side_effect = lambda path: [deployment] if path == 'deployments?environment=release-harmony-apptest' else [{'state': 'in_progress'}] if path == 'deployments/1/statuses' else []
        remote.api.return_value = {'id': 2}
        client.versions.return_value = [{'versionId': '123', 'testType': 3, 'releaseType': 6, 'state': 0,
                                        'packageList': [{'packageId': '456'}]}]
        reconcile(remote, client)
        body = remote.api.call_args_list[0].kwargs['data']
        self.assertEqual(body['payload']['binding'], payload['binding'])
        self.assertEqual(body['ref'], deployment['sha'])
        self.assertEqual(body['payload']['details']['availability'], 'APPROVED')
        self.assertEqual(remote.api.call_args.kwargs['data']['state'], 'success')


if __name__ == '__main__': unittest.main()
