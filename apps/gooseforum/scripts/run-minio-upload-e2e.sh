#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
work=$(mktemp -d)
server_pid=
cleanup() {
  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Official RELEASE.2025-10-15T17-29-55Z, including its security fix.
# https://github.com/minio/minio/releases/tag/RELEASE.2025-10-15T17-29-55Z
version=v0.0.0-20251015172955-9e49d5e7a648
go mod download -json "github.com/minio/minio@$version" > "$work/module.json"
python3 - "$work/module.json" <<'PY'
import json
import sys
with open(sys.argv[1]) as file:
    module = json.load(file)
expected = {
    'Path': 'github.com/minio/minio',
    'Version': 'v0.0.0-20251015172955-9e49d5e7a648',
    'Sum': 'h1:6TdolSCLSs2nwm8i0PpWDqf9iX2Ty9WQK8wmr7dCnUM=',
}
if any(module.get(key) != value for key, value in expected.items()):
    raise SystemExit('MinIO module provenance/checksum mismatch')
print('Verified official MinIO module:', module['Version'], module['Sum'])
PY
GOBIN="$work/bin" go install "github.com/minio/minio@$version"
go version -m "$work/bin/minio"

# Synthetic, short-lived credentials are used only by this loopback instance.
export MINIO_ROOT_USER=yourtj-e2e
export MINIO_ROOT_PASSWORD
MINIO_ROOT_PASSWORD=$(openssl rand -hex 20)
export MINIO_BROWSER=off MINIO_UPDATE=off
"$work/bin/minio" server --address 127.0.0.1:19000 --console-address 127.0.0.1:19001 "$work/data" > "$work/minio.log" 2>&1 &
server_pid=$!
ready=false
for _ in $(seq 1 60); do
  if ! kill -0 "$server_pid" 2>/dev/null; then
    echo 'MinIO exited before readiness' >&2
    cat "$work/minio.log" >&2
    exit 1
  fi
  if curl --noproxy '*' --silent --fail --max-time 1 --connect-timeout 1 http://127.0.0.1:19000/minio/health/ready > /dev/null; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  echo 'MinIO did not become ready' >&2
  exit 1
fi
cd "$root/apps/gooseforum"
YOURTJ_MINIO_E2E=1 go test -tags nodynamic,miniointegration ./app/http/routes -run '^TestDirectUploadMinIOReplay$' -count=1 -v -timeout=3m
