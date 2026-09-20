#!/usr/bin/env bash
set -euo pipefail

# Build a physical-device debug APK against the shared dev backend.
# Keep the endpoint in build-time defines so local emulator/test defaults stay local.
SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
FORUM_APP_DIR="$SCRIPT_ROOT/apps/mobile/packages/forum_app"
DEV_API_URL="${YOURTJ_DEV_API_URL:-https://dev.yourtj.de}"
DEV_OIDC_ISSUER="${YOURTJ_DEV_OIDC_ISSUER:-$DEV_API_URL/api/oauth}"
DEV_CLIENT_ID="${YOURTJ_DEV_CLIENT_ID:-yourtj-mobile}"

cd "$FORUM_APP_DIR"
flutter build apk --debug --split-per-abi \
  --dart-define=YOURTJ_API_BASE_URL="$DEV_API_URL" \
  --dart-define=YOURTJ_OIDC_ISSUER="$DEV_OIDC_ISSUER" \
  --dart-define=YOURTJ_OIDC_CLIENT_ID="$DEV_CLIENT_ID"

printf '\nDev APKs:\n'
for apk in build/app/outputs/flutter-apk/app-*-debug.apk; do
  printf '  %s\n' "$FORUM_APP_DIR/$apk"
done
