"""Required-reason declarations for the native app, extension and bundled SDK."""
from pathlib import Path
import plistlib
import unittest
import tempfile
import shutil
from build_ios import validate_privacy_resources

ROOT = Path(__file__).resolve().parents[2]

class PrivacyManifestTest(unittest.TestCase):
    def test_export_rejects_missing_widget_or_sdk_manifest(self):
        source = ROOT / 'apps/mobile/packages/forum_app/ios/Runner/PrivacyInfo.xcprivacy'
        with tempfile.TemporaryDirectory() as temporary:
            app = Path(temporary) / 'Runner.app'
            paths = [app / 'PrivacyInfo.xcprivacy',
                     app / 'PlugIns/ScheduleWidgets.appex/PrivacyInfo.xcprivacy',
                     app / 'home_widget_home_widget.bundle/PrivacyInfo.xcprivacy']
            for path in paths:
                path.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, path)
            validate_privacy_resources(app)
            for path in paths:
                path.unlink()
                with self.assertRaises(ValueError):
                    validate_privacy_resources(app)
                shutil.copyfile(source, path)

    def test_native_bundles_declare_app_group_user_defaults(self):
        paths = [
            'apps/mobile/packages/forum_app/ios/Runner/PrivacyInfo.xcprivacy',
            'apps/mobile/packages/forum_app/ios/ScheduleWidgets/PrivacyInfo.xcprivacy',
            'apps/mobile/third_party/home_widget/ios/home_widget/Sources/home_widget/PrivacyInfo.xcprivacy',
        ]
        for path in paths:
            with self.subTest(bundle=path):
                manifest = plistlib.loads((ROOT / path).read_bytes())
                categories = {item['NSPrivacyAccessedAPIType']: item['NSPrivacyAccessedAPITypeReasons']
                              for item in manifest['NSPrivacyAccessedAPITypes']}
                self.assertIn('1C8F.1', categories['NSPrivacyAccessedAPICategoryUserDefaults'])
                self.assertFalse(manifest['NSPrivacyTracking'])

    def test_native_targets_include_manifest_resources(self):
        project = (ROOT / 'apps/mobile/packages/forum_app/ios/Runner.xcodeproj/project.pbxproj').read_text()
        for target in ['Runner', 'ScheduleWidgets']:
            self.assertIn(f'{target} PrivacyInfo.xcprivacy in Resources', project)
        package = ROOT / 'apps/mobile/third_party/home_widget/ios/home_widget/Package.swift'
        self.assertIn('.process("PrivacyInfo.xcprivacy")', package.read_text())
        podspec = ROOT / 'apps/mobile/third_party/home_widget/ios/home_widget.podspec'
        self.assertIn('s.resource_bundles', podspec.read_text())
