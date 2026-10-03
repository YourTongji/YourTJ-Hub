#!/usr/bin/env python3
"""Select the exact GoReleaser Linux executable used in downloadable archives."""
import hashlib
import json
from pathlib import Path
import shutil
import sys
import subprocess
import tempfile


def stage(dist, destination, source_sha, version):
    artifacts = json.loads((dist / 'artifacts.json').read_text(encoding='utf-8'))
    matches = [a for a in artifacts if a['type'] == 'Binary' and a.get('goos') == 'linux' and a.get('goarch') == 'amd64']
    if len(matches) != 1:
        raise ValueError('Expected exactly one GoReleaser linux/amd64 binary')
    raw = Path(matches[0]['path'])
    # GoReleaser paths may be absolute on the original runner; re-root at dist for recovery.
    parts = raw.parts
    if 'dist' not in parts:
        raise ValueError('Artifact path is outside the distribution directory')
    relative = Path(*parts[parts.index('dist') + 1:])
    if '..' in relative.parts:
        raise ValueError('Unsafe artifact path')
    binary = dist / relative
    # Actions artifact downloads restore files as 0644, including this executable.
    binary.chmod(0o755)
    # A trimpath binary does not retain -ldflags in go version -m. Read the values
    # actually compiled into the executable. An isolated cwd contains legacy package
    # initialization (default config/log creation); the JSON file excludes shutdown logs.
    with tempfile.TemporaryDirectory() as temporary:
        identity = Path(temporary) / 'identity.json'
        subprocess.run([str(binary.resolve()), 'version', '--output', str(identity)],
                       cwd=temporary, capture_output=True, text=True, timeout=30, check=True)
        build_info = json.loads(identity.read_text(encoding='utf-8'))
    for field, value in [('version', version), ('commit', source_sha)]:
        if build_info.get(field) != value:
            raise ValueError('Binary build metadata does not match approved ' + field)
    destination.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(binary, destination / 'yourtj-hub')
    (destination / 'yourtj-hub').chmod(0o755)
    receipt = {'sourceSha': source_sha, 'version': version, 'binarySha256': hashlib.sha256(binary.read_bytes()).hexdigest()}
    (destination / 'binary.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    return receipt


if __name__ == '__main__':
    stage(Path(sys.argv[1]), Path(sys.argv[2]), sys.argv[3], sys.argv[4])
