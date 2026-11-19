"""Restore only this run's preserved APP Test intent, never infer a missing upload."""
import json
import os
from pathlib import Path

from github import GitHub
from model import require
from rc import read_json_artifact


def main():
    github = GitHub()
    run = os.environ['RECEIPT_RUN_ID']
    require(run.isdigit(), 'Invalid original receipt run')
    rows = [a for a in github.pages(f'actions/runs/{run}/artifacts', 'artifacts')
            if a['name'].startswith('harmony-upload-')]
    if rows:
        latest = max(rows, key=lambda a: int(a['name'].rsplit('-', 1)[1]))
        record = read_json_artifact(github, run, latest['name'], '.harmony-upload.json')
        require(record.get('sourceSha') == os.environ['SOURCE_SHA'], 'Upload source changed')
        Path('.harmony-upload.json').write_text(json.dumps(record) + '\n')
    else:
        require(os.environ.get('RECOVER') != 'true' and os.environ.get('GITHUB_RUN_ATTEMPT') == '1',
                'Upload outcome unknown without its original receipt; reconcile before recovery')


if __name__ == '__main__': main()
