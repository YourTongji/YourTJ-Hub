import unittest
import json
import os
from copy import deepcopy
from pathlib import Path
import subprocess
from tempfile import TemporaryDirectory
import textwrap
from unittest.mock import patch
import prepare_server as server
from prepare_server import next_version


class ServerVersionTest(unittest.TestCase):
    def test_mobile_and_non_release_tags_do_not_affect_server(self):
        self.assertEqual(next_version(['v1.9.0', 'v1.10.2', 'mobile-v99.0.0', 'v2.0.0-beta'], 'patch'), 'v1.10.3')
        self.assertEqual(next_version([], 'minor'), 'v0.1.0')
        self.assertEqual(next_version(['v1.9.0'], 'major'), 'v2.0.0')
        with self.assertRaises(ValueError): next_version([], 'invalid')


class ServerPromotionTest(unittest.TestCase):
    def setUp(self):
        environment = patch.dict(os.environ, GITHUB_REF='', GITHUB_SHA='')
        environment.start()
        self.addCleanup(environment.stop)

    def test_existing_pr_continues_to_merge_in_the_same_run(self):
        def read(*args):
            if args == ('git', 'rev-parse', 'origin/dev'):
                return 'dev-sha'
            if args == ('git', 'rev-parse', 'origin/main'):
                return 'main-sha'
            if args[:3] == ('gh', 'pr', 'list'):
                return json.dumps([{'number': 607, 'url': 'https://github.com/YourTongji/YourTJ-Hub/pull/607'}])
            if args == ('git', 'show', '-s', '--format=%P', 'merged-sha'):
                return 'main-sha dev-sha'
            self.fail(f'Unexpected command: {args}')

        with patch.object(server.subprocess, 'run') as run, \
             patch.object(server, 'read', side_effect=read), \
             patch.object(server, 'wait_for_merge', create=True, return_value='merged-sha') as merge:
            run.return_value.returncode = 1
            self.assertEqual(server.ensure_promoted(), 'merged-sha')
        merge.assert_called_once_with(607, 'dev-sha', 'main-sha')

    def test_new_pr_is_created_and_merged_without_a_second_dispatch(self):
        def read(*args):
            if args[:2] == ('git', 'rev-parse'):
                return 'dev-sha' if args[2] == 'origin/dev' else 'main-sha'
            if args[:3] == ('gh', 'pr', 'list'):
                return '[]'
            if args[:3] == ('gh', 'pr', 'create'):
                body = Path(args[-1]).read_text()
                self.assertIn('dev-sha', body)
                return 'https://github.com/YourTongji/YourTJ-Hub/pull/608'
            if args[:2] == ('git', 'show'):
                return 'main-sha dev-sha'
            self.fail(f'Unexpected command: {args}')

        with patch.object(server.subprocess, 'run') as run, \
             patch.object(server, 'read', side_effect=read), \
             patch.object(server, 'wait_for_merge', return_value='merged-sha') as merge:
            run.return_value.returncode = 1
            self.assertEqual(server.ensure_promoted(), 'merged-sha')
        merge.assert_called_once_with(608, 'dev-sha', 'main-sha')

    def test_equal_trees_release_the_captured_main_commit(self):
        with patch.object(server.subprocess, 'run') as run, \
             patch.object(server, 'read', side_effect=['dev-sha', 'main-sha']), \
             patch.object(server, 'wait_for_merge') as merge:
            run.return_value.returncode = 0
            self.assertEqual(server.ensure_promoted(), 'main-sha')
        merge.assert_not_called()

    def test_failed_diff_does_not_promote(self):
        with patch.object(server.subprocess, 'run') as run, \
             patch.object(server, 'read', side_effect=['dev-sha', 'main-sha']), \
             self.assertRaisesRegex(RuntimeError, 'compare'):
            run.return_value.returncode = 128
            server.ensure_promoted()

    def test_dev_dispatch_does_not_include_later_commits(self):
        with patch.dict(os.environ, GITHUB_REF='refs/heads/dev', GITHUB_SHA='original-sha'), \
             patch.object(server, 'read', side_effect=['new-dev-sha', 'main-sha']), \
             patch.object(server.subprocess, 'run') as run, \
             self.assertRaisesRegex(RuntimeError, 'dispatched'):
            server.ensure_promoted()
        run.assert_not_called()

    def test_main_racing_merge_is_not_published(self):
        with patch.object(server.subprocess, 'run') as run, \
             patch.object(server, 'read', side_effect=[
                 'dev-sha', 'main-sha', '[{"number":607,"url":"pr-url"}]', 'new-main dev-sha']), \
             patch.object(server, 'wait_for_merge', return_value='merged-sha'), \
             self.assertRaisesRegex(RuntimeError, 'unexpected parents'):
            run.return_value.returncode = 1
            server.ensure_promoted()


def release_pr():
    return {
        'number': 607, 'state': 'open', 'merged': False, 'draft': False,
        'mergeable': True, 'mergeable_state': 'clean', 'merge_commit_sha': 'merged-sha',
        'created_at': '2026-09-10T01:00:00Z',
        'head': {'ref': 'dev', 'sha': 'dev-sha', 'repo': {'full_name': server.REPOSITORY}},
        'base': {'ref': 'main', 'sha': 'main-sha', 'repo': {'full_name': server.REPOSITORY}},
    }


def ci_run(path, **updates):
    run = {
        'id': 100, 'path': path, 'event': 'pull_request', 'head_branch': 'dev',
        'head_sha': 'dev-sha', 'head_repository': {'full_name': server.REPOSITORY},
        'created_at': '2026-09-10T01:01:00Z', 'pull_requests': [],
        'status': 'completed', 'conclusion': 'success', 'html_url': f'https://github.com/run/{path}',
    }
    run.update(updates)
    return run


class ServerCITest(unittest.TestCase):
    def setUp(self):
        self.runs = [ci_run(path) for path in sorted(server.REQUIRED_WORKFLOWS)]

    def ready(self, runs):
        # Split across pages to guard against silently missing checks after page 1.
        with patch.object(server, 'api', return_value=[{'workflow_runs': runs[:2]}, {'workflow_runs': runs[2:]}]):
            return server.ci_ready(release_pr(), 'dev-sha', '2026-09-10T01:00:00Z')

    def test_all_core_workflows_are_required_even_without_branch_protection(self):
        self.assertTrue(self.ready(self.runs))
        self.assertFalse(self.ready([]))
        self.assertFalse(self.ready(self.runs[:-1]))

    def test_pending_workflow_waits_for_its_dependent_jobs(self):
        self.runs[0].update(status='in_progress', conclusion=None)
        self.assertFalse(self.ready(self.runs))

    def test_optional_workflow_also_blocks_while_running_or_failed(self):
        optional = ci_run('.github/workflows/ci-mobile.yml', status='queued', conclusion=None)
        self.assertFalse(self.ready(self.runs + [optional]))
        for conclusion in ['failure', 'cancelled', 'timed_out', 'skipped', 'action_required', None]:
            with self.subTest(conclusion=conclusion), self.assertRaisesRegex(RuntimeError, 'Release CI'):
                self.ready(self.runs + [dict(optional, status='completed', conclusion=conclusion)])

    def test_latest_workflow_run_wins_over_earlier_success_or_failure(self):
        path = self.runs[0]['path']
        for ordering in [False, True]:
            runs = self.runs + [ci_run(path, id=101, status='in_progress', conclusion=None)]
            self.assertFalse(self.ready(list(reversed(runs)) if ordering else runs))
        self.assertTrue(self.ready(self.runs + [ci_run(path, id=99, conclusion='failure')]))

    def test_other_events_heads_prs_and_stale_runs_do_not_satisfy_ci(self):
        for override in [
                {'event': 'push'}, {'head_sha': 'old-dev'}, {'head_branch': 'other'},
                {'head_repository': {'full_name': 'someone/fork'}},
                {'created_at': '2026-09-10T00:59:59Z'}, {'pull_requests': [{'number': 999}]}]:
            with self.subTest(override=override):
                runs = deepcopy(self.runs)
                runs[0].update(override)
                self.assertFalse(self.ready(runs))
        self.runs[0]['pull_requests'] = [{'number': 607}]
        self.assertTrue(self.ready(self.runs))

    def test_non_ci_automation_does_not_block_release(self):
        self.assertTrue(self.ready(self.runs + [ci_run('.github/workflows/release-to-main.yml', status='in_progress')]))


class ServerMergeTest(unittest.TestCase):
    def setUp(self):
        self.pr = release_pr()
        self.pr_reads = []
        self.merges = []
        self.merge_response = {'merged': True, 'sha': 'merged-sha'}
        self.api_mock = patch.object(server, 'api', side_effect=self.api).start()
        self.sleep = patch.object(server.time, 'sleep').start()
        self.clock = patch.object(server.time, 'monotonic', side_effect=range(100)).start()
        self.addCleanup(patch.stopall)

    def api(self, path, *args):
        if path == 'commits/main-sha':
            return {'commit': {'committer': {'date': '2026-09-10T00:00:00Z'}}}
        if path == 'pulls/607':
            return self.pr_reads.pop(0) if self.pr_reads else self.pr
        if path == 'pulls/607/merge':
            self.merges.append(args)
            return self.merge_response
        self.fail(f'Unexpected API call: {path}, {args}')

    def test_wait_then_merge_uses_locked_head_and_preserves_dev_branch(self):
        with patch.object(server, 'ci_ready', side_effect=[False, True]):
            self.assertEqual(server.wait_for_merge(607, 'dev-sha', 'main-sha'), 'merged-sha')
        self.sleep.assert_called_once_with(15)
        self.assertEqual(self.merges, [('--method', 'PUT', '-f', 'sha=dev-sha', '-f', 'merge_method=merge')])

    def test_failed_ci_never_merges(self):
        with patch.object(server, 'ci_ready', side_effect=RuntimeError('Release CI failure')), \
             self.assertRaisesRegex(RuntimeError, 'CI failure'):
            server.wait_for_merge(607, 'dev-sha', 'main-sha')
        self.assertEqual(self.merges, [])

    def test_closed_draft_conflicting_or_changed_pr_never_merges(self):
        variants = [
            dict(self.pr, state='closed'), dict(self.pr, draft=True),
            dict(self.pr, mergeable_state='dirty'),
            dict(self.pr, head=dict(self.pr['head'], sha='new-dev')),
            dict(self.pr, base=dict(self.pr['base'], sha='new-main')),
            dict(self.pr, head=dict(self.pr['head'], repo={'full_name': 'someone/fork'})),
        ]
        for pr in variants:
            with self.subTest(pr=pr), patch.object(server, 'ci_ready') as ready, self.assertRaises(RuntimeError):
                self.pr = pr
                server.wait_for_merge(607, 'dev-sha', 'main-sha')
            ready.assert_not_called()
        self.assertEqual(self.merges, [])

    def test_source_change_during_ci_read_is_caught_before_merge(self):
        self.pr_reads = [self.pr, dict(self.pr, head=dict(self.pr['head'], sha='new-dev'))]
        with patch.object(server, 'ci_ready', return_value=True), self.assertRaisesRegex(RuntimeError, 'changed'):
            server.wait_for_merge(607, 'dev-sha', 'main-sha')
        self.assertEqual(self.merges, [])

    def test_review_requirements_wait_and_timeout_without_bypass(self):
        self.pr['mergeable_state'] = 'blocked'
        with patch.object(server, 'ci_ready', return_value=True), self.assertRaises(TimeoutError):
            server.wait_for_merge(607, 'dev-sha', 'main-sha', timeout=3)
        self.assertEqual(self.merges, [])

    def test_missing_ci_times_out_without_merge(self):
        with patch.object(server, 'ci_ready', return_value=False), self.assertRaises(TimeoutError):
            server.wait_for_merge(607, 'dev-sha', 'main-sha', timeout=3)
        self.assertEqual(self.merges, [])

    def test_external_merge_still_waits_for_ci(self):
        self.pr.update(merged=True, state='closed')
        with patch.object(server, 'ci_ready', side_effect=[False, True]):
            self.assertEqual(server.wait_for_merge(607, 'dev-sha', 'main-sha'), 'merged-sha')
        self.assertEqual(self.merges, [])
        self.sleep.assert_called_once()

    def test_merge_refusal_or_api_error_does_not_return_a_release_commit(self):
        self.merge_response = {'merged': False, 'message': 'Head changed'}
        with patch.object(server, 'ci_ready', return_value=True), self.assertRaisesRegex(RuntimeError, 'Head changed'):
            server.wait_for_merge(607, 'dev-sha', 'main-sha')
        with patch.object(server, 'api', side_effect=subprocess.CalledProcessError(1, 'gh')), \
             self.assertRaises(subprocess.CalledProcessError):
            server.wait_for_merge(607, 'dev-sha', 'main-sha')


class ServerTagTest(unittest.TestCase):
    def test_tag_uses_merge_result_instead_of_moving_main(self):
        calls = []
        def read(*args):
            calls.append(args)
            if args == ('git', 'tag', '--list'):
                return 'v1.2.3\nmobile-v99.0.0'
            if args == ('git', 'tag', '--points-at', 'merged-sha'):
                return ''
            if args[:2] == ('gh', 'api'):
                return '{}'
            self.fail(f'Unexpected command: {args}')
        with TemporaryDirectory() as folder:
            output = Path(folder) / 'output'
            with patch.dict(os.environ, BUMP='minor', GITHUB_OUTPUT=str(output)), \
                 patch.object(server.subprocess, 'run'), \
                 patch.object(server, 'ensure_promoted', return_value='merged-sha'), \
                 patch.object(server, 'read', side_effect=read):
                server.main()
            self.assertEqual(output.read_text(), 'tag=v1.3.0\n')
        self.assertIn('sha=merged-sha', calls[-1])
        self.assertIn('ref=refs/tags/v1.3.0', calls[-1])

    def test_duplicate_tag_does_not_reserve_another_version(self):
        with patch.dict(os.environ, BUMP='patch'), patch.object(server.subprocess, 'run'), \
             patch.object(server, 'ensure_promoted', return_value='merged-sha'), \
             patch.object(server, 'read', side_effect=['v1.2.3', 'v1.2.3']) as read, \
             self.assertRaisesRegex(ValueError, 'already has'):
            server.main()
        self.assertEqual(read.call_count, 2)

    def test_promotion_failure_never_creates_a_tag(self):
        with patch.object(server.subprocess, 'run'), \
             patch.object(server, 'ensure_promoted', side_effect=RuntimeError('CI failed')), \
             patch.object(server, 'read') as read, self.assertRaises(RuntimeError):
            server.main()
        read.assert_not_called()


def workflow_shell(filename, step_name):
    lines = (Path(__file__).resolve().parents[2] / '.github/workflows' / filename).read_text().splitlines()
    start = lines.index(f'      - name: {step_name}')
    start = lines.index('        run: |', start) + 1
    end = start
    while end < len(lines) and (not lines[end].strip() or lines[end].startswith('          ')):
        end += 1
    return textwrap.dedent('\n'.join(lines[start:end]))


class ServerDeploymentTest(unittest.TestCase):
    def test_dispatch_deploys_published_tag_even_when_main_moves(self):
        shell = workflow_shell('release-to-main.yml', '触发生产部署(deploy-main)')
        with TemporaryDirectory() as folder:
            stub = Path(folder) / 'gh'
            stub.write_text('#!/bin/sh\nprintf "%s\\n" "$@" > "$CALL_LOG"\n')
            stub.chmod(0o755)
            log = Path(folder) / 'log'
            env = dict(os.environ, PATH=folder + os.pathsep + os.environ['PATH'],
                       CALL_LOG=str(log), RELEASE_TAG='v1.2.4', GITHUB_REF='refs/heads/dev')
            subprocess.run(['bash', '-e', '-c', shell], env=env, check=True, capture_output=True)
            self.assertEqual(log.read_text().splitlines(), ['workflow', 'run', 'deploy-main.yml', '--ref', 'v1.2.4'])

    def test_deploy_rejects_branches_mobile_tags_and_non_main_commits(self):
        shell = workflow_shell('deploy-main.yml', 'Require a server release tag on main')
        with TemporaryDirectory() as folder:
            stub = Path(folder) / 'git'
            stub.write_text('#!/bin/sh\nprintf "%s\\n" "$@" > "$CALL_LOG"\nexit "$GIT_RESULT"\n')
            stub.chmod(0o755)
            log = Path(folder) / 'log'
            for ref, git_result, succeeds in [
                    ('refs/tags/v1.2.3', '0', True), ('refs/heads/main', '0', False),
                    ('refs/tags/mobile-v1.2.3', '0', False), ('refs/tags/v1.2.3-beta', '0', False),
                    ('refs/tags/v1.2.3', '1', False)]:
                with self.subTest(ref=ref, git_result=git_result):
                    env = dict(os.environ, PATH=folder + os.pathsep + os.environ['PATH'], CALL_LOG=str(log),
                               GITHUB_REF=ref, GITHUB_SHA='published-sha', GIT_RESULT=git_result)
                    result = subprocess.run(['bash', '-e', '-c', shell], env=env, capture_output=True)
                    self.assertEqual(result.returncode == 0, succeeds)
            self.assertEqual(log.read_text().splitlines(), ['merge-base', '--is-ancestor', 'published-sha', 'origin/main'])
