#!/usr/bin/env bash
# backup-r2.sh — 生产主库 + 文件库 + 密钥的 Cloudflare R2 加密异地备份。
#
# 组成:
#   1. pgBackRest 物理基础备份(周全量 + 日差异, AES-256 加密 + LZ4 压缩, 配合连续 WAL 实现 PITR)
#   2. 可选逻辑兜底 pg_dump (-Fc) + pg_dumpall --globals-only(写入 0700 临时区并经 pg_restore -l 校验)
#   3. sqlite3 .backup 一致性快照 file.db + 不可再生密钥/配置(OIDC/APNs/agent-state/config/OpenResty)
#      经 restic AES-256 去重加密推送到 Cloudflare R2
#
# usage: backup-r2.sh [--pg-type auto|full|diff|incr] [--skip-pgbackrest] [--skip-restic]
set -euo pipefail
umask 077

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
R2_ENV_FILE="${R2_ENV_FILE:-$ROOT/backup/r2.env}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PG_CONTAINER="${PG_CONTAINER:-yourtj-postgres}"
PG_TYPE="auto"
SKIP_PGBACKREST=0
SKIP_RESTIC=0

while [ $# -gt 0 ]; do
  case "$1" in
    --pg-type)
      PG_TYPE="${2:?--pg-type requires auto|full|diff|incr}"
      shift 2
      ;;
    --skip-pgbackrest)
      SKIP_PGBACKREST=1
      shift
      ;;
    --skip-restic)
      SKIP_RESTIC=1
      shift
      ;;
    -h|--help)
      echo "usage: $0 [--pg-type auto|full|diff|incr] [--skip-pgbackrest] [--skip-restic]"
      exit 0
      ;;
    *)
      echo "backup-r2: unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

case "$PG_TYPE" in
  auto|full|diff|incr) ;;
  *) echo "backup-r2: invalid --pg-type '$PG_TYPE' (expect auto|full|diff|incr)" >&2; exit 2 ;;
esac

log() { printf '[backup-r2 %s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

# shellcheck source=pgdsn.sh
source "$SCRIPT_DIR/pgdsn.sh"

send_webhook() {
  local level="$1" text="$2"
  [ -n "${BACKUP_ALERT_WEBHOOK_URL:-}" ] || return 0
  command -v curl >/dev/null 2>&1 || return 0
  local payload
  payload="$(python3 - "$level" "$text" <<'PY'
import json, socket, sys
level, msg = sys.argv[1], sys.argv[2]
host = socket.gethostname()
body = {
    "service": "yourtj-backup-r2",
    "host": host,
    "level": level,
    "message": msg,
    "content": f"[yourtj-backup-r2][{level}][{host}] {msg}",
    "text": f"[yourtj-backup-r2][{level}][{host}] {msg}",
}
print(json.dumps(body))
PY
)"
  curl -fsS --max-time 15 -X POST -H 'Content-Type: application/json' \
    -d "$payload" "$BACKUP_ALERT_WEBHOOK_URL" >/dev/null 2>&1 || true
}

[ -f "$R2_ENV_FILE" ] || { echo "backup-r2: missing $R2_ENV_FILE" >&2; exit 1; }
# shellcheck disable=SC1090
source "$R2_ENV_FILE"

require_secret() {
  local name="$1" val="${!1:-}"
  if [ -z "$val" ] || [[ "$val" == REPLACE_* ]]; then
    echo "backup-r2: required setting $name is unset or placeholder in $R2_ENV_FILE" >&2
    exit 1
  fi
}

require_secret R2_ACCOUNT_ID
require_secret R2_BUCKET
require_secret R2_ACCESS_KEY_ID
require_secret R2_SECRET_ACCESS_KEY
if [ "$SKIP_RESTIC" -eq 0 ]; then
  require_secret RESTIC_PASSWORD
fi

LOCK_DIR="${BACKUP_LOCK_DIR:-$ROOT/backup/.backup-r2.lock}"
mkdir -p "$(dirname "$LOCK_DIR")"
if ! mkdir "$LOCK_DIR" 2>/dev/null; then
  echo "backup-r2: another backup run holds lock $LOCK_DIR" >&2
  exit 1
fi

STAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/yourtj-r2-stage.XXXXXX")"
chmod 0700 "$STAGE_DIR"

on_exit() {
  local rc=$?
  rm -rf "$STAGE_DIR"
  rmdir "$LOCK_DIR" 2>/dev/null || true
  if [ "$rc" -ne 0 ]; then
    log "FAILED (exit=$rc)"
    send_webhook "error" "backup-r2 failed with exit code $rc"
  fi
  exit "$rc"
}
trap on_exit EXIT

PG_USER="${POSTGRES_USER:-}"
if [ -z "$PG_USER" ] && [ -f "$ROOT/.env" ]; then
  PG_USER="$(grep -E '^POSTGRES_USER=' "$ROOT/.env" | head -1 | cut -d= -f2- || true)"
fi
PG_USER="${PG_USER:-yourtj}"

MAIN_CFG="$ROOT/main/config.toml"
PG_DB=""
if [ -f "$MAIN_CFG" ] && grep -Eq '^\s*connection\s*=\s*"postgres"' "$MAIN_CFG"; then
  PG_URL="$(pg_toml_url "$MAIN_CFG")"
  [ -n "$PG_URL" ] || { echo "backup-r2: $MAIN_CFG missing [db.default].url" >&2; exit 1; }
  PG_DB="$(pg_dsn_dbname "$PG_URL")"
fi
PG_DB="${PG_DB:-yourtj_main}"

# 1. PostgreSQL physical backup via pgBackRest (pairs with continuous WAL archiving for PITR)
if [ "$SKIP_PGBACKREST" -eq 0 ] && [ "${PGBACKREST_ENABLED:-true}" = "true" ]; then
  STANZA="${PGBACKREST_STANZA:-main}"
  log "running pgBackRest check (stanza=$STANZA)"
  docker exec -u postgres "$PG_CONTAINER" pgbackrest --stanza="$STANZA" check

  RESOLVED_PG_TYPE="$PG_TYPE"
  if [ "$RESOLVED_PG_TYPE" = "auto" ]; then
    FULL_DOW="${PG_FULL_BACKUP_DOW:-7}"
    TODAY_DOW="$(date +%u)"
    HAS_BACKUP="$(docker exec -u postgres "$PG_CONTAINER" pgbackrest --stanza="$STANZA" info --output=json \
      | python3 -c 'import json,sys; data=json.load(sys.stdin); print("1" if data and data[0].get("backup") else "0")')"
    if [ "$HAS_BACKUP" = "0" ] || [ "$TODAY_DOW" = "$FULL_DOW" ]; then
      RESOLVED_PG_TYPE="full"
    else
      RESOLVED_PG_TYPE="diff"
    fi
  fi

  log "running pgBackRest backup (stanza=$STANZA type=$RESOLVED_PG_TYPE)"
  docker exec -u postgres "$PG_CONTAINER" pgbackrest --stanza="$STANZA" backup --type="$RESOLVED_PG_TYPE"
  log "pgBackRest backup succeeded (type=$RESOLVED_PG_TYPE)"
else
  log "pgBackRest step skipped"
fi

# 2. Optional logical PG dump (-Fc) + globals into private staging dir (verified before upload)
if [ "$SKIP_RESTIC" -eq 0 ] && [ "${LOGICAL_PG_BACKUP:-true}" = "true" ]; then
  mkdir -p "$STAGE_DIR/postgres"
  FEED_PRIVACY="$SCRIPT_DIR/feed-privacy.py"
  [ -f "$FEED_PRIVACY" ] || { echo "backup-r2: $FEED_PRIVACY not found" >&2; exit 1; }
  FEED_FLAGS_RAW="$(python3 "$FEED_PRIVACY" --pg-flags)" || { echo "backup-r2: feed-privacy.py failed" >&2; exit 1; }
  [ -n "$FEED_FLAGS_RAW" ] || { echo "backup-r2: feed-privacy.py returned empty flags" >&2; exit 1; }
  FEED_PG_FLAGS=()
  while IFS= read -r flag; do
    [ -n "$flag" ] && FEED_PG_FLAGS+=("$flag")
  done <<< "$FEED_FLAGS_RAW"
  DUMP_OUT="$STAGE_DIR/postgres/${PG_DB}.dump"
  GLOBALS_OUT="$STAGE_DIR/postgres/globals.sql"
  log "taking logical pg_dump ($PG_DB, custom compressed format)"
  docker exec "$PG_CONTAINER" pg_dump --format=custom --compress=6 "${FEED_PG_FLAGS[@]}" -U "$PG_USER" -d "$PG_DB" > "$DUMP_OUT"
  chmod 0600 "$DUMP_OUT"
  [ -s "$DUMP_OUT" ] || { echo "backup-r2: logical pg_dump produced empty file" >&2; exit 1; }
  docker exec -i "$PG_CONTAINER" pg_restore --list < "$DUMP_OUT" >/dev/null
  docker exec "$PG_CONTAINER" pg_dumpall --globals-only -U "$PG_USER" > "$GLOBALS_OUT"
  chmod 0600 "$GLOBALS_OUT"
  log "logical pg_dump verified ($(wc -c < "$DUMP_OUT") bytes)"
fi

# 3. Consistent SQLite file.db snapshot + non-reproducible secrets/configs -> restic on R2
if [ "$SKIP_RESTIC" -eq 0 ]; then
  command -v restic >/dev/null 2>&1 || { echo "backup-r2: restic not installed" >&2; exit 1; }

  MAIN_FILE_DB="$ROOT/main/storage/database/file.db"
  if [ -f "$MAIN_FILE_DB" ]; then
    mkdir -p "$STAGE_DIR/main-storage/database"
    log "taking consistent sqlite3 .backup of main file.db"
    sqlite3 "$MAIN_FILE_DB" ".backup '$STAGE_DIR/main-storage/database/file.db'"
    chmod 0600 "$STAGE_DIR/main-storage/database/file.db"
  fi

  copy_if_exists() {
    local src="$1" rel="$2"
    [ -e "$src" ] || return 0
    local dst="$STAGE_DIR/$rel"
    mkdir -p "$(dirname "$dst")"
    cp -a "$src" "$dst"
  }

  copy_if_exists "$ROOT/main/storage/oidc" "main-storage/oidc"
  copy_if_exists "$ROOT/main/storage/push" "main-storage/push"
  copy_if_exists "$ROOT/main/storage/agent-state" "main-storage/agent-state"
  copy_if_exists "$ROOT/main/config.toml" "config/main-config.toml"
  copy_if_exists "$ROOT/.env" "config/dot-env"
  copy_if_exists "$ROOT/docker-compose.yaml" "config/docker-compose.yaml"

  # Optional 1Panel OpenResty configs & TLS certs (if present on host)
  OPENRESTY_ROOT="${OPENRESTY_BACKUP_DIR:-/opt/1panel/apps/openresty/openresty}"
  if [ -d "$OPENRESTY_ROOT/conf" ]; then
    copy_if_exists "$OPENRESTY_ROOT/conf" "openresty/conf"
  fi
  if [ -d "$OPENRESTY_ROOT/www/sites" ]; then
    # Exclude access/error logs to avoid bloating backups
    mkdir -p "$STAGE_DIR/openresty/sites"
    tar -C "$OPENRESTY_ROOT/www/sites" --exclude='*/log' --exclude='*/log/*' -cf - . \
      | tar -C "$STAGE_DIR/openresty/sites" -xf -
  fi

  export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
  export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
  export RESTIC_PASSWORD
  export RESTIC_REPOSITORY="s3:https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com/${R2_BUCKET}/${R2_RESTIC_PREFIX:-restic}"

  log "uploading encrypted deduplicated snapshot via restic"
  restic backup "$STAGE_DIR" \
    --host "${RESTIC_HOST_TAG:-yourtj-prod}" \
    --tag yourtj-main \
    --exclude-caches

  if [ "${RESTIC_SKIP_PRUNE:-false}" != "true" ]; then
    log "applying restic retention policy (daily=${RESTIC_KEEP_DAILY:-14} weekly=${RESTIC_KEEP_WEEKLY:-8} monthly=${RESTIC_KEEP_MONTHLY:-6})"
    restic forget \
      --host "${RESTIC_HOST_TAG:-yourtj-prod}" \
      --tag yourtj-main \
      --keep-daily "${RESTIC_KEEP_DAILY:-14}" \
      --keep-weekly "${RESTIC_KEEP_WEEKLY:-8}" \
      --keep-monthly "${RESTIC_KEEP_MONTHLY:-6}" \
      --prune
  else
    log "RESTIC_SKIP_PRUNE=true: skipping forget/prune (retention managed externally)"
  fi
fi

log "backup-r2 completed successfully"
if [ "${BACKUP_NOTIFY_ON_SUCCESS:-false}" = "true" ]; then
  send_webhook "info" "backup-r2 completed successfully"
fi
