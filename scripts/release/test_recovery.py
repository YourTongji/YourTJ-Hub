"""Recovery keeps original build identity and cannot silently roll back a newer release."""
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import workflow
from model import ReleaseError
from test_model import candidate


class RecoveryTests(unittest.TestCase):
    def invoke(self, live, prior, recover=True):
        manifest = candidate()
        binding = {'contentDigest': 'reviewed'}
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / 'output'
            with patch.dict(os.environ, CHANNEL='android', RECOVER=str(recover).lower(), GITHUB_RUN_ID='200', GITHUB_OUTPUT=str(output)), \
                 patch('sys.argv', ['workflow.py', 'start', '--candidate', manifest['candidateId']]), \
                 patch('workflow.authorize', return_value=(manifest, binding)), \
                 patch('workflow.latest_receipt', return_value=prior), \
                 patch('workflow.reservations', return_value=[]), \
                 patch('workflow.baselines', return_value={'android': live}), \
                 patch('workflow.record') as record:
                workflow.main()
                return output.read_text(), record.call_args.args[-1]

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
