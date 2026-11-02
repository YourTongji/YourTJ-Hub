#!/usr/bin/env python3
"""One explicit operator confirmation seals the RC and final notes before promotion."""
import argparse
import json
import os
from pathlib import Path
import tempfile

from cycle_control import human, output
from github import GitHub, git
from model import require, validate_request_paths
from rc import read_json_artifact, read_record, reusable
from state import latest_receipt

WORKFLOW = '.github/workflows/release-confirm.yml'
FIELDS = ('candidateId', 'sourceSha', 'version', 'buildNumber', 'displayBuild', 'iosBuildNumber', 'rcRun', 'channels')


def validate_seal(seal, manifest, head, content_digest, control, run_id, repository):
    require(control.get('path') == WORKFLOW and control.get('event') == 'workflow_dispatch'
            and control.get('head_branch') == 'main'
            and control.get('head_repository', {}).get('full_name', '').lower() == repository.lower(),
            'Confirmation must come from the trusted-main Confirm workflow')
    require(seal.get('schemaVersion') == 1 and seal.get('repository') == repository
            and str(seal.get('run')) == str(run_id) and seal.get('controllerSha') == control['head_sha']
            and seal.get('actor') == control.get('actor', {}).get('login'), 'Invalid confirmation provenance')
    require(all(seal.get(k) == manifest.get(k) for k in FIELDS)
            and seal.get('head') == head and seal.get('contentDigest') == content_digest,
            'Confirmation differs from this RC/source/notes/head/scope; confirm the new snapshot')


def read_confirmation(github, manifest, pr, content_digest):
    for row in github.pages('deployments?environment=release-confirmation'):
        pointer = row.get('payload') or {}
        if pointer.get('candidateId') != manifest['candidateId'] or pointer.get('head') != pr['head']['sha']:
            continue
        run_id, attempt = str(pointer.get('run', '')), pointer.get('attempt')
        require(run_id.isdigit() and type(attempt) is int and attempt > 0, 'Invalid confirmation pointer')
        control = github.api('actions/runs/' + run_id)
        seal = read_json_artifact(github, run_id, f'release-confirmation-{attempt}', 'confirmation.json')
        validate_seal(seal, manifest, pr['head']['sha'], content_digest, control, run_id, github.repository)
        require(seal.get('attempt') == attempt, 'Confirmation attempt mismatch')
        for actor in {seal['actor'], seal['triggeringActor']}:
            human(github, actor)
        return {'approvedHead': pr['head']['sha'], 'mergeSha': pr.get('merge_commit_sha'),
                'reviewIds': [], 'reviewers': [seal['actor']], 'confirmationRun': run_id}
    raise ValueError('Use Release / Confirm for this exact RC and final release-note head')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=['seal', 'record'])
    parser.add_argument('--candidate', default=os.environ.get('CANDIDATE'))
    args = parser.parse_args()
    github = GitHub()
    if args.command == 'record':
        seal = json.loads(Path('confirmation.json').read_text())
        require(seal['run'] == os.environ['GITHUB_RUN_ID'], 'Wrong confirmation run')
        deployment = github.api('deployments', method='POST', data={
            'ref': seal['sourceSha'], 'environment': 'release-confirmation', 'auto_merge': False,
            'required_contexts': [], 'production_environment': False,
            'payload': {k: seal[k] for k in ['candidateId', 'head', 'run', 'attempt']}})
        github.api(f"deployments/{deployment['id']}/statuses", method='POST', data={
            'state': 'success', 'auto_inactive': False,
            'log_url': f"https://github.com/{github.repository}/actions/runs/{seal['run']}"})
        return
    from controller import load_candidate, trusted_request, candidate_binding
    from build_identity import verify_production_number
    for actor in {os.environ['GITHUB_ACTOR'], os.environ.get('GITHUB_TRIGGERING_ACTOR', os.environ['GITHUB_ACTOR'])}:
        human(github, actor)
    control = github.api('actions/runs/' + os.environ['GITHUB_RUN_ID'])
    rows = github.pages(f'pulls?state=open&base=main&head={github.repository.split("/")[0]}:codex/release/{args.candidate}')
    rows = [p for p in rows if trusted_request(p, github)]
    require(len(rows) == 1, 'Confirm needs the open prepared RC request')
    pr = github.api(f"pulls/{rows[0]['number']}")
    require(trusted_request(pr, github), 'Foreign release request')
    validate_request_paths(pr, github.files(pr['number'], pr['changed_files']), args.candidate, False)
    git('fetch', 'origin', 'main', f"refs/pull/{pr['number']}/head", '--tags')
    with tempfile.TemporaryDirectory() as temp:
        manifest, folder = load_candidate(args.candidate, pr['head']['sha'], Path(temp))
        require(manifest['schemaVersion'] == 4 and manifest['rcRun'], 'New releases confirm a production RC; direct builds are disabled')
        git('merge-base', '--is-ancestor', manifest['sourceSha'], 'origin/main')
        verify_production_number(github, manifest)
        rc = read_record(github, manifest['rcRun'], manifest)
        for channel in manifest['channels']:
            reusable(rc, channel)
            require(latest_receipt(github, args.candidate, channel) is None,
                    'Publication already started; recover the original artifacts instead of confirming again')
        checks = github.pages(f"commits/{pr['head']['sha']}/check-runs", 'check_runs')
        gate = [c for c in checks if c['name'] == 'ci-required' and c.get('app', {}).get('slug') == 'github-actions']
        require(gate and max(gate, key=lambda c: c['id'])['conclusion'] == 'success', 'The RC request needs successful metadata CI')
        binding = candidate_binding(manifest, folder, {'approvedHead': pr['head']['sha']})
    seal = {k: manifest[k] for k in FIELDS} | {
        'schemaVersion': 1, 'repository': github.repository, 'head': pr['head']['sha'],
        'contentDigest': binding['contentDigest'], 'actor': os.environ['GITHUB_ACTOR'],
        'triggeringActor': os.environ.get('GITHUB_TRIGGERING_ACTOR', os.environ['GITHUB_ACTOR']),
        'run': os.environ['GITHUB_RUN_ID'], 'attempt': int(os.environ['GITHUB_RUN_ATTEMPT']),
        'controllerSha': control['head_sha']}
    validate_seal(seal, manifest, pr['head']['sha'], binding['contentDigest'], control, seal['run'], github.repository)
    Path('confirmation.json').write_text(json.dumps(seal) + '\n')
    output({'pr': pr['number'], 'head': pr['head']['sha']})
    with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as file:
        file.write(f"Confirmed `{args.candidate}` / `{manifest['displayBuild']}`. RC run `{manifest['rcRun']}`. "
                   f"Notes/head `{pr['head']['sha']}`. Promotion uses the original artifacts.\n")


if __name__ == '__main__': main()
