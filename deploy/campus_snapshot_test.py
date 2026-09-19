"""Exercise the real SQLite snapshot script against disposable local fixtures."""
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import unittest
class CampusSnapshotTests(unittest.TestCase):
    def test_school_credentials_are_removed_before_snapshot_install(self):
        with tempfile.TemporaryDirectory(prefix='yourtj-campus-snapshot-') as temp:
            root = Path(temp)
            for instance in ['main', 'dev']:
                (root / instance / 'storage/database').mkdir(parents=True)
                (root / instance / 'config.toml').write_text('[db.default]\nconnection="sqlite"\n')
            source = root / 'main/storage/database/sqlite.db'
            with sqlite3.connect(source) as db:
                db.executescript('CREATE TABLE campus_identity_bindings(user_id INTEGER, sealed TEXT); INSERT INTO campus_identity_bindings VALUES(1, "fictional-secret"); CREATE TABLE ordinary(value TEXT); INSERT INTO ordinary VALUES("keep");')
            tools = root / 'bin'
            tools.mkdir()
            docker = tools / 'docker'
            docker.write_text('#!/bin/sh\nexit 0\n')
            docker.chmod(0o700)
            script = Path(__file__).parent / 'scripts/sync-db-from-main.sh'
            env = dict(os.environ, YOURTJ_ROOT=str(root), PATH=str(tools)+os.pathsep+os.environ['PATH'])
            subprocess.run(['bash',str(script)],env=env,check=True,capture_output=True)
            with sqlite3.connect(root / 'dev/storage/database/sqlite.db') as db:
                self.assertEqual(db.execute('SELECT count(*) FROM campus_identity_bindings').fetchone()[0],0)
                self.assertEqual(db.execute('SELECT value FROM ordinary').fetchone()[0],'keep')
            with sqlite3.connect(source) as db:
                self.assertEqual(db.execute('SELECT count(*) FROM campus_identity_bindings').fetchone()[0],1)
if __name__ == '__main__': unittest.main()
