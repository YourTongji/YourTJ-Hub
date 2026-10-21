"""Failed Xcode builds must retain diagnostics independently of successful IPAs."""
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]


class IOSDiagnosticsTest(unittest.TestCase):
    def test_archive_failure_has_a_log_only_upload_that_runs_after_failure(self):
        for workflow in ['release-preview-ios.yml', 'release-ios.yml']:
            with self.subTest(workflow=workflow):
                text = (ROOT / '.github/workflows' / workflow).read_text()
                steps = re.split(r'(?m)^      - ', text)
                uploads = [step for step in steps if 'uses: actions/upload-artifact@' in step
                           and 'if: always()' in step and '/release/*.log' in step]
                self.assertEqual(len(uploads), 1, 'Archive failure currently discards its only error log')
                self.assertIn('include-hidden-files: true', uploads[0])
                self.assertIn('if-no-files-found: ignore', uploads[0])
                self.assertNotIn('*.ipa', uploads[0], 'Failed archive logs are not a recoverable signed IPA')
                self.assertNotIn('mobileprovision', uploads[0])
