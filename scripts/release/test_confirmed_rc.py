"""New requests cannot publish unbuilt or reuse another identity's confirmation."""
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock, patch
from confirmation import read_confirmation
from model import validate_candidate, ReleaseError, REPOSITORY
from test_model import candidate
from rc import validate_identity

class ConfirmedRCTests(unittest.TestCase):
    def test_schema_four_requires_real_daily_id_and_selected_rc(self):
        manifest = candidate() | {'schemaVersion': 4, 'rcRun': '100',
                                 'displayBuild': '202611030001', 'iosBuildNumber': '202611030001'}
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for name in manifest['notes'].values(): (root / name).write_text('[DRAFT: notes]')
            changelog = {'schemaVersion': 1, 'version': manifest['version'], 'buildNumber': 15,
                         'displayBuild': manifest['displayBuild'], 'highlights': [], 'breaking': [],
                         'requiredActions': [], 'evidence': {}, 'testflightNotes': []}
            (root/'changelog.json').write_text(json.dumps(changelog))
            (root/'evidence.json').write_text(json.dumps({'schemaVersion': 1, 'sourceSha': manifest['sourceSha'], 'evidence': []}))
            validate_candidate(manifest, root, draft=True)
            for delta in [{'displayBuild': '202602300001'}, {'displayBuild': '202611030000'},
                          {'displayBuild': '2026110342'}, {'iosBuildNumber': 'bad'}]:
                with self.subTest(delta=delta), self.assertRaises(ReleaseError):
                    validate_candidate(manifest | delta, root, draft=True)
            with self.assertRaisesRegex(ReleaseError, 'completed production RC'):
                validate_candidate(manifest | {'rcRun': None}, root)

    def test_mutable_pointer_without_trusted_immutable_seal_cannot_authorize(self):
        remote = Mock(repository=REPOSITORY)
        manifest = candidate() | {'displayBuild': '202611030001', 'iosBuildNumber': '202611030001', 'rcRun': '100'}
        pr = {'head': {'sha': 'b'*40}, 'merge_commit_sha': 'c'*40}
        pointer = {'candidateId': manifest['candidateId'], 'head': 'b'*40, 'run': '200', 'attempt': 1}
        remote.pages.return_value = [{'payload': pointer}]
        control = {'path': '.github/workflows/release-confirm.yml', 'event': 'workflow_dispatch',
                   'head_branch': 'main', 'head_sha': 'd'*40, 'actor': {'login': 'operator'},
                   'head_repository': {'full_name': REPOSITORY}}
        remote.api.return_value = control
        seal = {k:manifest.get(k) for k in ('candidateId','sourceSha','version','buildNumber','displayBuild','iosBuildNumber','rcRun','channels')}
        seal |= {'schemaVersion': 1, 'repository': REPOSITORY, 'head': 'b'*40, 'contentDigest': 'e'*64,
                 'run': '200', 'attempt': 1, 'actor': 'operator', 'triggeringActor': 'operator', 'controllerSha': 'd'*40}
        with patch('confirmation.read_json_artifact', return_value=seal) as artifact, patch('confirmation.human') as human:
            binding = read_confirmation(remote, manifest, pr, 'e'*64)
            self.assertEqual(binding['confirmationRun'], '200')
            artifact.assert_called_once_with(remote, '200', 'release-confirmation-1', 'confirmation.json')
            human.assert_called_once_with(remote, 'operator')
            with self.assertRaises(ReleaseError): read_confirmation(remote, manifest, pr, 'f'*64)
            remote.api.return_value = control | {'head_branch': 'dev'}
            with self.assertRaises(ReleaseError): read_confirmation(remote, manifest, pr, 'e'*64)

    def test_existing_build_metadata_promotion_keeps_original_rc_identity(self):
        original = {'candidateId': 'mobile-1.0.18-42', 'sourceSha': 'a'*40, 'product': 'mobile',
                    'version': '1.0.18', 'buildNumber':42, 'displayBuild':'202611030001',
                    'iosBuildNumber':'202611030001', 'environment':'production', 'preview':False}
        promotion = original | {'candidateId':'mobile-1.0.18-42-store-200', 'operation':'promote-ios',
                                'existingRelease':{'candidateId':original['candidateId'],'buildId':'apple-build'}}
        validate_identity(original, promotion)
        with self.assertRaises(ReleaseError): validate_identity(original, promotion | {'iosBuildNumber':'202611030002'})

if __name__ == '__main__': unittest.main()
