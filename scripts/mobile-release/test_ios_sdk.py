"""Reject IPA bundles that cannot provide the requested Duo SDK behavior."""
from pathlib import Path
import plistlib
import tempfile
import unittest
import zipfile

import build_ios


class IOSSDKTest(unittest.TestCase):
    def test_export_entry_rejects_an_old_sdk_before_signing_checks(self):
        with tempfile.TemporaryDirectory() as directory:
            ipa = Path(directory) / 'old-sdk.ipa'
            with zipfile.ZipFile(ipa, 'w') as archive:
                for path in ('Runner.app', 'Runner.app/PlugIns/ScheduleWidgets.appex'):
                    archive.writestr(f'Payload/{path}/Info.plist', plistlib.dumps({
                        'DTSDKName': 'iphoneos26.5', 'MinimumOSVersion': '15.0',
                    }))
            with self.assertRaisesRegex(ValueError, 'iOS 27.1'):
                build_ios.validate_exported_ipa(ipa, '4HJTS3G3T2')

    def test_validates_the_app_and_embedded_widget_build_metadata(self):
        with tempfile.TemporaryDirectory() as directory:
            app = Path(directory) / 'Runner.app'
            widget = app / 'PlugIns/ScheduleWidgets.appex'
            widget.mkdir(parents=True)
            valid = {'DTSDKName': 'iphoneos27.1', 'MinimumOSVersion': '15.0'}
            for bundle in (app, widget):
                (bundle / 'Info.plist').write_bytes(plistlib.dumps(valid))
            build_ios.validate_build_metadata(app)
            for bundle in (app, widget):
                for invalid in (
                    {**valid, 'DTSDKName': 'iphoneos26.5'},
                    {**valid, 'DTSDKName': 'iphonesimulator27.1'},
                    {**valid, 'MinimumOSVersion': '14.0'},
                    {'MinimumOSVersion': '15.0'},
                ):
                    with self.subTest(bundle=bundle.name, metadata=invalid):
                        (bundle / 'Info.plist').write_bytes(plistlib.dumps(invalid))
                        with self.assertRaises(ValueError):
                            build_ios.validate_build_metadata(app)
                        (bundle / 'Info.plist').write_bytes(plistlib.dumps(valid))
            newer = {'DTSDKName': 'iphoneos27.2', 'MinimumOSVersion': '16.0'}
            for bundle in (app, widget):
                (bundle / 'Info.plist').write_bytes(plistlib.dumps(newer))
            build_ios.validate_build_metadata(app)
