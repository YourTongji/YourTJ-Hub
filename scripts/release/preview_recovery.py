"""Recover an already processed dev iOS preview, without signing or uploading again."""
import io
import json
import os
import re
import subprocess
import zipfile

from model import VERSION, require


def read_identity(github, cycle, run_id, purpose):
    require(purpose == 'dev' and run_id.isdigit(), 'Recovery requires an original dev preview run ID')
    run = github.api(f'actions/runs/{run_id}')
    require(run.get('path') == '.github/workflows/release-prerelease.yml'
            and run.get('event') == 'workflow_dispatch' and run.get('head_branch') == 'main'
            and run.get('head_repository', {}).get('full_name', '').lower() == github.repository.lower()
            and run.get('status') == 'completed' and run.get('conclusion') in {'failure', 'success'},
            'Recovery requires a completed trusted-main preview')
    jobs = {}
    for job in sorted(github.pages(f'actions/runs/{run_id}/jobs?filter=all', 'jobs'), key=lambda j: j['id']):
        jobs[job['name']] = job
    require(jobs.get('validated', {}).get('conclusion') == 'success'
            and any(s['name'] == 'Build App Store IPA' and s['conclusion'] == 'success'
                    for s in jobs.get('ios / preview', {}).get('steps', [])),
            'Original preview source checks and signed IPA build must have succeeded')
    name = f"preview-apple-status-{run['run_attempt']}"
    artifacts = [a for a in github.pages(f'actions/runs/{run_id}/artifacts', 'artifacts') if a['name'] == name]
    require(len(artifacts) == 1 and not artifacts[0]['expired'] and artifacts[0]['size_in_bytes'] < 100_000,
            'Original Apple preview receipt is missing or expired')
    archive = subprocess.check_output(['gh', 'api',
        f'repos/{github.repository}/actions/artifacts/{artifacts[0]["id"]}/zip'])
    with zipfile.ZipFile(io.BytesIO(archive)) as zipped:
        require(zipped.namelist() == ['.preview-ios.json']
                and zipped.getinfo('.preview-ios.json').file_size < 50_000, 'Invalid Apple preview receipt')
        receipt = json.loads(zipped.read('.preview-ios.json'))
    require(receipt.get('stage') in {'processed', 'assigned', 'available_for_testing'}
            and receipt.get('environment') == purpose and receipt.get('source') == cycle['source']
            and receipt.get('buildId'), 'Recovery requires the exact processed source/environment/Apple build')
    require(VERSION.fullmatch(receipt.get('version', ''))
            and re.fullmatch(r'[1-9][0-9]*', str(receipt.get('buildNumber', ''))),
            'Invalid original preview version/build')
    refs = github.pages(f"git/matching-refs/tags/preview-{cycle['id']}-{run_id}-")
    matches = []
    for ref in refs:
        if ref['object']['type'] != 'tag' or not re.fullmatch(
                rf"refs/tags/preview-{re.escape(cycle['id'])}-{run_id}-[1-9][0-9]*-dev", ref['ref']):
            continue
        tag = github.api(f"git/tags/{ref['object']['sha']}")
        data = json.loads(tag['message'])
        if (tag['object'].get('sha') == cycle['source'] and tag['object'].get('type') == 'commit') and all(
                data.get(k) == v for k, v in {'source': cycle['source'], 'cycle': cycle['id'],
                    'environment': purpose, 'preview': True, 'version': receipt['version']}.items()) \
                and str(data.get('buildNumber')) == str(receipt['buildNumber']):
            matches.append(ref['ref'].removeprefix('refs/tags/'))
    require(len(matches) == 1, 'Apple receipt does not match one immutable preview identity')
    return {'version': receipt['version'], 'number': str(receipt['buildNumber']),
            'tag': matches[0], 'apple_build': receipt['buildId']}


def main():
    from cycle import current, live
    from cycle_store import load
    from github import GitHub
    import time
    github = GitHub()
    cycle = current(load(github))
    require(cycle and cycle['id'] == os.environ.get('CYCLE_ID')
            and cycle['source'] == os.environ.get('SOURCE_SHA')
            and os.environ.get('ARTIFACTS') == 'ios',
            'iOS recovery requires the current cycle, exact SHA and ios-only selection')
    live(cycle, int(time.time()))
    print(json.dumps(read_identity(github, cycle, os.environ['IOS_EXISTING_RUN'], os.environ['PURPOSE'])))


if __name__ == '__main__':
    main()
