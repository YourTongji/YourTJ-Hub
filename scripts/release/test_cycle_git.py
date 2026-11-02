"""Real merge trees prove that main can advance without changing release source."""
import subprocess
import tempfile
from pathlib import Path
import unittest
from unittest.mock import Mock, patch
from cycle_control import reviewed_main_pr
from model import REPOSITORY, ReleaseError


class CycleGitTests(unittest.TestCase):
    def test_clean_integration_keeps_exact_source_and_manual_merge_edits_require_qa(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            def git(*args):
                return subprocess.check_output(['git', '-C', temporary, *args], text=True, stderr=subprocess.DEVNULL).strip()
            git('init', '-q', '-b', 'main')
            git('config', 'user.name', 'Fixture'); git('config', 'user.email', 'fixture@example.org')
            (root / 'base').write_text('base')
            git('add', '.'); git('commit', '-qm', 'base')
            git('checkout', '-qb', 'candidate')
            (root / 'app').write_text('tested source')
            git('add', '.'); git('commit', '-qm', 'candidate')
            source = git('rev-parse', 'HEAD')
            git('checkout', '-q', 'main')
            (root / 'unpublished').write_text('main continues independently')
            git('add', '.'); git('commit', '-qm', 'main metadata')
            git('merge', '--no-ff', '-qm', 'integrate candidate', 'candidate')
            merge = git('rev-parse', 'HEAD')
            git('update-ref', 'refs/remotes/origin/main', merge)
            pr = {'number': 12, 'merged_at': 'today', 'merge_commit_sha': merge,
                  'head': {'sha': source, 'repo': {'full_name': REPOSITORY}},
                  'base': {'ref': 'main'}, 'user': {'login': 'author'}}
            review = {'user': {'login': 'reviewer'}, 'state': 'APPROVED', 'commit_id': source}
            remote = Mock(repository=REPOSITORY)
            def pages(endpoint):
                if endpoint.startswith('commits/'):
                    return [{**pr, 'base': {'ref': 'dev'}}]
                return [pr] if endpoint == 'pulls?state=closed&base=main' else [review]
            remote.pages.side_effect = pages
            remote.maintainers.return_value = {'reviewer'}
            with patch('cycle_control.git', side_effect=git):
                reviewed_main_pr(remote, source)
                self.assertNotEqual(source, merge)
                (root / 'app').write_text('untested conflict resolution')
                git('add', '.'); git('commit', '--amend', '-qm', 'manual merge change')
                pr['merge_commit_sha'] = git('rev-parse', 'HEAD')
                git('update-ref', 'refs/remotes/origin/main', pr['merge_commit_sha'])
                with self.assertRaisesRegex(ReleaseError, 'Integration changed code'):
                    reviewed_main_pr(remote, source)


if __name__ == '__main__': unittest.main()
