"""Exercise draft publication and remote byte confirmation without GitHub writes."""
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import publish_web as publisher


class WebPublishTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(); self.addCleanup(self.temp.cleanup)
        previous = Path.cwd(); os.chdir(self.temp.name); self.addCleanup(os.chdir, previous)
        self.notes = Path('.release-approved/web.zh-CN.md'); self.notes.parent.mkdir()
        self.notes.write_bytes('修复 Web 编辑器。\n'.encode())
        self.assets = Path('.source/apps/gooseforum/dist'); self.assets.mkdir(parents=True)
        for name in ['linux-amd64.tar.gz', 'linux-arm64.tar.gz', 'darwin-amd64.tar.gz', 'darwin-arm64.tar.gz', 'windows-amd64.zip', 'windows-arm64.zip', 'checksums.txt']:
            (self.assets / name).write_bytes(('original ' + name).encode())
        env = patch.dict(os.environ, RELEASE_TAG='v1.0.0', RELEASE_NOTES_DIGESTS=json.dumps({self.notes.name: hashlib.sha256(self.notes.read_bytes()).hexdigest()}))
        env.start(); self.addCleanup(env.stop)
        self.release = None; self.calls = []; self.corrupt = False; self.interrupt = False
        gh = patch.object(publisher, 'gh', side_effect=self.gh); gh.start(); self.addCleanup(gh.stop)
        find = patch.object(publisher, 'find_release', side_effect=lambda tag: self.release); find.start(); self.addCleanup(find.stop)

    def gh(self, *args):
        self.calls.append(args)
        if args[:2] == ('release', 'create'):
            self.assertIn('--draft', args); self.assertIn('--verify-tag', args)
            self.release = {'draft': True, 'body': self.notes.read_text(encoding='utf-8'), 'assets': []}
        elif args[:2] == ('release', 'upload'):
            if self.interrupt and len(self.release['assets']) == 2:
                self.interrupt = False
                raise RuntimeError('interrupted')
            path = Path(args[3])
            self.release['assets'].append({'name': path.name, 'digest': 'sha256:' + ('0' * 64 if self.corrupt else hashlib.sha256(path.read_bytes()).hexdigest())})
        elif args[:2] == ('release', 'edit'):
            self.assertEqual(len(self.release['assets']), 7)
            self.release['draft'] = False
        else: self.fail(str(args))

    def test_partial_upload_recovers_without_overwriting_and_only_then_flips_draft(self):
        self.interrupt = True
        with self.assertRaisesRegex(RuntimeError, 'interrupted'): publisher.main()
        self.assertTrue(self.release['draft'])
        publisher.main()
        self.assertFalse(self.release['draft'])
        self.assertEqual(len(self.release['assets']), 7)
        uploads = len([c for c in self.calls if c[:2] == ('release', 'upload')])
        publisher.main()
        self.assertEqual(len([c for c in self.calls if c[:2] == ('release', 'upload')]), uploads)

    def test_conflicting_immutable_bytes_are_refused(self):
        publisher.main(); self.calls.clear()
        (self.assets / 'windows-amd64.zip').write_bytes(b'rebuilt bytes')
        with self.assertRaisesRegex(ValueError, 'overwrite'): publisher.main()
        self.assertEqual(self.calls, [])

    def test_unconfirmed_remote_digest_never_publishes_draft(self):
        self.corrupt = True
        with self.assertRaisesRegex(ValueError, 'digest not confirmed'): publisher.main()
        self.assertTrue(self.release['draft'])
        self.assertFalse(any(c[:2] == ('release', 'edit') for c in self.calls))

    def test_incomplete_archive_inventory_never_publishes(self):
        (self.assets / 'checksums.txt').unlink()
        with self.assertRaisesRegex(ValueError, 'six platform archives'): publisher.main()
        self.assertFalse(any(c[:2] == ('release', 'edit') for c in self.calls))

    def test_null_or_different_remote_notes_fail_actionably(self):
        for body in (None, 'Different notes'):
            self.release = {'body': body, 'draft': True, 'assets': []}
            with self.subTest(body=body), self.assertRaisesRegex(ValueError, 'differs from approved'): publisher.main()
        self.assertEqual(self.calls, [])
