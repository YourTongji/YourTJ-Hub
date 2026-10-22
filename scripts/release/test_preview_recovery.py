import io
import json
import unittest
from unittest.mock import Mock, patch
import zipfile

from model import REPOSITORY, ReleaseError
from preview_recovery import read_identity


class PreviewRecoveryTests(unittest.TestCase):
    def setUp(self):
        self.cycle = {'id': '42', 'source': 'a'*40}
        self.run = {'path': '.github/workflows/release-prerelease.yml', 'event': 'workflow_dispatch',
                    'head_branch': 'main', 'head_repository': {'full_name': REPOSITORY},
                    'status': 'completed', 'conclusion': 'failure', 'run_attempt': 1}
        self.receipt = {'stage': 'processed', 'source': 'a'*40, 'environment': 'dev',
                        'version': '1.0.17', 'buildNumber': '21', 'buildId': 'apple-build'}
        self.tag = {'object': {'type': 'commit', 'sha': 'a'*40}, 'message': json.dumps({
            'source': 'a'*40, 'cycle': '42', 'version': '1.0.17', 'buildNumber': 21,
            'environment': 'dev', 'preview': True})}
        self.jobs = [{'id': 1, 'name': 'validated', 'conclusion': 'success'},
                     {'id': 2, 'name': 'ios / preview', 'steps': [
                         {'name': 'Build App Store IPA', 'conclusion': 'success'}]}]
        self.artifact = {'name': 'preview-apple-status-1', 'id': 10, 'expired': False, 'size_in_bytes': 500}
        self.remote = Mock(repository=REPOSITORY)
        self.remote.api.side_effect = lambda path: self.run if path == 'actions/runs/50' else self.tag
        self.remote.pages.side_effect = lambda path, *args: (
            self.jobs if '/jobs?' in path else [self.artifact] if '/artifacts' in path else
            [{'ref': 'refs/tags/preview-42-50-1-dev', 'object': {'sha': 'tag-id', 'type': 'tag'}}])

    def recover(self, **kwargs):
        archive = io.BytesIO()
        with zipfile.ZipFile(archive, 'w') as zipped:
            zipped.writestr('.preview-ios.json', json.dumps(self.receipt))
        with patch('preview_recovery.subprocess.check_output', return_value=archive.getvalue()):
            return read_identity(self.remote, self.cycle, kwargs.get('run', '50'), kwargs.get('purpose', 'dev'))

    def test_processed_preview_reuses_only_original_apple_identity(self):
        self.assertEqual(self.recover(), {'version': '1.0.17', 'number': '21',
                                         'tag': 'preview-42-50-1-dev', 'apple_build': 'apple-build'})

    def test_untrusted_or_incomplete_runs_are_rejected(self):
        for key, value in [('head_branch', 'dev'), ('event', 'pull_request'),
                           ('status', 'in_progress'), ('conclusion', 'cancelled'),
                           ('path', '.github/workflows/other.yml')]:
            with self.subTest(key=key), patch.dict(self.run, {key: value}), self.assertRaises(ReleaseError):
                self.recover()
        self.jobs[0]['conclusion'] = 'failure'
        with self.assertRaises(ReleaseError): self.recover()

    def test_receipt_must_match_fixed_source_environment_and_reserved_identity(self):
        for key, value in [('source', 'b'*40), ('environment', 'production'),
                           ('stage', 'processing'), ('version', '1.0.18'), ('buildNumber', '22'), ('buildId', '')]:
            with self.subTest(key=key), patch.dict(self.receipt, {key: value}), self.assertRaises(ReleaseError):
                self.recover()

    def test_missing_expired_receipt_or_unbuilt_ipa_cannot_recover(self):
        self.artifact['expired'] = True
        with self.assertRaises(ReleaseError): self.recover()
        self.artifact['expired'] = False
        self.jobs[1]['steps'][0]['conclusion'] = 'skipped'
        with self.assertRaises(ReleaseError): self.recover()

    def test_wrong_cycle_tag_source_and_production_are_rejected(self):
        with self.assertRaises(ReleaseError): self.recover(purpose='production')
        with self.assertRaises(ReleaseError): self.recover(run='not-a-run')
        self.tag['object']['sha'] = 'b'*40
        with self.assertRaises(ReleaseError): self.recover()
