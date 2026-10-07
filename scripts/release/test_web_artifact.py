"""Stage a real Go executable, including restoration on a different runner path."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from web_artifact import stage


class WebArtifactTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.temp.cleanup)
        cls.root = Path(cls.temp.name)
        (cls.root / 'go.mod').write_text('module example.org/release\n\ngo 1.20\n', encoding='utf-8')
        (cls.root / 'buildinfo').mkdir()
        (cls.root / 'buildinfo/info.go').write_text('package buildinfo\nvar Version, Commit string\n', encoding='utf-8')
        (cls.root / 'main.go').write_text('package main\nimport ("fmt"; "example.org/release/buildinfo")\nfunc main(){fmt.Println(buildinfo.Version, buildinfo.Commit)}\n', encoding='utf-8')
        (cls.root / '.gitignore').write_text('dist/\n', encoding='utf-8')
        subprocess.run(['git', 'init', '-q', str(cls.root)], check=True, capture_output=True)
        subprocess.run(['git', 'add', '.'], cwd=cls.root, check=True, capture_output=True)
        subprocess.run(['git', '-c', 'user.name=Release fixture', '-c', 'user.email=fixture@example.org',
                        'commit', '-qm', 'fixture'], cwd=cls.root, check=True, capture_output=True)
        cls.sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=cls.root, text=True).strip()
        cls.dist = cls.root / 'dist'
        binary = cls.dist / 'unix_linux_amd64_v1/yourtj-hub'
        binary.parent.mkdir(parents=True)
        subprocess.run(['go', 'build', '-trimpath', '-ldflags', '-s -w -X example.org/release/buildinfo.Version=1.0.50 -X example.org/release/buildinfo.Commit=' + cls.sha,
                        '-o', str(binary), '.'], cwd=cls.root, env={**os.environ, 'GOOS': 'linux', 'GOARCH': 'amd64', 'CGO_ENABLED': '0', 'GOWORK': 'off'}, check=True, capture_output=True)
        (cls.dist / 'artifacts.json').write_text(json.dumps([{'type': 'Binary', 'goos': 'linux', 'goarch': 'amd64',
            'path': '/previous-runner/project/dist/unix_linux_amd64_v1/yourtj-hub'}]), encoding='utf-8')
        cls.metadata = {'version': '1.0.50', 'commit': cls.sha}
        (cls.dist / 'metadata.json').write_text(json.dumps(cls.metadata), encoding='utf-8')
        cls.original = binary.read_bytes()

    def test_stages_original_bytes_from_goreleaser_inventory(self):
        binary = self.dist / 'unix_linux_amd64_v1/yourtj-hub'
        self.assertEqual(subprocess.check_output([str(binary)], text=True).strip(), '1.0.50 ' + self.sha)
        self.assertNotIn('-ldflags', subprocess.check_output(['go', 'version', '-m', str(binary)], text=True))
        result = stage(self.dist, self.root / 'bin', self.sha, '1.0.50')
        self.assertEqual((self.root / 'bin/yourtj-hub').read_bytes(), self.original)
        self.assertEqual(result['binarySha256'], hashlib.sha256(self.original).hexdigest())
        if os.name != 'nt':  # Windows does not expose POSIX executable permission bits.
            self.assertTrue((self.root / 'bin/yourtj-hub').stat().st_mode & 0o111)

    def test_similar_version_and_different_source_are_not_the_approved_binary(self):
        for sha, version in [(self.sha, '1.0.5'), ('b' * 40, '1.0.50')]:
            with self.subTest(version=version), self.assertRaises(ValueError):
                stage(self.dist, self.root / 'wrong', sha, version)

    def test_binary_revision_cannot_be_substituted_by_distribution_metadata(self):
        other_sha = 'b' * 40
        metadata = self.dist / 'metadata.json'
        metadata.write_text(json.dumps(self.metadata | {'commit': other_sha}), encoding='utf-8')
        try:
            with self.assertRaisesRegex(ValueError, 'Binary.*source'):
                stage(self.dist, self.root / 'wrong', other_sha, '1.0.50')
        finally:
            metadata.write_text(json.dumps(self.metadata), encoding='utf-8')
