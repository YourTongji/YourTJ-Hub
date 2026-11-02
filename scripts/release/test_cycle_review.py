"""Regression coverage for hotfix admission, queued merges and transition recovery."""
import os
from pathlib import Path
import re
import tempfile
import unittest
from unittest.mock import Mock, patch
from cycle import empty_state, transition
from cycle_control import freeze_verdict, main, refresh_checks
from model import ReleaseError


class CycleReviewTests(unittest.TestCase):
    def dispatch(self, action='hotfix', baselines=None, groups=(), serialized='true'):
        remote = Mock()
        remote.api.return_value = {'state': 'open', 'object': {'sha': 'a' * 40}}
        remote.pages.return_value = list(groups)
        with tempfile.NamedTemporaryFile() as summary, patch.dict(os.environ, {
                'GITHUB_ACTOR': 'alice', 'GITHUB_TRIGGERING_ACTOR': 'alice', 'GITHUB_RUN_ID': '42',
                'ACTION': action, 'CYCLE_ID': '', 'SOURCE_SHA': 'a' * 40, 'BASE_SHA': 'b' * 40,
                'ISSUE': '1240', 'CHANNELS': 'web,android', 'NOTE': 'Fix outage',
                'PRODUCTION_SERIALIZED': serialized, 'GITHUB_STEP_SUMMARY': summary.name}), \
             patch('sys.argv', ['cycle_control.py', 'change']), \
             patch('cycle_control.GitHub', return_value=remote), patch('cycle_control.human'), \
             patch('cycle_control.load', return_value=empty_state()), patch('cycle_control.fetch_source'), \
             patch('cycle_control.git'), patch('cycle_control.reservations', return_value=[]), \
             patch('cycle_control.baselines', return_value=baselines or {
                 'web': {'sourceSha': 'b' * 40}, 'android': {'sourceSha': 'b' * 40}}), \
             patch('cycle_control.output'), patch('cycle_control.save') as save:
            main()
            return save.call_args.args[1]

    def test_hotfix_rejects_another_selected_channels_different_or_unknown_baseline(self):
        for other in ['c' * 40, None]:
            with self.subTest(other=other), self.assertRaisesRegex(ReleaseError, 'Every hotfix channel'):
                self.dispatch(baselines={'web': {'sourceSha': 'b' * 40}, 'android': {'sourceSha': other}})
        self.assertEqual(self.dispatch()['active'], '42')

    def test_hotfix_cannot_take_over_without_the_production_queue(self):
        with self.assertRaisesRegex(ReleaseError, 'production publication queue'):
            self.dispatch(serialized='false')

    def test_freeze_cannot_start_with_an_existing_merge_group(self):
        for action in ['start', 'hotfix']:
            with self.subTest(action=action), self.assertRaisesRegex(ReleaseError, 'merge queue'):
                self.dispatch(action=action, groups=[{'ref': 'refs/heads/gh-readonly-queue/dev/pr-1',
                                                     'object': {'sha': 'd' * 40}}])

    def test_freeze_verdict_is_the_job_result_for_prs_and_merge_groups(self):
        frozen = transition(empty_state(), 'start', 'alice', 100, cycle='42', source='a' * 40,
                            issue=1240, channels=['web'])
        pr = {'pull_request': {'number': 7, 'base': {'ref': 'dev'}, 'head': {'sha': 'c' * 40}}}
        group = {'merge_group': {'base_ref': 'refs/heads/dev', 'head_sha': 'd' * 40}}
        self.assertTrue(freeze_verdict(empty_state(), pr, 200)[0])
        self.assertTrue(freeze_verdict(empty_state(), group, 200)[0])
        self.assertEqual(freeze_verdict(frozen, pr, 200), (False, 'Dev is frozen. Only an owner-admitted '
                                                                 'blocker at this exact PR head may merge.'))
        self.assertFalse(freeze_verdict(frozen, group, 200)[0])
        admitted = transition(frozen, 'admit', 'alice', 150, cycle='42', pr=7, head='c' * 40, note='blocker')
        self.assertTrue(freeze_verdict(admitted, pr, 200)[0])
        moved = {'pull_request': {**pr['pull_request'], 'head': {'sha': 'e' * 40}}}
        self.assertFalse(freeze_verdict(admitted, moved, 200)[0])
        self.assertTrue(freeze_verdict(frozen, {}, 200)[0])

    def test_check_pr_exits_nonzero_when_blocked(self):
        frozen = transition(empty_state(), 'start', 'alice', 100, cycle='42', source='a' * 40,
                            issue=1240, channels=['web'])
        event = '{"pull_request": {"number": 7, "base": {"ref": "dev"}, "head": {"sha": "%s"}}}' % ('c' * 40)
        for snapshot, blocked in [(frozen, True), (empty_state(), False)]:
            with self.subTest(blocked=blocked), tempfile.NamedTemporaryFile('w', suffix='.json') as path:
                path.write(event)
                path.flush()
                remote = Mock()
                with patch.dict(os.environ, {'GITHUB_EVENT_PATH': path.name}), \
                     patch('sys.argv', ['cycle_control.py', 'check-pr']), \
                     patch('cycle_control.GitHub', return_value=remote), \
                     patch('cycle_control.load', return_value=snapshot):
                    if blocked:
                        with self.assertRaisesRegex(SystemExit, 'Dev is frozen'):
                            main()
                    else:
                        main()
                remote.api.assert_not_called()

    def test_refresh_reruns_latest_completed_freeze_runs_without_writing_check_runs(self):
        remote = Mock()
        group = {'ref': 'refs/heads/gh-readonly-queue/dev/pr-1', 'object': {'sha': 'd' * 40}}
        prs = [{'number': n, 'base': {'ref': 'dev'}, 'state': 'open', 'head': {'sha': s * 40}}
               for n, s in [(1, 'c'), (2, 'e'), (3, 'f')]]
        runs = {
            'c' * 40: [{'id': 10, 'event': 'pull_request_target', 'head_sha': 'c' * 40, 'status': 'completed'},
                       {'id': 11, 'event': 'pull_request_target', 'head_sha': 'c' * 40, 'status': 'completed'},
                       {'id': 12, 'event': 'push', 'head_sha': 'c' * 40, 'status': 'completed'}],
            'e' * 40: [{'id': 20, 'event': 'pull_request_target', 'head_sha': 'e' * 40, 'status': 'in_progress'}],
            'd' * 40: [{'id': 30, 'event': 'merge_group', 'head_sha': 'd' * 40, 'status': 'completed'}],
        }

        def pages(endpoint, key=None):
            if endpoint.startswith('pulls?'):
                return prs
            if endpoint.startswith('actions/workflows/release-freeze.yml/runs?head_sha='):
                return runs.get(endpoint.split('=')[1], [])
            return [group]
        remote.pages.side_effect = pages
        refresh_checks(remote)
        calls = [(c.args[0], c.kwargs.get('method')) for c in remote.api.call_args_list]
        self.assertEqual(calls, [('actions/runs/11/rerun', 'POST'), ('actions/runs/30/rerun', 'POST')])
        self.assertFalse(any('check-runs' in c[0] for c in calls))

    def test_freeze_workflow_job_is_named_like_the_required_check(self):
        root = Path(__file__).resolve().parents[2] / '.github/workflows'
        workflow = (root / 'release-freeze.yml').read_text()
        self.assertIn('    name: release-freeze\n', workflow)
        self.assertNotIn('strategy:', workflow)
        self.assertNotIn('checks: write', workflow)
        self.assertNotIn('cycle_control.py check-pr', workflow)
        self.assertIn('Dev delivery is continuous', workflow)
        source = (Path(__file__).resolve().parent / 'cycle_control.py').read_text()
        self.assertNotIn('check-runs', source)

    def test_failed_projection_is_a_separate_rerunnable_job(self):
        workflow = (Path(__file__).resolve().parents[2] / '.github/workflows/release-control.yml').read_text()
        jobs = dict(re.findall(r'^  ([\w-]+):\n(.*?)(?=^  [\w-]+:|\Z)',
                               workflow.split('\njobs:\n')[1], re.M | re.S))
        for name in ['control', 'refresh']:
            self.assertNotIn('cycle_control.py checks', jobs[name])
            self.assertNotIn('preview_status.py', jobs[name])
            self.assertIn('Seal release state', jobs[name])
        self.assertIn('needs: [control, refresh]', jobs['project'])
        self.assertIn('cycle_control.py checks', jobs['project'])
        self.assertIn('preview_status.py', jobs['project'])
        self.assertIn('actions: write', jobs['project'])


if __name__ == '__main__':
    unittest.main()
