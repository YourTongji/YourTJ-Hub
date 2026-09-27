"""Validate client-only JPush configuration before producing signed Android builds.

ANDROID_PUSH_JSON is a secret JSON object with JPUSH_APPKEY, an optional VENDORS
array, and client identifiers for selected adapters. Provider server secrets are
deliberately rejected. FCM's Firebase app configuration is passed separately.
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
    'fcm': [],
}

ANDROID_PACKAGE = 'tj.yourtj.forum_app'

def properties(config):
    allowed = {'JPUSH_APPKEY', 'VENDORS'} | {key for keys in VENDORS.values() for key in keys}
    if set(config) - allowed:
        raise ValueError('Unknown Android push keys; never include provider server credentials')
    if not re.fullmatch(r'[a-fA-F0-9]{24}', config.get('JPUSH_APPKEY', '')):
        raise ValueError('ANDROID_PUSH_JSON requires a valid JPUSH_APPKEY')
    vendors = config.get('VENDORS', [])
    if not isinstance(vendors, list) or any(not isinstance(v, str) or v not in VENDORS for v in vendors):
        raise ValueError('VENDORS must contain only supported offline push adapters')
    required = {'JPUSH_APPKEY'} | {key for vendor in vendors for key in VENDORS[vendor]}
    for key in required:
        if not isinstance(config.get(key), str) or not re.fullmatch(r'[A-Za-z0-9_.:-]+', config[key]):
            raise ValueError(f'Missing or invalid Android push client parameter: {key}')
    return '\n'.join(f'{key}={config[key]}' for key in sorted(required)) + '\nVENDORS=' + ','.join(vendors) + '\n'

def fcm_config(raw):
    try:
        data = json.loads(raw)
    except (TypeError, json.JSONDecodeError):
        raise ValueError('FCM_GOOGLE_SERVICES_JSON must be valid Firebase Android configuration JSON') from None
    clients = data.get('client', []) if isinstance(data, dict) else []
    package_names = set()
    if isinstance(clients, list):
        for client in clients:
            info = client.get('client_info') if isinstance(client, dict) else None
            android = info.get('android_client_info') if isinstance(info, dict) else None
            package_name = android.get('package_name') if isinstance(android, dict) else None
            if isinstance(package_name, str):
                package_names.add(package_name)
    if ANDROID_PACKAGE not in package_names:
        raise ValueError(f'FCM_GOOGLE_SERVICES_JSON must include package {ANDROID_PACKAGE}')
    return raw

def main():
    config = json.loads(os.environ.get('ANDROID_PUSH_JSON', '{}'))
    output = properties(config)
    android = ROOT / 'apps/mobile/packages/forum_app/android'
    fcm_raw = None
    if 'fcm' in config.get('VENDORS', []):
        fcm_raw = fcm_config(os.environ.get('FCM_GOOGLE_SERVICES_JSON', ''))
    if 'huawei' in config.get('VENDORS', []):
        raw = os.environ.get('HUAWEI_AGCONNECT_JSON', '')
        data = json.loads(raw)
        if data.get('client', {}).get('package_name') != ANDROID_PACKAGE:
            raise ValueError('Huawei configuration must match the Android package')
        destination = android / 'app/agconnect-services.json'
        destination.write_text(raw)
        destination.chmod(0o600)
    if fcm_raw is not None:
        destination = android / 'app/google-services.json'
        destination.write_text(fcm_raw)
        destination.chmod(0o600)
    destination = android / 'push.properties'
    destination.write_text(output)
    destination.chmod(0o600)
    print('Android push client configuration validated (values withheld)')

if __name__ == '__main__':
    main()
