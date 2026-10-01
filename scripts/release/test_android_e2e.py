"""Exercise the real Android publisher across partial upload, retries and immutable identity."""
import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'mobile-release'))
import publish_android as publisher


class AndroidPublishTest(unittest.TestCase):
    def test_partial_upload_resumes_original_apks_and_rejects_replacement(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            sdk = root / 'sdk/build-tools/37.0.0'; sdk.mkdir(parents=True)
            source = root / 'apps/mobile/packages/forum_app/build/app/outputs/flutter-apk'; source.mkdir(parents=True)
            certificate = 'a' * 64
            (root / 'apps/mobile/release-config.json').write_text(json.dumps({'androidCertificateSha256': certificate}))
            for abi in publisher.ABIS:
                (source / f'app-{abi}-release.apk').write_bytes(('original signed fixture ' + abi).encode())
            note = root / 'android.zh-CN.md'; note.write_text('修复 Android 图片选择后显示异常。')
            release, writes, interrupt = None, [], True
            def gh(*args, **kwargs):
                nonlocal release, interrupt
                if args[:2] == ('release','view'):
                    return json.dumps({'databaseId': 1}) if release else None
                if args[:2] == ('release','create'):
                    release = {'id': 1, 'draft': True, 'body': Path(args[args.index('--notes-file')+1]).read_text(), 'assets': []}
                    writes.append('create'); return ''
                if args[0] == 'api': return json.dumps(release)
                if args[:2] == ('release','upload'):
                    file = Path(args[3])
                    if interrupt and len(release['assets']) == 1:
                        interrupt = False
                        raise RuntimeError('simulated interrupted transport')
                    release['assets'].append({'name':file.name,'digest':'sha256:'+hashlib.sha256(file.read_bytes()).hexdigest()})
                    writes.append(file.name); return ''
                if args[:2] == ('release','edit'):
                    release['draft'] = False; return ''
                raise AssertionError(args)
            def apk_tool(args, **kwargs):
                if args[0].endswith('apksigner'):
                    return 'Signer #1 certificate SHA-256 digest: '+certificate
                abi = Path(args[-1]).name.removeprefix('app-').removesuffix('-release.apk')
                code = publisher.android_version_code('16', abi)
                return f"package: name='tj.yourtj.forum_app' versionCode='{code}' versionName='1.0.15'\nnative-code: '{abi}'"
            env = {'MOBILE_VERSION':'1.0.15','MOBILE_BUILD_NUMBER':'16','RELEASE_TAG':'mobile-v1.0.15',
                   'ANDROID_HOME':str(root/'sdk'),'ANDROID_NOTES_PATH':str(note)}
            with patch.object(publisher, 'ROOT', root), patch.object(publisher, 'gh', side_effect=gh), patch.object(publisher.subprocess, 'check_output', side_effect=apk_tool), patch.dict(os.environ, env), patch.object(sys, 'argv', ['publish_android.py']):
                with self.assertRaisesRegex(RuntimeError, 'interrupted'): publisher.main()
                self.assertTrue(release['draft'])
                publisher.main()
                self.assertFalse(release['draft'])
                self.assertEqual(len(release['assets']), 4)
                self.assertEqual(release['body'].strip(), note.read_text())
                previous = list(writes); publisher.main(); self.assertEqual(writes, previous)
                (source / 'app-arm64-v8a-release.apk').write_bytes(b'different signed bytes')
                with self.assertRaisesRegex(ValueError, 'different bytes'): publisher.main()
                self.assertEqual(writes, previous)


if __name__ == '__main__': unittest.main()
