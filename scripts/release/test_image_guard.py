import subprocess
import unittest
from unittest.mock import patch
from image_guard import require_absent
from model import ReleaseError


class ImageGuardTests(unittest.TestCase):
    reference = 'ghcr.io/yourtongji/yourtj-hub:main-v0.0.50'

    def test_existing_tag_cannot_be_rebuilt_after_receipt_loss(self):
        with patch('image_guard.subprocess.run', return_value=subprocess.CompletedProcess([], 0, '{"digest":"original"}', '')):
            with self.assertRaisesRegex(ReleaseError, 'Refusing to overwrite'):
                require_absent(self.reference)

    def test_only_an_explicit_absent_manifest_allows_the_first_push(self):
        for absence in ('no such manifest: ' + self.reference + '\n', 'manifest unknown\n'):
            with patch('image_guard.subprocess.run', return_value=subprocess.CompletedProcess([], 1, '', absence)):
                require_absent(self.reference)
        for message in ('unauthorized', 'Too Many Requests', 'timeout', 'registry unavailable'):
            with patch('image_guard.subprocess.run', return_value=subprocess.CompletedProcess([], 1, '', message)), self.assertRaises(ReleaseError):
                require_absent(self.reference)
