"""Promote only identity-matched APP Test receipts after Huawei actually approves them."""
import json
import os
from pathlib import Path
import re
import sys

from github import GitHub
from controller import load_candidate
from catalog import candidate_content_digest
from state import latest_receipt, record
import tempfile
from model import require

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'mobile-release'))
from harmony_api import APP_ID, Client


def availability(version, details):
    require(str(version.get('versionId')) == str(details.get('versionId'))
            and version.get('testType') == 3 and version.get('releaseType') == 6,
            'Unexpected Huawei invitation identity')
    packages = version.get('packageList') or []
    require(len(packages) == 1 and str(packages[0].get('packageId')) == str(details.get('packageId')),
            'Huawei package differs from the original upload receipt')
    state = version.get('state')
    return 'success' if state == 0 else 'failure' if state in {1, 2, 10, 11, 13} else 'in_progress'


def import_initial_release(github, versions):
    candidate = 'mobile-1.0.18-36'
    if latest_receipt(github, candidate, 'harmony-apptest'):
        return
    with tempfile.TemporaryDirectory() as temp:
        manifest, folder = load_candidate(candidate, 'HEAD', Path(temp))
        evidence = json.loads((folder / 'evidence.json').read_text())
        details = evidence['evidence'][0]['receipt']
        require(details['appId'] == APP_ID and manifest['sourceSha'] == evidence['sourceSha'],
                'Initial Harmony archive identity changed')
        version = versions.get(details['versionId'])
        require(version is not None, 'Initial Harmony invitation missing')
        packages = version.get('packageList') or []
        require(len(packages) == 1 and str(packages[0].get('versionCode')) == str(manifest['buildNumber'])
                and packages[0].get('versionName') == manifest['version'], 'Initial Harmony native identity changed')
        if availability(version, details) != 'success':
            return  # Unknown/expired availability cannot bootstrap an approved historical receipt.
        record(github, manifest, {'sourceSha': manifest['sourceSha'], 'contentDigest': candidate_content_digest(folder)},
               'harmony-apptest', 'success', details | {'sourceSha': manifest['sourceSha'], 'availability': 'APPROVED'})


def reconcile(github, client):
    versions = {str(v['versionId']): v for v in client.versions()}
    if os.environ.get('HARMONY_IMPORT_INITIAL') == 'true':
        import_initial_release(github, versions)
    for environment in ['release-harmony-apptest', 'dev-harmonyos']:
        seen = set()
        for deployment in github.pages('deployments?environment=' + environment):
            payload = deployment.get('payload') or {}
            key = payload.get('candidateId') if environment.startswith('release-') else payload.get('run')
            if not key or key in seen:
                continue
            seen.add(key)
            statuses = github.pages(f"deployments/{deployment['id']}/statuses")
            if not statuses or statuses[0]['state'] != 'in_progress':
                continue
            details = payload.get('details') or {}
            if details.get('availability') != 'SUBMITTED':
                continue
            require(details.get('appId') == APP_ID and re.fullmatch(r'[0-9a-f]{64}', details.get('appSha256', ''))
                    and details.get('sourceSha') == deployment['sha'], 'Malformed Harmony publication receipt')
            version = versions.get(str(details.get('versionId')))
            require(version is not None, 'Original Huawei version missing; reconcile manually')
            state = availability(version, details)
            if state == 'in_progress':
                continue
            updated = payload | {'details': details | {'availability': 'APPROVED' if state == 'success' else 'UNAVAILABLE'}}
            # GitHub deployment payloads are immutable; retain the exact original binding in the new receipt.
            new = github.api('deployments', method='POST', data={
                'ref': deployment['sha'], 'environment': environment, 'auto_merge': False,
                'required_contexts': [], 'production_environment': False, 'payload': updated})
            github.api(f"deployments/{new['id']}/statuses", method='POST', data={
                'state': state, 'auto_inactive': False, 'description': updated['details']['availability']})


if __name__ == '__main__':
    try:
        reconcile(GitHub(), Client(json.loads(os.environ['HARMONY_SERVICE_ACCOUNT_JSON'])))
    except (ValueError, KeyError, OSError, TypeError):
        raise SystemExit('Harmony reconciliation stopped; no unknown version may be marked available')
