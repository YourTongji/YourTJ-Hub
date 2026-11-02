"""Reject IPA bundles that cannot provide the requested Duo SDK behavior."""
from pathlib import Path
import plistlib
import tempfile
import unittest
from unittest.mock import patch
import zipfile

import build_ios


class IOSSDKTest(unittest.TestCase):
    def test_exported_rc_ipa_requires_the_explicit_dev_exception(self):
        with tempfile.TemporaryDirectory() as directory:
            ipa = Path(directory) / 'rc.ipa'
            metadata = {'DTSDKName': 'iphoneos27.2', 'DTXcodeBuild': '27B48', 'MinimumOSVersion': '15.0'}
            privacy = {'NSPrivacyAccessedAPITypes': [{
                'NSPrivacyAccessedAPIType': 'NSPrivacyAccessedAPICategoryUserDefaults',
                'NSPrivacyAccessedAPITypeReasons': ['1C8F.1'],
            }]}
            with zipfile.ZipFile(ipa, 'w') as archive:
                for bundle in ('Runner.app', 'Runner.app/PlugIns/ScheduleWidgets.appex'):
                    archive.writestr(f'Payload/{bundle}/Info.plist', plistlib.dumps(metadata))
                for bundle in ('Runner.app', 'Runner.app/PlugIns/ScheduleWidgets.appex', 'Runner.app/home_widget.bundle'):
                    archive.writestr(f'Payload/{bundle}/PrivacyInfo.xcprivacy', plistlib.dumps(privacy))
            team = '4HJTS3G3T2'
            app_entitlements = {'application-identifier': f'{team}.{build_ios.BUNDLE_ID}',
                                'aps-environment': 'production', 'com.apple.developer.applesignin': ['Default'],
                                'com.apple.security.application-groups': [build_ios.WIDGET_APP_GROUP]}
            widget_entitlements = {'application-identifier': f'{team}.{build_ios.WIDGET_BUNDLE_ID}',
                                   'com.apple.security.application-groups': [build_ios.WIDGET_APP_GROUP]}
            with patch('build_ios.subprocess.check_output', side_effect=[
                    plistlib.dumps(app_entitlements), plistlib.dumps(widget_entitlements)]) as codesign:
                with self.assertRaises(ValueError):
                    build_ios.validate_exported_ipa(ipa, team)
                codesign.assert_not_called()
                build_ios.validate_exported_ipa(ipa, team, allow_release_candidate=True)

    def test_dev_preview_accepts_the_reviewed_rc_but_production_stays_strict(self):
        with tempfile.TemporaryDirectory() as directory:
            app = Path(directory) / 'Runner.app'
            widget = app / 'PlugIns/ScheduleWidgets.appex'
            widget.mkdir(parents=True)
            valid = {'DTSDKName': 'iphoneos27.2', 'DTXcodeBuild': '27B48', 'MinimumOSVersion': '15.0'}
            for bundle in (app, widget):
                (bundle / 'Info.plist').write_bytes(plistlib.dumps(valid))
            build_ios.validate_build_metadata(app, allow_release_candidate=True)
            with self.assertRaises(ValueError):
                build_ios.validate_build_metadata(app)
            for bundle in (app, widget):
                for invalid in (
                    {**valid, 'DTXcodeBuild': '27B47'},
                    {**valid, 'DTSDKName': 'iphoneos27.1'},
                    {**valid, 'DTSDKName': 'iphonesimulator27.2'},
                    {**valid, 'MinimumOSVersion': '14.0'},
                ):
                    with self.subTest(bundle=bundle.name, metadata=invalid):
                        (bundle / 'Info.plist').write_bytes(plistlib.dumps(invalid))
                        with self.assertRaises(ValueError):
                            build_ios.validate_build_metadata(app, allow_release_candidate=True)
                        (bundle / 'Info.plist').write_bytes(plistlib.dumps(valid))

    def test_export_entry_rejects_an_old_sdk_before_signing_checks(self):
        with tempfile.TemporaryDirectory() as directory:
            ipa = Path(directory) / 'old-sdk.ipa'
            with zipfile.ZipFile(ipa, 'w') as archive:
                for path in ('Runner.app', 'Runner.app/PlugIns/ScheduleWidgets.appex'):
                    archive.writestr(f'Payload/{path}/Info.plist', plistlib.dumps({
                        'DTSDKName': 'iphoneos26.5', 'MinimumOSVersion': '15.0',
                    }))
            with self.assertRaisesRegex(ValueError, 'iOS 27.2'):
                build_ios.validate_exported_ipa(ipa, '4HJTS3G3T2')

    def test_validates_the_app_and_embedded_widget_build_metadata(self):
        with tempfile.TemporaryDirectory() as directory:
            app = Path(directory) / 'Runner.app'
            widget = app / 'PlugIns/ScheduleWidgets.appex'
            widget.mkdir(parents=True)
            valid = {'DTSDKName': 'iphoneos27.2', 'DTXcodeBuild': '27B52', 'MinimumOSVersion': '15.0'}
            for bundle in (app, widget):
                (bundle / 'Info.plist').write_bytes(plistlib.dumps(valid))
            build_ios.validate_build_metadata(app)
            for bundle in (app, widget):
                for invalid in (
                    {**valid, 'DTSDKName': 'iphoneos26.5'},
                    {**valid, 'DTSDKName': 'iphoneos27.1'},
                    {**valid, 'DTXcodeBuild': '27B48'},
                    {**valid, 'DTSDKName': 'iphonesimulator27.2'},
                    {**valid, 'MinimumOSVersion': '14.0'},
                    {'MinimumOSVersion': '15.0'},
                ):
                    with self.subTest(bundle=bundle.name, metadata=invalid):
                        (bundle / 'Info.plist').write_bytes(plistlib.dumps(invalid))
                        with self.assertRaises(ValueError):
                            build_ios.validate_build_metadata(app)
                        (bundle / 'Info.plist').write_bytes(plistlib.dumps(valid))
            newer = {**valid, 'MinimumOSVersion': '16.0'}
            for bundle in (app, widget):
                (bundle / 'Info.plist').write_bytes(plistlib.dumps(newer))
            build_ios.validate_build_metadata(app)
