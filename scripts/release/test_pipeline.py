"""Behavioral integration checks: real Git evidence, strict remote errors and rendered drafts."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from collect import collect
from controller import prepare, source_sha
from github import GitHub
from model import ReleaseError, digest, validate_candidate
from notes import render
from cli import execute, parser
from test_model import candidate


class FakeGitHub:
    repository = 'YourTongji/YourTJ-Hub'
    def api(self, endpoint, **kwargs):
        raise AssertionError('Unexpected remote side effect or lookup: ' + endpoint)
    def dispatch(self, workflow, inputs):
        return {'workflow': workflow, 'inputs': inputs}


class PipelineTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def test_external_agent_dry_run_does_not_contact_a_remote(self):
        args = parser().parse_args(['prepare', '--scope', 'mobile', '--ios-destination', 'testflight', '--json'])
        value = execute(args, FakeGitHub())
        self.assertTrue(value['dryRun'])
        self.assertEqual(value['inputs']['ios_destination'], 'testflight')
        args = parser().parse_args(['prepare', '--scope', 'android', '--ios-destination', 'app-store', '--apply'])
        with self.assertRaises(ReleaseError):
            execute(args, FakeGitHub())

    def test_agent_can_inspect_remote_candidate_without_changing_checkout(self):
        manifest = candidate()
        class Remote(FakeGitHub):
            def pages(self, endpoint):
                return [] if endpoint.startswith('deployments?') else [{'number': 12}]
            def api(self, endpoint, **kwargs):
                return {'number': 12, 'head': {'sha': 'a' * 40, 'repo': {'full_name': self.repository}}}
        def materialize(candidate_id, sha, folder, draft):
            folder = Path(folder)
            (folder / 'manifest.json').write_text(json.dumps(manifest), encoding='utf-8')
            for name in manifest['notes'].values(): (folder / name).write_text('[DRAFT: human review required]', encoding='utf-8')
        args = parser().parse_args(['status', '--candidate', manifest['candidateId'], '--json'])
        with patch('cli.git') as git, patch('cli.load_candidate', side_effect=materialize):
            result = execute(args, Remote())
        git.assert_called_once_with('fetch', 'origin', 'refs/pull/12/head')
        self.assertEqual(result['sourceSha'], manifest['sourceSha'])
        self.assertEqual(result['channels'], {'android': None, 'ios-testflight': None})

    def test_prepare_preserves_explicit_disclosures_without_guessing_from_keywords(self):
        manifest = candidate()
        disclosure = {'id': 'privacy', 'channels': ['android'], 'text': 'Explicit reviewed disclosure'}
        manifest['requiredDisclosures'] = [disclosure]
        request = {'evidence': [{'paths': ['apps/mobile/packages/forum_app/lib/src/analytics/visitor_analytics.dart'],
                                 'channels': ['ios-testflight'], 'detail': '- enabled\n+ disabled'}]}
        with patch('controller.collect', return_value=(request, {'schemaVersion': 1})):
            result = prepare(manifest, FakeGitHub(), self.root / 'prepared')
        self.assertEqual(result['requiredDisclosures'], [disclosure])
        self.assertEqual(manifest['requiredDisclosures'], [disclosure])

    def test_model_output_cannot_change_a_channel_or_overwrite_a_human_edit(self):
        manifest = candidate()
        request = {'evidence': [{'id': 'android-fix', 'channels': ['android']}, {'id': 'ios-fix', 'channels': ['ios-testflight']}]}
        raw = json.dumps(request).encode()
        (self.root / 'evidence.json').write_text(json.dumps({'schemaVersion': 1, 'sourceSha': manifest['sourceSha']}), encoding='utf-8')
        response = {'schemaVersion': 1, 'promptVersion': 1, 'model': 'oryn/test', 'inputSha256': hashlib.sha256(raw).hexdigest(),
                    'output': {'schemaVersion': 1, 'entries': [{'channel': 'android', 'text': '修复 Android 相机返回。', 'evidenceIds': ['android-fix']},
                                {'channel': 'ios-testflight', 'text': 'Test iOS widgets.', 'evidenceIds': ['ios-fix']}], 'uncertainties': []}}
        bad = copy.deepcopy(response)
        bad['output']['entries'][0]['evidenceIds'] = ['ios-fix']
        with self.assertRaises(ReleaseError): render(manifest, self.root, bad, raw)
        self.assertFalse((self.root / 'android.zh-CN.md').exists())
        render(manifest, self.root, response, raw)
        validate_candidate(manifest, self.root)
        note = self.root / 'android.zh-CN.md'
        note.write_text('人工修改后的 Android 说明。', encoding='utf-8')
        with self.assertRaises(ReleaseError): render(manifest, self.root, response, raw)
        self.assertEqual(note.read_text(encoding='utf-8'), '人工修改后的 Android 说明。')
        response['inputSha256'] = '0' * 64
        with self.assertRaises(ReleaseError): render(manifest, self.root, response, raw, replace=True)

    def test_schema_two_oryn_rerender_preserves_all_original_bytes_without_replace(self):
        manifest = candidate() | {"schemaVersion": 2}
        evidence_items = [{"id": "android-fix", "channels": ["android"]},
                          {"id": "ios-fix", "channels": ["ios-testflight"]}]
        request = {"evidence": evidence_items}
        evidence = {"schemaVersion": 1, "sourceSha": manifest["sourceSha"], "evidence": evidence_items}
        folder = self.root / "prepared"
        with patch("controller.collect", return_value=(request, evidence)):
            prepared_input = prepare(manifest, FakeGitHub(), folder)
        raw = json.dumps(prepared_input).encode()
        response = {"schemaVersion": 1, "promptVersion": 1, "model": "oryn/test",
                    "inputSha256": hashlib.sha256(raw).hexdigest(),
                    "output": {"schemaVersion": 1, "entries": [
                        {"channel": "android", "text": "修复 Android 相机返回。", "evidenceIds": ["android-fix"]},
                        {"channel": "ios-testflight", "text": "Test iOS widgets.", "evidenceIds": ["ios-fix"]}],
                        "uncertainties": []}}
        render(manifest, folder, response, raw)
        protected = {path.name: path.read_bytes() for path in folder.iterdir()}
        (folder / "android.zh-CN.md").write_text("Human reviewed Android copy.\n", encoding="utf-8")
        before_retry = {path.name: path.read_bytes() for path in folder.iterdir()}
        with self.assertRaises(ReleaseError):
            render(manifest, folder, response, raw)
        self.assertEqual({path.name: path.read_bytes() for path in folder.iterdir()}, before_retry)
        self.assertIn("changelog.json", protected)
        render(manifest, folder, response, raw, replace=True)
        self.assertEqual((folder / "android.zh-CN.md").read_text(encoding="utf-8").strip(),
                         "### 本次更新\n\n- **修复 Android 相机返回**：修复 Android 相机返回。")

    def test_net_diff_accounts_for_reverts_renames_and_platform_baselines(self):
        def git(*args):
            return subprocess.check_output(['git', '-C', str(self.root), *args], text=True).strip()
        git('init', '-q')
        env = {**os.environ, 'GIT_AUTHOR_NAME': 'Fixture', 'GIT_AUTHOR_EMAIL': 'fixture@example.org',
               'GIT_COMMITTER_NAME': 'Fixture', 'GIT_COMMITTER_EMAIL': 'fixture@example.org'}
        def commit(message):
            git('add', '.')
            subprocess.run(['git', '-C', str(self.root), 'commit', '-qm', message], env=env, check=True)
            return git('rev-parse', 'HEAD')
        path = self.root / 'apps/mobile/packages/forum_app/ios/Widget.swift'
        path.parent.mkdir(parents=True)
        path.write_text('old widget\n', encoding='utf-8')
        base = commit('initial')
        shared = self.root / 'apps/mobile/packages/forum_app/lib/shared.dart'
        shared.parent.mkdir(); shared.write_text('temporary feature\n', encoding='utf-8'); commit('temporary')
        shared.unlink(); commit('revert temporary')
        path.write_text('new widget\n', encoding='utf-8'); source = commit('widget behavior')
        manifest = candidate() | {'sourceSha': source, 'baselines': {'android': {'tag': 'mobile-v1.0.14', 'sourceSha': base},
                                                                           'ios-testflight': {'tag': 'mobile-v1.0.13', 'sourceSha': base}}}
        with patch('collect.git', side_effect=git):
            request, evidence = collect(manifest, FakeGitHub())
        self.assertEqual(len(request['evidence']), 1)
        self.assertEqual(request['evidence'][0]['channels'], ['ios-testflight'])
        self.assertNotIn('shared.dart', json.dumps(evidence['inventories']))
        # Same source can have different real channel baselines.
        manifest['baselines']['ios-testflight']['sourceSha'] = source
        with patch('collect.git', side_effect=git):
            request, _ = collect(manifest, FakeGitHub())
        self.assertEqual(request['evidence'], [])

    def test_remote_errors_never_become_missing_or_success(self):
        for code in (401, 403, 500):
            result = subprocess.CompletedProcess([], 1, '', f'failed (HTTP {code})')
            with patch('github.subprocess.run', return_value=result), self.assertRaises(ReleaseError):
                GitHub().api('releases/tags/v1.0.0', missing=True)
        with patch('github.subprocess.run', return_value=subprocess.CompletedProcess([], 1, '', 'failed (HTTP 404)')):
            self.assertIsNone(GitHub().api('releases/tags/v1.0.0', missing=True))

    def test_all_pages_and_both_rename_paths_are_inspected(self):
        github = GitHub()
        first = [{'filename': f'file-{i}', 'status': 'modified'} for i in range(100)]
        second = [{'filename': 'ios/new.swift', 'previous_filename': 'android/old.kt'}]
        with patch.object(github, 'api', side_effect=[first, second]) as api:
            paths = github.files(1)
        self.assertEqual(len(paths), 102)
        self.assertIn('android/old.kt', paths)
        self.assertIn('page=2', api.call_args_list[-1].args[0])


if __name__ == '__main__': unittest.main()
