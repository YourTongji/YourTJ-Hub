"""Store copy stays readable without changing already approved candidate bytes."""
import copy
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from controller import prepare
from model import FILES, ReleaseError, render_changelog, validate_candidate, validate_changelog
from notes import render, render_structured
from test_model import candidate


def changelog():
    def item(identity, kind, platforms, title, summary):
        return dict(id=identity, kind=kind, platforms=platforms, title=title, summary=summary)
    return dict(schemaVersion=1, notesFormatVersion=2, version='1.0.15', buildNumber=15,
                highlights=[item('android', 'feature', ['android'], '通知权限', 'Android 权限提示。'),
                            item('widget', 'feature', ['ios-app-store'], '课表小组件', '隐藏已结束课程。'),
                            item('images', 'fix', ['ios-app-store'], '图片上传', '修复图片上传。'),
                            item('login', 'fix', ['ios-app-store'], '修复授权过期', '修复授权过期。')],
                breaking=[], requiredActions=[item('privacy', 'improvement', ['ios-app-store'],
                                                   '统计说明', '访问统计自动开启。')],
                testflightNotes=[dict(id='beta', text='Test the iOS widget.', evidenceIds=['proof'])],
                evidence={key: ['proof'] for key in ('android', 'widget', 'images', 'login', 'privacy', 'beta')})


class NoteFormatTests(unittest.TestCase):
    def test_store_has_plain_sections_bullets_and_required_copy_first(self):
        self.assertEqual(render_changelog(changelog(), 'ios-app-store'),
                         '重要提示\n\n• 统计说明：访问统计自动开启。\n\n'
                         '新功能\n\n• 课表小组件：隐藏已结束课程。\n\n'
                         '问题修复\n\n• 图片上传：修复图片上传。\n• 修复授权过期。')

    def test_legacy_bytes_android_and_testflight_are_stable(self):
        current = changelog()
        legacy = copy.deepcopy(current)
        legacy.pop('notesFormatVersion')
        self.assertEqual(render_changelog(legacy, 'ios-app-store'),
                         '统计说明：访问统计自动开启。\n课表小组件：隐藏已结束课程。\n'
                         '图片上传：修复图片上传。\n修复授权过期。')
        for channel in ('android', 'ios-testflight'):
            self.assertEqual(render_changelog(current, channel), render_changelog(legacy, channel))
        self.assertEqual(render_changelog(current, 'android'), '### 新功能\n\n- **通知权限**：Android 权限提示。')
        self.assertEqual(render_changelog(current, 'ios-testflight'), 'Test the iOS widget.')

    def test_unknown_format_cannot_validate_or_render(self):
        for version in (0, 3, '2', True, None):
            value = changelog() | {'notesFormatVersion': version}
            with self.subTest(version=version):
                with self.assertRaisesRegex(ReleaseError, 'notes format'):
                    validate_changelog(value, '1.0.15', 15, ['android', 'ios-app-store', 'ios-testflight'])
                with self.assertRaisesRegex(ReleaseError, 'notes format'):
                    render_changelog(value, 'ios-app-store')

    def test_prepare_and_explicit_render_use_current_format_with_strict_checks(self):
        manifest = candidate() | dict(schemaVersion=2,
            channels=['android', 'ios-app-store', 'ios-testflight'],
            notes={c: FILES[c] for c in ('android', 'ios-app-store', 'ios-testflight')},
            baselines={c: dict(tag=None, sourceSha=None) for c in ('android', 'ios-app-store', 'ios-testflight')},
            requiredDisclosures=[dict(id='privacy', channels=['ios-app-store'], text='访问统计自动开启。')])
        evidence = dict(schemaVersion=1, sourceSha=manifest['sourceSha'],
                        evidence=[dict(id='proof', channels=manifest['channels'])])
        with tempfile.TemporaryDirectory() as temporary:
            folder = Path(temporary) / 'candidate'
            with patch('controller.collect', return_value=({}, evidence)):
                prepare(manifest, None, folder)
            self.assertEqual(json.loads((folder / 'changelog.json').read_text())['notesFormatVersion'], 2)
            value = changelog()
            (folder / 'changelog.json').write_text(json.dumps(value))
            render_structured(manifest, folder)
            original = validate_candidate(manifest, folder)
            before = {p.name: p.read_bytes() for p in folder.iterdir()}
            with self.assertRaisesRegex(ReleaseError, 'Preserving human edits'):
                render_structured(manifest, folder)
            self.assertEqual(before, {p.name: p.read_bytes() for p in folder.iterdir()})
            render_structured(manifest, folder, replace=True)
            self.assertEqual(original, validate_candidate(manifest, folder))
            (folder / FILES['ios-app-store']).write_text('访问统计自动开启。\nOther copy')
            with self.assertRaisesRegex(ReleaseError, 'differ from'):
                validate_candidate(manifest, folder)
            # Format overhead counts towards Apple's existing 4000-character gate.
            value['highlights'] += [dict(id=f'long-{i}', kind='feature', platforms=['ios-app-store'],
                                        title='Long', summary='x' * 390) for i in range(10)]
            value['evidence'].update({f'long-{i}': ['proof'] for i in range(10)})
            (folder / 'changelog.json').write_text(json.dumps(value))
            with self.assertRaisesRegex(ReleaseError, 'oversized note'):
                render_structured(manifest, folder, replace=True)

    def test_oryn_draft_respects_host_format_and_existing_candidate_version(self):
        manifest = candidate() | dict(schemaVersion=2, channels=['ios-app-store'],
            notes={'ios-app-store': FILES['ios-app-store']},
            baselines={'ios-app-store': dict(tag=None, sourceSha=None)})
        request = dict(evidence=[dict(id='proof', channels=['ios-app-store'])])
        evidence = dict(schemaVersion=1, sourceSha=manifest['sourceSha'], **request)
        raw = json.dumps(request).encode()
        response = dict(schemaVersion=1, promptVersion=1, model='oryn/test',
            inputSha256=hashlib.sha256(raw).hexdigest(),
            output=dict(schemaVersion=1, uncertainties=[], entries=[dict(channel='ios-app-store',
                text='课表小组件：隐藏已结束课程。', evidenceIds=['proof'], kind='feature',
                title='课表小组件', summary='隐藏已结束课程。')]))
        for version in (None, 1, 2):
            with self.subTest(version=version), tempfile.TemporaryDirectory() as temporary:
                folder = Path(temporary) / 'candidate'
                with patch('controller.collect', return_value=(request, evidence)):
                    prepare(manifest, None, folder)
                path = folder / 'changelog.json'
                scaffold = json.loads(path.read_text())
                if version is None:
                    scaffold.pop('notesFormatVersion')
                else:
                    scaffold['notesFormatVersion'] = version
                path.write_text(json.dumps(scaffold))
                render(manifest, folder, response, raw)
                validate_candidate(manifest, folder)
                rendered = (folder / FILES['ios-app-store']).read_text()
                line = '课表小组件：隐藏已结束课程。\n'
                self.assertEqual(rendered, '新功能\n\n• ' + line if version == 2 else line)
                self.assertEqual(json.loads(path.read_text()).get('notesFormatVersion', 1), version or 1)

    def test_empty_channel_remains_empty_and_does_not_gain_filler_headings(self):
        value = changelog() | dict(highlights=[], requiredActions=[])
        self.assertEqual(render_changelog(value, 'ios-app-store'), '')


if __name__ == '__main__':
    unittest.main()
