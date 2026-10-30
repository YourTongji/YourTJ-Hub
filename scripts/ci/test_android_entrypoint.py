"""Exercise the native workflow with a successful build but missing Kotlin code."""
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from check_android_entrypoint import find_apkanalyzer, verify


ROOT = Path(__file__).resolve().parents[2]
ENTRYPOINT = 'tj.yourtj.forum_app.MainActivity'


class AndroidEntrypointWorkflowTests(unittest.TestCase):
    def run_build(self, dex_output, analyzer_exit=0):
        workflow = (ROOT / '.github/workflows/ci-mobile-native.yml').read_text()
        step = workflow.split('      - name: Compile and shrink Android with all seven push adapters\n', 1)[1]
        step = step.split('\n      - name:', 1)[0]
        script = '\n'.join(line[10:] for line in step.split('        run: |\n', 1)[1].splitlines())
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            app = root / 'apps/mobile/packages/forum_app'
            (app / 'android').mkdir(parents=True)
            shutil.copytree(ROOT / 'scripts/ci', root / 'scripts/ci')
            bin_dir = root / 'bin'
            bin_dir.mkdir()
            programs = {
                'keytool': 'pass',
                'flutter': "from pathlib import Path\np = Path('build/app/outputs/flutter-apk/app-arm64-v8a-release.apk')\np.parent.mkdir(parents=True)\np.write_bytes(b'fixture APK')",
                'apkanalyzer': "import os, sys\nfrom pathlib import Path\nPath(os.environ['ANALYZER_ARGS']).write_text('\\n'.join(sys.argv[1:]))\nprint(os.environ['DEX_OUTPUT'])\nsys.exit(int(os.environ['ANALYZER_EXIT']))",
            }
            for name, program in programs.items():
                executable = bin_dir / name
                executable.write_text(f'#!{sys.executable}\n{program}\n')
                executable.chmod(0o755)
            env = os.environ | {
                'PATH': f'{bin_dir}{os.pathsep}{os.environ["PATH"]}',
                'RUNNER_TEMP': temporary,
                'DEX_OUTPUT': dex_output,
                'ANALYZER_EXIT': str(analyzer_exit),
                'ANALYZER_ARGS': str(root / 'analyzer-args'),
            }
            result = subprocess.run(['bash', '-euo', 'pipefail', '-c', script], cwd=app,
                                    env=env, capture_output=True, text=True, timeout=30)
            args_file = root / 'analyzer-args'
            return result, args_file.read_text().splitlines() if args_file.exists() else []

    def test_missing_entrypoint_rejects_successful_flutter_build(self):
        result, _ = self.run_build('C d 3 3 123 io.flutter.SomePlugin')
        self.assertNotEqual(result.returncode, 0, 'Flutter success must not hide missing Kotlin entrypoint')
        self.assertIn(ENTRYPOINT, result.stderr)

    def test_defined_entrypoint_passes_and_inspects_all_dex_files(self):
        result, args = self.run_build(f'P d 8 8 256 tj.yourtj.forum_app\n  C d 8 8 256 {ENTRYPOINT}')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(args, ['dex', 'packages', '--defined-only',
                                'build/app/outputs/flutter-apk/app-arm64-v8a-release.apk'])

    def test_reference_or_inner_class_is_not_a_definition(self):
        for output in (f'C r 0 2 10 {ENTRYPOINT}', f'C d 1 1 10 {ENTRYPOINT}$Companion',
                       f'M d 1 1 10 {ENTRYPOINT}', ''):
            with self.subTest(output=output):
                result, _ = self.run_build(output)
                self.assertNotEqual(result.returncode, 0)

    def test_inspector_failure_does_not_accept_partial_output(self):
        result, _ = self.run_build(f'C d 8 8 256 {ENTRYPOINT}', analyzer_exit=1)
        self.assertNotEqual(result.returncode, 0)


class AndroidEntrypointToolTests(unittest.TestCase):
    def test_path_inspector_takes_precedence(self):
        with patch('check_android_entrypoint.shutil.which', return_value='/sdk/bin/apkanalyzer'):
            self.assertEqual(find_apkanalyzer(), '/sdk/bin/apkanalyzer')

    def test_sdk_inspector_fallback_requires_an_executable(self):
        with tempfile.TemporaryDirectory() as temporary:
            inspector = Path(temporary) / 'cmdline-tools/latest/bin/apkanalyzer'
            inspector.parent.mkdir(parents=True)
            inspector.write_text('#!/bin/sh\n')
            for variable in ('ANDROID_HOME', 'ANDROID_SDK_ROOT'):
                with self.subTest(variable=variable), \
                     patch('check_android_entrypoint.shutil.which', return_value=None), \
                     patch.dict(os.environ, {variable: temporary}, clear=True):
                    inspector.chmod(0o644)
                    with self.assertRaisesRegex(ValueError, 'apkanalyzer is required'):
                        find_apkanalyzer()
                    inspector.chmod(0o755)
                    self.assertEqual(find_apkanalyzer(), str(inspector))

    def test_missing_inspector_fails_instead_of_skipping(self):
        with patch('check_android_entrypoint.shutil.which', return_value=None), \
             patch.dict(os.environ, {}, clear=True):
            with self.assertRaisesRegex(ValueError, 'apkanalyzer is required'):
                find_apkanalyzer()

    def test_missing_apk_fails_before_inspection(self):
        with tempfile.TemporaryDirectory() as temporary, \
             patch('check_android_entrypoint.subprocess.run') as run:
            with self.assertRaisesRegex(ValueError, 'APK not found'):
                verify(Path(temporary) / 'missing.apk')
            run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
