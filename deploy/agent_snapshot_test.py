"""Run production credential isolation against a disposable snapshot and state."""
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import unittest

class AgentSnapshotTests(unittest.TestCase):
    def test_tokens_secrets_tasks_and_epoch_are_isolated_before_dev_restart(self):
        with tempfile.TemporaryDirectory(prefix='yourtj-agent-isolation-') as temporary:
            root = Path(temporary)
            for instance in ['main','dev']:
                (root / instance / 'storage/database').mkdir(parents=True)
                (root / instance / 'config.toml').write_text('[db.default]\nconnection="sqlite"\n')
            state_dir = root / 'dev/storage/agent-state'
            state_dir.mkdir()
            (state_dir/'state.json').write_text(json.dumps({'instanceId':'inst_dev','streamEpoch':'old','apiEnabled':True,'producerEnabled':True,'webhookEnabled':True}))
            source = root / 'main/storage/database/sqlite.db'
            with sqlite3.connect(source) as db:
                db.executescript("CREATE TABLE agents(enabled INTEGER,token_hash TEXT,secret_ciphertext TEXT,previous_secret_ciphertext TEXT,webhook_enabled INTEGER,events_enabled INTEGER); INSERT INTO agents VALUES(1,'production-hash','sealed','sealed-old',1,1); CREATE TABLE task_queue(type TEXT,status INTEGER,last_error TEXT); INSERT INTO task_queue VALUES('agent-webhook.deliver',0,''); INSERT INTO task_queue VALUES('email.activation',0,''); CREATE TABLE agent_events(id TEXT); INSERT INTO agent_events VALUES('evt_production');")
            tools=root/'bin';tools.mkdir();docker=tools/'docker';docker.write_text('#!/bin/sh\nexit 0\n');docker.chmod(0o700)
            script=Path(__file__).parent/'scripts/sync-db-from-main.sh'
            env=dict(os.environ,YOURTJ_ROOT=str(root),PATH=str(tools)+os.pathsep+os.environ['PATH'])
            subprocess.run(['bash',str(script)],env=env,check=True,capture_output=True)
            with sqlite3.connect(root/'dev/storage/database/sqlite.db') as db:
                self.assertEqual(db.execute('SELECT * FROM agents').fetchone(),(0,'','','',0,0))
                self.assertEqual(db.execute("SELECT status FROM task_queue WHERE type='agent-webhook.deliver'").fetchone()[0],3)
                self.assertEqual(db.execute("SELECT status FROM task_queue WHERE type='email.activation'").fetchone()[0],0)
                self.assertEqual(db.execute('SELECT count(*) FROM agent_events').fetchone()[0],0)
            state=json.loads((state_dir/'state.json').read_text())
            self.assertEqual(state['instanceId'],'inst_dev');self.assertNotEqual(state['streamEpoch'],'old')
            self.assertFalse(state['apiEnabled']);self.assertFalse(state['producerEnabled']);self.assertFalse(state['webhookEnabled'])
            with sqlite3.connect(source) as db:self.assertEqual(db.execute('SELECT enabled FROM agents').fetchone()[0],1)

if __name__=='__main__':unittest.main()
