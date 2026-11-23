#!/usr/bin/env bash
# check-backup-health.sh — PostgreSQL WAL 归档、pgBackRest 与 restic 备份健康巡检。
#
# 检查项:
#   1. 宿主机磁盘使用率 <= MAX_DISK_USAGE_PCT (默认 85%)
#   2. PostgreSQL pg_wal 目录大小 <= MAX_PG_WAL_MB (默认 1024 MB)
#   3. archive_mode=on 且 pg_stat_archiver 无持续失败、最新归档时间 <= MAX_WAL_ARCHIVE_AGE_MIN (默认 45 分钟)
#   4. pgBackRest 最新有效备份距今 <= MAX_BACKUP_AGE_HOURS (默认 36 小时)
#   5. restic 最新快照距今 <= MAX_BACKUP_AGE_HOURS (可用 --skip-restic 跳过)
#
# usage: check-backup-health.sh [--skip-restic]
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
R2_ENV_FILE="${R2_ENV_FILE:-$ROOT/backup/r2.env}"
PG_CONTAINER="${PG_CONTAINER:-yourtj-postgres}"
SKIP_RESTIC=0

while [ $# -gt 0 ]; do
  case "$1" in
    --skip-restic) SKIP_RESTIC=1; shift ;;
    -h|--help) echo "usage: $0 [--skip-restic]"; exit 0 ;;
    *) echo "check-backup-health: unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [ -f "$R2_ENV_FILE" ]; then
  # shellcheck disable=SC1090
  source "$R2_ENV_FILE"
fi

PG_USER="${POSTGRES_USER:-}"
if [ -z "$PG_USER" ] && [ -f "$ROOT/.env" ]; then
  PG_USER="$(grep -E '^POSTGRES_USER=' "$ROOT/.env" | head -1 | cut -d= -f2- || true)"
fi
PG_USER="${PG_USER:-yourtj}"

MAX_WAL_ARCHIVE_AGE_MIN="${MAX_WAL_ARCHIVE_AGE_MIN:-45}"
MAX_BACKUP_AGE_HOURS="${MAX_BACKUP_AGE_HOURS:-36}"
MAX_DISK_USAGE_PCT="${MAX_DISK_USAGE_PCT:-85}"
MAX_PG_WAL_MB="${MAX_PG_WAL_MB:-1024}"
STANZA="${PGBACKREST_STANZA:-main}"

FAILURES=()
ok() { echo "[ok] $*"; }
fail() { echo "[FAIL] $*" >&2; FAILURES+=("$*"); }

send_webhook() {
  local text="$1"
  [ -n "${BACKUP_ALERT_WEBHOOK_URL:-}" ] || return 0
  command -v curl >/dev/null 2>&1 || return 0
  local payload
  payload="$(python3 - "$text" <<'PY'
import json, socket, sys
msg = sys.argv[1]
host = socket.gethostname()
print(json.dumps({
    "service": "yourtj-backup-health",
    "host": host,
    "level": "error",
    "message": msg,
    "content": f"[yourtj-backup-health][FAIL][{host}] {msg}",
    "text": f"[yourtj-backup-health][FAIL][{host}] {msg}",
}))
PY
)"
  curl -fsS --max-time 15 -X POST -H 'Content-Type: application/json' \
    -d "$payload" "$BACKUP_ALERT_WEBHOOK_URL" >/dev/null 2>&1 || true
}

# 1. Host disk usage
DISK_PCT="$(df -P "$ROOT" | awk 'NR==2 {gsub(/%/,"",$5); print $5}')"
if [ -n "$DISK_PCT" ] && [ "$DISK_PCT" -le "$MAX_DISK_USAGE_PCT" ]; then
  ok "disk usage on $ROOT is ${DISK_PCT}% (limit ${MAX_DISK_USAGE_PCT}%)"
else
  fail "disk usage on $ROOT is ${DISK_PCT:-unknown}% (limit ${MAX_DISK_USAGE_PCT}%)"
fi

# 2. pg_wal directory size
PG_WAL_KB="$(docker exec "$PG_CONTAINER" du -sk /var/lib/postgresql/data/pg_wal 2>/dev/null | awk '{print $1}' || true)"
if [ -n "$PG_WAL_KB" ]; then
  PG_WAL_MB="$((PG_WAL_KB / 1024))"
  if [ "$PG_WAL_MB" -le "$MAX_PG_WAL_MB" ]; then
    ok "pg_wal size is ${PG_WAL_MB} MB (limit ${MAX_PG_WAL_MB} MB)"
  else
    fail "pg_wal size is ${PG_WAL_MB} MB (exceeds limit ${MAX_PG_WAL_MB} MB; possible WAL archive stall)"
  fi
else
  fail "could not read pg_wal size from container $PG_CONTAINER"
fi

# 3. PostgreSQL archive_mode + pg_stat_archiver
if [ "${PGBACKREST_ENABLED:-true}" = "true" ]; then
  ARCHIVE_ROW="$(docker exec "$PG_CONTAINER" psql -U "$PG_USER" -d postgres -tAF '|' -c \
    "SELECT current_setting('archive_mode'), archived_count, failed_count, coalesce(extract(epoch from last_archived_time)::bigint, 0), coalesce(extract(epoch from last_failed_time)::bigint, 0), extract(epoch from now())::bigint FROM pg_stat_archiver;" 2>/dev/null || true)"
  if [ -z "$ARCHIVE_ROW" ]; then
    fail "failed to query pg_stat_archiver from $PG_CONTAINER"
  else
    IFS='|' read -r arch_mode arch_cnt fail_cnt last_ok_ep last_fail_ep now_ep <<< "$ARCHIVE_ROW"
    if [ "$arch_mode" != "on" ]; then
      fail "PostgreSQL archive_mode is '$arch_mode' (expected 'on')"
    elif [ "$arch_cnt" -le 0 ] || [ "$last_ok_ep" -le 0 ]; then
      fail "PostgreSQL has never archived a WAL segment (archived_count=$arch_cnt failed_count=$fail_cnt)"
    else
      age_min="$(( (now_ep - last_ok_ep) / 60 ))"
      if [ "$fail_cnt" -gt 0 ] && [ "$last_fail_ep" -ge "$last_ok_ep" ]; then
        fail "WAL archiver failing (failed_count=$fail_cnt, last_failed_epoch=$last_fail_ep >= last_archived_epoch=$last_ok_ep)"
      elif [ "$age_min" -gt "$MAX_WAL_ARCHIVE_AGE_MIN" ]; then
        fail "last archived WAL is ${age_min} min old (limit ${MAX_WAL_ARCHIVE_AGE_MIN} min)"
      else
        ok "WAL archiving healthy (archived=$arch_cnt, last=${age_min}m ago, failed=$fail_cnt)"
      fi
    fi
  fi

  # 4. pgBackRest backup freshness
  INFO_JSON="$(docker exec -u postgres "$PG_CONTAINER" pgbackrest --stanza="$STANZA" info --output=json 2>/dev/null || true)"
  if [ -z "$INFO_JSON" ]; then
    fail "pgbackrest info failed for stanza=$STANZA"
  else
    PGB_CHECK="$(INFO_JSON="$INFO_JSON" python3 - "$MAX_BACKUP_AGE_HOURS" <<'PY'
import json, os, sys, time
max_hours = int(sys.argv[1])
data = json.loads(os.environ["INFO_JSON"])
if not data:
    print("FAIL|empty pgbackrest info")
    sys.exit(0)
stanza = data[0]
status = stanza.get("status", {})
if status.get("code", 1) != 0:
    print(f"FAIL|stanza status code={status.get('code')} message={status.get('message')}")
    sys.exit(0)
backups = [b for b in stanza.get("backup", []) if not b.get("error", False)]
if not backups:
    print("FAIL|no valid pgbackrest backups found")
    sys.exit(0)
latest = max(backups, key=lambda b: b.get("timestamp", {}).get("stop", 0))
stop_ts = int(latest.get("timestamp", {}).get("stop", 0))
age_hours = (time.time() - stop_ts) / 3600.0
label = latest.get("label", "unknown")
btype = latest.get("type", "unknown")
if age_hours > max_hours:
    print(f"FAIL|latest pgbackrest backup {label} ({btype}) is {age_hours:.1f}h old (limit {max_hours}h)")
else:
    print(f"OK|latest pgbackrest backup {label} ({btype}) is {age_hours:.1f}h old (total={len(backups)})")
PY
)"
    if [[ "$PGB_CHECK" == OK\|* ]]; then
      ok "${PGB_CHECK#OK|}"
    else
      fail "${PGB_CHECK#FAIL|}"
    fi
  fi
fi

# 5. restic snapshot freshness
if [ "$SKIP_RESTIC" -eq 0 ] && [ -n "${R2_ACCOUNT_ID:-}" ] && [ -n "${R2_BUCKET:-}" ] && [ -n "${RESTIC_PASSWORD:-}" ]; then
  if command -v restic >/dev/null 2>&1; then
    export AWS_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID:-}"
    export AWS_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY:-}"
    export RESTIC_PASSWORD
    export RESTIC_REPOSITORY="s3:https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com/${R2_BUCKET}/${R2_RESTIC_PREFIX:-restic}"
    SNAP_JSON="$(restic snapshots --tag yourtj-main --latest 1 --json 2>/dev/null || true)"
    if [ -z "$SNAP_JSON" ]; then
      fail "restic snapshots query failed"
    else
      RESTIC_CHECK="$(SNAP_JSON="$SNAP_JSON" python3 - "$MAX_BACKUP_AGE_HOURS" <<'PY'
import datetime, json, os, sys
max_hours = int(sys.argv[1])
snaps = json.loads(os.environ["SNAP_JSON"])
if not snaps:
    print("FAIL|no restic snapshots found for tag yourtj-main")
    sys.exit(0)
latest = snaps[-1]
ts_raw = latest.get("time", "")
if ts_raw.endswith("Z"):
    ts_raw = ts_raw[:-1] + "+00:00"
dt = datetime.datetime.fromisoformat(ts_raw)
age_hours = (datetime.datetime.now(datetime.timezone.utc) - dt.astimezone(datetime.timezone.utc)).total_seconds() / 3600.0
sid = latest.get("short_id", latest.get("id", "unknown")[:8])
if age_hours > max_hours:
    print(f"FAIL|latest restic snapshot {sid} is {age_hours:.1f}h old (limit {max_hours}h)")
else:
    print(f"OK|latest restic snapshot {sid} is {age_hours:.1f}h old")
PY
)"
      if [[ "$RESTIC_CHECK" == OK\|* ]]; then
        ok "${RESTIC_CHECK#OK|}"
      else
        fail "${RESTIC_CHECK#FAIL|}"
      fi
    fi
  else
    fail "restic binary not installed"
  fi
fi

if [ "${#FAILURES[@]}" -gt 0 ]; then
  SUMMARY="$(printf '%s; ' "${FAILURES[@]}")"
  send_webhook "${SUMMARY%; }"
  exit 1
fi
exit 0
