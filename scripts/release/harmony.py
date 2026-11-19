"""APP Test adapter for the existing RC, Confirm, recovery and dev identities."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import sys
import time
from zipfile import ZipFile

from model import require

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'mobile-release'))
from harmony_api import APP_ID, BUNDLE, Client
from publish_harmony import publish, verify_group


def app_identity(directory, version, number, expected=''):
    apps = list(Path(directory).glob('*.app'))
    require(len(apps) == 1, 'Expected one original signed Harmony APP')
    app = apps[0]
    with app.open('rb') as file:
        sha = hashlib.file_digest(file, 'sha256').hexdigest()
    require(app.with_suffix('.sha256').read_text() == f'{sha}  {app.name}\n', 'Original APP checksum changed')
    with ZipFile(app) as archive:
        info = json.loads(archive.read('pack.info'))['summary']['app']
    require(info['bundleName'] == BUNDLE and info['version']['name'] == version
            and str(info['version']['code']) == str(number), 'Original APP version/bundle changed')
    require(not expected or sha == expected, 'APP differs from the selected RC digest')
    return app, sha


def configuration(detail, group, notes, now, share=True):
    require(re.fullmatch(r'[0-9a-f]{32}', group or ''), 'Configure the explicit APP Test group ID')
    require(0 < len(notes.strip()) <= 1024 and '同济大学' not in notes, 'Invalid reviewed Harmony release notes')
    languages = []
    for language in detail['languages']:
        require('同济大学' not in language['appDesc'], 'Review inherited app description before submitting')
        languages.append({'language': language['lang'], 'appDesc': language['appDesc'], 'newFeatures': notes})
    require(languages, 'Inherited app languages are missing')
    inherited = (detail.get('openTestInfo') or {}).get('testTaskInfo', {}).get('groupInfos', [])
    groups = [{'groupId': old['groupId'], 'bind': 0} for old in inherited if old.get('groupId') != group]
    groups.append({'groupId': group, 'bind': 1})
    return {'languagesList': languages, 'openTestInfo': {
        'startTime': now * 1000, 'endTime': (now + 30 * 86400) * 1000,
        'testDesc': 'YourTJ HarmonyOS', 'testTaskInfo': {
            'groupInfos': groups, 'needShareLink': int(share),
            'displayArea': '1', 'needNotify': 1}}}


def publish_original(client, app, receipt, source, sha, group, notes, share=True):
    require(re.fullmatch(r'[0-9a-f]{32}', group or ''), 'Configure the explicit APP Test group ID')
    require(0 < len(notes.strip()) <= 1024 and '同济大学' not in notes, 'Invalid reviewed Harmony release notes')
    if receipt.exists():
        previous = json.loads(receipt.read_text())
        require(previous.get('sha256') == sha and previous.get('sourceSha') == source
                and previous.get('groupId') == group, 'Original upload/group receipt changed')
        require(previous.get('versionId'), 'Draft creation outcome unknown; reconcile Huawei before recovery')
        version_id = previous['versionId']
    else:
        # Persist intent before every non-idempotent call. An ambiguous response requires reconciliation.
        receipt.write_text(json.dumps({'sha256': sha, 'sourceSha': source, 'groupId': group, 'phase': 'creating'}) + '\n')
        created = client.request('POST', '/api/publish/v2/test/app/version', {
            'releaseType': 6, 'testType': 3, 'testDesc': 'YourTJ HarmonyOS', 'onshelfSelfDetect': 0}, query={})
        version_id = str(created['versionId'])
        receipt.write_text(json.dumps({'appId': APP_ID, 'versionId': version_id, 'sha256': sha,
                                       'sourceSha': source, 'groupId': group, 'phase': 'prepared'}) + '\n')
    version = client.invitation(version_id)
    if version['state'] == 7:
        detail = client.request('GET', '/api/publish/v3/app-info', query={'releaseType': 6, 'versionId': version_id})
        config = configuration(detail, group, notes, int(time.time()), share=share)
        client.request('PUT', '/api/publish/v2/test/app/version', {'versionId': version_id, **config}, query={})
        verify_group(client, version_id, group)
    publish(client, app, version_id, receipt, source)
    record = json.loads(receipt.read_text())
    require(record.get('packageId'), 'Original upload has no package ID')
    current = client.invitation(version_id)
    packages = current.get('packageList') or []
    require(len(packages) == 1 and str(packages[0].get('packageId')) == record['packageId'],
            'Huawei package binding changed')
    require(current['state'] in {0, 3, 4, 12}, 'Huawei did not accept the test submission')
    return {'appId': APP_ID, 'versionId': version_id, 'packageId': record['packageId'],
            'appSha256': sha, 'sourceSha': source, 'availability': 'APPROVED' if current['state'] == 0 else 'SUBMITTED'}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=['inspect', 'publish', 'preflight'])
    args = parser.parse_args()
    if args.command == 'preflight':
        if os.environ.get('RC_BUILD') != 'true':
            group = os.environ.get('HARMONY_TEST_GROUP_ID', '')
            require(re.fullmatch(r'[0-9a-f]{32}', group), 'Configure the explicit APP Test group before building')
            if os.environ.get('AUTOMATIC_DEV') == 'true':
                require(group != os.environ.get('HARMONY_PRODUCTION_GROUP_ID') and os.environ.get('HARMONY_PRODUCTION_GROUP_ID'),
                        'Dev builds require a separate explicit internal group')
        client = Client(json.loads(os.environ['HARMONY_SERVICE_ACCOUNT_JSON']))
        require(all(int(p['versionCode']) < int(os.environ['MOBILE_BUILD_NUMBER'])
                    for v in client.versions() for p in (v.get('packageList') or [])),
                'Native build number already uploaded; recover its original artifact')
        return
    app, sha = app_identity(os.environ['HARMONY_APP_DIRECTORY'], os.environ['MOBILE_VERSION'],
                            os.environ['MOBILE_BUILD_NUMBER'], os.environ.get('EXPECTED_APP_SHA256', ''))
    details = {'appSha256': sha}
    if args.command == 'publish':
        group = os.environ['HARMONY_TEST_GROUP_ID']
        if os.environ.get('AUTOMATIC_DEV') == 'true':
            require(group != os.environ.get('HARMONY_PRODUCTION_GROUP_ID') and os.environ.get('HARMONY_PRODUCTION_GROUP_ID'),
                    'Dev distribution requires a separate explicit internal group')
        client = Client(json.loads(os.environ['HARMONY_SERVICE_ACCOUNT_JSON']))
        require(all(int(p['versionCode']) <= int(os.environ['MOBILE_BUILD_NUMBER'])
                    for v in client.versions() for p in (v.get('packageList') or [])),
                'A newer Harmony build exists; prepare a fresh RC instead of rebuilding or publishing this identity')
        details = publish_original(client, app, Path('.harmony-upload.json'), os.environ['SOURCE_SHA'], sha,
                                   group, Path(os.environ['HARMONY_NOTES_PATH']).read_text().strip(),
                                   share=os.environ.get('AUTOMATIC_DEV') != 'true')
    Path('.release-result.json').write_text(json.dumps(details) + '\n')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError, TypeError):
        raise SystemExit('Harmony publication stopped; preserve original APP/receipt and reconcile Huawei status')
