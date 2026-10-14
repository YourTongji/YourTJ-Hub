import pathlib
import subprocess
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).with_name('sync-db-from-main.sh').read_text()


class SnapshotRetentionTest(unittest.TestCase):
    def test_failed_sync_keeps_old_snapshots(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            directory = root / 'snapshots/dev-prev'
            directory.mkdir(parents=True)
            for day in range(1, 9):
                (directory / f'file-202610{day:02d}_030000.db').write_text('keep')
            main = root / 'main/storage/database'
            main.mkdir(parents=True)
            (main / 'file.db').write_text('source')
            commands = root / 'bin'
            commands.mkdir()
            for name, body in [('docker', 'exit 0'), ('sqlite3', 'exit 1'),
                               ('python3', 'exit 0')]:
                command = commands / name
                command.write_text('#!/bin/sh\n' + body + '\n')
                command.chmod(0o755)
            result = subprocess.run(['bash', str(pathlib.Path(__file__).with_name('sync-db-from-main.sh'))],
                                    env={'PATH': str(commands) + ':/usr/bin:/bin', 'YOURTJ_ROOT': tmp},
                                    capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertEqual(len(list(directory.iterdir())), 8)

    def test_postgres_success_prunes_existing_snapshots(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            directory = root / 'snapshots/dev-prev'
            directory.mkdir(parents=True)
            for day in range(1, 9):
                (directory / f'file-202609{day:02d}_030000.db').write_text('old')
            for instance in ('main', 'dev'):
                storage = root / instance / 'storage/database'
                storage.mkdir(parents=True)
                (storage / 'file.db').write_text(instance)
                (root / instance / 'config.toml').write_text(
                    '[db.default]\nconnection = "postgres"\nurl = "postgres://localhost/' + instance + '"\n')
            commands = root / 'bin'
            commands.mkdir()
            mocks = {'docker': '#!/bin/sh\ncat >/dev/null\n',
                     'sqlite3': '#!/usr/bin/env python3\nimport shutil,sys\n'
                                'shutil.copyfile(sys.argv[1], sys.argv[2].split("\'")[1])\n'}
            for name, body in mocks.items():
                command = commands / name
                command.write_text(body)
                command.chmod(0o755)
            result = subprocess.run(['bash', str(pathlib.Path(__file__).with_name('sync-db-from-main.sh'))],
                                    env={'PATH': str(commands) + ':/usr/bin:/bin', 'YOURTJ_ROOT': tmp},
                                    capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(len(list(directory.iterdir())), 5)
            self.assertEqual((root / 'dev/storage/database/file.db').read_text(), 'main')
            self.assertEqual((root / 'main/storage/database/file.db').read_text(), 'main')
            self.assertFalse((directory / 'file-20260901_030000.db').exists())

    def test_retention(self):
        start = SCRIPT.index('prune_dev_snapshots() {')
        function = SCRIPT[start:SCRIPT.index('\n}\n', start) + 3]
        for count in (0, 3, 8):
            with self.subTest(count=count), tempfile.TemporaryDirectory() as tmp:
                directory = pathlib.Path(tmp) / 'snapshots/dev-prev'
                directory.mkdir(parents=True)
                for day in range(1, count + 1):
                    for label in ('sqlite', 'file'):
                        (directory / f'{label}-202610{day:02d}_030000.db').write_text('fixture')
                unknown = directory / 'file-not-a-snapshot.db'
                unknown.write_text('keep')
                link = directory / 'file-20260901_030000.db'
                link.symlink_to(unknown)
                subprocess.run(['bash', '-eu', '-c', function + '\nprune_dev_snapshots', 'test', tmp],
                               env={'PATH': '/usr/bin:/bin', 'ROOT': tmp}, check=True)
                expected = {f'{label}-202610{day:02d}_030000.db'
                            for day in range(max(1, count - 4), count + 1)
                            for label in ('sqlite', 'file')}
                self.assertEqual({p.name for p in directory.iterdir()},
                                 expected | {unknown.name, link.name})


if __name__ == '__main__':
    unittest.main()
