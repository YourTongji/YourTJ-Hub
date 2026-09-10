#!/usr/bin/env python3
"""Install the pinned official ASC CLI after checking its release digest."""
import hashlib
from pathlib import Path
import platform
import sys
import urllib.request

VERSION = "5.0.0"
DIGESTS = {
    "macOS_arm64": "7e1d5dfafa053555f4db63478dbcba6f2a39b1563b2171ca3f4b6404f27afbb0",
    "macOS_amd64": "b5ce1901558f26b56fe2dd08138e760d647f2772a8e75047197e0ef5b2199263",
    "linux_amd64": "76dc06fab91b0f6db73f42bb977fa1f61817b5ee5cb0958a38408f1aceeb3415",
    "linux_arm64": "f457c466e869bcf1824795f493d4b9169644d7784f0dd124dedd08461c3d5099",
}


def main():
    system = {"Darwin": "macOS", "Linux": "linux"}[platform.system()]
    architecture = {"arm64": "arm64", "aarch64": "arm64", "x86_64": "amd64"}[platform.machine()]
    target = f"{system}_{architecture}"
    url = f"https://github.com/rorkai/App-Store-Connect-CLI/releases/download/{VERSION}/asc_{VERSION}_{target}"
    with urllib.request.urlopen(url, timeout=120) as response:
        data = response.read(100 * 1024 * 1024)
    if hashlib.sha256(data).hexdigest() != DIGESTS[target]:
        raise ValueError("ASC binary checksum mismatch")
    destination = Path(sys.argv[1])
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_bytes(data)
    destination.chmod(0o755)
    print(f"Installed verified ASC {VERSION}: {destination}")


if __name__ == "__main__":
    main()
