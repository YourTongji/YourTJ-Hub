"""Number allocation and native uploads must share the same ordering boundary."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2] / '.github/workflows'

class PackingQueueTests(unittest.TestCase):
    def test_dev_and_rc_hold_one_lock_from_allocation_through_native_upload(self):
        dev = (ROOT/'dev-mobile.yml').read_text()
        rc = (ROOT/'release-rc.yml').read_text()
        for text in (dev, rc):
            self.assertIn('release-mobile-packaging', text.split('\njobs:', 1)[0])
            self.assertIn('queue: max', text.split('\njobs:', 1)[0])

    def test_existing_apple_build_promotion_does_not_wait_for_native_compilation(self):
        text = (ROOT/'release-ios.yml').read_text().split('\njobs:', 1)[0]
        self.assertIn("inputs.promotion && 'release-apple-metadata-6809457637'", text)

if __name__ == '__main__': unittest.main()
