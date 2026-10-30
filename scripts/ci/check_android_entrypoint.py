#!/usr/bin/env python3
"""Reject an APK that built successfully without the app's Kotlin entrypoint."""
import argparse
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys


ENTRYPOINT = 'tj.yourtj.forum_app.MainActivity'


def find_apkanalyzer():
    executable = shutil.which('apkanalyzer')
    if executable:
        return executable
    for variable in ('ANDROID_HOME', 'ANDROID_SDK_ROOT'):
        if os.environ.get(variable):
            candidate = Path(os.environ[variable]) / 'cmdline-tools/latest/bin/apkanalyzer'
            if candidate.is_file() and os.access(candidate, os.X_OK):
                return str(candidate)
    raise ValueError('apkanalyzer is required: add Android SDK Command-Line Tools to PATH '
                     'or set ANDROID_HOME to the SDK with cmdline-tools/latest installed')


def verify(apk):
    if not apk.is_file():
        raise ValueError(f'Android APK not found: {apk}')
    result = subprocess.run([find_apkanalyzer(), 'dex', 'packages', '--defined-only', str(apk)],
                            capture_output=True, text=True, check=True, timeout=120)
    # Android's documented class-tree output is type/state/method counts/size/name.
    # Inspect definitions across every DEX, not a string/reference or an R8 mapping.
    # https://developer.android.com/tools/apkanalyzer#commands
    definition = rf'^\s*C\s+d\s+\d+\s+\d+\s+\d+\s+{re.escape(ENTRYPOINT)}\s*$'
    if not re.search(definition, result.stdout, re.MULTILINE):
        raise ValueError(f'APK does not define {ENTRYPOINT}; check Kotlin compilation and '
                         'the Android entrypoint before accepting this build')
    print(f'Android APK defines {ENTRYPOINT}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('apk', type=Path)
    args = parser.parse_args()
    try:
        verify(args.apk)
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print(f'Android entrypoint verification failed: {error}', file=sys.stderr)
        if isinstance(error, subprocess.CalledProcessError) and error.stderr:
            print(error.stderr[-3000:], file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
