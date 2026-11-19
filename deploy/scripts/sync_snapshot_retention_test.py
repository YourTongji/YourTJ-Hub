import json
import os
import pathlib
import sqlite3
import subprocess
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).with_name('sync-db-from-main.sh')
ROOT_NAME = 'snapshots/dev-prev'


def write_executable(path, body):
    path.write_text(body)
    path.chmod(0o755)


def make_marker_db(path, marker):
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists():
        path.unlink()
    with sqlite3.connect(path) as db:
        db.execute('CREATE TABLE marker(v TEXT)')
        db.execute('INSERT INTO marker VALUES (?)', (marker,))


def marker_of(path):
    with sqlite3.connect(path) as db:
        return db.execute('SELECT v FROM marker').fetchone()[0]


class SyncWithoutDevPrevTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.tmp.name)
        self.commands = self.root / 'bin'
        self.commands.mkdir()
        self.docker_log = self.root / 'docker.log'
        write_executable(
            self.commands / 'docker',
            '#!/bin/sh\nprintf "%s\\n" "$*" >> "$DOCKER_LOG"\ncat >/dev/null\nexit 0\n',
        )
        self.env = {
            'PATH': str(self.commands) + os.pathsep + '/usr/bin:/bin',
            'YOURTJ_ROOT': str(self.root),
            'DOCKER_LOG': str(self.docker_log),
        }

    def tearDown(self):
        self.tmp.cleanup()

    def run_sync(self):
        return subprocess.run(
            ['bash', str(SCRIPT)],
            env=self.env,
            capture_output=True,
            text=True,
        )

    def docker_text(self):
        if not self.docker_log.exists():
            return ''
        return self.docker_log.read_text()

    def write_config(self, instance, connection, url):
        path = self.root / instance / 'config.toml'
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(
            '[db.default]\nconnection = "%s"\nurl = "%s"\n' % (connection, url)
        )

    def test_sqlite_mode_copies_main_to_dev_and_skips_dev_prev(self):
        make_marker_db(self.root / 'main/storage/database/sqlite.db', 'main-sqlite')
        make_marker_db(self.root / 'main/storage/database/file.db', 'main-file')
        make_marker_db(self.root / 'dev/storage/database/sqlite.db', 'old-dev-sqlite')
        make_marker_db(self.root / 'dev/storage/database/file.db', 'old-dev-file')
        self.write_config('main', 'sqlite', '')
        self.write_config('dev', 'sqlite', '')

        result = self.run_sync()

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(marker_of(self.root / 'dev/storage/database/sqlite.db'), 'main-sqlite')
        self.assertEqual(marker_of(self.root / 'dev/storage/database/file.db'), 'main-file')
        self.assertEqual(marker_of(self.root / 'main/storage/database/sqlite.db'), 'main-sqlite')
        self.assertEqual(marker_of(self.root / 'main/storage/database/file.db'), 'main-file')
        self.assertFalse((self.root / ROOT_NAME).exists())
        log = self.docker_text()
        self.assertIn('stop dev', log)
        self.assertNotIn('pg_dump', log)
        state = json.loads((self.root / 'dev/storage/agent-state/state.json').read_text())
        self.assertFalse(state['apiEnabled'])

    def test_postgres_mode_copies_file_db_and_skips_dev_prev(self):
        make_marker_db(self.root / 'main/storage/database/file.db', 'main-file')
        make_marker_db(self.root / 'dev/storage/database/file.db', 'old-dev-file')
        self.write_config('main', 'postgres', 'postgres://localhost/yourtj_main')
        self.write_config('dev', 'postgres', 'postgres://localhost/yourtj_dev')
        previous = self.root / ROOT_NAME
        previous.mkdir(parents=True)
        (previous / 'file-20261001_030000.db').write_text('already-there')

        result = self.run_sync()

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(marker_of(self.root / 'dev/storage/database/file.db'), 'main-file')
        self.assertEqual(marker_of(self.root / 'main/storage/database/file.db'), 'main-file')
        self.assertEqual(
            sorted(path.name for path in previous.iterdir()),
            ['file-20261001_030000.db'],
        )
        log = self.docker_text()
        self.assertLess(log.index('stop dev'), log.index('pg_dump'))
        self.assertIn('yourtj_main', log)
        self.assertIn('yourtj_dev', log)
        state = json.loads((self.root / 'dev/storage/agent-state/state.json').read_text())
        self.assertFalse(state['apiEnabled'])

    def test_mode_mismatch_exits_before_stopping_dev(self):
        self.write_config('main', 'postgres', 'postgres://localhost/yourtj_main')
        self.write_config('dev', 'sqlite', '')

        result = self.run_sync()

        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.docker_text(), '')
        self.assertFalse((self.root / ROOT_NAME).exists())

    def test_invalid_postgres_dsn_exits_before_stopping_dev(self):
        self.write_config('main', 'postgres', '')
        self.write_config('dev', 'postgres', 'postgres://localhost/yourtj_dev')

        result = self.run_sync()

        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.docker_text(), '')
        self.assertFalse((self.root / ROOT_NAME).exists())

    def test_failed_sqlite_backup_leaves_dev_and_old_copies_unchanged(self):
        make_marker_db(self.root / 'main/storage/database/sqlite.db', 'main-sqlite')
        make_marker_db(self.root / 'dev/storage/database/sqlite.db', 'old-dev-sqlite')
        write_executable(self.commands / 'sqlite3', '#!/bin/sh\nexit 1\n')
        previous = self.root / ROOT_NAME
        previous.mkdir(parents=True)
        (previous / 'sqlite-20261001_030000.db').write_text('already-there')

        result = self.run_sync()

        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(marker_of(self.root / 'dev/storage/database/sqlite.db'), 'old-dev-sqlite')
        self.assertEqual(
            sorted(path.name for path in previous.iterdir()),
            ['sqlite-20261001_030000.db'],
        )
        self.assertIn('stop dev', self.docker_text())


if __name__ == '__main__':
    unittest.main()
