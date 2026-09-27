"""Check Xcode's effective widget signing settings after Flutter bootstrap."""
import json
import os
from pathlib import Path
import subprocess
import sys
import unittest


APP = Path.cwd() / 'apps/mobile/packages/forum_app'


@unittest.skipUnless(sys.platform == 'darwin' and os.environ.get('IOS_EXISTING_BUILD_ONLY') != 'true',
                     'requires macOS; publisher-only recovery does not build')
class IOSSigningSettingsTest(unittest.TestCase):
    def test_widget_release_profiles_use_the_manual_signing_configuration(self):
        self.assertTrue((APP / 'ios/Flutter/Generated.xcconfig').is_file(),
                        'Run flutter pub get in forum_app before checking native signing settings')
        config = APP / 'ios/Flutter/WidgetReleaseSigning.xcconfig'
        previous = config.read_bytes()
        profile = '00000000-0000-0000-0000-000000000001'
        try:
            config.write_text(
                '#include "Generated.xcconfig"\n'
                'DEVELOPMENT_TEAM = 4HJTS3G3T2\nCODE_SIGN_STYLE = Manual\n'
                'CODE_SIGN_IDENTITY = Apple Distribution\n'
                f'PROVISIONING_PROFILE_SPECIFIER = {profile}\n'
            )
            for configuration in ('Release', 'Profile'):
                with self.subTest(configuration=configuration):
                    result = subprocess.run(
                        ['xcodebuild', '-project', 'ios/Runner.xcodeproj', '-target', 'ScheduleWidgets',
                         '-configuration', configuration, '-sdk', 'iphoneos', '-showBuildSettings', '-json'],
                        cwd=APP, capture_output=True, text=True, timeout=180,
                    )
                    self.assertEqual(result.returncode, 0, result.stderr)
                    settings = next(item['buildSettings'] for item in json.loads(result.stdout)
                                    if item['target'] == 'ScheduleWidgets')
                    self.assertEqual(settings['CODE_SIGN_STYLE'], 'Manual')
                    self.assertEqual(settings['PROVISIONING_PROFILE_SPECIFIER'], profile)
                    self.assertEqual(settings['CODE_SIGN_IDENTITY'], 'Apple Distribution')
        finally:
            config.write_bytes(previous)
