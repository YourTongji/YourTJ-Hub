"""Exercise the real selector with mock Xcode command output, without signing."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class IOSToolchainTest(unittest.TestCase):
    def select(self, purpose, build, sdk='27.2'):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for command, variable in [('xcodebuild', 'TEST_XCODE_VERSION'), ('xcrun', 'TEST_SDK_VERSION')]:
                executable = root / command
                executable.write_text('#!/bin/bash\nprintf "%s\\n" "$' + variable + '"\n')
                executable.chmod(0o755)
            output = root / 'github-env'
            result = subprocess.run(['bash', str(Path(__file__).with_name('select_ios_toolchain.sh'))],
                                    env={**os.environ, 'PATH': f'{root}:/usr/bin:/bin',
                                         'DEVELOPER_DIR': str(root), 'GITHUB_ENV': str(output),
                                         'YOURTJ_IOS_BUILD_PURPOSE': purpose,
                                         'TEST_XCODE_VERSION': f'Xcode 27.2\nBuild version {build}',
                                         'TEST_SDK_VERSION': sdk}, capture_output=True, text=True)
            return result.returncode, output.read_text() if output.exists() else ''

    def test_release_build_works_for_both_purposes(self):
        for purpose in ['dev', 'production']:
            with self.subTest(purpose=purpose):
                code, output = self.select(purpose, '27B52')
                self.assertEqual(code, 0)
                self.assertIn('DEVELOPER_DIR=', output)

    def test_reviewed_rc_works_only_for_dev(self):
        self.assertEqual(self.select('dev', '27B48')[0], 0)
        self.assertEqual(self.select('production', '27B48'), (1, ''))

    def test_unknown_purpose_build_or_old_sdk_cannot_select_signing_tools(self):
        for purpose, build, sdk in [('other', '27B52', '27.2'), ('dev', '27B47', '27.2'),
                                    ('dev', '27B48', '27.1')]:
            with self.subTest(purpose=purpose, build=build, sdk=sdk):
                self.assertEqual(self.select(purpose, build, sdk), (1, ''))
