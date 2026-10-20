#!/usr/bin/env python3
"""Trusted-main release-cycle controller. All writes run under instance-mutate-dev."""
import argparse
import json
import os
from pathlib import Path
import time
from cycle import current, transition, can_merge, authorize_source, preview_artifacts, required_artifacts
from cycle_store import load, save
from github import GitHub, git
from model import require, SHA, ReleaseError
from state import reservations, baselines


def human(github, actor, *, admin=False):
    # Repository permission includes the identity; do not infer authority from labels.
    permission = github.api(f'collaborators/{actor}/permission')
    require(permission.get('permission') in {'admin', 'maintain', 'write'}
            and permission.get('user', {}).get('type') == 'User', 'A human maintainer must initiate this action')
    require(not admin or permission['permission'] == 'admin', 'Only a repository admin may perform this action')


def fetch_source(source):
    require(SHA.fullmatch(source or ''), 'Use a full commit SHA')
    git('fetch', 'origin', 'main', 'dev', source, '--tags')
    require(git('rev-parse', f'{source}^{{commit}}') == source, 'Source is not a commit')


def includes_hotfixes(state, source):
    for hotfix in state['requiredHotfixes']:
        git('merge-base', '--is-ancestor', hotfix, source)


def reviewed_main_pr(github, source):
    """Preserve source ancestry and reject hand-edited/conflict-resolved merge trees."""
    git('merge-base', '--is-ancestor', source, 'origin/main')
    prs = github.pages(f'commits/{source}/pulls')
    prs = [p for p in prs if p.get('merged_at') and p['base']['ref'] == 'main'
           and p['head']['sha'] == source and p['head']['repo']['full_name'].lower() == github.repository.lower()]
    require(prs, 'The exact source must enter main through a reviewed PR preserving ancestry')
    pr = max(prs, key=lambda p: p['number'])
    reviews = github.pages(f"pulls/{pr['number']}/reviews")
    latest = {}
    for review in reviews:
        if review.get('state') != 'COMMENTED':
            latest[review['user']['login']] = review
    maintainers = github.maintainers(reviews)
    require(any(r['state'] == 'APPROVED' and r['commit_id'] == source and user in maintainers
                and user != pr['user']['login'] for user, r in latest.items()),
            'An eligible human must approve the exact source PR head')
    merge = pr['merge_commit_sha']
    parents = git('show', '-s', '--format=%P', merge).split()
    require(source in parents and len(parents) == 2, 'Use a merge commit that preserves the candidate SHA')
    expected_tree = git('merge-tree', '--write-tree', parents[0], source).splitlines()[0]
    require(expected_tree == git('show', '-s', '--format=%T', merge),
            'Integration changed code: form and test a new candidate')


def reviewed_preview_source(github, source):
    prs = github.pages(f'commits/{source}/pulls')
    prs = [p for p in prs if p['base']['ref'] in {'dev', 'main'}
           and p.get('head', {}).get('repo', {})
           and p['head']['repo']['full_name'].lower() == github.repository.lower()
           and (p['head']['sha'] == source or p.get('merged_at') and p.get('merge_commit_sha') == source)]
    for pr in prs:
        reviews = github.pages(f"pulls/{pr['number']}/reviews")
        latest = {}
        for r in reviews:
            if r.get('state') != 'COMMENTED':
                latest[r['user']['login']] = r
        eligible = github.maintainers(reviews)
        if any(r['state'] == 'APPROVED' and r.get('commit_id') == pr['head']['sha']
               and who in eligible and who != pr['user']['login'] for who, r in latest.items()):
            return
    require(False, 'Preview source needs an eligible human review of its exact PR head before signing')


def record_preview_bypass(github, cycle, *, run, purpose, actor, triggering_actor, reason, now):
    reason = reason.strip()
    require(purpose == 'dev' and 0 < len(reason) <= 1000,
            'Source-review bypass needs a reason (1-1000 characters) and is limited to dev previews')
    human(github, actor, admin=True)
    human(github, triggering_actor, admin=True)
    cycle.setdefault('previewReviewBypasses', {})[run] = {
        'cycle': cycle['id'], 'source': cycle['source'], 'run': run, 'purpose': purpose,
        'actor': actor, 'triggeringActor': triggering_actor, 'reason': reason, 'at': now}


def authorize_preview_source(github, cycle, purpose, run):
    grant = cycle.get('previewReviewBypasses', {}).get(run)
    if not grant:
        return reviewed_preview_source(github, cycle['source'])
    require(purpose == 'dev' and grant['purpose'] == purpose and grant['cycle'] == cycle['id']
            and grant['source'] == cycle['source'] and grant['run'] == run
            and grant['actor'] == os.environ['GITHUB_ACTOR'] and grant['reason'].strip(),
            'Source-review bypass does not authorize this preview source/cycle/run')
    human(github, grant['actor'], admin=True)
    human(github, os.environ.get('GITHUB_TRIGGERING_ACTOR', grant['actor']), admin=True)


def production_gate(github, source, channels, manifest=None, binding=None, promotion=False):
    state = load(github)
    # Closed QA can recover an already reserved formal release, not allocate a
    # fresh version from an old source without another freeze/regression cycle.
    reserved = bool(manifest and github.api(f"git/ref/tags/{manifest['tag']}", missing=True))
    if reserved and binding and not any(c['source'] == source for c in state['cycles'].values()):
        # A release-data merge predating this controller is immutable historical
        # evidence. Absence of a cycle alone is NEVER a legacy exemption.
        from controller import authorize, verify_reservation
        original, approval = manifest, binding
        if manifest['operation'] == 'promote-ios':
            import tempfile
            with tempfile.TemporaryDirectory() as folder:
                original, approval = authorize(manifest['existingRelease']['candidateId'], github, Path(folder) / 'original')
            require(channels == ['ios-app-store'] and original['sourceSha'] == source,
                    'Historical promotion must retain the original source and Apple build')
        elif promotion:
            require(channels == ['ios-app-store'] and 'ios-testflight' in original['channels'],
                    'Historical promotion requires an original TestFlight release')
        else:
            require(set(channels) <= set(original['channels']), 'Historical recovery cannot add a channel')
        fetch_source(source)
        pr = github.api(f"pulls/{approval['pr']}")
        merge = pr.get('merge_commit_sha')
        require(merge and not git('ls-tree', merge, '--', 'scripts/release/cycle.py'),
                'This release was prepared under cycle rules; missing QA cannot use historical recovery')
        require(not state.get('pendingHotfixes') and (not current(state) or current(state)['kind'] != 'hotfix'),
                'Hotfix owns publication; finish its backflow before historical recovery')
        includes_hotfixes(state, source)
        verify_reservation(original, approval, github)
        from build_identity import verify_production_number
        verify_production_number(github, original)
        return None
    cycle = authorize_source(state, source, channels, int(time.time()), allow_closed=reserved)
    if cycle.get('completion'):
        request_id = manifest['candidateId'] if manifest else None
        if manifest and manifest.get('operation') == 'promote-ios' and channels == ['ios-app-store']:
            request_id = manifest['existingRelease']['candidateId']
        require(manifest and request_id == cycle['completion']['candidate']
                and set(channels) <= set(cycle['completion']['channels']),
                'This cycle ended with another formal request; start a new cycle for later platforms')
    fetch_source(source)
    if cycle['kind'] != 'hotfix':
        includes_hotfixes(state, source)
    reviewed_main_pr(github, source)
    if manifest:
        from build_identity import verify_production_number
        verify_production_number(github, manifest)
    return cycle


def verify_build_run(github, run_id, artifact):
    run = github.api(f'actions/runs/{run_id}')
    require(run.get('path') == '.github/workflows/release-prerelease.yml'
            and run.get('head_branch') == 'main' and run.get('event') == 'workflow_dispatch'
            and run.get('head_repository', {}).get('full_name', '').lower() == github.repository.lower()
            and run.get('status') == 'completed' and run.get('conclusion') in {'success', 'failure'},
            'Use a completed trusted-main prerelease run')
    # Reuse a successful artifact even when another selected platform failed.
    # Include retained jobs from earlier attempts, but the newest job wins.
    jobs = {}
    for job in sorted(github.pages(f'actions/runs/{run_id}/jobs?filter=all', 'jobs'), key=lambda j: j['id']):
        jobs[job['name']] = job['conclusion']
    name = {'web': 'dev / deploy', 'android': 'android / preview', 'ios': 'ios / preview'}[artifact]
    require(jobs.get('validated') == 'success' and jobs.get(name) == 'success',
            f'Prerelease {artifact} validation and distribution must succeed before QA')


def cycle_receipts(github, cycle, channel):
    return [r for r in github.pages(f"deployments?environment=release-{channel}&sha={cycle['source']}")
            if r.get('payload', {}).get('details', {}).get('cycleId') == cycle['id']]


def reconcile_abort(github, cycle, apple):
    """Preserve facts, including uncertainty; abort is neither success nor rollback."""
    outcome = {}
    tags = reservations()
    for channel in cycle['channels']:
        receipts = cycle_receipts(github, cycle, channel)
        statuses = github.pages(f"deployments/{receipts[0]['id']}/statuses") if receipts else []
        row = {'receiptId': receipts[0]['id'] if receipts else None,
               'receiptState': statuses[0]['state'] if statuses else 'none',
               'details': receipts[0].get('payload', {}).get('details', {}) if receipts else {}}
        try:
            actual = baselines(github, [channel], tags, apple)[channel]
        except ReleaseError:
            # External API outages must not strand an emergency exit. Unknown
            # stays unknown, preserves hotfix backflow, and never authorizes QA.
            actual = {'state': 'unknown'}
        row.update(actualSource=actual.get('sourceSha'), actualTag=actual.get('tag'), actualState=actual.get('state'))
        if channel.startswith('ios-'):
            matching = {(t['tag'].removeprefix('mobile-v'), str(t['buildNumber']))
                        for t in tags if t['tag'].startswith('mobile-v') and t['sourceSha'] == cycle['source']}
            row['appleHistory'] = [item for item in (apple or {}).get(
                'storeVersions' if channel == 'ios-app-store' else 'testflightBuilds', [])
                if (item['version'], str(item['buildNumber'])) in matching]
        outcome[channel] = row
    return outcome


def qa_builds(github, cycle, purpose):
    previews = cycle.get('previews', {}).get(purpose, {})
    require(required_artifacts(cycle['channels'], purpose) <= previews.keys(), 'Build every required preview artifact before QA')
    require(previews, 'No preview artifacts selected')
    for artifact, run in previews.items():
        verify_build_run(github, run, artifact)
    return previews


def queued_dev_groups(github):
    # Merge-group checks belong to these synthetic SHAs, not the constituent PR heads.
    return github.pages('git/matching-refs/heads/gh-readonly-queue/dev/')


def require_drained_queue(github):
    require(not queued_dev_groups(github), 'Drain the dev merge queue before freezing a candidate')


FREEZE_WORKFLOW = 'release-freeze.yml'
FREEZE_BLOCKED = 'Dev is frozen. Only an owner-admitted blocker at this exact PR head may merge.'
QUEUE_BLOCKED = 'Merge queue must drain before selecting a candidate.'


def freeze_verdict(state, event, now):
    """Decide the `release-freeze` job result for one pull_request(_target)/merge_group event.

    The verdict is the job's own conclusion: branch rulesets only accept the check
    run that GitHub Actions creates for a job, never one written through the Checks API.
    """
    if event.get('merge_group'):
        group = event['merge_group']
        return group['base_ref'] != 'refs/heads/dev' or current(state) is None, QUEUE_BLOCKED
    pr = event.get('pull_request')
    if not pr:
        # Pushes and other events carry no PR head to admit; they are never gated.
        return True, ''
    if pr['base']['ref'] != 'dev':
        return True, ''
    return can_merge(state, pr['number'], pr['head']['sha'], now), FREEZE_BLOCKED


def latest_freeze_run(github, sha, events):
    runs = [r for r in github.pages(f'actions/workflows/{FREEZE_WORKFLOW}/runs?head_sha={sha}', 'workflow_runs')
            if r.get('event') in events and r.get('head_sha') == sha]
    return max(runs, key=lambda r: r['id']) if runs else None


def rerun_freeze(github, sha, events, label):
    run = latest_freeze_run(github, sha, events)
    if run is None:
        print(f'{label}: no release-freeze run found for {sha}; push or reopen to trigger one')
        return False
    if run.get('status') != 'completed':
        # A queued/in-progress run waits on instance-mutate-dev and reads the new state itself.
        print(f"{label}: release-freeze run {run['id']} is {run.get('status')}; it will evaluate current state")
        return False
    github.api(f"actions/runs/{run['id']}/rerun", method='POST')
    print(f"{label}: re-running release-freeze run {run['id']} for {sha}")
    return True


def refresh_checks(github, state=None, number=None):
    """Re-run the latest release-freeze job for open dev PR heads and queued merge groups."""
    prs = [github.api(f'pulls/{number}')] if number else github.pages('pulls?state=open&base=dev')
    for pr in prs:
        if pr['base']['ref'] != 'dev' or pr['state'] != 'open':
            continue
        rerun_freeze(github, pr['head']['sha'], {'pull_request_target', 'pull_request'}, f"PR #{pr['number']}")
    if number is None:
        for sha in {group['object']['sha'] for group in queued_dev_groups(github)}:
            rerun_freeze(github, sha, {'merge_group'}, 'merge group')


def output(values):
    with open(os.environ['GITHUB_OUTPUT'], 'a') as out:
        for key, value in values.items():
            require('\n' not in str(value), 'Invalid workflow output')
            out.write(f'{key}={value}\n')


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('command', choices=['change', 'begin-build', 'check-build', 'check-dev', 'check-pr', 'checks', 'status', 'refresh', 'apple-needed'])
    args = parser.parse_args()
    github = GitHub()
    state = load(github)
    if args.command == 'apple-needed':
        c = state['cycles'].get(os.environ.get('CYCLE_ID'))
        channels = c['channels'] if c else os.environ.get('CHANNELS', '').split(',')
        output({'needed': str(any(c.startswith('ios-') for c in channels)).lower()})
        return
    if args.command == 'status':
        print(json.dumps(state, indent=2, ensure_ascii=False))
        return
    if args.command == 'check-pr':
        event = json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text())
        allowed, reason = freeze_verdict(state, event, int(time.time()))
        if not allowed:
            raise SystemExit(f'release-freeze: {reason}')
        print('release-freeze: allowed. Normal review and CI still apply.')
        return
    if args.command == 'checks':
        refresh_checks(github, state)
        return
    if args.command == 'check-dev':
        source = os.environ['SOURCE_SHA']
        c = current(state)
        if os.environ.get('CYCLE_ID'):
            require(c and c['id'] == os.environ['CYCLE_ID'] and c['source'] == source
                    and c.get('previews', {}).get('dev', {}).get('web', c['builds'].get('dev')) == os.environ['BUILD_RUN'], 'Dev candidate was superseded')
            from cycle import live
            live(c, int(time.time()))
            require(c['qa'] is None and c['productionQa'] is None,
                    'QA already recorded; start a new dev prerelease run to invalidate QA before resetting dev data')
        else:
            require(not c, 'Automatic dev deployment paused by release freeze')
        require(github.api('git/ref/heads/dev')['object']['sha'] == source or c and c['kind'] == 'hotfix',
                'Dev head drifted; update the candidate explicitly')
        return
    if args.command == 'refresh':
        state['lastEvent'] = {'action': 'refresh', 'cycle': state['active'] or '-',
                              'actor': os.environ['GITHUB_ACTOR'], 'at': int(time.time()), 'note': 'Renew state seal'}
        save(github, state)
        return
    if args.command == 'change' and os.environ['ACTION'] == 'complete':
        from cycle_completion import complete_release
        state, status = complete_release(github, state, os.environ['RELEASE_RUN'], int(time.time()))
        state.setdefault('lastEvent', {'action': 'complete', 'cycle': '-', 'actor': os.environ['GITHUB_ACTOR'],
                                      'at': int(time.time()), 'note': status})
        save(github, state)
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as out:
            out.write(f"Release run {os.environ['RELEASE_RUN']}: {status}. "
                      "Cycle completion does not assert Apple approval or public availability.\n")
        return
    cycle_id = os.environ.get('CYCLE_ID') or os.environ['GITHUB_RUN_ID']
    if args.command == 'check-build':
        c = current(state)
        purpose = os.environ['PURPOSE']
        artifact = os.environ['PREVIEW_ARTIFACT']
        require(c and c['id'] == cycle_id and c['source'] == os.environ['SOURCE_SHA']
                and c.get('previews', {}).get(purpose, {}).get(artifact) == os.environ['GITHUB_RUN_ID'], 'Build was cancelled or superseded')
        from cycle import live
        live(c, int(time.time()))
        authorize_preview_source(github, c, purpose, os.environ['GITHUB_RUN_ID'])
        return
    actor = os.environ['GITHUB_ACTOR']
    human(github, actor)
    human(github, os.environ.get('GITHUB_TRIGGERING_ACTOR', actor))
    action = 'build' if args.command == 'begin-build' else os.environ['ACTION']
    if action in {'hotfix', 'abort'}:
        require(os.environ.get('PRODUCTION_SERIALIZED') == 'true',
                'Hotfix and abort must hold the production publication queue')
    if action in {'start', 'hotfix'} or action == 'build' and cycle_id not in state['cycles']:
        require_drained_queue(github)
    data = {'note': os.environ.get('NOTE', ''), 'source': os.environ.get('SOURCE_SHA', ''),
            'assignee': os.environ.get('ASSIGNEE', ''), 'base': os.environ.get('BASE_SHA', '')}
    for env, key in [('UNTIL', 'until'), ('ISSUE', 'issue'), ('PR', 'pr')]:
        if os.environ.get(env):
            data[key] = int(os.environ[env])
    if action == 'reassign':
        human(github, actor, admin=True)
        human(github, os.environ.get('GITHUB_TRIGGERING_ACTOR', actor), admin=True)
        human(github, data['assignee'])
        data['admin'] = True
    if action in {'start', 'hotfix', 'advance', 'backflow'}:
        fetch_source(data['source'])
        advancing_hotfix = action == 'advance' and state['cycles'].get(cycle_id, {}).get('kind') == 'hotfix'
        if action != 'hotfix' and not advancing_hotfix:
            includes_hotfixes(state, data['source'])
            require(github.api('git/ref/heads/dev')['object']['sha'] == data['source'], 'Select the current dev SHA')
        if advancing_hotfix:
            git('merge-base', '--is-ancestor', state['cycles'][cycle_id]['base'], data['source'])
        if action in {'start', 'hotfix'}:
            data['channels'] = os.environ['CHANNELS'].split(',')
            issue = github.api(f"issues/{data['issue']}")
            require('pull_request' not in issue and issue['state'] == 'open', 'Use an open release tracking issue')
        if action == 'hotfix':
            apple = json.loads(Path('.apple-state.json').read_text()) if Path('.apple-state.json').exists() else None
            deployed = baselines(github, data['channels'], reservations(), apple)
            require(all(deployed[c].get('sourceSha') and deployed[c]['sourceSha'] == data['base']
                        for c in data['channels']),
                    'Every hotfix channel must share the actual distributed base; split different baselines into separate hotfixes')
            git('merge-base', '--is-ancestor', data['base'], data['source'])
        if action == 'backflow':
            c = state['cycles'][cycle_id]
            git('merge-base', '--is-ancestor', c['source'], data['source'])
            reviewed_preview_source(github, data['source'])
            runs = github.pages(f"actions/workflows/ci.yml/runs?branch=dev&head_sha={data['source']}&event=push", 'workflow_runs')
            require(runs and max(runs, key=lambda r: r['id']).get('conclusion') == 'success',
                    'The merged dev SHA needs successful CI before backflow is complete')
            # Explicit owner attestation covers the post-merge regression checklist.
    if action == 'admit':
        pr = github.api(f"pulls/{data['pr']}")
        require(pr['base']['ref'] == 'dev' and pr['state'] == 'open', 'Admit only an open dev fix PR')
        data['head'] = pr['head']['sha']
    if action == 'offer':
        human(github, data['assignee'])
    if action == 'abort':
        git('fetch', 'origin', 'main', '--tags')
        c = state['cycles'][cycle_id]
        apple = json.loads(Path('.apple-state.json').read_text()) if Path('.apple-state.json').exists() else None
        data['outcome'] = reconcile_abort(github, c, apple)
        data['needs_backflow'] = any(row['receiptId'] or row['actualSource'] == c['source']
                                     or row['actualState'] == 'unknown' or row.get('appleHistory')
                                     for row in data['outcome'].values())
    if action in {'cancel', 'advance'} or action == 'build' and os.environ['PURPOSE'] == 'dev' and cycle_id in state['cycles']:
        c = state['cycles'][cycle_id]
        for channel in c['channels']:
            receipts = cycle_receipts(github, c, channel)
            require(not receipts, 'Production already started; reconcile its channels or start a hotfix instead of replacing the candidate')
    if action == 'build':
        artifacts = preview_artifacts(os.environ.get('ARTIFACTS'), os.environ['PURPOSE'])
        if os.environ['PURPOSE'] == 'dev' and 'web' not in artifacts:
            c = state['cycles'].get(cycle_id)
            deployed = c.get('previews', {}).get('dev', {}).get('web') if c else None
            require(deployed, 'Deploy this SHA to dev with web selected before mobile-only previews')
            verify_build_run(github, deployed, 'web')
        if cycle_id not in state['cycles']:
            require(os.environ['PURPOSE'] == 'dev', 'Start with dev prerelease')
            fetch_source(data['source'])
            includes_hotfixes(state, data['source'])
            require(github.api('git/ref/heads/dev')['object']['sha'] == data['source'], 'Select the current dev SHA')
            issue = github.api(f"issues/{data['issue']}")
            require('pull_request' not in issue and issue['state'] == 'open', 'Use an open release tracking issue')
            state = transition(state, 'start', actor, int(time.time()), cycle=cycle_id,
                               source=data['source'], issue=data['issue'], channels=os.environ['CHANNELS'].split(','))
        data.update(run=os.environ['GITHUB_RUN_ID'], purpose=os.environ['PURPOSE'], artifacts=artifacts)
    if action in {'qa', 'production-qa'}:
        c = state['cycles'][cycle_id]
        purpose = 'dev' if action == 'qa' else 'production'
        data['run'] = c['builds'].get(purpose)
        require(data['run'], 'No candidate build')
        qa_builds(github, c, purpose)
        if c['kind'] != 'hotfix':
            require(github.api('git/ref/heads/dev')['object']['sha'] == c['source'], 'Dev moved; QA must restart')
    state = transition(state, action, actor, int(time.time()), cycle=cycle_id, **data)
    bypass_reason = os.environ.get('SOURCE_REVIEW_BYPASS_REASON', '')
    if action == 'build' and bypass_reason:
        record_preview_bypass(github, state['cycles'][cycle_id], run=data['run'], purpose=data['purpose'],
                              actor=actor, triggering_actor=os.environ.get('GITHUB_TRIGGERING_ACTOR', actor),
                              reason=bypass_reason, now=int(time.time()))
    save(github, state)
    c = state['cycles'][cycle_id]
    output({'source': c['source'], 'cycle': cycle_id, 'owner': c['owner']})
    if action == 'build':
        output({'artifacts': json.dumps(data['artifacts']),
                'mobile_artifacts': json.dumps([a for a in data['artifacts'] if a != 'web'])})
    with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as out:
        out.write(f"Release cycle `{cycle_id}` · owner @{c['owner']} · issue #{c['issue']}\n\n"
                  f"Source `{c['source']}` · {c['phase']} · frozen: {c['frozen']}\n\n"
                  "A successful upload is not device QA. The owner records the device checklist before approval.\n")
        if action == 'build' and bypass_reason:
            out.write('\nAdmin source-review bypass (dev preview only):\n\n```json\n' +
                      json.dumps(c['previewReviewBypasses'][data['run']], ensure_ascii=False) + '\n```\n')


if __name__ == '__main__':
    main()
