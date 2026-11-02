"""Confirm authorizes one exact RC and note snapshot, without a second build."""
import unittest
from confirmation import validate_seal
from model import ReleaseError, REPOSITORY


class ConfirmationTests(unittest.TestCase):
    def test_seal_cannot_authorize_another_source_head_notes_or_channel(self):
        manifest = {'candidateId': 'mobile-1.0.18-41', 'sourceSha': 'a' * 40, 'version': '1.0.18',
                    'buildNumber': 41, 'displayBuild': '202611030001', 'rcRun': '100',
                    'channels': ['android', 'ios-app-store']}
        seal = manifest | {'schemaVersion': 1, 'repository': REPOSITORY, 'head': 'b' * 40,
                           'contentDigest': 'c' * 64, 'actor': 'operator', 'run': '200',
                           'attempt': 1, 'controllerSha': 'd' * 40, 'triggeringActor': 'operator'}
        control = {'path': '.github/workflows/release-confirm.yml', 'event': 'workflow_dispatch',
                   'head_branch': 'main', 'head_sha': 'd' * 40, 'actor': {'login': 'operator'},
                   'head_repository': {'full_name': REPOSITORY}}
        validate_seal(seal, manifest, 'b' * 40, 'c' * 64, control, '200', REPOSITORY)
        for delta in [{'sourceSha': 'e' * 40}, {'head': 'e' * 40}, {'contentDigest': 'f' * 64},
                      {'rcRun': '101'}, {'channels': ['ios-app-store']}, {'buildNumber': 42},
                      {'displayBuild': '202611030002'}, {'actor': 'other'}, {'controllerSha': 'e' * 40}]:
            with self.subTest(delta=delta), self.assertRaises(ReleaseError):
                validate_seal(seal | delta, manifest, 'b' * 40, 'c' * 64, control, '200', REPOSITORY)
        for delta in [{'head_branch': 'dev'}, {'event': 'push'}, {'path': '.github/workflows/ci.yml'},
                      {'head_repository': {'full_name': 'fork/repo'}}]:
            with self.subTest(delta=delta), self.assertRaises(ReleaseError):
                validate_seal(seal, manifest, 'b' * 40, 'c' * 64, control | delta, '200', REPOSITORY)


if __name__ == '__main__': unittest.main()
