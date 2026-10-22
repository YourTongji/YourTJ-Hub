#!/usr/bin/env python3
"""Explicit admin maintenance: add Associated Domains and replace only Runner's profile.

Uses ASC 5.0.0's profiles and bundle-ids interfaces:
https://github.com/rorkai/App-Store-Connect-CLI/tree/5.0.0/internal/cli/profiles
No certificate, old profile, Widget capability, or store release is revoked/changed.
"""
import base64
import json
import os
from pathlib import Path
import plistlib
import re
import subprocess
import tempfile

from build_ios import BUNDLE_ID, validate_profile
from publish_ios import asc, resource

TEAM = '4HJTS3G3T2'
DOMAINS = ['webcredentials:f.yourtj.de', 'webcredentials:dev.yourtj.de']


def refresh(old, name):
    validate_profile(old, TEAM)
    # Apple's bundle IDs are case-insensitive; do not rely on experimental API filtering.
    # https://developer.apple.com/help/glossary/bundle-id/
    inventory = asc('bundle-ids', 'list', '--fields', 'identifier,seedId', '--paginate')['data']
    bundles = [b for b in inventory if b['attributes'].get('identifier', '').lower() == BUNDLE_ID.lower()]
    if len(bundles) != 1:
        identifiers = [b['attributes'].get('identifier') for b in inventory]
        raise ValueError(f'Expected one existing Runner bundle ID; found {identifiers}')
    bundle = bundles[0]
    if bundle['attributes'].get('seedId') != TEAM:
        raise ValueError(f'Runner bundle seed {bundle["attributes"].get("seedId")} differs from team {TEAM}')
    profiles = asc('profiles', 'list', '--profile-type', 'IOS_APP_STORE', '--paginate')['data']
    matches = [p for p in profiles if p['attributes'].get('uuid', '').lower() == old['UUID'].lower()]
    if len(matches) != 1:
        raise ValueError(f'Configured Runner profile {old["UUID"]} is not uniquely present in Apple')
    existing = asc('profiles', 'view', '--id', matches[0]['id'], '--include', 'bundleId,certificates')
    relations = existing['data']['relationships']
    if relations['bundleId']['data']['id'] != bundle['id']:
        raise ValueError('Configured profile does not belong to Runner')
    certificates = [c['id'] for c in relations['certificates']['data']]
    certificate_bytes = {base64.b64decode(c['attributes']['certificateContent'], validate=True)
                         for c in existing.get('included', [])
                         if c['type'] == 'certificates' and c['id'] in certificates}
    if not certificates or certificate_bytes != set(old['DeveloperCertificates']):
        raise ValueError('Apple certificate identities do not match the configured profile')
    capabilities = asc('bundle-ids', 'capabilities', 'list', '--bundle', bundle['id'], '--paginate')['data']
    if not any(c['attributes']['capabilityType'] == 'ASSOCIATED_DOMAINS' for c in capabilities):
        asc('bundle-ids', 'capabilities', 'add', '--bundle', bundle['id'], '--capability', 'ASSOCIATED_DOMAINS')
    prior = [p for p in profiles if p['attributes'].get('name') == name]
    if len(prior) > 1:
        raise ValueError('Ambiguous previous profile maintenance result')
    if prior:
        result = resource(asc('profiles', 'view', '--id', prior[0]['id']))
    else:
        result = resource(asc('profiles', 'create', '--name', name, '--profile-type', 'IOS_APP_STORE',
                              '--bundle', bundle['id'], '--certificate', ','.join(certificates)))
    return base64.b64decode(result['attributes']['profileContent'], validate=True)


def decode(path):
    return plistlib.loads(subprocess.check_output(
        ['security', 'cms', '-D', '-i', str(path)], stderr=subprocess.DEVNULL))


def main():
    os.umask(0o077)
    run = os.environ['GITHUB_RUN_ID']
    if not re.fullmatch(r'[1-9]\d*', run):
        raise ValueError('Expected an Actions run ID')
    # No profile or authentication bytes enter logs or the artifact before validation.
    with tempfile.TemporaryDirectory() as folder:
        old_path = Path(folder) / 'old.mobileprovision'
        old_path.write_bytes(base64.b64decode(os.environ['IOS_PROFILE_BASE64'], validate=True))
        old = decode(old_path)
        updated_path = Path(folder) / 'updated.mobileprovision'
        updated_path.write_bytes(refresh(old, 'YourTJ Associated Domains ' + run))
        updated = decode(updated_path)
        validate_profile(updated, TEAM, associated_domains=DOMAINS)
        if set(updated['DeveloperCertificates']) != set(old['DeveloperCertificates']):
            raise ValueError('Replacement changed the signing certificate identities')
        output = Path('.ios-profile-refresh')
        output.mkdir(exist_ok=True)
        (output / 'Runner.mobileprovision').write_bytes(updated_path.read_bytes())
        facts = {'bundle': BUNDLE_ID, 'team': TEAM, 'oldUuid': old['UUID'],
                 'newUuid': updated['UUID'], 'name': updated['Name'],
                 'expires': updated['ExpirationDate'].isoformat(), 'run': run}
        (output / 'metadata.json').write_text(json.dumps(facts, indent=2) + '\n')
        print(json.dumps(facts))


if __name__ == '__main__':
    main()
