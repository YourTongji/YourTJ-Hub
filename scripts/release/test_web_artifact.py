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
    def test_original_archives_are_preserved_before_staging_can_fail(self):
        workflow = (Path(__file__).resolve().parents[2] / '.github/workflows/release-web.yml').read_text()
        self.assertLess(workflow.index('name: Preserve original distribution archives'),
                        workflow.index('name: Stage the exact production binary and its digest'))

    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.addClassCleanup(cls.temp.cleanup)
        cls.root = Path(cls.temp.name)
        (cls.root / 'go.mod').write_text('module example.org/release\n\ngo 1.20\n', encoding='utf-8')
        (cls.root / 'buildinfo').mkdir()
        (cls.root / 'buildinfo/info.go').write_text('package buildinfo\nvar Version, Commit string\n', encoding='utf-8')
        (cls.root / 'main.go').write_text('package main\nimport ("encoding/json"; "fmt"; "os"; "example.org/release/buildinfo")\nfunc main(){if len(os.Args)!=4 || os.Args[1]!="version" || os.Args[2]!="--output" {os.Exit(1)}; data,_:=json.Marshal(map[string]string{"version":buildinfo.Version,"commit":buildinfo.Commit}); if err:=os.WriteFile(os.Args[3],data,0600); err!=nil {panic(err)}; fmt.Println("shutdown logs") }\n', encoding='utf-8')
        cls.dist = cls.root / 'dist'
        binary = cls.dist / 'unix_linux_amd64_v1/yourtj-hub'
        binary.parent.mkdir(parents=True)
        cls.sha = 'a' * 40
        # Use production's trimpath/stripping flags: go version -m omits linker flags.
        # Execute a native fixture on each test host; production stages Linux on Linux.
        subprocess.run(['go', 'build', '-trimpath', '-ldflags', '-s -w -X example.org/release/buildinfo.Version=1.0.50 -X example.org/release/buildinfo.Commit=' + cls.sha,
                        '-o', str(binary), '.'], cwd=cls.root, env={**os.environ, 'CGO_ENABLED': '0', 'GOWORK': 'off'}, check=True, capture_output=True)
        (cls.dist / 'artifacts.json').write_text(json.dumps([{'type': 'Binary', 'goos': 'linux', 'goarch': 'amd64',
            'path': '/previous-runner/project/dist/unix_linux_amd64_v1/yourtj-hub'}]), encoding='utf-8')
        cls.original = binary.read_bytes()

    def test_stages_original_bytes_from_goreleaser_inventory(self):
        # download-artifact restores regular files without their executable bit.
        (self.dist / 'unix_linux_amd64_v1/yourtj-hub').chmod(0o644)
        result = stage(self.dist, self.root / 'bin', self.sha, '1.0.50')
        self.assertEqual((self.root / 'bin/yourtj-hub').read_bytes(), self.original)
        self.assertEqual(result['binarySha256'], hashlib.sha256(self.original).hexdigest())
        if os.name != 'nt':  # Windows does not expose POSIX executable permission bits.
            self.assertTrue((self.root / 'bin/yourtj-hub').stat().st_mode & 0o111)

    def test_similar_version_and_different_source_are_not_the_approved_binary(self):
        for sha, version in [(self.sha, '1.0.5'), ('b' * 40, '1.0.50')]:
            with self.subTest(version=version), self.assertRaises(ValueError):
                stage(self.dist, self.root / 'wrong', sha, version)
