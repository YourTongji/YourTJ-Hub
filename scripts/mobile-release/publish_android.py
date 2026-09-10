#!/usr/bin/env python3
"""Verify signed APKs, then add immutable-name assets to a mobile GitHub Release."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]
REPOSITORY = "YourTongji/YourTJ-Hub"
ABIS = ("arm64-v8a", "armeabi-v7a", "x86_64")


def android_version_code(number, abi):
    # Flutter 3.44.9 split-per-abi codes; validate the actual APK before publishing.
    code = int(number) + {"armeabi-v7a": 1000, "arm64-v8a": 2000, "x86_64": 4000}[abi]
    if int(number) <= 0 or code > 2100000000:
        raise ValueError("Android version code is outside the supported range")
    return str(code)


def check_package(badging, signatures, version, number, certificate):
    match = re.search(r"package: name='([^']+)' versionCode='([^']+)' versionName='([^']+)'", badging)
    if not match or match.groups() != ("tj.yourtj.forum_app", number, version):
        raise ValueError("APK package/version does not match this release")
    # Build Tools 37 labels certificates by scheme (e.g. "V2 Signer:").
    # The same identity can appear under multiple schemes. Public-key hashes and
    # source-stamp certificates are not APK signer identities.
    fingerprints = re.findall(
        r"^(?:Signer #\d+|V[123](?:\.[12])? Signer:) certificate SHA-256 digest: ([0-9a-fA-F]{64})$",
        signatures, re.M)
    if not fingerprints:
        raise ValueError("No supported APK signer certificate fingerprint found in apksigner output")
    if {fingerprint.lower() for fingerprint in fingerprints} != {certificate.lower()}:
        raise ValueError("APK is not signed by the YourTJ release certificate")


def gh(*args, allow_missing=False):
    result = subprocess.run(["gh", *args], capture_output=True, text=True)
    if result.returncode:
        if allow_missing and ("404" in result.stderr or result.stderr.strip() == "release not found"):
            return None
        raise RuntimeError(result.stderr.strip())
    return result.stdout


def find_release(tag):
    # The tag REST endpoint does not reliably expose drafts. gh release view
    # resolves drafts too; use the immutable database ID for subsequent API reads.
    resolved = gh("release", "view", tag, "--repo", REPOSITORY, "--json", "databaseId", allow_missing=True)
    if resolved is None:
        return None
    release_id = json.loads(resolved)["databaseId"]
    return json.loads(gh("api", f"repos/{REPOSITORY}/releases/{release_id}"))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--verify-only", action="store_true")
    args = parser.parse_args()
    version = os.environ["MOBILE_VERSION"]
    number = os.environ["MOBILE_BUILD_NUMBER"]
    tag = os.environ["RELEASE_TAG"]
    if not re.fullmatch(r"\d+\.\d+\.\d+", version) or not re.fullmatch(r"[1-9]\d*", number) or tag != f"mobile-v{version}":
        raise ValueError("Invalid mobile release identity")
    config = json.loads((ROOT / "apps/mobile/release-config.json").read_text())
    tools = Path(os.environ.get("ANDROID_HOME") or os.environ["ANDROID_SDK_ROOT"]) / "build-tools"
    versions = [p for p in tools.iterdir() if re.fullmatch(r"\d+(\.\d+)*", p.name)]
    toolchain = max(versions, key=lambda p: tuple(map(int, p.name.split('.'))))
    source = ROOT / "apps/mobile/packages/forum_app/build/app/outputs/flutter-apk"
    output = ROOT / "apps/mobile/build/release/android"
    output.mkdir(parents=True, exist_ok=True)
    assets = []
    for abi in ABIS:
        apk = source / f"app-{abi}-release.apk"
        signatures = subprocess.check_output([str(toolchain / "apksigner"), "verify", "--verbose", "--print-certs", str(apk)], text=True)
        badging = subprocess.check_output([str(toolchain / "aapt"), "dump", "badging", str(apk)], text=True)
        code = android_version_code(number, abi)
        check_package(badging, signatures, version, code, config["androidCertificateSha256"])
        if f"native-code: '{abi}'" not in badging:
            raise ValueError(f"APK contains an unexpected native architecture: {abi}")
        destination = output / f"YourTJ-{version}+{code}-{abi}.apk"
        shutil.copyfile(apk, destination)
        assets.append((destination, hashlib.sha256(destination.read_bytes()).hexdigest()))
    checksums = output / "SHA256SUMS.txt"
    checksums.write_text(''.join(f"{digest}  {path.name}\n" for path, digest in assets))
    assets.append((checksums, hashlib.sha256(checksums.read_bytes()).hexdigest()))

    if args.verify_only:
        print("Verified three signed APKs and wrote SHA256SUMS.txt")
        return

    release = find_release(tag)
    if release is None:
        notes = output / "notes.md"
        notes.write_text((ROOT / "apps/mobile/store/zh-Hans/metadata.json").read_text())
        metadata = json.loads(notes.read_text())
        notes.write_text(metadata["whatsNew"] + "\n\nAndroid：按设备架构下载安装 APK。iOS：通过 TestFlight 或 App Store 分发，审核状态以 Apple 为准。\n")
        gh("release", "create", tag, "--repo", REPOSITORY, "--verify-tag", "--draft", "--latest=false",
           "--title", f"YourTJ {version}", "--notes-file", str(notes))
        release = find_release(tag)
        if release is None:
            raise RuntimeError("Created release is not discoverable yet; retry to resume its draft")
    endpoint = f"repos/{REPOSITORY}/releases/{release['id']}"
    existing = {asset["name"]: asset for asset in release["assets"]}
    for path, digest in assets:
        if path.name in existing:
            if existing[path.name].get("digest") != "sha256:" + digest:
                raise ValueError(f"Release asset already exists with different bytes: {path.name}; never overwrite an APK")
            continue
        gh("release", "upload", tag, str(path), "--repo", REPOSITORY)
    # GitHub computes asset digests independently of the runner. Wait for those
    # digests before exposing a draft to the update client.
    for attempt in range(12):
        uploaded = {asset["name"]: asset for asset in json.loads(gh("api", endpoint))["assets"]}
        if all(uploaded.get(path.name, {}).get("digest") == "sha256:" + digest for path, digest in assets):
            break
        if attempt == 11:
            raise ValueError("GitHub asset digest verification failed; release remains a draft")
        time.sleep(5)
    gh("release", "edit", tag, "--repo", REPOSITORY, "--draft=false", "--latest=false")
    print(f"Verified APK release: https://github.com/{REPOSITORY}/releases/tag/{tag}")


if __name__ == "__main__":
    main()
