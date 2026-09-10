import unittest
from prepare_push import properties

class PushConfigTest(unittest.TestCase):
    def setUp(self):
        self.config = {'JPUSH_APPKEY': 'a'*24, 'VENDORS': ['honor'], 'HONOR_APPID': '123'}
    def test_client_properties(self):
        self.assertIn('HONOR_APPID=123\n', properties(self.config))
        self.assertIn('VENDORS=honor', properties(self.config))
    def test_rejects_missing_oem_client_keys(self):
        self.config.pop('HONOR_APPID')
        with self.assertRaises(ValueError): properties(self.config)
    def test_rejects_server_secrets_and_property_injection(self):
        self.config['MASTER_SECRET'] = 'never-in-apk'
        with self.assertRaises(ValueError): properties(self.config)
        self.config.pop('MASTER_SECRET')
        self.config['HONOR_APPID'] = '123\nMASTER_SECRET=bad'
        with self.assertRaises(ValueError): properties(self.config)
    def test_release_must_configure_an_offline_channel(self):
        self.config['VENDORS'] = []
        with self.assertRaises(ValueError): properties(self.config)
