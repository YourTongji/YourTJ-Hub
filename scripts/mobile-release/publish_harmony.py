"""Upload and submit an existing YourTJ invitation draft; preserve upload identity on retry."""
import hashlib
import json
import os
from pathlib import Path
import re
import time
from zipfile import ZipFile

from harmony_api import APP_ID, BUNDLE, Client


def publish(client, app, version_id, receipt, source):
    if not re.fullmatch(r'[0-9a-f]{40}', source):
        raise ValueError('A full source SHA is required')
    version = client.invitation(version_id)
    with ZipFile(app) as archive:
        identity = json.loads(archive.read('pack.info'))['summary']['app']
    if identity['bundleName'] != BUNDLE:
        raise ValueError('Unexpected Harmony bundle identity')
    with app.open('rb') as file:
        digest = hashlib.file_digest(file, 'sha256').hexdigest()
    expected = {'appId': APP_ID, 'versionId': version_id, 'sha256': digest, 'sourceSha': source}
    if receipt.exists():
        record = json.loads(receipt.read_text())
        if any(record.get(key) != value for key, value in expected.items()):
            raise ValueError('Receipt does not match the original APP, source and invitation draft')
        if record.get('phase') == 'submitted':
            print('Original APP Test submission already recorded; inspect Huawei review status')
            return
        if version.get('state') != 7:
            raise ValueError('Invitation state changed; reconcile Huawei status before retrying')
    else:
        if version.get('state') != 7:
            raise ValueError('Invitation must be in Prepare to submit state')
        if os.environ.get('GITHUB_RUN_ATTEMPT', '1') != '1':
            raise ValueError('Retry has no upload receipt; reconcile the original upload manually')
        package, uploaded_digest = client.upload(app)
        if uploaded_digest != digest:
            raise ValueError('APP changed during upload')
        record = {**expected, 'packageId': package, 'phase': 'uploaded'}
        receipt.write_text(json.dumps(record, indent=2) + '\n')
    package = record.get('packageId', '')
    if not isinstance(package, str) or not re.fullmatch(r'[0-9]{1,32}', package):
        raise ValueError('Invalid package identity in receipt')
    for _ in range(60):
        states = client.request('GET', '/api/publish/v3/package/compile/status', query={'pkgIds': package})['pkgStateList']
        matching = [item for item in states if str(item.get('pkgId')) == package]
        if len(matching) != 1:
            raise ValueError('Huawei returned an unexpected package status identity')
        status = matching[0].get('successStatus')
        if status == 0:
            break
        if status not in (1,):
            raise ValueError(f'Harmony package legality failed (status {status}); inspect Huawei report')
        time.sleep(10)
    else:
        raise ValueError('Harmony package parsing timed out; recover using the saved receipt')
    client.invitation(version_id)
    client.request('PUT', '/api/publish/v2/test/app/version',
                   {'versionId': version_id, 'pkgId': package}, query={})
    bound = client.invitation(version_id).get('packageList') or []
    if len(bound) != 1 or str(bound[0].get('packageId')) != package:
        raise ValueError('Invitation did not bind the original uploaded package')
    client.request('POST', '/api/publish/v2/test/app/version/submit', {'versionId': version_id}, query={})
    record['phase'] = 'submitted'
    receipt.write_text(json.dumps(record, indent=2) + '\n')
    print('YourTJ APP Test submitted for review; approval and availability are not confirmed')


if __name__ == '__main__':
    try:
        client = Client(json.loads(Path(os.environ['HARMONY_SERVICE_ACCOUNT_PATH']).read_text()))
        publish(client, Path(os.environ['HARMONY_APP_PATH']), os.environ['HARMONY_TEST_VERSION_ID'],
                Path(os.environ['HARMONY_RECEIPT_PATH']), os.environ['SOURCE_SHA'])
    except (ValueError, KeyError, OSError):
        raise SystemExit('Harmony APP Test publication stopped; preserve its receipt and inspect Huawei status')
