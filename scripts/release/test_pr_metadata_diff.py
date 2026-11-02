"""Queued PR checks use the proposed branch, not unrelated newer main changes."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from test_model import candidate
import workflow

class PRMetadataDiffTests(unittest.TestCase):
    def test_main_advances_between_event_and_synthetic_merge_checkout(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary)
            def git(*args):
                return subprocess.check_output(['git','-C',str(root),*args],text=True,stderr=subprocess.DEVNULL).strip()
            git('init','-q','-b','main');git('config','user.name','Fixture');git('config','user.email','fixture@example.org')
            (root/'README.md').write_text('base');git('add','.');git('commit','-qm','base');base=git('rev-parse','HEAD')
            git('switch','-qc','request')
            manifest=candidate();folder=root/'releases/requests'/manifest['candidateId'];folder.mkdir(parents=True)
            (folder/'manifest.json').write_text(json.dumps(manifest))
            (folder/'evidence.json').write_text(json.dumps({'schemaVersion':1,'sourceSha':manifest['sourceSha']}))
            (folder/'android.zh-CN.md').write_text('改善聊天显示。')
            (folder/'testflight.en-US.txt').write_text('Verify chat rendering.')
            git('add','.');git('commit','-qm','release request');head=git('rev-parse','HEAD')
            git('switch','-q','main');(root/'controller.py').write_text('new main controller');git('add','.');git('commit','-qm','advance main')
            target=git('rev-parse','HEAD')
            git('merge','--no-ff','--no-edit','request')
            event=root/'event.json';event.write_text(json.dumps({'pull_request':{'number':1,'base':{'sha':base},'head':{'sha':head,'ref':'codex/release/'+manifest['candidateId']}}}))
            previous=Path.cwd()
            try:
                os.chdir(root)
                with patch.dict(os.environ,GITHUB_EVENT_PATH=str(event)), patch('sys.argv',['workflow.py','validate-pr']), patch('workflow.GitHub'):
                    workflow.main()
                # CI must keep this a metadata-only check as well: newer main
                # executable inputs do not request another source/native matrix.
                import contextlib, io, sys
                sys.path.insert(0, str(Path(__file__).resolve().parents[1]/'ci'))
                import select_inputs
                with patch.dict(os.environ,GITHUB_EVENT_NAME='pull_request',GITHUB_EVENT_PATH=str(event),GITHUB_OUTPUT=''), patch('sys.argv',['select_inputs.py','--output',str(root/'plan.json')]), contextlib.redirect_stdout(io.StringIO()):
                    select_inputs.main()
                plan=json.loads((root/'plan.json').read_text())
                self.assertEqual({k for k,v in plan.items() if v['run']},{'docs','govulncheck','release'})
                # Updating the request from main must not turn known main edits
                # into author changes when the event's base SHA is old.
                git('switch','-q','request');git('merge','--no-edit',target)
                updated=git('rev-parse','HEAD')
                git('switch','-qc','synthetic-updated',target);git('merge','--no-ff','--no-edit','request')
                data=json.loads(event.read_text());data['pull_request']['head']['sha']=updated;event.write_text(json.dumps(data))
                with patch.dict(os.environ,GITHUB_EVENT_PATH=str(event)), patch('sys.argv',['workflow.py','validate-pr']), patch('workflow.GitHub'):
                    workflow.main()
                # A real extra change on the PR branch must still be rejected.
                git('switch','-q','request');(root/'foreign.md').write_text('unrelated');git('add','foreign.md');git('commit','-qm','foreign edit')
                data=json.loads(event.read_text());data['pull_request']['head']['sha']=git('rev-parse','HEAD');event.write_text(json.dumps(data))
                from model import ReleaseError
                with patch.dict(os.environ,GITHUB_EVENT_PATH=str(event)), patch('sys.argv',['workflow.py','validate-pr']), patch('workflow.GitHub'), self.assertRaises(ReleaseError):
                    workflow.main()
            finally:
                os.chdir(previous)

if __name__=='__main__':unittest.main()
