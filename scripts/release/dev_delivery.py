#!/usr/bin/env python3
"""Continuous dev delivery. No release owner, freeze, QA admission or expiry."""
import argparse
import json
import os
from pathlib import Path
import re
import tempfile
import time

from build_identity import reserve_display_id, reserve_number
from cycle_control import human, output
from github import GitHub, git, run
from model import SHA, next_identity, require
from rc import read_json_artifact
from state import reservations

WORKFLOW = '.github/workflows/dev-distribute.yml'


def admitted_request(request, repository, source):
    return (request.get('path') == '.github/workflows/dev-request.yml' and request.get('event') == 'push'
            and request.get('head_branch') == 'dev' and request.get('head_sha') == source
            and request.get('head_repository', {}).get('full_name', '').lower() == repository.lower())


def current_source(github, source):
    require(SHA.fullmatch(source or ''), 'Dev delivery needs a full source SHA')
    return github.api('git/ref/heads/dev')['object']['sha'] == source


def trusted_run(github, run_id):
    require(str(run_id).isdigit(), 'Invalid dev distribution run')
    value = github.api(f'actions/runs/{run_id}')
    require(value.get('path') == WORKFLOW and value.get('event') == 'workflow_dispatch'
            and value.get('head_branch') == 'main'
            and value.get('head_repository', {}).get('full_name', '').lower() == github.repository.lower(),
            'Dev signing must use the trusted-main distribution workflow')
    return value


def identity(github):
    run_id = os.environ['GITHUB_RUN_ID']
    value = read_json_artifact(github, run_id, 'dev-identity', 'dev-identity.json')
    control = trusted_run(github, run_id)
    require(value['run'] == run_id and value['controllerSha'] == control['head_sha']
            and value['source'] == os.environ['SOURCE_SHA'] and value['environment'] == 'dev',
            'Dev artifact source/run/config changed')
    for env, field in [('MOBILE_VERSION', 'version'), ('MOBILE_BUILD_NUMBER', 'number'), ('MOBILE_BUILD_ID', 'displayBuild')]:
        require(env not in os.environ or os.environ[env] == str(value[field]), 'Dev build inputs changed')
    require(not os.environ.get('IOS_BUILD_NUMBER') or os.environ['IOS_BUILD_NUMBER'] == value.get('iosNumber'),
            'Native dev iOS build number changed')
    return value


def project_status(source, platforms, now):
    return {'schema': 2, 'updatedAt': now, 'source': source, 'platforms': platforms}


def latest_platforms(github):
    platforms = {}
    for name in ['web', 'android', 'ios']:
        for item in github.pages(f'deployments?environment=dev-{name}'):
            payload = item.get('payload') or {}
            if payload.get('kind') != 'continuous-dev':
                continue
            status = github.pages(f"deployments/{item['id']}/statuses")
            if status and status[0]['state'] == 'success':
                platforms[name] = payload
                break
    return platforms


def publish_status(github):
    source = github.api('git/ref/heads/dev')['object']['sha']
    status = project_status(source, latest_platforms(github), int(time.time()))
    lines = ['# Development builds', '', f'Current dev source: `{source}`', '',
             'Web and mobile follow dev automatically. Platform builds finish independently; these are development packages.']
    for name, data in status['platforms'].items():
        lines += ['', f"## {name}", f"Source: `{data['source']}` · build {data.get('displayBuild', 'web')}",
                  f"[Distribution run](https://github.com/{github.repository}/actions/runs/{data['run']})"]
        if name == 'android':
            lines.append(f"[Signed APK downloads](https://github.com/{github.repository}/releases/tag/{data['tag']})")
        elif name == 'ios':
            lines.append('Install from the configured internal TestFlight group.')
    lines += ['', 'Development builds connect to dev.yourtj.de. Production RCs and formal releases use separate artifacts.']
    release = github.api('releases/tags/dev-latest', missing=True)
    data = {'name': 'Latest development builds', 'body': '\n'.join(lines), 'prerelease': True,
            'draft': False, 'make_latest': 'false'}
    if release:
        require(release['prerelease'] and not release['draft'], 'Dev channel must remain isolated from stable releases')
        github.api(f"releases/{release['id']}", method='PATCH', data=data)
    else:
        github.api('releases', method='POST', data=data | {'tag_name': 'dev-latest', 'target_commitish': source})
    with tempfile.TemporaryDirectory() as temp:
        path = Path(temp) / 'dev-status.json'
        path.write_text(json.dumps(status) + '\n')
        run('gh', 'release', 'upload', 'dev-latest', str(path), '--repo', github.repository, '--clobber')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=['admit', 'identity', 'start', 'check-build', 'current', 'record', 'status'])
    args = parser.parse_args()
    github = GitHub()
    source = os.environ.get('SOURCE_SHA', '')
    if args.command == 'admit':
        control = trusted_run(github, os.environ['GITHUB_RUN_ID'])
        source = source or github.api('git/ref/heads/dev')['object']['sha']
        request = os.environ.get('REQUEST_RUN', '')
        if request:
            require(request.isdigit() and admitted_request(github.api(f'actions/runs/{request}'), github.repository, source),
                    'Automatic dev delivery needs the matching same-repository dev push request')
        else:
            for actor in {os.environ['GITHUB_ACTOR'], os.environ.get('GITHUB_TRIGGERING_ACTOR', os.environ['GITHUB_ACTOR'])}:
                human(github, actor)
        output({'source': source, 'eligible': str(current_source(github, source)).lower(), 'controller': control['head_sha']})
    elif args.command == 'identity':
        require(current_source(github, source), 'Dev mobile build was superseded before allocation')
        run_id = os.environ['GITHUB_RUN_ID']
        control = trusted_run(github, run_id)
        artifacts = github.pages(f'actions/runs/{run_id}/artifacts', 'artifacts')
        if any(a['name'] == 'dev-identity' for a in artifacts):
            value = identity(github)
            fresh = False
        else:
            git('fetch', 'origin', source, '--tags')
            pubspec = git('show', f'{source}:apps/mobile/packages/forum_app/pubspec.yaml')
            floor = re.search(r'(?m)^version:\s*(\d+\.\d+\.\d+)\+([1-9]\d*)\s*$', pubspec)
            require(floor, 'Mobile version/build floor missing')
            version, number = next_identity('mobile', 'patch', reservations(), floor[1], int(floor[2]))
            display = reserve_display_id(github, source, 'dev')
            reserve_number(github, source, version, number, 'dev-preview', display_build=display, ios_build_number=display)
            value = {'source': source, 'version': version, 'number': number, 'displayBuild': display,
                     'iosNumber': display, 'environment': 'dev', 'tag': 'dev-' + display, 'run': run_id, 'controllerSha': control['head_sha']}
            tag = github.api('git/tags', method='POST', data={'tag': value['tag'], 'object': source, 'type': 'commit',
                                                           'message': json.dumps(value)})
            github.api('git/refs', method='POST', data={'ref': 'refs/tags/' + value['tag'], 'sha': tag['sha']})
            fresh = True
        Path('dev-identity.json').write_text(json.dumps(value) + '\n')
        output({key: value[key] for key in ['version', 'number', 'displayBuild', 'iosNumber', 'tag']} | {'fresh': str(fresh).lower()})
    elif args.command == 'start':
        value = identity(github)
        name = os.environ['PREVIEW_ARTIFACT']
        require(name in {'android', 'ios'}, 'Invalid dev platform')
        artifacts = github.pages(f"actions/runs/{value['run']}/artifacts", 'artifacts')
        originals = [a for a in artifacts if a['name'] == f"preview-{name}-{value['version']}-{value['number']}"]
        require(len(originals) <= 1 and all(not a['expired'] for a in originals), 'Original dev archive expired/ambiguous')
        build_id = ''
        if name == 'android' and not originals:
            require(github.api('releases/tags/' + value['tag'], missing=True) is None,
                    'Dev APKs were already distributed; restore their original archive or start a new dev build')
        if name == 'ios':
            records = [a for a in artifacts if a['name'].startswith('preview-apple-status-')]
            if records:
                item = max(records, key=lambda a: int(a['name'].rsplit('-', 1)[1]))
                previous = read_json_artifact(github, value['run'], item['name'], '.preview-ios.json')
                require(previous['source'] == source and previous['environment'] == 'dev', 'Dev Apple receipt changed')
                build_id = previous.get('buildId', '')
                require(originals or build_id, 'Original dev IPA missing; use a new build instead of recompiling')
        output({'build_required': str(not originals and not build_id).lower(), 'existing_build_id': build_id})
    elif args.command == 'check-build':
        identity(github)  # An admitted in-flight upload may safely finish after a newer push.
    elif args.command == 'current':
        output({'eligible': str(current_source(github, source)).lower()})
    elif args.command == 'record':
        name = os.environ['DEV_PLATFORM']
        require(name in {'web', 'android', 'ios'}, 'Invalid development platform')
        value = {'source': source, 'run': os.environ['GITHUB_RUN_ID']}
        if name != 'web':
            value |= identity(github)
        deployment = github.api('deployments', method='POST', data={
            'ref': source, 'environment': 'dev-' + name, 'auto_merge': False, 'required_contexts': [],
            'production_environment': False, 'payload': value | {'kind': 'continuous-dev'}})
        github.api(f"deployments/{deployment['id']}/statuses", method='POST', data={
            'state': 'success', 'auto_inactive': False,
            'log_url': f"https://github.com/{github.repository}/actions/runs/{value['run']}"})
    else:
        publish_status(github)


if __name__ == '__main__': main()
