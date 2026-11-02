import os
import tempfile
import unittest
from unittest.mock import Mock, patch

import cycle_control
from cli import execute, parser
from cycle import empty_state, transition
from model import ReleaseError


SHA = 'a' * 40


class PreviewBypassTests(unittest.TestCase):
    def setUp(self):
        self.remote = Mock()
        self.remote.api.return_value = {'permission': 'admin', 'user': {'type': 'User'}}
        self.cycle = {'id': '42', 'source': SHA}

    def record(self, **kwargs):
        values = dict(run='50', purpose='dev', actor='alice', triggering_actor='alice',
                      reason='Maintainer explicitly authorized this preview', now=100)
        cycle_control.record_preview_bypass(self.remote, self.cycle, **(values | kwargs))

    def test_admin_override_is_audited_and_only_authorizes_its_source_cycle_run(self):
        self.record()
        grant = self.cycle['previewReviewBypasses']['50']
        self.assertEqual((grant['source'], grant['cycle'], grant['run'], grant['actor'], grant['at']),
                         (SHA, '42', '50', 'alice', 100))
        with patch.dict(os.environ, GITHUB_ACTOR='alice', GITHUB_TRIGGERING_ACTOR='alice'), \
             patch('cycle_control.reviewed_preview_source', side_effect=ReleaseError('Review missing')) as review:
            cycle_control.authorize_preview_source(self.remote, self.cycle, 'dev', '50')
            review.assert_not_called()
            for mutation, purpose, run in [({'source': 'b'*40}, 'dev', '50'),
                                            ({'id': '43'}, 'dev', '50'),
                                            ({}, 'production', '50'), ({}, 'dev', '51')]:
                with self.subTest(mutation=mutation, purpose=purpose, run=run), self.assertRaises(ReleaseError):
                    cycle_control.authorize_preview_source(self.remote, self.cycle | mutation, purpose, run)

    def test_override_requires_admin_initiator_and_admin_rerun_actor(self):
        for actor in ('alice', 'bob'):
            with self.subTest(actor=actor):
                self.remote.api.side_effect = lambda path: {
                    'permission': 'write' if path == f'collaborators/{actor}/permission' else 'admin',
                    'user': {'type': 'User'}}
                with self.assertRaises(ReleaseError):
                    self.record(triggering_actor='bob')
                self.assertNotIn('previewReviewBypasses', self.cycle)
        self.remote.api.side_effect = None
        self.record()
        self.remote.api.return_value = {'permission': 'write', 'user': {'type': 'User'}}
        with patch.dict(os.environ, GITHUB_ACTOR='alice', GITHUB_TRIGGERING_ACTOR='alice'), self.assertRaises(ReleaseError):
            cycle_control.authorize_preview_source(self.remote, self.cycle, 'dev', '50')

    def test_missing_reason_or_production_rehearsal_cannot_create_override(self):
        for values in ({'reason': ' '}, {'reason': 'x'*1001}, {'purpose': 'production'}):
            with self.subTest(values=values), self.assertRaises(ReleaseError):
                self.record(**values)

    def test_normal_preview_keeps_exact_human_review(self):
        with patch('cycle_control.reviewed_preview_source', side_effect=ReleaseError('Review missing')) as review:
            with self.assertRaises(ReleaseError):
                cycle_control.authorize_preview_source(self.remote, self.cycle, 'dev', '50')
            review.assert_called_once_with(self.remote, SHA)

    def test_cli_passes_explicit_reason_and_rejects_production(self):
        args = ['prerelease', '--source', SHA, '--issue', '1240',
                '--source-review-bypass-reason', 'Authorized emergency preview']
        request = execute(parser().parse_args(args), self.remote)
        self.assertEqual(request['inputs']['source_review_bypass_reason'], 'Authorized emergency preview')
        self.remote.dispatch.assert_not_called()
        with self.assertRaises(SystemExit):
            execute(parser().parse_args(args + ['--cycle', '42', '--purpose', 'production']), self.remote)

    def test_controller_seals_the_override_and_signing_rechecks_it(self):
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['web', 'android'])
        self.remote.pages.return_value = []
        env = {'CYCLE_ID': '42', 'GITHUB_RUN_ID': '50', 'GITHUB_ACTOR': 'alice',
               'GITHUB_TRIGGERING_ACTOR': 'alice', 'SOURCE_SHA': SHA, 'PURPOSE': 'dev',
               'ARTIFACTS': 'web,android', 'SOURCE_REVIEW_BYPASS_REASON': 'Explicit admin authorization'}
        with tempfile.NamedTemporaryFile() as summary, \
             patch.dict(os.environ, env | {'GITHUB_STEP_SUMMARY': summary.name}), \
             patch('sys.argv', ['cycle_control.py', 'begin-build']), \
             patch('cycle_control.GitHub', return_value=self.remote), \
             patch('cycle_control.load', return_value=state), patch('cycle_control.time.time', return_value=110), \
             patch('cycle_control.output'), patch('cycle_control.save') as save:
            cycle_control.main()
            sealed = save.call_args.args[1]
            self.assertEqual(sealed['cycles']['42']['previewReviewBypasses']['50']['source'], SHA)
            summary.seek(0)
            self.assertIn(b'Explicit admin authorization', summary.read())
        with patch.dict(os.environ, env | {'PREVIEW_ARTIFACT': 'android'}), \
             patch('sys.argv', ['cycle_control.py', 'check-build']), \
             patch('cycle_control.GitHub', return_value=self.remote), \
             patch('cycle_control.load', return_value=sealed), patch('cycle_control.time.time', return_value=110), \
             patch('cycle_control.reviewed_preview_source', side_effect=ReleaseError('No review')):
            cycle_control.main()
            sealed['cycles']['42']['previewReviewBypasses']['50']['source'] = 'b'*40
            with self.assertRaises(ReleaseError):
                cycle_control.main()

    def test_preview_bypass_does_not_supply_formal_production_review(self):
        self.record()
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['web'])
        state['cycles']['42'].update(qa={'source': SHA},
                                     previewReviewBypasses=self.cycle['previewReviewBypasses'])
        with patch('cycle_control.load', return_value=state), patch('cycle_control.time.time', return_value=110), \
             patch('cycle_control.fetch_source'), patch('cycle_control.includes_hotfixes'), \
             patch('cycle_control.reviewed_main_pr', side_effect=ReleaseError('Formal review required')) as review:
            with self.assertRaisesRegex(ReleaseError, 'Formal review required'):
                cycle_control.production_gate(self.remote, SHA, ['web'])
            review.assert_called_once_with(self.remote, SHA)


if __name__ == '__main__':
    unittest.main()
