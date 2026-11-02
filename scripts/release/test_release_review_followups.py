import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock, patch
from cli import execute, parser
from model import REPOSITORY
from rc import resolve

ROOT=Path(__file__).resolve().parents[2]/'.github/workflows'

class ReleaseReviewFollowups(unittest.TestCase):
    def test_non_ios_rc_scopes_ignore_unrelated_default_ios_destination(self):
        for scope in ['web','android']:
            result=execute(parser().parse_args(['rc','--scope',scope]), Mock())
            self.assertEqual(result['inputs']['ios_destination'],'testflight')

    def test_metadata_promotion_verifies_original_reserved_tag(self):
        remote=Mock(repository=REPOSITORY)
        candidate='mobile-1.0.18-43-store-200'
        pr={'number':1,'state':'open','base':{'ref':'main'},'head':{'repo':{'full_name':REPOSITORY},'ref':'codex/release/'+candidate,'sha':'b'*40}}
        manifest={'candidateId':candidate,'schemaVersion':4,'operation':'promote-ios','rcRun':'100','sourceSha':'a'*40,'tag':'mobile-v1.0.18','channels':['ios-app-store']}
        remote.pages.return_value=[pr]
        remote.api.side_effect=lambda path,**kw:pr if path=='pulls/1' else {'object':{'type':'tag','sha':'c'*40}}
        with tempfile.TemporaryDirectory() as temp, patch('rc.git'), patch('controller.load_candidate',return_value=(manifest,Path(temp))), patch('build_identity.verify_production_number'), patch('state.latest_receipt',return_value=None), patch('controller.verify_reservation') as verify:
            resolve(remote,candidate,Path(temp))
            verify.assert_called_once()
            self.assertEqual(verify.call_args.args[0],manifest)

    def test_each_platform_refreshes_status_without_waiting_for_other_platforms(self):
        for name in ['release-preview-android.yml','release-preview-ios.yml']:
            text=(ROOT/name).read_text()
            self.assertIn('uses: ./.github/workflows/dev-status.yml',text)
            self.assertIn('needs: preview',text)
        text=(ROOT/'dev-distribute.yml').read_text()
        self.assertIn('needs: web',text)
        status=(ROOT/'dev-status.yml').read_text()
        self.assertIn('group: dev-public-status',status)
        self.assertIn('queue: max',status)

    def test_existing_candidate_product_overrides_unrelated_scope_for_lock(self):
        rc=(ROOT/'release-rc.yml').read_text().split('\njobs:',1)[0]
        self.assertIn("inputs.candidate == '' && inputs.scope == 'web'",rc)

if __name__=='__main__':unittest.main()
