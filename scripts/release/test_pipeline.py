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
from controller import notes_banner, prepare, source_sha
from github import GitHub
from model import FILES, ReleaseError, digest, validate_candidate
from notes import draft_notes, render
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
                         "### 体验改进\n\n- **修复 Android 相机返回。**")

    def test_feature_prs_on_the_merged_side_are_linked_to_their_evidence(self):
        def git(*args):
            return subprocess.check_output(['git', '-C', str(self.root), *args], text=True).strip()
        env = {**os.environ, 'GIT_AUTHOR_NAME': 'Fixture', 'GIT_AUTHOR_EMAIL': 'fixture@example.org',
               'GIT_COMMITTER_NAME': 'Fixture', 'GIT_COMMITTER_EMAIL': 'fixture@example.org'}
        def run(*args):
            subprocess.run(['git', '-C', str(self.root), *args], env=env, check=True, capture_output=True)
        git('init', '-q', '-b', 'main')
        app = self.root / 'apps/mobile/packages/forum_app/lib'
        app.mkdir(parents=True)
        (app / 'feed.dart').write_text('feed\n', encoding='utf-8'); run('add', '.'); run('commit', '-qm', 'initial')
        base = git('rev-parse', 'HEAD')
        run('checkout', '-qb', 'dev')
        run('checkout', '-qb', 'feat/stickers')
        (app / 'stickers.dart').write_text('stickers\n', encoding='utf-8'); run('add', '.'); run('commit', '-qm', 'feat: stickers')
        run('checkout', '-q', 'dev')
        run('merge', '-q', '--no-ff', 'feat/stickers', '-m', 'Merge pull request #7 from Org/feat/stickers')
        (app / 'feed.dart').write_text('feed fixed\n', encoding='utf-8'); run('add', '.')
        run('commit', '-qm', 'fix: feed scrolling (#9)')
        run('checkout', '-q', 'main')
        run('merge', '-q', '--no-ff', 'dev', '-m', 'Merge pull request #8 from Org/dev')
        source = git('rev-parse', 'HEAD')
        manifest = candidate() | {'sourceSha': source, 'channels': ['android'], 'notes': {'android': 'android.zh-CN.md'},
                                  'baselines': {'android': {'tag': 'mobile-v1.0.14', 'sourceSha': base}},
                                  'requiredDisclosures': [{'id': 'Analytics', 'channels': ['android'], 'text': '访问统计默认开启。'}]}
        class Titles(FakeGitHub):
            def api(self, endpoint, **kwargs):
                number = int(endpoint.rsplit('/', 1)[1])
                return {'title': {7: 'feat: stickers', 9: 'fix: feed scrolling'}[number], 'html_url': f'https://example.org/{number}'}
        with patch('collect.git', side_effect=git):
            request, evidence = collect(manifest, Titles())
        by_path = {item['paths'][0].rsplit('/', 1)[1]: item['id'] for item in request['evidence']}
        self.assertEqual(evidence['evidencePullRequests'], {by_path['stickers.dart']: [7], by_path['feed.dart']: [9]})
        self.assertEqual({pr['number']: pr['branch'] for pr in evidence['pullRequests']}, {7: 'Org/feat/stickers', 9: None})
        # The model request is unchanged: no PR links and no disclosure evidence.
        self.assertNotIn('evidencePullRequests', request)
        self.assertEqual(len(request['evidence']), 2)
        self.assertEqual([item.get('disclosureId') for item in evidence['evidence']][-1], 'Analytics')

    def test_oryn_drafts_become_grouped_stable_user_facing_entries(self):
        manifest = candidate() | {'schemaVersion': 2, 'channels': ['android'], 'notes': {'android': 'android.zh-CN.md'},
                                  'baselines': {'android': {'tag': 'mobile-v1.0.14', 'sourceSha': 'b' * 40}},
                                  'requiredDisclosures': [{'id': 'Visit analytics', 'channels': ['android'],
                                                           'text': '访问统计默认开启，可在关于页面查看说明。'}]}
        items = [{'id': f'e{i}', 'channels': ['android']} for i in range(5)]
        request = {'evidence': items}
        record = {'schemaVersion': 1, 'sourceSha': manifest['sourceSha'],
                  'evidence': items + [{'id': 'disclosure-x', 'channels': ['android'], 'disclosureId': 'Visit analytics'}],
                  'pullRequests': [{'number': 7, 'title': 'feat: stickers', 'branch': 'Org/feat/stickers'},
                                   {'number': 9, 'title': 'Feed scrolling', 'branch': 'Org/fix/feed'},
                                   {'number': 11, 'title': 'feat!: new sign-in', 'branch': 'Org/feat/sign-in'},
                                   {'number': 12, 'title': 'chore: patch image decoder vulnerability', 'branch': None}],
                  'evidencePullRequests': {'e0': [7], 'e1': [9], 'e2': [11], 'e3': [12]}}
        folder = self.root / 'structured'
        folder.mkdir()
        (folder / 'evidence.json').write_text(json.dumps(record), encoding='utf-8')
        raw = json.dumps(request).encode()
        entries = [
            {'channel': 'android', 'text': '私信贴纸：在聊天里发送和收藏贴纸。', 'evidenceIds': ['e0']},
            {'channel': 'android', 'text': '修复信息流滚动时偶尔跳回顶部的问题。', 'evidenceIds': ['e1']},
            {'channel': 'android', 'text': '需要重新登录。登录方式升级，旧的登录状态会失效。', 'evidenceIds': ['e2']},
            {'channel': 'android', 'text': '图片解码更安全。', 'evidenceIds': ['e3']},
            {'channel': 'android', 'text': '列表更紧凑。', 'evidenceIds': ['e4'], 'kind': 'improvement',
             'title': '更紧凑的列表', 'summary': '一屏能看到更多帖子。'},
        ]
        response = {'schemaVersion': 1, 'promptVersion': 1, 'model': 'oryn/test',
                    'inputSha256': hashlib.sha256(raw).hexdigest(),
                    'output': {'schemaVersion': 1, 'entries': entries, 'uncertainties': []}}
        render(manifest, folder, response, raw)
        changelog = json.loads((folder / 'changelog.json').read_text(encoding='utf-8'))
        rows = {e['id']: e for group in ('highlights', 'breaking', 'requiredActions') for e in changelog[group]}
        self.assertEqual(rows['pr-7'], {'id': 'pr-7', 'title': '私信贴纸', 'summary': '在聊天里发送和收藏贴纸。',
                                            'platforms': ['android'], 'kind': 'feature'})
        self.assertEqual(rows['pr-9']['kind'], 'fix')
        self.assertEqual([e['id'] for e in changelog['breaking']], ['pr-11'])
        self.assertEqual(rows['pr-12']['kind'], 'security')
        self.assertEqual((rows['pr-11']['title'], rows['pr-11']['summary']), ('需要重新登录', '登录方式升级，旧的登录状态会失效。'))
        self.assertEqual(rows[next(k for k in rows if k.startswith('oryn-'))]['title'], '更紧凑的列表')
        self.assertEqual([e['id'] for e in changelog['requiredActions']], ['visit-analytics'])
        note = (folder / 'android.zh-CN.md').read_text(encoding='utf-8')
        self.assertEqual([line for line in note.splitlines() if line.startswith('###')],
                         ['### 重要提示', '### 安全更新', '### 新功能', '### 体验改进', '### 问题修复'])
        self.assertIn('- **图片解码更安全。**', note)
        # Publishing validation: verbatim disclosure, structured facts and rendered text agree.
        validate_candidate(manifest, folder)

    def draft_fixture(self, channels):
        manifest = candidate() | {'schemaVersion': 2, 'channels': channels, 'notes': {c: FILES[c] for c in channels},
                                  'baselines': {c: {'tag': 'mobile-v1.0.14', 'sourceSha': 'b' * 40} for c in channels}}
        folder = self.root / 'approved' / 'candidate'
        folder.mkdir(parents=True)
        items = [{'id': f'e-{c}', 'channels': [c]} for c in channels]
        (folder / 'evidence.json').write_text(json.dumps({'schemaVersion': 1, 'sourceSha': manifest['sourceSha'],
                                                          'evidence': items}), encoding='utf-8')
        (folder / 'changelog.json').write_text(json.dumps({
            'schemaVersion': 1, 'version': manifest['version'], 'buildNumber': manifest['buildNumber'],
            'highlights': [], 'breaking': [], 'requiredActions': [], 'evidence': {}, 'testflightNotes': []}), encoding='utf-8')
        for name in manifest['notes'].values():
            (folder / name).write_text('[DRAFT: human review required — complete from evidence.json]\n', encoding='utf-8')
        return manifest, folder, json.dumps({'evidence': items}).encode()

    @staticmethod
    def oryn(raw, entries):
        return {'schemaVersion': 1, 'promptVersion': 1, 'model': 'oryn/test', 'inputSha256': hashlib.sha256(raw).hexdigest(),
                'output': {'schemaVersion': 1, 'entries': entries, 'uncertainties': []}}

    def run_draft(self, manifest, folder, raw, outcomes, **options):
        output = self.root / 'approved' / 'oryn-output.json'
        pending = list(outcomes)
        def run_oryn(timeout):
            outcome = pending.pop(0)
            if isinstance(outcome, Exception):
                raise outcome
            output.write_text(outcome if isinstance(outcome, str) else json.dumps(outcome), encoding='utf-8')
        waits = []
        status = draft_notes(manifest, folder, raw, output, run_oryn, sleep=waits.append, **options)
        return status, waits, pending

    def test_oryn_failures_and_malformed_drafts_are_retried_until_every_channel_has_notes(self):
        manifest, folder, raw = self.draft_fixture(['android'])
        good = [{'channel': 'android', 'text': '私信贴纸：在聊天里发送和收藏贴纸。', 'evidenceIds': ['e-android']}]
        status, waits, _ = self.run_draft(manifest, folder, raw, [
            subprocess.CalledProcessError(1, ['bun', 'script/release-notes.ts']),
            '{"truncated": ',
            self.oryn(raw, [good[0] | {'channel': 'ios-app-store'}]),
            self.oryn(raw, good)], attempts=4)
        self.assertEqual(status['status'], 'complete')
        self.assertEqual(len(status['attempts']), 4)
        self.assertTrue(all(a['error'] for a in status['attempts'][:3]))
        self.assertEqual(len(waits), 3)
        self.assertIn('- **私信贴纸**：在聊天里发送和收藏贴纸。', (folder / 'android.zh-CN.md').read_text(encoding='utf-8'))
        validate_candidate(manifest, folder)

    def test_exhausted_oryn_retries_leave_an_unpublishable_human_draft(self):
        manifest, folder, raw = self.draft_fixture(['android'])
        before = {p.name: p.read_bytes() for p in folder.iterdir()}
        status, waits, _ = self.run_draft(manifest, folder, raw, [subprocess.TimeoutExpired(['bun'], 60)] * 3)
        self.assertEqual((status['status'], status['missing'], len(status['attempts']), len(waits)),
                         ('failed', ['android'], 3, 2))
        self.assertEqual({p.name: p.read_bytes() for p in folder.iterdir()}, before)
        with self.assertRaises(ReleaseError):
            validate_candidate(manifest, folder)
        # A spent time budget stops before another provider call is started.
        clock = iter([0, 0, 2000, 2000, 2000])
        status, _, pending = self.run_draft(manifest, folder, raw, [OSError('provider down')] * 3,
                                            budget=2030, clock=lambda: next(clock))
        self.assertEqual((status['status'], len(pending)), ('failed', 2))

    def test_a_draft_missing_a_channel_is_retried_then_kept_for_human_completion(self):
        manifest, folder, raw = self.draft_fixture(['android', 'ios-testflight'])
        partial = self.oryn(raw, [{'channel': 'android', 'text': '列表更紧凑。', 'evidenceIds': ['e-android']}])
        status, _, _ = self.run_draft(manifest, folder, raw, [partial] * 3)
        self.assertEqual((status['status'], status['missing'], len(status['attempts'])),
                         ('partial', ['ios-testflight'], 3))
        self.assertIn('列表更紧凑', (folder / 'android.zh-CN.md').read_text(encoding='utf-8'))
        self.assertIn('[DRAFT:', (folder / 'testflight.en-US.txt').read_text(encoding='utf-8'))
        with self.assertRaises(ReleaseError):
            validate_candidate(manifest, folder)

    def test_release_request_names_incomplete_oryn_notes(self):
        manifest = candidate()
        self.assertIn('Oryn was unavailable. Missing: android, ios-testflight.', notes_banner(manifest, None))
        failed = {'status': 'failed', 'missing': ['android'], 'stopped': None,
                  'attempts': [{'attempt': 1, 'error': 'TimeoutExpired: `bun` timed out'}]}
        self.assertIn("no usable draft after 1 attempt(s) (last error: TimeoutExpired: 'bun' timed out). Missing: android.",
                      notes_banner(manifest, failed))
        complete = {'status': 'complete', 'missing': [], 'stopped': None, 'attempts': [{'attempt': 1, 'error': None}]}
        self.assertNotIn('[!WARNING]', notes_banner(manifest, complete))

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
