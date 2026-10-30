#!/usr/bin/env bash
# Execute the deploy artifact as the runtime's unprivileged user, without host
# libraries, configuration, writes or networking. Optional platform is for local
# cross-architecture reproduction; CI uses its native Linux architecture.
set -euo pipefail
binary="${1:?usage: check-runtime-binary.sh binary [platform]}"
binary="$(cd "$(dirname "$binary")" && pwd)/$(basename "$binary")"
test -x "$binary"
if [ -n "${2:-}" ]; then set -- --platform "$2"; else set --; fi
docker run --rm "$@" --network none --read-only --user 1000:1000 \
  --mount "type=bind,src=$binary,dst=/app/yourtj-hub,readonly" \
  --entrypoint /app/yourtj-hub alpine:3.24 --help >/dev/null
echo 'Alpine runtime: executable starts without host libraries'
