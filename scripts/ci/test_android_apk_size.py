"""Exercise the size gate's CLI and the native workflow's actual invocation."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from check_android_apk_size import BUILD_FILES, build_config_sha256


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / 'scripts/ci/check_android_apk_size.py'
BASELINE = ROOT / 'scripts/ci/android_apk_size_baseline.json'


class AndroidApkSizeTests(unittest.TestCase):
    def check(self, metadata, baseline=None):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            current = directory / 'build-metadata.json'
            current.write_text(json.dumps(metadata))
            baseline_path = BASELINE
            if baseline is not None:
                baseline_path = directory / 'baseline.json'
                baseline_path.write_text(json.dumps(baseline))
            return subprocess.run([sys.executable, str(SCRIPT), str(current),
                                   '--baseline', str(baseline_path)],
                                  capture_output=True, text=True, timeout=10)

    def metadata(self, size=48833689):
        return {'gitSha': 'a' * 40, 'flutterVersion': '3.44.9',
                'buildMode': 'release', 'targetPlatform': 'android-arm64',
                'androidBuildConfigSha256': '6e17addbb2b096550e2a63d012d13e5bbed432960cd5d999886601a11c268afe',
                'apkBytes': size}

    def test_recorded_baseline_and_exact_three_percent_boundary(self):
        for size in (48833689, 50298699):
            with self.subTest(size=size):
                result = self.check(self.metadata(size))
                self.assertEqual(result.returncode, 0, result.stderr)
        result = self.check(self.metadata(50298700))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('exceeds', result.stderr)

    def test_mismatched_toolchain_mode_or_abi_requires_reviewed_baseline(self):
        for field, value in (('flutterVersion', '3.45.0'), ('buildMode', 'debug'),
                             ('targetPlatform', 'android-x64'),
                             ('androidBuildConfigSha256', 'b' * 64)):
            with self.subTest(field=field):
                result = self.check(self.metadata() | {field: value})
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(field, result.stderr)

    def test_build_fingerprint_covers_gradle_configuration(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            android = root / 'apps/mobile/packages/forum_app/android'
            for name in BUILD_FILES:
                path = android / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('original')
            initial = build_config_sha256(root)
            for name in BUILD_FILES:
                with self.subTest(name=name):
                    path = android / name
                    path.write_text('changed')
                    self.assertNotEqual(build_config_sha256(root), initial)
                    path.write_text('original')
            (android / BUILD_FILES[0]).unlink()
            with self.assertRaises(FileNotFoundError):
                build_config_sha256(root)

    def test_invalid_baseline_size_fails_closed(self):
        for size in (0, True, '48833689'):
            with self.subTest(size=size):
                result = self.check(self.metadata(), self.metadata(size))
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('apkBytes', result.stderr)

    def test_missing_invalid_and_boolean_sizes_fail_closed(self):
        for size in (0, -1, True, '48833689', 48833689.0, None):
            with self.subTest(size=size):
                result = self.check(self.metadata(size))
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('apkBytes', result.stderr)
        metadata = self.metadata()
        del metadata['apkBytes']
        self.assertNotEqual(self.check(metadata).returncode, 0)

    def test_malformed_and_missing_metadata_fail_closed(self):
        for content in ('{', '[]'):
            with tempfile.TemporaryDirectory() as temporary:
                path = Path(temporary) / 'metadata.json'
                path.write_text(content)
                result = subprocess.run([sys.executable, str(SCRIPT), str(path)],
                                        capture_output=True, text=True, timeout=10)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn('Traceback', result.stderr)
        with tempfile.TemporaryDirectory() as temporary:
            result = subprocess.run([sys.executable, str(SCRIPT), str(Path(temporary) / 'missing')],
                                    capture_output=True, text=True, timeout=10)
            self.assertNotEqual(result.returncode, 0)

    def test_native_workflow_rejects_oversized_report_after_upload(self):
        workflow = (ROOT / '.github/workflows/ci-mobile-native.yml').read_text()
        title = '      - name: Check Android release APK size budget\n'
        self.assertIn(title, workflow)
        self.assertLess(workflow.index('      - name: Upload Android release size analysis\n'),
                        workflow.index(title))
        step = workflow.split(title, 1)[1].split('\n      - name:', 1)[0]
        self.assertIn("if: matrix.platform == 'android'", step)
        self.assertNotIn('continue-on-error', step)
        script = '\n'.join(line[10:] for line in step.split('        run: |\n', 1)[1].splitlines())
        with tempfile.TemporaryDirectory() as temporary:
            report = Path(temporary) / 'mobile-release-size/build-metadata.json'
            report.parent.mkdir()
            report.write_text(json.dumps(self.metadata(50298700)))
            result = subprocess.run(['bash', '-euo', 'pipefail', '-c', script], cwd=ROOT,
                                    env=os.environ | {'RUNNER_TEMP': temporary},
                                    capture_output=True, text=True, timeout=10)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('exceeds', result.stderr)


if __name__ == '__main__':
    unittest.main()
