#!/usr/bin/env python3
"""Replay historical source through Prepare's draft path without publication."""
import json
from pathlib import Path
from controller import prepare, write_json
from github import GitHub, git
from model import require, validate_candidate
import argparse


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--verify', action='store_true')
    parser.add_argument('--scope', choices=['mobile', 'ios', 'web'], default='mobile')
    args = parser.parse_args()
    folder = Path('.release-smoke')
    if args.verify:
        manifest = json.loads((folder / 'candidate/manifest.json').read_text(encoding='utf-8'))
        validate_candidate(manifest, folder / 'candidate')
        status = json.loads((folder / 'notes-status.json').read_text(encoding='utf-8'))
        assert status['status'] == 'complete', 'The smoke must cover every requested channel'
        print('Oryn draft passes candidate validation. This is not human approval; no publication performed.')
        return
    base = git('rev-parse', 'mobile-v1.0.13^{commit}')
    source = git('rev-parse', 'mobile-v1.0.14^{commit}')
    manifest = {'schemaVersion': 2, 'candidateId': 'mobile-1.0.14-14', 'product': 'mobile',
                'sourceSha': source, 'version': '1.0.14', 'buildNumber': 14, 'tag': 'mobile-v1.0.14',
                'channels': ['android', 'ios-testflight', 'ios-app-store'], 'operation': 'release', 'existingRelease': None,
                'baselines': {c: {'tag': 'mobile-v1.0.13', 'sourceSha': base} for c in ['android','ios-testflight','ios-app-store']},
                'notes': {'android': 'android.zh-CN.md', 'ios-testflight': 'testflight.en-US.txt', 'ios-app-store': 'ios.zh-Hans.txt'},
                'serverRequirement': None, 'requiredDisclosures': []}
    if args.scope == 'ios':
        manifest['channels'] = ['ios-testflight']
        manifest['notes'] = {'ios-testflight': 'testflight.en-US.txt'}
        manifest['baselines'] = {'ios-testflight': manifest['baselines']['ios-testflight']}
    elif args.scope == 'web':
        # This range includes forum UI changes and operator evidence; status-only releases do not.
        manifest.update(candidateId='web-0.0.46', product='web', version='0.0.46', buildNumber=None,
                        tag='v0.0.46', sourceSha=git('rev-parse', 'v0.0.46^{commit}'), channels=['web'],
                        notes={'web': 'web.zh-CN.md'}, baselines={'web': {
                            'tag': 'v0.0.45', 'sourceSha': git('rev-parse', 'v0.0.45^{commit}')}})
    # This is a replay comparison, not a claim that this was Apple's historical live baseline.
    request = prepare(manifest, GitHub(), folder / 'candidate')
    require(all(any(channel in entry['channels'] for entry in request['evidence']) for channel in request['channels']),
            'Smoke fixture must contain evidence for every requested channel')
    write_json(folder / 'oryn-input.json', request)
    print('Read-only historical source replay collected; Apple distribution history is not inferred.')


if __name__ == '__main__':
    main()
