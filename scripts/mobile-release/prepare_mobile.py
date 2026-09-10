#!/usr/bin/env python3
"""Reserve patch/minor/major releases on reviewed main; tags own build identity."""
import json
import os
import re
import subprocess

from prepare_server import ensure_promoted, read

REPOSITORY = 'YourTongji/YourTJ-Hub'
PUBSPEC = 'apps/mobile/packages/forum_app/pubspec.yaml'
TAG = re.compile(r'mobile-v(\d+\.\d+\.\d+)')
MAX_BUILD = 2099996000  # Leave room for Flutter's largest ABI offset.


def parse_pubspec(text):
    match = re.search(r'^version: (\d+\.\d+\.\d+)\+([1-9]\d*)$', text, re.M)
    if not match or int(match[2]) > MAX_BUILD:
        raise ValueError('Invalid pubspec version/build floor')
    return match[1], int(match[2])


def parse_annotation(tag, message):
    data = json.loads(message)
    if not isinstance(data, dict):
        raise ValueError('Invalid release annotation')
    version, number = data.get('version'), data.get('buildNumber')
    if (data.get('schema') != 1 or not TAG.fullmatch(tag) or
            tag != f'mobile-v{version}' or type(number) is not int or not 1 <= number <= MAX_BUILD):
        raise ValueError('Tag annotation does not contain a valid matching release identity')
    return version, number


def next_release(floor, history, bump):
    versions = [floor, *history]
    major, minor, patch = max(tuple(map(int, version.split('.'))) for version, _ in versions)
    if bump == 'patch': patch += 1
    elif bump == 'minor': minor, patch = minor + 1, 0
    elif bump == 'major': major, minor, patch = major + 1, 0, 0
    else: raise ValueError('Expected patch, minor or major')
    number = max(number for _, number in versions) + 1
    if number > MAX_BUILD:
        raise ValueError('Mobile build number exhausted')
    return f'{major}.{minor}.{patch}', number


def tag_identity(tag):
    if not TAG.fullmatch(tag):
        raise ValueError('Expected an existing mobile-vX.Y.Z tag')
    ref = f'refs/tags/{tag}'
    sha = read('git', 'rev-parse', ref + '^{commit}')
    subprocess.run(['git', 'merge-base', '--is-ancestor', sha, 'origin/main'], check=True)
    if read('git', 'cat-file', '-t', ref) == 'tag':
        version, number = parse_annotation(tag, read('git', 'for-each-ref', '--format=%(contents)', ref))
    else:
        # Recovery compatibility for manually created lightweight tags from the old workflow.
        version, number = parse_pubspec(read('git', 'show', f'{sha}:{PUBSPEC}'))
        if tag != f'mobile-v{version}':
            raise ValueError('Legacy lightweight tag must match its committed pubspec')
    return sha, version, number


def main():
    bump = os.environ.get('BUMP', 'patch')
    resume = os.environ.get('RESUME_TAG', '').strip()
    recover_ios = os.environ.get('RECOVER_IOS') == 'true'
    testflight_only = os.environ.get('TESTFLIGHT_ONLY') == 'true'
    if recover_ios and not resume:
        raise ValueError('iOS recovery requires an existing release tag')
    if bump not in {'patch', 'minor', 'major'} or (resume and not TAG.fullmatch(resume)):
        raise ValueError('Invalid release input')
    if os.environ['GITHUB_REF'] not in {'refs/heads/main', 'refs/heads/dev'}:
        raise ValueError('Dispatch releases only from main or dev')
    subprocess.run(['git', 'fetch', 'origin', 'dev', 'main', '--tags'], check=True)
    if resume:
        tag_identity(resume)  # Reject unknown/unreviewed tags before any dispatch.
    elif not ensure_promoted():
        return
    if os.environ['GITHUB_REF'] == 'refs/heads/dev':
        # Signing secrets are intentionally restricted to main. Dispatch is non-blocking;
        # this preparation run exits before the serialized main run acquires the group.
        args = ['gh', 'workflow', 'run', 'release-mobile.yml', '--ref', 'main', '-f', f'bump={bump}']
        if resume: args.extend(['-f', f'tag={resume}'])
        if recover_ios: args.extend(['-f', 'recover_ios=true'])
        if testflight_only: args.extend(['-f', 'testflight_only=true'])
        read(*args)
        print('Mobile release dispatched on main; follow its Release / mobile run.')
        return
    if resume:
        tag = resume
        sha, version, number = tag_identity(tag)
    else:
        sha = read('git', 'rev-parse', 'origin/main')
        tags = [tag for tag in read('git', 'tag', '--list', 'mobile-v*').splitlines() if TAG.fullmatch(tag)]
        history = []
        for tag in tags:
            previous_sha, version, number = tag_identity(tag)
            if previous_sha == sha:
                raise ValueError(f'This main commit already has {tag}; use the optional tag field to resume it')
            history.append((version, number))
        floor = parse_pubspec(read('git', 'show', f'{sha}:{PUBSPEC}'))
        version, number = next_release(floor, history, bump)
        tag = f'mobile-v{version}'
        message = json.dumps({'schema': 1, 'version': version, 'buildNumber': number})
        annotation = json.loads(read('gh', 'api', '--method', 'POST', f'repos/{REPOSITORY}/git/tags',
                                     '-f', f'tag={tag}', '-f', f'message={message}', '-f', f'object={sha}', '-f', 'type=commit'))
        print(f'Reserving {tag}, build {number}, source {sha}', flush=True)
        read('gh', 'api', '--method', 'POST', f'repos/{REPOSITORY}/git/refs',
             '-f', f'ref=refs/tags/{tag}', '-f', f'sha={annotation["sha"]}')
    with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
        output.write(f'sha={sha}\nversion={version}\nnumber={number}\ntag={tag}\n')
    print(f'Release identity: {tag}, build {number}')


if __name__ == '__main__':
    main()
