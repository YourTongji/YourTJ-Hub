"""Recovery keeps original build identity and cannot silently roll back a newer release."""
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
import workflow
from model import ReleaseError
from test_model import candidate


class RecoveryTests(unittest.TestCase):
    def invoke(self, live, prior, recover=True, channel='android', remote=None):
        manifest = candidate() | {'channels': [channel]}
        if channel == 'web':
            manifest = manifest | {'tag': 'v1.0.15', 'baselines': {'web': live}}
        binding = {'contentDigest': 'reviewed'}
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / 'output'
            with patch.dict(os.environ, CHANNEL=channel, RECOVER=str(recover).lower(), GITHUB_RUN_ID='200', GITHUB_OUTPUT=str(output)), \
                 patch('workflow.GitHub', return_value=remote), \
                 patch('sys.argv', ['workflow.py', 'start', '--candidate', manifest['candidateId']]), \
                 patch('workflow.authorize', return_value=(manifest, binding)), \
                 patch('workflow.latest_receipt', return_value=prior), \
                 patch('workflow.reservations', return_value=[]), \
                 patch('workflow.baselines', return_value={channel: live}), \
                 patch('workflow.record') as record:
                workflow.main()
                return output.read_text(encoding='utf-8'), record.call_args.args[-1]

    def web_remote(self, *, artifacts=None, upload='skipped', release=None, run_status='completed'):
        remote = Mock()
        remote.pages.side_effect = [artifacts or [], [{'name': 'web / assets', 'steps': [
            {'name': 'Preserve original distribution archives', 'conclusion': upload}]}]]
        remote.api.side_effect = [{'status': run_status, 'conclusion': 'failure'}, release]
        return remote

    def web_prior(self, **details):
        return {'deployment': {'payload': {'binding': {'contentDigest': 'reviewed'},
                    'details': {'buildRunId': '100', **details}}}}

    def test_web_can_repeat_failed_attempt_before_any_artifact_was_saved(self):
        output, details = self.invoke({}, self.web_prior(), channel='web', remote=self.web_remote())
        self.assertIn('build_required=true', output)
        self.assertIn('build_run_id=200', output)
        self.assertEqual(details['buildRunId'], '200')

    def test_web_reuses_retained_archives(self):
        artifact = {'name': 'web-v1.0.15', 'expired': False}
        output, details = self.invoke({}, self.web_prior(), channel='web',
                                      remote=self.web_remote(artifacts=[artifact]))
        self.assertIn('build_required=false', output)
        self.assertEqual(details['buildRunId'], '100')

    def test_web_does_not_rebuild_expired_or_previously_saved_or_published_identity(self):
        cases = [({}, {'artifacts': [{'name': 'web-v1.0.15', 'expired': True}]}),
                 ({}, {'upload': 'success'}), ({}, {'release': {'assets': []}}),
                 ({'binarySha256': 'c' * 64}, {}), ({}, {'run_status': 'in_progress'})]
        for details, state in cases:
            with self.subTest(details=details, state=state), self.assertRaises(ReleaseError):
                self.invoke({}, self.web_prior(**details), channel='web', remote=self.web_remote(**state))

    def test_web_artifact_query_failure_cannot_authorize_another_build(self):
        remote = self.web_remote()
        remote.pages.side_effect = ReleaseError('API unavailable')
        with self.assertRaisesRegex(ReleaseError, 'API unavailable'):
            self.invoke({}, self.web_prior(), channel='web', remote=remote)

    def test_web_deleted_archives_from_an_older_attempt_cannot_be_rebuilt(self):
        remote = self.web_remote()
        remote.pages.side_effect = [[], [
            {'name': 'web / assets', 'steps': [
                {'name': 'Preserve original distribution archives', 'conclusion': state}]}
            for state in ('success', 'skipped')]]
        with self.assertRaisesRegex(ReleaseError, 'were saved'):
            self.invoke({}, self.web_prior(), channel='web', remote=remote)

    def test_web_reuses_image_without_querying_unneeded_archives(self):
        remote = self.web_remote()
        remote.pages.side_effect = ReleaseError('API unavailable')
        prior = self.web_prior(image='main-v1.0.15@sha256:' + 'c' * 64, binarySha256='d' * 64)
        output, details = self.invoke({}, prior, channel='web', remote=remote)
        self.assertIn('build_required=false', output)
        self.assertEqual(details['image'], prior['deployment']['payload']['details']['image'])
        remote.pages.assert_not_called()

    def test_recovery_reuses_first_build_run(self):
        prior = {'deployment': {'payload': {'binding': {'contentDigest': 'reviewed'},
                    'details': {'buildRunId': '100', 'assets': ['immutable.apk']}}}}
        output, details = self.invoke(candidate()['baselines']['android'], prior)
        self.assertIn('build_run_id=100', output)
        self.assertEqual(details['assets'], ['immutable.apk'])
        self.assertEqual(details['runId'], '200')

    def test_changed_review_or_missing_original_artifact_receipt_blocks_recovery(self):
        prior = {'deployment': {'payload': {'binding': {'contentDigest': 'changed'}, 'details': {'buildRunId': '100'}}}}
        for previous in (prior, None):
            with self.assertRaises(ReleaseError): self.invoke(candidate()['baselines']['android'], previous)

    def test_recheck_inside_platform_lock_rejects_advanced_distribution(self):
        with self.assertRaisesRegex(ReleaseError, 'Distribution advanced'):
            self.invoke({'tag': 'mobile-v1.0.16', 'sourceSha': 'd' * 40}, None, recover=False)

    def test_deploy_inputs_must_equal_the_reviewed_image_receipt(self):
        manifest = candidate() | {'channels': ['web']}
        binding = {'contentDigest': 'reviewed'}
        receipt = {'deployment': {'payload': {'binding': binding,
                    'details': {'image': 'main-v1.0.15@sha256:' + 'b' * 64, 'binarySha256': 'c' * 64}}}}
        with patch('sys.argv', ['workflow.py', 'verify-deploy']), patch('workflow.authorize', return_value=(manifest, binding)), \
             patch('workflow.verify_reservation'), patch('workflow.latest_receipt', return_value=receipt), \
             patch.dict(os.environ, IMAGE_REF=receipt['deployment']['payload']['details']['image'], BINARY_SHA='c' * 64,
                        SOURCE_SHA=manifest['sourceSha'], VERSION=manifest['version']):
            workflow.main()
            for key in ('IMAGE_REF', 'BINARY_SHA', 'SOURCE_SHA', 'VERSION'):
                with patch.dict(os.environ, {key: 'different'}), self.assertRaises(ReleaseError):
                    workflow.main()
