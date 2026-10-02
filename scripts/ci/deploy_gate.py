#!/usr/bin/env python3
"""A privileged workflow_run consumer admits only current, successful dev push CI."""
import json
import os
from pathlib import Path
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'release'))
from github import GitHub
from model import require


def admitted(run, repository, current_sha):
    return (run.get('event') == 'push' and run.get('conclusion') == 'success'
            and run.get('head_branch') == 'dev' and run.get('head_sha') == current_sha
            and run.get('head_repository', {}).get('full_name', '').lower() == repository.lower()
            and run.get('name') == 'CI / Verify' and run.get('path') == '.github/workflows/ci.yml')


def main():
    github = GitHub()
    event = json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text())
    run = github.api(f"actions/runs/{event['workflow_run']['id']}")
    current = github.api('git/ref/heads/dev')['object']['sha']
    eligible = admitted(run, github.repository, current)
    if os.environ.get('GITHUB_OUTPUT'):
        with open(os.environ['GITHUB_OUTPUT'], 'a') as out:
            out.write(f"eligible={str(eligible).lower()}\nsha={run['head_sha']}\n")
    if not eligible:
        print('No deployment: obsolete, failed or non-dev-push CI run.')


if __name__ == '__main__':
    main()
