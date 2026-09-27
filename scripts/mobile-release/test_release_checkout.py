"""Run native-source checks from the separate release-tool checkout used by iOS."""
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


class ReleaseCheckoutTest(unittest.TestCase):
    def test_sparse_tools_validate_the_release_source_working_directory(self):
        tools = Path(__file__).resolve().parent
        with tempfile.TemporaryDirectory() as temporary:
            checkout = Path(temporary) / '.release-tools/scripts/mobile-release'
            checkout.mkdir(parents=True)
            for name in ('build_ios.py', 'test_android_push_consent.py', 'test_privacy_manifests.py'):
                shutil.copyfile(tools / name, checkout / name)
            result = subprocess.run(
                [sys.executable, '-m', 'unittest', 'discover', '-s', str(checkout), '-p', 'test_*.py'],
                cwd=Path.cwd(), capture_output=True, text=True, timeout=30,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
