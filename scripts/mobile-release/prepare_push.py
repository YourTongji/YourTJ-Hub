"""Validate client-only JPush configuration before producing signed Android builds.

ANDROID_PUSH_JSON is a secret JSON object with JPUSH_APPKEY, VENDORS (array), and
OEM client identifiers. Provider server secrets are deliberately rejected.
"""
import json
import os
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
VENDORS = {
    'xiaomi': ['XIAOMI_APPID', 'XIAOMI_APPKEY'],
    'oppo': ['OPPO_APPID', 'OPPO_APPKEY', 'OPPO_APPSECRET'],
    'vivo': ['VIVO_APPID', 'VIVO_APPKEY'],
    'honor': ['HONOR_APPID'],
    'meizu': ['MEIZU_APPID', 'MEIZU_APPKEY'],
    'huawei': [],
}

def properties(config):
    allowed = {'JPUSH_APPKEY', 'VENDORS'} | {key for keys in VENDORS.values() for key in keys}
    if set(config) - allowed:
        raise ValueError('Unknown Android push keys; never include provider server credentials')
    if not re.fullmatch(r'[a-fA-F0-9]{24}', config.get('JPUSH_APPKEY', '')):
        raise ValueError('ANDROID_PUSH_JSON requires a valid JPUSH_APPKEY')
    vendors = config.get('VENDORS', [])
    if not isinstance(vendors, list) or not vendors or any(v not in VENDORS for v in vendors):
        raise ValueError('Select at least one configured OEM offline channel in VENDORS')
    required = {'JPUSH_APPKEY'} | {key for vendor in vendors for key in VENDORS[vendor]}
    for key in required:
        if not isinstance(config.get(key), str) or not re.fullmatch(r'[A-Za-z0-9_.:-]+', config[key]):
            raise ValueError(f'Missing or invalid Android push client parameter: {key}')
    return '\n'.join(f'{key}={config[key]}' for key in sorted(required)) + '\nVENDORS=' + ','.join(vendors) + '\n'

def main():
    config = json.loads(os.environ.get('ANDROID_PUSH_JSON', '{}'))
    output = properties(config)
    android = ROOT / 'apps/mobile/packages/forum_app/android'
    if 'huawei' in config['VENDORS']:
        raw = os.environ.get('HUAWEI_AGCONNECT_JSON', '')
        data = json.loads(raw)
        if data.get('client', {}).get('package_name') != 'tj.yourtj.forum_app':
            raise ValueError('Huawei configuration must match the Android package')
        destination = android / 'app/agconnect-services.json'
        destination.write_text(raw)
        destination.chmod(0o600)
    destination = android / 'push.properties'
    destination.write_text(output)
    destination.chmod(0o600)
    print('Android push client configuration validated (values withheld)')

if __name__ == '__main__':
    main()
