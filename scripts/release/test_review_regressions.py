"""Review regressions at preparation and receipt trust boundaries."""
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
import controller
import state
import workflow
from model import ReleaseError, REPOSITORY
from test_model import candidate


def request(branch, repository=REPOSITORY):
    return {'number': 1, 'html_url': 'https://example.org/pr/1',
            'head': {'ref': branch, 'repo': {'full_name': repository} if repository else None}}


class ReviewRegressions(unittest.TestCase):
    def test_open_slot_ignores_forks_and_deleted_repositories(self):
        for repository in ('attacker/YourTJ-Hub', None):
            remote = Mock(repository=REPOSITORY)
            remote.pages.return_value = [request('codex/release/mobile-1.0.15-16', repository)]
            self.assertIsNone(controller.open_request(remote, 'mobile'))
        remote.pages.return_value = [request('codex/release/mobile-1.0.15-16', REPOSITORY.lower())]
        self.assertEqual(controller.open_request(remote, 'mobile')['number'], 1)

    def test_closed_fork_cannot_poison_next_identity(self):
        remote = Mock(repository=REPOSITORY)
        remote.pages.side_effect = lambda endpoint: [] if 'state=open' in endpoint else [
            request('codex/release/mobile-999.0.0-999999999999', 'attacker/YourTJ-Hub'),
            request('codex/release/mobile-1.0.20-21')]
        with patch('controller.source_sha', return_value='a' * 40), patch('controller.reservations', return_value=[]), \
             patch('controller.git', return_value='version: 1.0.14+15'), patch('controller.baselines', return_value={}):
            manifest = controller.plan('android', 'patch', 'main', 'testflight', remote)
        self.assertEqual((manifest['version'], manifest['buildNumber']), ('1.0.21', 22))

    def test_receipt_rejects_changed_binding_before_recording(self):
        manifest = candidate()
        previous = {'deployment': {'payload': {'binding': {'contentDigest': 'old'}, 'details': {'buildRunId': '1'}}}}
        with patch('sys.argv', ['workflow.py', 'receipt', '--candidate', manifest['candidateId']]), \
             patch('workflow.authorize', return_value=(manifest, {'contentDigest': 'new'})), \
             patch('workflow.latest_receipt', return_value=previous), patch('workflow.record') as record, \
             patch.dict(os.environ, CHANNEL='android', GITHUB_RUN_ID='2', GITHUB_RUN_ATTEMPT='1', RELEASE_STATE='success'):
            with self.assertRaisesRegex(ReleaseError, 'approval/content changed'):
                workflow.main()
            record.assert_not_called()

    def test_malformed_successful_baseline_is_actionable(self):
        remote = Mock()
        for payload in ({}, None, 'external payload'):
            remote.pages.side_effect = [[{'id': 1, 'sha': 'a' * 40, 'payload': payload}], [{'state': 'success'}]]
            with self.subTest(payload=payload), self.assertRaisesRegex(ReleaseError, 'Malformed.*receipt'):
                state.successful_baseline(remote, 'web')

    def test_apple_publication_rechecks_baseline_and_accepts_exact_retry(self):
        manifest = candidate()
        channel = 'ios-testflight'
        frozen = manifest['baselines'][channel]
        with tempfile.TemporaryDirectory() as temporary:
            file = Path(temporary) / 'apple.json'; file.write_text('{}', encoding='utf-8')
            with patch('sys.argv', ['workflow.py', 'verify-apple', '--apple-state', str(file)]), \
                 patch.dict(os.environ, APPLE_CHANNELS=json.dumps([channel])), \
                 patch('workflow.authorize', return_value=(manifest, {})), patch('workflow.reservations', return_value=[]):
                for live, allowed in ((frozen, True), ({'tag': manifest['tag'], 'sourceSha': manifest['sourceSha']}, True),
                                      (frozen | {'buildId': 'unexpected'}, False), ({'tag': 'mobile-v9.0.0', 'sourceSha': 'c' * 40}, False)):
                    with self.subTest(live=live), patch('workflow.baselines', return_value={channel: live}):
                        if allowed: workflow.main()
                        else:
                            with self.assertRaisesRegex(ReleaseError, 'Apple distribution advanced'): workflow.main()
                with patch.dict(os.environ, APPLE_CHANNELS='["ios-app-store"]'), self.assertRaisesRegex(ReleaseError, 'Unapproved'):
                    workflow.main()

    def test_reservation_requires_unchanged_distribution_before_creating_tag(self):
        manifest = candidate()
        with patch('sys.argv', ['workflow.py', 'reserve']), patch('workflow.authorize', return_value=(manifest, {})), \
             patch('workflow.latest_receipt', return_value=None), patch('workflow.reservations', return_value=[]), \
             patch('workflow.reserve') as reserve:
            with patch('workflow.baselines', return_value={'android': {'tag': 'mobile-v9.0.0', 'sourceSha': 'c' * 40}}):
                with self.assertRaisesRegex(ReleaseError, 'Distribution advanced'): workflow.main()
            reserve.assert_not_called()
            with patch('workflow.baselines', return_value=manifest['baselines']): workflow.main()
            reserve.assert_called_once()

    def test_reserve_creates_annotated_identity_and_retry_does_not_move_tag(self):
        manifest = candidate(); binding = {'contentDigest': 'reviewed'}
        remote = Mock()
        remote.api.side_effect = [None, {'sha': 'annotation'}, {}]
        controller.reserve(manifest, binding, remote)
        calls = remote.api.call_args_list
        metadata = json.loads(calls[1].kwargs['data']['message'])
        self.assertEqual(metadata['contentDigest'], 'reviewed')
        self.assertEqual(calls[1].kwargs['data']['object'], manifest['sourceSha'])
        self.assertEqual(calls[2].kwargs['data']['sha'], 'annotation')
        remote.reset_mock(); remote.api.side_effect = None; remote.api.return_value = {'object': {'type': 'tag', 'sha': 'annotation'}}
        with patch('controller.verify_reservation') as verify:
            controller.reserve(manifest, binding, remote)
            verify.assert_called_once()
        self.assertEqual(remote.api.call_count, 1)

    def test_recovery_missing_original_build_reports_actionable_error(self):
        prior = {'deployment': {'payload': {'binding': {'contentDigest': 'reviewed'}, 'details': {}}}}
        from test_recovery import RecoveryTests
        with self.assertRaisesRegex(ReleaseError, 'missing buildRunId'):
            RecoveryTests().invoke(candidate()['baselines']['android'], prior)
