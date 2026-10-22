"""Publisher unit tests must not write the workflow's real Apple receipt."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class ReceiptIsolationTest(unittest.TestCase):
    def test_publisher_tests_leave_workflow_receipt_untouched(self):
        with tempfile.TemporaryDirectory() as folder:
            receipt = Path(folder) / 'workflow-result.json'
            receipt.write_text('existing workflow receipt')
            result = subprocess.run(
                [sys.executable, '-m', 'unittest', 'test_publish_ios'],
                cwd=Path(__file__).resolve().parent,
                env={**os.environ, 'RELEASE_RESULT_PATH': str(receipt)},
                capture_output=True, text=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(receipt.read_text(), 'existing workflow receipt')
