#!/usr/bin/env python3
"""Production RCs reuse the platform builders and immutable Actions artifacts.

An RC is a build of an existing release request, not a second approval protocol.
Notes may still be edited; source/config/version/build identity cannot change.
"""
import argparse
import base64
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import zipfile

from github import GitHub, git
from model import FILES, require, digest

WORKFLOW = '.github/workflows/release-rc.yml'
IDENTITY_KEYS = ('candidateId', 'sourceSha', 'product', 'version', 'buildNumber')


def platform(channel):
    return 'ios' if channel.startswith('ios-') else channel


def artifact_name(identity, name):
    return ('web-v' + identity['version']) if name == 'web' else f"{name}-{identity['version']}-{identity['buildNumber']}"


def validate_identity(identity, expected):
    if expected.get('operation') == 'promote-ios':
        expected = expected | {'candidateId': expected['existingRelease']['candidateId']}
    keys = IDENTITY_KEYS + tuple(k for k in ['displayBuild', 'iosBuildNumber'] if k in expected)
    require(all(identity.get(k) == expected.get(k) for k in keys)
            and identity.get('environment') == 'production' and identity.get('preview') is False,
            'RC source/version/build/config differs from the selected release request')


def trusted_run(github, run_id, completed=False):
    require(str(run_id).isdigit(), 'Select an explicit RC run ID')
    run = github.api(f'actions/runs/{run_id}')
    require(run.get('path') == WORKFLOW and run.get('event') == 'workflow_dispatch'
            and run.get('head_branch') == 'main'
            and run.get('head_repository', {}).get('full_name', '').lower() == github.repository.lower(),
            'RC must come from the trusted-main RC workflow')
    if completed:
        require(run.get('status') == 'completed', 'Selected RC is still running')
    return run


def read_json_artifact(github, run_id, name, filename):
    rows = [a for a in github.pages(f'actions/runs/{run_id}/artifacts', 'artifacts') if a['name'] == name]
    require(len(rows) == 1 and not rows[0]['expired'] and rows[0]['size_in_bytes'] < 1_000_000,
            'Original RC record missing/expired; never rebuild a published identity')
    data = subprocess.check_output(['gh', 'api', f'repos/{github.repository}/actions/artifacts/{rows[0]["id"]}/zip'])
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        require(archive.namelist() == [filename] and archive.getinfo(filename).file_size < 500_000,
                'Invalid RC artifact contents')
        return json.loads(archive.read(filename))


def read_record(github, run_id, manifest):
    run = trusted_run(github, run_id, completed=True)
    result = read_json_artifact(github, run_id, f"rc-record-{run['run_attempt']}", 'rc-record.json')
    identity = result['identity']
    validate_identity(identity, manifest)
    if manifest.get('operation') == 'promote-ios':
        require(result.get('platforms', {}).get('ios', {}).get('buildId') == manifest['existingRelease']['buildId'],
                'Store promotion must use the original RC Apple build')
    require(identity['run'] == str(run_id) and identity['controllerSha'] == run['head_sha'],
            'RC record run/controller mismatch')
    # Collection seals only successful platform results. A later request-attachment
    # failure must not invalidate those original bytes or force another build.
    require(result.get('platforms') and set(result['platforms']) == set(identity['platforms']),
            'RC platform record mismatch')
    for name, details in result['platforms'].items():
        require(name in {'web', 'android', 'ios'} and details.get('buildRunId') == str(run_id)
                and details.get('artifact') == artifact_name(identity, name), 'RC artifact identity mismatch')
        reusable(result, name)
    return result


def reusable(record, channel):
    name = platform(channel)
    require(name in record['platforms'], 'Selected RC has no artifact for this platform')
    result = record['platforms'][name]
    if name == 'ios':
        require(result.get('buildId'), 'RC has no original Apple build ID')
    elif name == 'web':
        require(result.get('image') and result.get('binarySha256'), 'RC has no original image/binary')
    else:
        require(result.get('assets'), 'RC has no original APK digests')
    return result


def resolve(github, candidate, folder, linked_run=None):
    from controller import load_candidate, trusted_request
    from build_identity import verify_production_number
    from model import ID
    require(isinstance(candidate, str) and ID.fullmatch(candidate), 'Invalid RC candidate ID')
    rows = github.pages(f'pulls?state=all&base=main&head={github.repository.split("/")[0]}:codex/release/{candidate}')
    rows = [p for p in rows if trusted_request(p, github)]
    require(len(rows) == 1 and (rows[0].get('state') == 'open' or rows[0].get('merged_at')),
            'RC needs an open or merged same-repository release request')
    pr = github.api(f"pulls/{rows[0]['number']}")
    require(trusted_request(pr, github) and pr['base']['ref'] == 'main'
            and pr['head']['ref'] == f'codex/release/{candidate}', 'Untrusted RC request')
    git('fetch', 'origin', 'main', f"refs/pull/{pr['number']}/head", '--tags')
    manifest, folder = load_candidate(candidate, pr['head']['sha'], folder, draft=True)
    require(manifest['schemaVersion'] in {3, 4} and (manifest['operation'] == 'promote-ios'
            or manifest['rcRun'] in {None, linked_run}),
            'Build a fresh schema-3 request; promotion requests cannot create another RC')
    require(pr.get('state') == 'open' or manifest['rcRun'] is not None,
            'Start RC before merging the request so its final head can select the artifacts')
    git('merge-base', '--is-ancestor', manifest['sourceSha'], 'origin/main')
    verify_production_number(github, manifest)
    from state import latest_receipt
    require(all(latest_receipt(github, candidate, c) is None for c in manifest['channels']),
            'Formal publication already started; recover its original artifacts instead of building an RC')
    existing = github.api('git/ref/tags/' + manifest['tag'], missing=True)
    if manifest['operation'] == 'promote-ios':
        from controller import verify_reservation
        verify_reservation(manifest, {}, github, existing)
    else:
        require(existing is None, 'Final tag already reserved; continue formal publication instead of building an RC')
    return manifest, folder


def output(values):
    with open(os.environ['GITHUB_OUTPUT'], 'a') as file:
        for key, value in values.items():
            require('\n' not in str(value) and '\r' not in str(value), 'Invalid workflow output')
            file.write(f'{key}={value}\n')


def rc_tag(identity):
    return f"rc-{identity['candidateId']}-{identity['run']}"


def current_identity(github, manifest):
    run_id = os.environ['GITHUB_RUN_ID']
    run = trusted_run(github, run_id)
    value = read_json_artifact(github, run_id, 'rc-identity', 'rc-identity.json')
    validate_identity(value, manifest)
    require(value['run'] == run_id and value['controllerSha'] == run['head_sha'], 'RC build binding changed')
    return value


def prepare_build(github, candidate, folder):
    folder = Path(folder)
    with tempfile.TemporaryDirectory() as temp:
        manifest, original = resolve(github, candidate, Path(temp), linked_run=os.environ['GITHUB_RUN_ID'])
        require(not folder.is_symlink(), 'Unsafe RC notes directory')
        folder.mkdir(exist_ok=True, parents=True)
        for file in original.iterdir():
            require(not (folder / file.name).is_symlink(), 'Unsafe RC notes file')
            shutil.copyfile(file, folder / file.name)
    identity = current_identity(github, manifest)
    for env, key in [('SOURCE_SHA', 'sourceSha'), ('MOBILE_VERSION', 'version'), ('MOBILE_BUILD_NUMBER', 'buildNumber')]:
        require(env not in os.environ or os.environ[env] == str(identity[key] or ''), 'RC builder inputs changed')
    require(not os.environ.get('MOBILE_BUILD_ID') or os.environ['MOBILE_BUILD_ID'] == identity.get('displayBuild'),
            'RC display build ID changed')
    require(not os.environ.get('IOS_BUILD_NUMBER') or os.environ['IOS_BUILD_NUMBER'] ==
            str(identity.get('iosBuildNumber') or identity['buildNumber']), 'RC native iOS number changed')
    # RC distribution uses deterministic build-identification notes. These never
    # replace the release request's human-editable final platform notes.
    text = f"Release candidate {manifest['version']}. Build {manifest['buildNumber']}. Source {manifest['sourceSha']}."
    names = set(FILES.values()) | {'operators.zh-CN.md'}
    for name in names:
        require(not (folder / name).is_symlink(), 'Unsafe RC notes file')
        (folder / name).write_text(text + '\n')
    binding = {'contentDigest': digest(identity), 'notesDigests': {
        name: hashlib.sha256((folder / name).read_bytes()).hexdigest() for name in FILES.values()}}
    return manifest, identity, binding


def handle(command, github, candidate, folder):
    """Existing platform workflows keep one build implementation for RC and release."""
    from controller import emit_outputs
    manifest, identity, binding = prepare_build(github, candidate, folder)
    if command == 'authorize':
        emit_outputs(manifest, binding | {'approvedHead': ''})
        output({'rc_tag': rc_tag(identity), 'cycle': '', 'content_digest': binding['contentDigest']})
    elif command in {'verify-publication', 'verify-apple'}:
        return
    elif command == 'start':
        name = platform(os.environ['CHANNEL'])
        require(name in identity['platforms'], 'Platform outside selected RC scope')
        rows = [a for a in github.pages(f"actions/runs/{identity['run']}/artifacts", 'artifacts')
                if a['name'] == artifact_name(identity, name)]
        require(len(rows) <= 1 and all(not a['expired'] for a in rows), 'Original RC artifact expired/ambiguous')
        records = [a for a in github.pages(f"actions/runs/{identity['run']}/artifacts", 'artifacts')
                   if a['name'].startswith(f'rc-result-{name}-')]
        require(rows or not records,
                'Original RC artifact missing after a platform attempt; prepare a fresh candidate instead of rebuilding')
        details = {}
        if records:
            artifact = max(records, key=lambda row: int(row['name'].rsplit('-', 1)[1]))
            attempt = artifact['name'].rsplit('-', 1)[1]
            previous = read_json_artifact(github, identity['run'], artifact['name'], f'{name}-{attempt}.json')
            require(previous['identity'] == identity, 'RC retry identity changed')
            details = previous['details']
        output({'build_required': str(not rows).lower(), 'build_run_id': identity['run'],
                'reuse_rc': 'false', 'binary': details.get('binarySha256', ''),
                'image': details.get('image', ''), 'existing_build_id': details.get('buildId', '')})
    elif command == 'receipt':
        name = platform(os.environ['CHANNEL'])
        require(name in identity['platforms'], 'Platform outside RC scope')
        result_file = Path(os.environ.get('RECEIPT_DETAILS', '.release-result.json'))
        details = json.loads(result_file.read_text()) if result_file.exists() else {}
        details = details.get(os.environ['CHANNEL'], details)
        folder = Path('rc-results')
        folder.mkdir(exist_ok=True)
        attempt = int(os.environ['GITHUB_RUN_ATTEMPT'])
        (folder / f'{name}-{attempt}.json').write_text(json.dumps({
            'identity': identity, 'platform': name, 'attempt': attempt,
            'success': os.environ['RELEASE_STATE'] in {'success', 'in_progress'},
            'details': details | {'buildRunId': identity['run'], 'artifact': artifact_name(identity, name)}}) + '\n')
    else:
        raise ValueError('Unsupported RC builder command: ' + command)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=['begin', 'collect', 'check-build', 'attach'])
    parser.add_argument('--candidate', default=os.environ.get('CANDIDATE'))
    args = parser.parse_args()
    github = GitHub()
    if args.command == 'attach':
        identity = json.loads(Path('rc-record.json').read_text())['identity']
        require(identity['run'] == os.environ['GITHUB_RUN_ID'], 'Wrong RC run')
        trusted_run(github, identity['run'])
        with tempfile.TemporaryDirectory() as temp:
            manifest, _ = resolve(github, identity['candidateId'], Path(temp), linked_run=identity['run'])
        validate_identity(identity, manifest)
        prefix = 'releases/requests/' + identity['candidateId'] + '/manifest.json'
        branch = 'codex/release/' + identity['candidateId']
        blob = github.api(f'contents/{prefix}?ref={branch}')
        current = json.loads(base64.b64decode(blob['content']))
        validate_identity(identity, current)
        require(current.get('rcRun') in {None, identity['run']}, 'Another RC already selected')
        if current.get('rcRun') is None:
            current['rcRun'] = identity['run']
            content = (json.dumps(current, ensure_ascii=False, indent=2) + '\n').encode()
            github.api('contents/' + prefix, method='PUT', data={
                'message': 'chore: select original RC artifacts for ' + identity['candidateId'],
                'branch': branch, 'sha': blob['sha'], 'content': base64.b64encode(content).decode()})
        return
    if args.command == 'check-build':
        prepare_build(github, args.candidate, Path('.release-approved'))
        return
    if args.command == 'begin':
        from cycle_control import human
        for actor in {os.environ['GITHUB_ACTOR'], os.environ['GITHUB_TRIGGERING_ACTOR']}:
            human(github, actor)
        run_id = os.environ['GITHUB_RUN_ID']
        run = trusted_run(github, run_id)
        with tempfile.TemporaryDirectory() as temp:
            manifest, _ = resolve(github, args.candidate, Path(temp), linked_run=run_id)
        if manifest['operation'] == 'promote-ios':
            require(manifest['schemaVersion'] == 4 and manifest['rcRun'],
                    'Legacy existing-build promotion uses its existing recovery entry')
            read_record(github, manifest['rcRun'], manifest)
            output({'candidate': args.candidate, 'platforms': '[]', 'metadata_only': 'true', 'new_identity': 'false'})
            return
        platforms = sorted({platform(c) for c in manifest['channels']})
        value = {k: manifest[k] for k in IDENTITY_KEYS} | {
            'environment': 'production', 'preview': False, 'run': run_id,
            'controllerSha': run['head_sha'], 'platforms': platforms}
        if 'displayBuild' in manifest:
            value['displayBuild'] = manifest['displayBuild']
            value['iosBuildNumber'] = manifest['iosBuildNumber']
        # A candidate owns one RC run; adding a new build uses a new request/number.
        prefix = f"rc-{args.candidate}-"
        refs = github.pages('git/matching-refs/tags/' + prefix)
        require(all(r['ref'] == 'refs/tags/' + rc_tag(value) for r in refs),
                'This candidate already has an RC; resume that run instead of rebuilding')
        existing = github.api('git/ref/tags/' + rc_tag(value), missing=True)
        if existing:
            original = current_identity(github, manifest)
            require(original == value, 'RC rerun identity changed')
        else:
            tag = github.api('git/tags', method='POST', data={'tag': rc_tag(value),
                'object': value['sourceSha'], 'type': 'commit', 'message': json.dumps(value)})
            github.api('git/refs', method='POST', data={'ref': 'refs/tags/' + rc_tag(value), 'sha': tag['sha']})
        Path('rc-identity.json').write_text(json.dumps(value) + '\n')
        output({'candidate': args.candidate, 'source': value['sourceSha'], 'version': value['version'],
                'number': value['buildNumber'] or '', 'tag': manifest['tag'],
                'channels': json.dumps(manifest['channels']), 'platforms': json.dumps(platforms),
                'display_build': manifest.get('displayBuild', ''),
                'ios_number': manifest.get('iosBuildNumber') or str(manifest['buildNumber'] or ''),
                'new_identity': str(not existing).lower()})
    else:
        identity = read_json_artifact(github, os.environ['GITHUB_RUN_ID'], 'rc-identity', 'rc-identity.json')
        results = {}
        for file in Path('rc-results').glob('*.json'):
            row = json.loads(file.read_text())
            require(row['identity'] == identity, 'Mixed RC result identities')
            name = row['platform']
            if name not in results or row['attempt'] > results[name]['attempt']:
                results[name] = row
        require(set(results) == set(identity['platforms']) and all(r['success'] for r in results.values()),
                'Every selected RC platform must finish; retry only failed jobs')
        record = {'identity': identity, 'platforms': {name: row['details'] for name, row in results.items()}}
        for name in results:
            reusable(record, name)
        Path('rc-record.json').write_text(json.dumps(record) + '\n')
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as file:
            file.write(f"RC `{identity['candidateId']}` ready. Set manifest `rcRun` to `{identity['run']}` "
                       "in the release request to publish these exact artifacts. No stable release was published.\n")


if __name__ == '__main__':
    main()
