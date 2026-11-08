"""Exercise the exact uploaded dev script bundle, without preinstalled helpers."""
from contextlib import closing
import json
import os
from pathlib import Path
import re
import shutil
import sqlite3
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]


class DevDeployBundleTest(unittest.TestCase):
    def test_uploaded_bundle_is_self_contained_and_isolates_copied_credentials(self):
        workflow = (ROOT / '.github/workflows/deploy-dev.yml').read_text()
        sources = re.search(r'source: "(deploy/scripts/[^"\n]+)"', workflow)[1].split(',')
        uploads = {Path(source).name: ROOT / source for source in sources}
        installs = re.findall(r'install -m \d+ "\$RUN_DIR/([^"\n]+)" /opt/yourtj/scripts/(\S+)', workflow)
        with tempfile.TemporaryDirectory(prefix='yourtj-deploy-bundle-') as temporary:
            root = Path(temporary)
            scripts = root / 'scripts'
            scripts.mkdir()
            for source, destination in installs:
                self.assertIn(source, uploads, f'{source} is installed but not uploaded')
                self.assertTrue(uploads[source].is_file(), f'{source} is absent from the trusted checkout')
                target = scripts / destination
                shutil.copyfile(uploads[source], target)
                target.chmod(0o755)
            for instance in ('main', 'dev'):
                (root / instance / 'storage/database').mkdir(parents=True)
                (root / instance / 'config.toml').write_text('[db.default]\nconnection="sqlite"\n')
            source = root / 'main/storage/database/sqlite.db'
            with closing(sqlite3.connect(source)) as db, db:
                db.executescript("CREATE TABLE agents(enabled INTEGER,token_hash TEXT);"
                                 "INSERT INTO agents VALUES(1,'production-token');"
                                 "CREATE TABLE feed_serve_log(id TEXT);"
                                 "INSERT INTO feed_serve_log VALUES('private-observation');")
            commands = root / 'bin'
            commands.mkdir()
            docker = commands / 'docker'
            docker.write_text('#!/bin/sh\nexit 0\n')
            docker.chmod(0o755)
            result = subprocess.run(['bash', str(scripts / 'sync-db-from-main.sh')],
                                    env=dict(os.environ, YOURTJ_ROOT=str(root),
                                             PATH=str(commands) + os.pathsep + os.environ['PATH']),
                                    capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            with closing(sqlite3.connect(root / 'dev/storage/database/sqlite.db')) as db, db:
                self.assertEqual(db.execute('SELECT * FROM agents').fetchone(), (0, ''))
                self.assertEqual(db.execute('SELECT count(*) FROM feed_serve_log').fetchone()[0], 0)
            state = json.loads((root / 'dev/storage/agent-state/state.json').read_text())
            self.assertFalse(state['apiEnabled'])
            self.assertFalse(state['producerEnabled'])
            self.assertFalse(state['webhookEnabled'])
            with closing(sqlite3.connect(source)) as db, db:
                self.assertEqual(db.execute('SELECT * FROM agents').fetchone(), (1, 'production-token'))
                self.assertEqual(db.execute('SELECT count(*) FROM feed_serve_log').fetchone()[0], 1)


if __name__ == '__main__':
    unittest.main()
