#!/bin/bash
set -euo pipefail

purpose="${YOURTJ_IOS_BUILD_PURPOSE:-production}"
if [[ "$purpose" != dev && "$purpose" != production ]]; then
  echo 'Unknown iOS build purpose' >&2
  exit 1
fi
export DEVELOPER_DIR="${DEVELOPER_DIR:-/Applications/Xcode_27.2.app/Contents/Developer}"
test -d "$DEVELOPER_DIR"
xcode_version="$(xcodebuild -version)"
printf '%s\n' "$xcode_version"
if [[ "$xcode_version" != $'Xcode 27.2\nBuild version 27B52' ]]; then
  if [[ "$purpose" != dev || "$xcode_version" != $'Xcode 27.2\nBuild version 27B48' ]]; then
    echo 'Expected Xcode 27.2 release (27B52), or the reviewed RC (27B48) for dev previews only.' >&2
    exit 1
  fi
fi
sdk_version="$(xcrun --sdk iphoneos --show-sdk-version)"
if [[ "$sdk_version" != '27.2' ]]; then
  echo "Expected iOS 27.2 SDK, found $sdk_version" >&2
  exit 1
fi
printf 'DEVELOPER_DIR=%s\n' "$DEVELOPER_DIR" >> "$GITHUB_ENV"
