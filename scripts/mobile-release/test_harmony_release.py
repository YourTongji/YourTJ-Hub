from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from unittest.mock import Mock
from zipfile import ZipFile
import json
import base64
import subprocess
import hashlib
import io

import build_harmony
import download_harmony_cli
import harmony_api
import publish_harmony


class HarmonyReleaseTests(unittest.TestCase):
    def test_app_test_build_is_production_release_without_preview(self):
        class BuildReached(Exception):
            pass

        with tempfile.TemporaryDirectory() as temp, \
                patch.object(build_harmony, 'ROOT', Path(temp)), \
                patch.dict('os.environ', {'MOBILE_VERSION': '1.0.18', 'MOBILE_BUILD_NUMBER': '36',
                                     'SOURCE_SHA': 'a' * 40, 'HARMONY_PROFILE_BASE64': '',
                                     'DEVECO_SDK_HOME': '/unused/sdk'}), \
                patch.object(build_harmony, 'validate_profile'), \
                patch.object(build_harmony, 'run', side_effect=BuildReached) as run:
            with self.assertRaises(BuildReached):
                build_harmony.main()
        command = run.call_args.args[0]
        self.assertIn('--release', command)
        self.assertIn('--target-platform=ohos-arm64', command)
        self.assertIn('--dart-define=YOURTJ_PREVIEW=false', command)
        self.assertIn('--dart-define=YOURTJ_ENVIRONMENT=production', command)
        self.assertIn('--dart-define=YOURTJ_API_BASE_URL=https://f.yourtj.de', command)

    def test_publisher_bundle_matches_native_application(self):
        self.assertEqual(json.loads((build_harmony.APP / 'ohos/AppScope/app.json5').read_text())['app']['bundleName'],
                         harmony_api.BUNDLE)

    def test_toolchain_checksum_failure_removes_only_its_partial_download(self):
        url = 'https://contentcenter-vali-drcn.dbankcdn.cn/tools/commandline-tools-linux-x64-26.0.0.865.zip?signature=private'
        with tempfile.TemporaryDirectory() as temp:
            destination = Path(temp) / 'cli.zip'
            with patch('urllib.request.build_opener') as opener:
                opener.return_value.open.return_value = io.BytesIO(b'corrupted archive')
                with self.assertRaises(ValueError) as raised:
                    download_harmony_cli.download(url, destination)
                self.assertNotIn('signature=private', str(raised.exception))
                self.assertFalse(destination.exists())
                destination.write_bytes(b'previous artifact')
                opener.return_value.open.return_value = io.BytesIO(b'new archive')
                with self.assertRaises(FileExistsError):
                    download_harmony_cli.download(url, destination)
                self.assertEqual(destination.read_bytes(), b'previous artifact')

    def test_jwt_has_ps256_signature_and_one_hour_expiry(self):
        with tempfile.TemporaryDirectory() as temp:
            key = Path(temp) / 'key.pem'
            subprocess.run(['openssl', 'genpkey', '-algorithm', 'RSA', '-pkeyopt', 'rsa_keygen_bits:2048',
                            '-out', str(key)], check=True, capture_output=True)
            credentials = {'key_id': 'test-key', 'sub_account': 'test-account', 'private_key': key.read_text()}
            token = harmony_api.authorization(credentials, now=1000)
            header, payload, signature = token.split('.')
            decode = lambda value: base64.urlsafe_b64decode(value + '=' * (-len(value) % 4))
            self.assertEqual(json.loads(decode(header)), {'alg': 'PS256', 'typ': 'JWT', 'kid': 'test-key'})
            self.assertEqual(json.loads(decode(payload)),
                             {'aud': harmony_api.AUDIENCE, 'iss': 'test-account', 'iat': 1000, 'exp': 4600})
            public = Path(temp) / 'public.pem'
            subprocess.run(['openssl', 'pkey', '-in', str(key), '-pubout', '-out', str(public)],
                           check=True, capture_output=True)
            sig = Path(temp) / 'signature'
            sig.write_bytes(decode(signature))
            result = subprocess.run(['openssl', 'dgst', '-sha256', '-verify', str(public),
                                     '-signature', str(sig), '-sigopt', 'rsa_padding_mode:pss',
                                     '-sigopt', 'rsa_pss_saltlen:digest'],
                                    input=f'{header}.{payload}'.encode(), capture_output=True)
            self.assertEqual(result.returncode, 0)

    def test_profile_requires_correct_bundle_and_approved_clipboard_acl(self):
        profile = {'type': 'release', 'bundle-info': {'bundle-name': harmony_api.BUNDLE},
                   'acls': {'allowed-acls': []}}
        with patch.object(build_harmony.subprocess, 'run') as command:
            command.return_value = subprocess.CompletedProcess([], 0, json.dumps(profile).encode(), b'')
            with self.assertRaisesRegex(ValueError, 'READ_PASTEBOARD'):
                build_harmony.validate_profile(b'signed profile')
            profile['acls']['allowed-acls'] = ['ohos.permission.READ_PASTEBOARD']
            command.return_value.stdout = json.dumps(profile).encode()
            build_harmony.validate_profile(b'signed profile')
            profile['bundle-info']['bundle-name'] = 'another.app'
            command.return_value.stdout = json.dumps(profile).encode()
            with self.assertRaisesRegex(ValueError, 'YourTJ'):
                build_harmony.validate_profile(b'signed profile')

    def test_invitation_rejects_formal_or_public_test_versions(self):
        client = object.__new__(harmony_api.Client)
        for kind, release in [(None, 1), (4, 6), (3, 1)]:
            client.versions = lambda: [{'versionId': '123', 'testType': kind, 'releaseType': release}]
            with self.assertRaisesRegex(ValueError, 'invitation'):
                client.invitation('123')

    def test_failed_legality_preserves_identity_without_submitting(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            app = directory / 'original.app'
            with ZipFile(app, 'w') as archive:
                archive.writestr('pack.info', json.dumps({'summary': {'app': {'bundleName': harmony_api.BUNDLE}}}))
            digest = hashlib.sha256(app.read_bytes()).hexdigest()
            client = Mock()
            client.invitation.return_value = {'state': 7}
            client.upload.return_value = ('456', digest)
            client.request.return_value = {'pkgStateList': [{'pkgId': '456', 'successStatus': 2}]}
            receipt = directory / 'receipt.json'
            with patch.dict('os.environ', {'GITHUB_RUN_ATTEMPT': '1'}):
                with self.assertRaisesRegex(ValueError, 'legality'):
                    publish_harmony.publish(client, app, '123', receipt, 'a' * 40)
                with self.assertRaisesRegex(ValueError, 'legality'):
                    publish_harmony.publish(client, app, '123', receipt, 'a' * 40)
            client.upload.assert_called_once()
            self.assertTrue(all(call.args[0] == 'GET' for call in client.request.call_args_list))
            self.assertEqual(json.loads(receipt.read_text())['packageId'], '456')

    def test_successful_publish_reuses_receipt_and_only_calls_test_submission(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            app = directory / 'original.app'
            with ZipFile(app, 'w') as archive:
                archive.writestr('pack.info', json.dumps({'summary': {'app': {'bundleName': harmony_api.BUNDLE}}}))
            client = Mock()
            client.invitation.return_value = {'state': 7, 'packageList': [{'packageId': '456'}]}
            client.upload.return_value = ('456', hashlib.sha256(app.read_bytes()).hexdigest())
            client.request.return_value = {'pkgStateList': [{'pkgId': '456', 'successStatus': 0}]}
            receipt = directory / 'receipt.json'
            with patch.dict('os.environ', {'GITHUB_RUN_ATTEMPT': '1'}):
                publish_harmony.publish(client, app, '123', receipt, 'a' * 40)
                publish_harmony.publish(client, app, '123', receipt, 'a' * 40)
            self.assertEqual(json.loads(receipt.read_text())['phase'], 'submitted')
            client.upload.assert_called_once()
            posts = [call.args for call in client.request.call_args_list if call.args[0] == 'POST']
            self.assertEqual(posts, [('POST', '/api/publish/v2/test/app/version/submit', {'versionId': '123'})])

    def test_app_container_is_signed_after_packing(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            unsigned = directory / 'unsigned.app'
            hap = directory / 'entry-default.hap'
            hap.write_bytes(b'verified signed HAP')
            info = {'summary': {'app': {'bundleName': build_harmony.BUNDLE}},
                    'packages': [{'name': 'entry-default'}]}
            with ZipFile(unsigned, 'w') as archive:
                archive.writestr('pack.info', json.dumps(info))
                archive.writestr('pac.json', '{}')
            output = directory / 'final.app'
            calls = []

            def command(args):
                if '--out-path' in args:
                    with ZipFile(args[args.index('--out-path') + 1], 'w') as archive:
                        archive.writestr('entry-default.hap', hap.read_bytes())
                else:
                    calls.append('verify')

            def signer(tool, keys, source, target):
                calls.append('sign')
                self.assertEqual(source.suffix, '.app')
                target.write_bytes(source.read_bytes())

            with patch.object(build_harmony, 'run', side_effect=command), \
                    patch.object(build_harmony, 'sign', side_effect=signer):
                build_harmony.package_app(unsigned, hap, directory, directory, output)
            self.assertEqual(calls, ['sign', 'verify'])
            with ZipFile(output) as archive:
                self.assertEqual(archive.read('entry-default.hap'), hap.read_bytes())

    def test_toolchain_rejects_windows_and_foreign_hosts_before_network(self):
        with patch('urllib.request.urlopen') as network:
            for url in ['https://example.com/commandline-tools-linux-x64-26.0.0.865.zip',
                        'https://contentcenter-vali-drcn.dbankcdn.cn/commandline-tools-windows-x64-26.0.0.865.zip']:
                with self.assertRaises(ValueError):
                    download_harmony_cli.download(url, Path('unused.zip'))
            network.assert_not_called()


if __name__ == '__main__':
    unittest.main()
