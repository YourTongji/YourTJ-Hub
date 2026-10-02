#!/usr/bin/env python3
"""Replay known mobile source ranges into Oryn input; no release/tag/Apple mutations."""
import json
from pathlib import Path
from collect import collect
from controller import prepare, write_json
from github import GitHub, git
from notes import render
import argparse


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--render', action='store_true')
    args = parser.parse_args()
    folder = Path('.release-smoke')
    if args.render:
        manifest = json.loads((folder / 'candidate/manifest.json').read_text(encoding='utf-8'))
        response = json.loads((folder / 'output.json').read_text(encoding='utf-8'))
        render(manifest, folder / 'candidate', response, (folder / 'input.json').read_bytes())
        print('Live Oryn replay rendered independent drafts. Human review and all publishing remain disabled.')
        return
    base = git('rev-parse', 'mobile-v1.0.13^{commit}')
    source = git('rev-parse', 'mobile-v1.0.14^{commit}')
    manifest = {'schemaVersion': 1, 'candidateId': 'mobile-1.0.14-14', 'product': 'mobile',
                'sourceSha': source, 'version': '1.0.14', 'buildNumber': 14, 'tag': 'mobile-v1.0.14',
                'channels': ['android', 'ios-testflight', 'ios-app-store'], 'operation': 'release', 'existingRelease': None,
                'baselines': {c: {'tag': 'mobile-v1.0.13', 'sourceSha': base} for c in ['android','ios-testflight','ios-app-store']},
                'notes': {'android': 'android.zh-CN.md', 'ios-testflight': 'testflight.en-US.txt', 'ios-app-store': 'ios.zh-Hans.txt'},
                'serverRequirement': None, 'requiredDisclosures': []}
    # This is a replay comparison, not a claim that this was Apple's historical live baseline.
    request = prepare(manifest, GitHub(), folder / 'candidate')
    write_json(folder / 'input.json', request)
    print('Read-only historical source replay collected; Apple distribution history is not inferred.')


if __name__ == '__main__':
    main()
