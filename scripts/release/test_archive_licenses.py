"""Exercise license delivery with the shipping GoReleaser archive configuration."""
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest
import zipfile


ROOT = Path(__file__).resolve().parents[2]
GORELEASER = os.environ.get('YOURTJ_GORELEASER') or shutil.which('goreleaser')


@unittest.skipUnless(GORELEASER, 'Install GoReleaser or set YOURTJ_GORELEASER')
class ArchiveLicenseTests(unittest.TestCase):
    def test_all_platform_archives_include_gpl_mit_and_scope_notice(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            app = root / 'apps/gooseforum'
            app.mkdir(parents=True)
            shutil.copyfile(ROOT / 'LICENSE', root / 'LICENSE')
            source = ROOT / 'apps/gooseforum'
            for pattern in ('LICENSE*', 'README*', '.goreleaser.yaml', 'scripts/windows/start.bat'):
                for path in source.glob(pattern):
                    destination = app / path.relative_to(source)
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(path, destination)
            # Only the executable is a fixture; file selection and archive formats
            # are the actual release configuration, including its parent-relative GPL path.
            (app / 'go.mod').write_text('module example.org/archive\n\ngo 1.26\n', encoding='utf-8')
            (app / 'main.go').write_text('package main\nfunc main() {}\n', encoding='utf-8')
            (root / '.gitignore').write_text('dist/\n', encoding='utf-8')
            subprocess.run(['git', 'init', '-q', str(root)], check=True, capture_output=True)
            subprocess.run(['git', 'add', '.'], cwd=root, check=True, capture_output=True)
            subprocess.run(['git', '-c', 'user.name=Archive fixture', '-c', 'user.email=fixture@example.org',
                            'commit', '-qm', 'fixture'], cwd=root, check=True, capture_output=True)
            result = subprocess.run([GORELEASER, 'release', '--snapshot', '--clean'], cwd=app,
                                    env={**os.environ, 'GOWORK': 'off'}, text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            archives = sorted((app / 'dist').glob('*.tar.gz')) + sorted((app / 'dist').glob('*.zip'))
            self.assertEqual(len(archives), 6, 'Expected all Linux, macOS and Windows archives')
            for archive in archives:
                with self.subTest(archive=archive.name):
                    if archive.suffix == '.zip':
                        with zipfile.ZipFile(archive) as packed:
                            files = {name: packed.read(name) for name in packed.namelist()}
                    else:
                        with tarfile.open(archive) as packed:
                            files = {entry.name: packed.extractfile(entry).read()
                                     for entry in packed.getmembers() if entry.isfile()}
                    self.assertIn('licenses/GPL-3.0-only/LICENSE', list(files), 'GPL text must accompany the executable')
                    self.assertEqual(files['licenses/GPL-3.0-only/LICENSE'], (ROOT / 'LICENSE').read_bytes())
                    self.assertEqual(files['LICENSE'], (source / 'LICENSE').read_bytes())
                    self.assertIn('LICENSES.md', list(files), 'The upstream MIT grant needs a scope notice')
                    self.assertEqual(files['LICENSES.md'], (source / 'LICENSES.md').read_bytes())
                    if archive.suffix == '.zip':
                        self.assertIn('start.bat', list(files))


if __name__ == '__main__':
    unittest.main()
