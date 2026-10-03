"""Prevent green releases with skipped publishers and preserve first-build identity."""
import json
from pathlib import Path
import re
import tempfile
import unittest
from unittest.mock import patch

import cli
import workflow
from model import ReleaseError
from test_model import candidate


ROOT = Path(__file__).resolve().parents[2]


class PublishDispatchTests(unittest.TestCase):
    def test_trusted_manual_entry_rejects_any_started_channel_before_source_checks(self):
        manifest = candidate()
        with patch('sys.argv', ['workflow.py', 'authorize', '--candidate', manifest['candidateId']]), \
             patch('workflow.authorize', return_value=(manifest, {})), \
             patch('workflow.emit_outputs') as emit, \
             patch.dict('os.environ', {'INITIAL_PUBLICATION': 'true'}, clear=True):
            with patch('workflow.latest_receipt', return_value=None) as receipts:
                workflow.main()
                self.assertEqual(receipts.call_count, len(manifest['channels']))
                emit.assert_called_once_with(manifest, {})
            emit.reset_mock()
            with patch('workflow.latest_receipt', return_value={'status': None}):
                with self.assertRaisesRegex(ReleaseError, 'Recover'):
                    workflow.main()
            emit.assert_not_called()
            with patch('workflow.latest_receipt', side_effect=RuntimeError('API unavailable')):
                with self.assertRaisesRegex(RuntimeError, 'API unavailable'):
                    workflow.main()
            emit.assert_not_called()
            with patch.dict('os.environ', {'INITIAL_PUBLICATION': 'false'}), \
                 patch('workflow.latest_receipt') as receipts:
                workflow.main()
                receipts.assert_not_called()
                emit.assert_called_once_with(manifest, {})

    def test_publishers_override_skip_propagation_but_require_successful_gates(self):
        workflow = (ROOT / '.github/workflows/release-publish.yml').read_text()
        for job in ('web', 'android', 'ios'):
            with self.subTest(job=job):
                # Read the job's own if, including its indented body.
                condition = re.search(rf'\n  {job}:\n    needs:.*\n(?:    #.*\n)*    if: (.*)', workflow)[1]
                self.assertIn('!cancelled()', condition)
                self.assertIn("needs.authorize.result == 'success'", condition)
                self.assertIn("needs.reserve.result == 'success'", condition)

    def test_first_publication_dispatch_does_not_turn_existing_builds_into_rebuilds(self):
        manifest = candidate()
        class Remote:
            def dispatch(self, workflow, inputs):
                return {'workflow': workflow, 'inputs': inputs}
        args = cli.parser().parse_args(['publish', '--candidate', manifest['candidateId']])
        with tempfile.TemporaryDirectory() as temporary:
            folder = Path(temporary)
            (folder / 'manifest.json').write_text(json.dumps(manifest))
            with patch('cli.validate_candidate', return_value='digest'), \
                 patch('cli.latest_receipt', return_value=None):
                result = cli.inspect_candidate(args, Remote(), folder)
                self.assertTrue(result['dryRun'])
                self.assertEqual(result['workflow'], 'release-publish.yml')
                args.apply = True
                self.assertEqual(cli.inspect_candidate(args, Remote(), folder)['inputs'],
                                 {'candidate': manifest['candidateId']})
            for prior in ({'status': None}, {'status': {'state': 'failure'}},
                          {'status': {'state': 'success'}}):
                with patch('cli.validate_candidate', return_value='digest'), \
                     patch('cli.latest_receipt', side_effect=[None, prior]):
                    with self.assertRaisesRegex(ReleaseError, 'Recover'):
                        cli.inspect_candidate(args, Remote(), folder)

    def test_summary_fails_if_a_selected_publisher_is_skipped(self):
        from verify_jobs import verify
        for channels, selected in ((['web'], ['web']), (['android'], ['android']),
                                   (['ios-testflight'], ['ios']),
                                   (['android', 'ios-testflight', 'ios-app-store'], ['android', 'ios'])):
            results = {job: {'result': 'success' if job in selected else 'skipped'}
                       for job in ('web', 'android', 'ios')}
            verify(channels, '', results)
            for job in selected:
                for state in ('skipped', 'failure', 'cancelled', None):
                    with self.subTest(channels=channels, job=job, state=state):
                        with self.assertRaisesRegex(ReleaseError, 'Publisher'):
                            verify(channels, '', results | {job: {'result': state}})
        verify(['android', 'ios-app-store'], 'android-alias',
               {'android': {'result': 'success'}, 'ios': {'result': 'skipped'}})
        verify(['android', 'ios-app-store'], 'ios-app-store',
               {'android': {'result': 'skipped'}, 'ios': {'result': 'success'}})
        with self.assertRaises(ReleaseError):
            verify(['web'], 'ios-app-store', {'ios': {'result': 'success'}})
