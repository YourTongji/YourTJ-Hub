import unittest
import json
import os
import tempfile
from unittest.mock import Mock, patch
import workflow
from cli import parser, execute
from cycle_control import production_gate, human, reviewed_preview_source, main as control
from model import ReleaseError, next_identity, REPOSITORY
from build_identity import verify_production_number
from cycle import empty_state, transition
from test_model import candidate
SHA = 'a' * 40


class CycleIntegrationTests(unittest.TestCase):
    def test_all_production_entrypoints_fail_before_side_effects_without_qa(self):
        manifest = candidate()
        for command in ['authorize', 'reserve', 'start', 'verify-publication', 'verify-deploy']:
            with self.subTest(command=command), patch('sys.argv', ['workflow.py', command]), \
                 patch('workflow.authorize', return_value=(manifest, {})), \
                 patch('workflow.production_gate', side_effect=ReleaseError('QA missing')) as gate, \
                 patch('workflow.reserve') as reserve, patch('workflow.record') as record, \
                 patch('workflow.emit_outputs') as emit:
                with self.assertRaisesRegex(ReleaseError, 'QA missing'): workflow.main()
                gate.assert_called_once()
                reserve.assert_not_called(); record.assert_not_called(); emit.assert_not_called()

    def test_preview_and_production_consume_unique_numbers_without_consuming_versions(self):
        version, number = next_identity('mobile', 'patch', [
            {'tag': 'mobile-v1.2.3', 'buildNumber': 24}, {'tag': 'mobile-build-30', 'buildNumber': 30}])
        self.assertEqual((version, number), ('1.2.4', 31))

    def test_dev_ipa_number_cannot_be_relabelled_by_a_reviewed_release_pr(self):
        remote = Mock()
        manifest = candidate()
        metadata = {'environment': 'production', 'version': manifest['version'], 'buildNumber': manifest['buildNumber']}
        annotation = {'object': {'type': 'commit', 'sha': manifest['sourceSha']}, 'message': json.dumps(metadata)}
        marker = {'object': {'type': 'tag', 'sha': 'tag'}}
        remote.api.side_effect = [marker, annotation]
        verify_production_number(remote, manifest)
        for environment in ['dev-preview', 'production-preview']:
            remote.api.side_effect = [marker, annotation | {'message': json.dumps(metadata | {'environment': environment})}]
            with self.assertRaisesRegex(ReleaseError, 'Preview/other-source'):
                verify_production_number(remote, manifest)

    def test_operator_commands_are_dry_run_without_remote_calls(self):
        remote = Mock()
        for argv in [['prerelease', '--source', SHA, '--issue', '1240'],
                     ['control', '--action', 'cancel', '--cycle', '42', '--note', 'Blocked'],
                     ['control', '--action', 'complete', '--release-run', '500', '--note', 'Retry completion']]:
            result = execute(parser().parse_args(argv), remote)
            self.assertTrue(result['dryRun'])
            self.assertEqual(result['ref'], 'main')
        remote.assert_not_called()
        self.assertEqual(remote.mock_calls, [])

    def test_completion_callback_can_run_as_bot_without_taking_release_ownership(self):
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['web'])
        with tempfile.NamedTemporaryFile() as summary, patch.dict(os.environ, {
                'ACTION': 'complete', 'RELEASE_RUN': '500', 'GITHUB_ACTOR': 'github-actions[bot]',
                'GITHUB_STEP_SUMMARY': summary.name}), patch('sys.argv', ['cycle_control.py', 'change']), \
             patch('cycle_control.load', return_value=state), patch('cycle_control.human') as human_check, \
             patch('cycle_completion.complete_release', return_value=(state, 'closed')) as complete, \
             patch('cycle_control.time.time', return_value=120), patch('cycle_control.save') as save:
            control()
            complete.assert_called_once()
            self.assertEqual(complete.call_args.args[2:], ('500', 120))
            human_check.assert_not_called()
            self.assertEqual(save.call_args.args[1]['cycles']['42']['owner'], 'alice')

    def test_bot_or_reader_cannot_own_a_release(self):
        remote = Mock()
        for permission, kind in [('write', 'Bot'), ('read', 'User')]:
            remote.api.return_value = {'permission': permission, 'user': {'type': kind}}
            with self.assertRaises(ReleaseError): human(remote, 'actor')

    def test_candidate_edits_cannot_replace_the_main_controller_or_human_review(self):
        remote = Mock(repository=REPOSITORY)
        remote.pages.side_effect = [[{'number': 12, 'base': {'ref': 'dev'}, 'head': {
            'sha': SHA, 'repo': {'full_name': REPOSITORY}}, 'user': {'login': 'author'}}],
            [{'user': {'login': 'reviewer'}, 'state': 'APPROVED', 'commit_id': 'b'*40}]]
        remote.maintainers.return_value = {'reviewer'}
        with self.assertRaises(ReleaseError): reviewed_preview_source(remote, SHA)

    def test_main_advancing_does_not_change_the_approved_source(self):
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['web'])
        state['cycles']['42']['qa'] = state['cycles']['42']['productionQa'] = {'source': SHA}
        remote = Mock()
        with patch('cycle_control.load', return_value=state), patch('cycle_control.time.time', return_value=110), \
             patch('cycle_control.fetch_source') as fetch, patch('cycle_control.includes_hotfixes') as hotfix, \
             patch('cycle_control.reviewed_main_pr') as reviewed:
            production_gate(remote, SHA, ['web'])
            fetch.assert_called_once_with(SHA)
            reviewed.assert_called_once_with(remote, SHA)
            hotfix.assert_called_once_with(state, SHA)
            with self.assertRaises(ReleaseError): production_gate(remote, 'c'*40, ['web'])

    def test_hotfix_revision_stays_on_actual_channel_base_instead_of_dev(self):
        state = transition(empty_state(), 'hotfix', 'alice', 100, cycle='42', source=SHA,
                           base='b'*40, note='Outage', issue=1240, channels=['android'])
        state['requiredHotfixes'] = ['d'*40]  # Another channel can have newer changes.
        remote = Mock()
        remote.api.return_value = {'object': {'sha': 'e'*40}}
        remote.pages.return_value = []
        with tempfile.NamedTemporaryFile() as summary, patch.dict(os.environ, {
                'CYCLE_ID': '42', 'GITHUB_RUN_ID': '43', 'GITHUB_ACTOR': 'alice',
                'ACTION': 'advance', 'SOURCE_SHA': 'c'*40, 'NOTE': 'Reviewed blocker fix',
                'GITHUB_STEP_SUMMARY': summary.name}), \
             patch('sys.argv', ['cycle_control.py', 'change']), \
             patch('cycle_control.GitHub', return_value=remote), patch('cycle_control.human'), \
             patch('cycle_control.load', return_value=state), patch('cycle_control.fetch_source'), \
             patch('cycle_control.includes_hotfixes') as required, patch('cycle_control.git') as ancestry, \
             patch('cycle_control.time.time', return_value=110), patch('cycle_control.output'), \
             patch('cycle_control.save') as save:
            control()
            ancestry.assert_called_once_with('merge-base', '--is-ancestor', 'b'*40, 'c'*40)
            required.assert_not_called()
            self.assertEqual(save.call_args.args[1]['cycles']['42']['source'], 'c'*40)

    def test_channel_hotfix_publication_does_not_pull_unrelated_dev_hotfixes(self):
        state = transition(empty_state(), 'hotfix', 'alice', 100, cycle='42', source=SHA,
                           base='b'*40, note='Outage', issue=1240, channels=['android'])
        state['cycles']['42']['qa'] = state['cycles']['42']['productionQa'] = {'source': SHA}
        state['requiredHotfixes'] = ['d'*40]
        with patch('cycle_control.load', return_value=state), patch('cycle_control.time.time', return_value=110), \
             patch('cycle_control.fetch_source'), patch('cycle_control.reviewed_main_pr'), \
             patch('cycle_control.includes_hotfixes') as required:
            production_gate(Mock(), SHA, ['android'])
            required.assert_not_called()

    def test_closed_cycle_can_recover_its_request_but_not_release_later_platforms(self):
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['web', 'android'])
        c = state['cycles']['42']
        c.update(phase='closed', frozen=False, qa={'source': SHA}, productionQa={'source': SHA},
                 completion={'candidate': 'web-1.2.3', 'channels': ['web']})
        state['active'] = None
        with patch('cycle_control.load', return_value=state), patch('cycle_control.fetch_source'), \
             patch('cycle_control.includes_hotfixes'), patch('cycle_control.reviewed_main_pr'):
            production_gate(Mock(), SHA, ['web'], manifest={'candidateId': 'web-1.2.3', 'tag': 'v1.2.3', 'product': 'web'})
            with self.assertRaisesRegex(ReleaseError, 'start a new cycle'):
                production_gate(Mock(), SHA, ['android'], manifest=candidate())

    def test_closed_cycle_allows_store_copy_promotion_of_completed_request_only(self):
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['ios-testflight', 'ios-app-store'])
        state['cycles']['42'].update(phase='closed', frozen=False, qa={'source': SHA},
            productionQa={'source': SHA}, completion={'candidate': 'mobile-original',
            'channels': ['ios-testflight', 'ios-app-store']})
        state['active'] = None
        manifest = candidate() | {'candidateId': 'mobile-store-copy', 'operation': 'promote-ios',
                                 'existingRelease': {'candidateId': 'mobile-original', 'buildId': 'apple-build'}}
        with patch('cycle_control.load', return_value=state), patch('cycle_control.fetch_source'), \
             patch('cycle_control.includes_hotfixes'), patch('cycle_control.reviewed_main_pr'), \
             patch('build_identity.verify_production_number'):
            production_gate(Mock(), SHA, ['ios-app-store'], manifest=manifest)
            manifest['existingRelease']['candidateId'] = 'another-release'
            with self.assertRaisesRegex(ReleaseError, 'start a new cycle'):
                production_gate(Mock(), SHA, ['ios-app-store'], manifest=manifest)

    def test_rerunning_only_dev_deploy_cannot_preserve_previous_qa(self):
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['web'])
        c = state['cycles']['42']
        c['builds']['dev'] = '42'
        c['qa'] = {'source': SHA}
        remote = Mock()
        remote.api.return_value = {'object': {'sha': SHA}}
        with patch.dict(os.environ, {'CYCLE_ID': '42', 'SOURCE_SHA': SHA, 'BUILD_RUN': '42'}), \
             patch('sys.argv', ['cycle_control.py', 'check-dev']), \
             patch('cycle_control.GitHub', return_value=remote), patch('cycle_control.load', return_value=state), \
             patch('cycle_control.time.time', return_value=110):
            with self.assertRaisesRegex(ReleaseError, 'QA already recorded'):
                control()

    def test_new_cycle_at_same_sha_can_cancel_without_inheriting_old_publication(self):
        state = transition(empty_state(), 'start', 'alice', 100, cycle='42', source=SHA,
                           issue=1240, channels=['web'])
        remote = Mock()
        remote.pages.return_value = [{'payload': {'details': {'cycleId': '41'}}}]
        with tempfile.NamedTemporaryFile() as summary, patch.dict(os.environ, {
                'CYCLE_ID': '42', 'GITHUB_RUN_ID': '43', 'GITHUB_ACTOR': 'alice',
                'ACTION': 'cancel', 'NOTE': 'Stop this new cycle', 'GITHUB_STEP_SUMMARY': summary.name}), \
             patch('sys.argv', ['cycle_control.py', 'change']), \
             patch('cycle_control.GitHub', return_value=remote), patch('cycle_control.human'), \
             patch('cycle_control.load', return_value=state), patch('cycle_control.output'), \
             patch('cycle_control.time.time', return_value=110), patch('cycle_control.save') as save:
            control()
            self.assertEqual(save.call_args.args[1]['cycles']['42']['phase'], 'cancelled')


if __name__ == '__main__': unittest.main()
