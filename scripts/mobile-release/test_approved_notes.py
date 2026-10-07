import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from approved_notes import read_notes


class ApprovedNotesTests(unittest.TestCase):
    def test_bytes_must_match_approval_even_when_shape_is_valid(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / 'android.zh-CN.md'
            path.write_bytes('Reviewed Android 改进。\n'.encode())
            hashes = {'android.zh-CN.md': hashlib.sha256(path.read_bytes()).hexdigest()}
            with patch.dict(os.environ, ANDROID_NOTES_PATH=str(path), RELEASE_NOTES_DIGESTS=json.dumps(hashes)):
                self.assertEqual(read_notes('ANDROID_NOTES_PATH'), 'Reviewed Android 改进。')
                path.write_bytes(b'Other valid but unreviewed notes\n')
                with self.assertRaisesRegex(ValueError, 'digest'):
                    read_notes('ANDROID_NOTES_PATH')
            with patch.dict(os.environ, ANDROID_NOTES_PATH=str(path), RELEASE_NOTES_DIGESTS='{}'):
                with self.assertRaisesRegex(ValueError, 'digest'):
                    read_notes('ANDROID_NOTES_PATH')

    def test_drafts_limits_plain_text_and_symlinks_fail_even_with_matching_digest(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / 'ios.zh-Hans.txt'
            for text in ('', '[DRAFT: incomplete]', 'x' * 4001, '# Heading', 'nul\x00text'):
                path.write_bytes(text.encode())
                with self.subTest(text=text[:25]), patch.dict(os.environ, IOS_STORE_NOTES_PATH=str(path), RELEASE_NOTES_DIGESTS=json.dumps({path.name: hashlib.sha256(path.read_bytes()).hexdigest()})):
                    with self.assertRaises(ValueError): read_notes('IOS_STORE_NOTES_PATH')
            target = Path(temporary) / 'target'; target.write_bytes(b'valid')
            path.unlink()
            try:
                path.symlink_to(target)
            except OSError:
                self.skipTest('Symlink creation is unavailable on this host')
            with patch.dict(os.environ, IOS_STORE_NOTES_PATH=str(path), RELEASE_NOTES_DIGESTS=json.dumps({path.name: hashlib.sha256(target.read_bytes()).hexdigest()})):
                with self.assertRaises(ValueError): read_notes('IOS_STORE_NOTES_PATH')
