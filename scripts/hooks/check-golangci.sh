#!/usr/bin/env bash
# golangci-lint pre-push gate: incremental against origin/dev when available,
# full run as fallback. Fails loud with install guidance when the binary is missing.
set -euo pipefail

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint not found — run: brew install golangci-lint" >&2
  exit 1
fi

cd apps/gooseforum

if git rev-parse --verify origin/dev >/dev/null 2>&1; then
  exec golangci-lint run --new-from-rev=origin/dev
fi

exec golangci-lint run
