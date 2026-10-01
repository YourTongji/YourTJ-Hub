import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


@unittest.skipIf(os.name == 'nt', 'Production deployment uses POSIX shell and executable permissions')
class DeploymentTest(unittest.TestCase):
    def test_immutable_digest_and_running_binary_are_checked(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / 'main').mkdir()
            (root / '.env').write_text('IMAGE_REPO=ghcr.io/yourtongji/yourtj-hub\nMAIN_TAG=main-old\n', encoding='utf-8')
            (root / 'docker-compose.yaml').write_text('services: {}\n', encoding='utf-8')
            (root / 'main/config.toml').write_text('safe=true\n', encoding='utf-8')
            (root / 'bin').mkdir()
            image = 'main-v1.2.3@sha256:' + 'd' * 64
            docker = root / 'bin/docker'
            docker.write_text('#!/bin/sh\ncase "$1" in\nexec) echo "' + 'a' * 64 + '  /app/yourtj-hub";;\ninspect) echo "ghcr.io/yourtongji/yourtj-hub:' + image + '";;\nesac\nexit 0\n', encoding='utf-8')
            docker.chmod(0o755)
            curl = root / 'bin/curl'
            curl.write_text('#!/bin/sh\nexit 0\n', encoding='utf-8'); curl.chmod(0o755)
            env = {**os.environ, 'YOURTJ_ROOT': tmp, 'PATH': str(root / 'bin') + ':' + os.environ['PATH'],
                   'EXPECTED_BINARY_SHA256': 'b' * 64}
            result = subprocess.run(['bash', str(ROOT / 'deploy/scripts/deploy.sh'), 'main', image], env=env, capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0, 'Wrong running binary must fail deployment, even with healthy HTTP')
            self.assertIn('MAIN_TAG=main-old', (root / '.env').read_text(encoding='utf-8'))
            self.assertFalse((root / 'main/.config.lock').exists())
