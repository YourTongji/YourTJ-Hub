#!/usr/bin/env python3
"""Publish archives without overwriting different bytes or regenerating notes."""
import hashlib
import json
import os
from pathlib import Path
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'mobile-release'))
from publish_android import gh, find_release
from approved_notes import read_notes


def main():
    tag = os.environ['RELEASE_TAG']
    notes = Path('.release-approved/web.zh-CN.md')
    os.environ['WEB_NOTES_PATH'] = str(notes)
    text = read_notes('WEB_NOTES_PATH')
    release = find_release(tag)
    if release is None:
        gh('release', 'create', tag, '--verify-tag', '--draft', '--title', tag, '--notes-file', str(notes))
        release = find_release(tag)
    if not release or release.get('body', '').strip() != text:
        raise ValueError('Existing web release differs from approved notes')
    existing = {a['name']: a for a in release['assets']}
    assets = [p for p in Path('.source/apps/gooseforum/dist').iterdir() if p.name.endswith(('.tar.gz', '.zip', 'checksums.txt'))]
    if len(assets) != 7:
        raise ValueError('Expected six platform archives plus checksums')
    for path in assets:
        digest = 'sha256:' + hashlib.sha256(path.read_bytes()).hexdigest()
        if path.name in existing:
            if existing[path.name].get('digest') != digest:
                raise ValueError('Refusing to overwrite web archive: ' + path.name)
        else:
            gh('release', 'upload', tag, str(path))
    uploaded = {a['name']: a for a in find_release(tag)['assets']}
    for path in assets:
        if uploaded[path.name].get('digest') != 'sha256:' + hashlib.sha256(path.read_bytes()).hexdigest():
            raise ValueError('GitHub archive digest not confirmed; retry original artifacts')
    gh('release', 'edit', tag, '--draft=false', '--latest')


if __name__ == '__main__':
    main()
