#!/usr/bin/env python3
"""Refresh fixed APK download aliases from verified, public mobile releases.

Only the mobile-latest channel is mutable; versioned releases remain untouched.
Run under release-mobile's concurrency group (or while no release is running).
See docs/decisions/0050-android-stable-download-links.md.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import tempfile
import time

from publish_android import ABIS, REPOSITORY, find_release, gh

ALIAS_TAG = 'mobile-latest'


def mobile_version(tag):
    match = re.fullmatch(r'mobile-v(\d+)\.(\d+)\.(\d+)', tag)
    return tuple(map(int, match.groups())) if match else None


def select_source(releases):
    candidates = [release for release in releases
                  if mobile_version(release['tag_name']) is not None
                  and not release['draft'] and not release['prerelease']]
    if not candidates:
        raise ValueError('No published stable mobile release is available')
    return max(candidates, key=lambda release: mobile_version(release['tag_name']))


def source_assets(source):
    tag = source['tag_name']
    if mobile_version(tag) is None or source.get('draft') or source.get('prerelease'):
        raise ValueError('Source must be a published stable mobile release')
    version = tag.removeprefix('mobile-v')
    selected = []
    numbers = set()
    for abi in ABIS:
        pattern = rf'YourTJ-{re.escape(version)}\+([1-9]\d*)-{re.escape(abi)}\.apk'
        matches = [(asset, re.fullmatch(pattern, asset['name'])) for asset in source['assets']]
        matches = [(asset, match) for asset, match in matches if match]
        if len(matches) != 1:
            raise ValueError(f'Expected one source APK for {abi}')
        asset, match = matches[0]
        code = int(match[1])
        number = code - {'arm64-v8a': 2000, 'armeabi-v7a': 1000, 'x86_64': 4000}[abi]
        if number <= 0 or code > 2100000000:
            raise ValueError('Invalid Android build number')
        numbers.add(number)
        digest = asset.get('digest') or ''
        if not re.fullmatch(r'sha256:[0-9a-f]{64}', digest) or asset.get('size', 0) <= 0:
            raise ValueError('Source APK lacks a GitHub SHA-256 digest or size')
        selected.append((abi, asset))
    if len(numbers) != 1:
        raise ValueError('Source APKs belong to different builds')
    return selected


def file_digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def stage_source(source, directory):
    assets = []
    for abi, asset in source_assets(source):
        gh('release', 'download', source['tag_name'], '--repo', REPOSITORY,
           '--pattern', asset['name'], '--dir', str(directory))
        downloaded = directory / asset['name']
        digest = file_digest(downloaded)
        if downloaded.stat().st_size != asset['size'] or f'sha256:{digest}' != asset['digest']:
            raise ValueError(f'Source APK verification failed for {abi}; aliases are unchanged')
        destination = directory / f'YourTJ-{abi}.apk'
        downloaded.rename(destination)
        assets.append((destination, digest))
    checksums = directory / 'SHA256SUMS.txt'
    checksums.write_text(''.join(f'{digest}  {path.name}\n' for path, digest in assets))
    assets.append((checksums, file_digest(checksums)))
    return assets


def source_marker(source):
    return f"<!-- yourtj-android-latest:{source['tag_name']} -->"


def publish_alias(source, assets, directory):
    alias = find_release(ALIAS_TAG)
    if alias is not None:
        if alias.get('tag_name') != ALIAS_TAG or alias.get('immutable') or not alias.get('prerelease'):
            raise ValueError('Refusing to overwrite a stable, immutable or unrelated release')
        marker = re.search(r'<!-- yourtj-android-latest:(mobile-v\d+\.\d+\.\d+) -->', alias.get('body', ''))
        if marker and mobile_version(marker[1]) > mobile_version(source['tag_name']):
            print('A newer Android alias is already published; skipping older recovery')
            return
        if not marker:
            raise ValueError('Existing alias has no recognized source marker')
    notes = directory / 'latest-notes.md'
    notes.write_text(
        source_marker(source) + '\n\n'
        f"最新版 Android APK：[原始版本 {source['tag_name']}](https://github.com/{REPOSITORY}/releases/tag/{source['tag_name']})。\n\n"
        '这里是固定下载入口，安装包与原始正式版本逐字节一致。'
        '本入口标记为 pre-release 仅用于区分下载渠道，不表示 APK 是测试版。'
        '更新时单个链接可能短暂不可用；此时请使用原始版本页面。\n\n'
        + '\n'.join(f'- [{abi}](https://github.com/{REPOSITORY}/releases/download/{ALIAS_TAG}/YourTJ-{abi}.apk)' for abi in ABIS)
        + '\n\nSHA256SUMS.txt 校验当前别名文件。源码请使用上方原始版本标签；本页自动生成的源码归档不代表最新 APK。\n'
    )
    if alias is None:
        commit = gh('api', f"repos/{REPOSITORY}/commits/{source['tag_name']}", '--jq', '.sha').strip()
        if not re.fullmatch(r'[0-9a-f]{40}', commit):
            raise ValueError('Cannot resolve the source release commit')
        gh('release', 'create', ALIAS_TAG, '--repo', REPOSITORY, '--target', commit,
           '--draft', '--prerelease', '--latest=false', '--title', 'YourTJ Android 最新版',
           '--notes-file', str(notes))
        alias = find_release(ALIAS_TAG)
        if alias is None:
            raise RuntimeError('Alias draft is not discoverable yet; retry to resume')
    existing = {asset['name']: asset for asset in alias['assets']}
    for path, digest in assets:
        if existing.get(path.name, {}).get('digest') == f'sha256:{digest}':
            continue
        # Clobber is confined to the rolling alias, never a mobile-v* release.
        gh('release', 'upload', ALIAS_TAG, str(path), '--repo', REPOSITORY, '--clobber')
    for attempt in range(12):
        uploaded = json.loads(gh('api', f"repos/{REPOSITORY}/releases/{alias['id']}"))
        digests = {asset['name']: asset.get('digest') for asset in uploaded['assets']}
        if all(digests.get(path.name) == f'sha256:{digest}' for path, digest in assets):
            break
        if attempt == 11:
            raise ValueError('Alias digest verification failed; retry from the original release')
        time.sleep(5)
    gh('release', 'edit', ALIAS_TAG, '--repo', REPOSITORY, '--draft=false',
       '--prerelease', '--latest=false', '--notes-file', str(notes))
    print(f'Android download: https://github.com/{REPOSITORY}/releases/download/{ALIAS_TAG}/YourTJ-arm64-v8a.apk')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--verify-only', action='store_true', help='Download and verify without changing releases')
    args = parser.parse_args()
    releases = json.loads(gh('api', f'repos/{REPOSITORY}/releases?per_page=100'))
    source = select_source(releases)
    with tempfile.TemporaryDirectory(prefix='yourtj-android-latest-') as temporary:
        directory = Path(temporary)
        assets = stage_source(source, directory)
        if args.verify_only:
            print(f"Verified aliases from {source['tag_name']}; no releases changed")
            return
        publish_alias(source, assets, directory)


if __name__ == '__main__':
    main()
