#!/usr/bin/env python3
"""Enforce the native CI arm64 release APK's fixed +3% size budget."""
import argparse
import hashlib
import json
from pathlib import Path
import sys


ROOT = Path(__file__).resolve().parents[2]
BASELINE = Path(__file__).with_name('android_apk_size_baseline.json')
BUILD_FILES = (
    'app/build.gradle.kts', 'build.gradle.kts', 'gradle.properties',
    'gradle/wrapper/gradle-wrapper.properties', 'settings.gradle.kts',
)


def build_config_sha256(root=ROOT):
    digest = hashlib.sha256()
    android = root / 'apps/mobile/packages/forum_app/android'
    for name in BUILD_FILES:
        digest.update(name.encode() + b'\0' + (android / name).read_bytes() + b'\0')
    return digest.hexdigest()


def read_metadata(path):
    value = json.loads(path.read_text())
    if not isinstance(value, dict):
        raise ValueError(f'{path.name} must contain a JSON object')
    size = value.get('apkBytes')
    if type(size) is not int or size <= 0:
        raise ValueError(f'{path.name}: apkBytes must be a positive integer')
    return value


def verify(current, baseline):
    for field in ('flutterVersion', 'buildMode', 'targetPlatform', 'androidBuildConfigSha256'):
        expected = baseline.get(field)
        if not isinstance(expected, str) or not expected or current.get(field) != expected:
            raise ValueError(f'{field} differs from the baseline or is missing; '
                             'review build compatibility and record a new measured baseline')
    # Integer comparison includes exactly +3%, without float rounding at byte boundaries.
    maximum = baseline['apkBytes'] * 103 // 100
    if current['apkBytes'] > maximum:
        raise ValueError(f"APK size {current['apkBytes']} bytes exceeds {maximum} bytes "
                         f"(baseline {baseline['apkBytes']} +3%); inspect the uploaded size report")
    print(f"APK size {current['apkBytes']} bytes <= {maximum} bytes "
          f"(baseline {baseline['apkBytes']} +3%)")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('metadata', type=Path, nargs='?')
    parser.add_argument('--baseline', type=Path, default=BASELINE)
    parser.add_argument('--build-config-sha256', action='store_true')
    args = parser.parse_args()
    if not args.build_config_sha256 and args.metadata is None:
        parser.error('metadata is required unless --build-config-sha256 is used')
    try:
        if args.build_config_sha256:
            print(build_config_sha256())
        else:
            verify(read_metadata(args.metadata), read_metadata(args.baseline))
    except (ValueError, OSError) as error:
        print(f'Android APK size verification failed: {error}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
