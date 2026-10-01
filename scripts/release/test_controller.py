"""Exercise the real Git approval boundary, including byte-preserving recovery."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from controller import authorize, load_candidate
from model import ReleaseError
from test_model import candidate


class ControllerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / 'repo'
        self.repo.mkdir()
        self.git('init', '-q')
        self.git('config', 'user.name', 'Fixture')
        self.git('config', 'user.email', 'fixture@example.org')
        (self.repo / 'README').write_text('source\n')
        self.git('add', '.')
        self.git('commit', '-qm', 'source')
        self.source = self.git('rev-parse', 'HEAD')
        self.manifest = candidate() | {'sourceSha': self.source}
        self.prefix = 'releases/requests/' + self.manifest['candidateId']
        folder = self.repo / self.prefix
        folder.mkdir(parents=True)
        self.bytes = b'  Human reviewed Android note.  \n\n'
        (folder / 'manifest.json').write_text(json.dumps(self.manifest))
        (folder / 'evidence.json').write_text(json.dumps({'schemaVersion': 1, 'sourceSha': self.source}))
        (folder / 'android.zh-CN.md').write_bytes(self.bytes)
        (folder / 'testflight.en-US.txt').write_text('Test only the iOS widget.\n')
        self.git('add', '.')
        self.git('commit', '-qm', 'release data')
        self.head = self.git('rev-parse', 'HEAD')
        self.git('update-ref', 'refs/remotes/origin/main', self.head)

    def git(self, *args):
        if args[0] == 'fetch':
            return ''
        return subprocess.check_output(['git', '-C', str(self.repo), *args], text=True).strip()

    def test_materialization_preserves_exact_reviewed_bytes_and_rejects_stale_files(self):
        destination = self.root / 'candidate'
        # Use the actual git blob reader in the repository under test.
        previous = Path.cwd()
        try:
            os.chdir(self.repo)
            load_candidate(self.manifest['candidateId'], self.head, destination)
            self.assertEqual((destination / 'android.zh-CN.md').read_bytes(), self.bytes)
            (destination / 'unexpected.txt').write_text('not reviewed')
            with self.assertRaises(ReleaseError):
                load_candidate(self.manifest['candidateId'], self.head, destination)
        finally:
            os.chdir(previous)

    def test_authorize_binds_human_head_and_refuses_altered_merge_or_ci(self):
        outer = self
        class Remote:
            repository = 'YourTongji/YourTJ-Hub'
            conclusion = 'success'
            def pages(self, endpoint, key=None):
                if endpoint.startswith('pulls?'): return [{'number': 123, 'merged_at': 'today'}]
                if endpoint.endswith('/reviews'):
                    return [{'id': 1, 'state': 'APPROVED', 'commit_id': outer.head,
                             'submitted_at': 'today', 'user': {'type': 'User', 'login': 'owner'}}]
                if endpoint.endswith('/check-runs'):
                    return [{'id': 1, 'name': 'ci-required', 'app': {'slug': 'github-actions'}, 'conclusion': self.conclusion}]
                raise AssertionError(endpoint)
            def api(self, endpoint):
                return {'number': 123, 'merged': True, 'changed_files': 4, 'merge_commit_sha': outer.git('rev-parse', 'origin/main'),
                        'head': {'sha': outer.head, 'ref': 'codex/release/' + outer.manifest['candidateId'],
                                 'repo': {'full_name': self.repository}}, 'base': {'ref': 'main'}}
            def files(self, number, expected):
                return [outer.prefix + '/' + name for name in ('manifest.json', 'evidence.json', 'android.zh-CN.md', 'testflight.en-US.txt')]
            def maintainers(self, reviews): return {'owner'}
        remote = Remote()
        previous = Path.cwd()
        try:
            os.chdir(self.repo)
            with patch('controller.git', side_effect=self.git):
                _, first = authorize(self.manifest['candidateId'], remote, self.root / 'approved')
                _, second = authorize(self.manifest['candidateId'], remote, self.root / 'approved')
                self.assertEqual(first['contentDigest'], second['contentDigest'])
                remote.conclusion = 'failure'
                with self.assertRaises(ReleaseError): authorize(self.manifest['candidateId'], remote, self.root / 'approved')
                remote.conclusion = 'success'
                (self.repo / self.prefix / 'android.zh-CN.md').write_text('Unreviewed merge edit')
                self.git('commit', '-qam', 'altered merge')
                self.git('update-ref', 'refs/remotes/origin/main', self.git('rev-parse', 'HEAD'))
                with self.assertRaises(ReleaseError): authorize(self.manifest['candidateId'], remote, self.root / 'approved')
        finally:
            os.chdir(previous)
