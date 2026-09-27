"""Check Android startup policy before packaging optional push SDKs."""
import unittest
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ANDROID = '{http://schemas.android.com/apk/res/android}'
TOOLS = '{http://schemas.android.com/tools}'


class AndroidPushConsentTest(unittest.TestCase):
    def test_optional_push_sdks_do_not_auto_register_before_consent(self):
        manifest = ROOT / 'apps/mobile/packages/forum_app/android/app/src/main/AndroidManifest.xml'
        app = ET.parse(manifest).getroot().find('application')
        metadata = {item.get(ANDROID + 'name'): item.get(ANDROID + 'value')
                    for item in app.findall('meta-data')}
        for name in ('firebase_messaging_auto_init_enabled', 'firebase_analytics_collection_enabled'):
            with self.subTest(name=name):
                self.assertEqual(metadata.get(name), 'false')
        init = next(item for item in app.findall('provider')
                    if item.get(ANDROID + 'name') == 'cn.jpush.android.service.InitProvider')
        self.assertEqual(init.get(TOOLS + 'node'), 'remove')
