#!/usr/bin/env bash
# install-r2-backup.sh — 在宿主机幂等初始化 Cloudflare R2 备份环境与 systemd 定时器。
#
# 功能:
#   1. 安装 restic（若缺失）
#   2. 创建 /opt/yourtj/backup（0700）与 /opt/yourtj/backup/pgbackrest（0750）
#   3. 若 /opt/yourtj/backup/r2.env 不存在，复制模板（0600）并提示填写
#   4. 若 r2.env 已填写，自动渲染 /opt/yourtj/backup/pgbackrest/pgbackrest.conf（0600，属主匹配 PG 容器 postgres UID:GID）
#   5. 安装 systemd units（yourtj-backup-r2.timer + yourtj-backup-health.timer）
#
# usage: sudo bash install-r2-backup.sh [--enable-timers]
set -euo pipefail
umask 077

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$DEPLOY_DIR/.." && pwd)"
PG_CONTAINER="${PG_CONTAINER:-yourtj-postgres}"
ENABLE_TIMERS=0

while [ $# -gt 0 ]; do
  case "$1" in
    --enable-timers) ENABLE_TIMERS=1; shift ;;
    -h|--help) echo "usage: $0 [--enable-timers]"; exit 0 ;;
    *) echo "install-r2-backup: unknown argument: $1" >&2; exit 2 ;;
  esac
done

# 1. Install restic if missing
if ! command -v restic >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq && apt-get install -y -qq restic
  else
    echo "install-r2-backup: please install restic manually" >&2
    exit 1
  fi
fi

# 2. Backup directories & scripts
mkdir -p "$ROOT/backup/pgbackrest" "$ROOT/scripts"
chmod 0700 "$ROOT/backup"
chmod 0750 "$ROOT/backup/pgbackrest"

for f in backup-r2.sh check-backup-health.sh verify-pitr-restore.sh backup-db.sh pgdsn.sh feed-privacy.py; do
  if [ -f "$SCRIPT_DIR/$f" ] && [ "$(realpath "$SCRIPT_DIR/$f")" != "$(realpath -m "$ROOT/scripts/$f")" ]; then
    install -m 0755 "$SCRIPT_DIR/$f" "$ROOT/scripts/$f"
  fi
done
if [ -f "$SCRIPT_DIR/feed-raw-tables.json" ] && [ "$(realpath "$SCRIPT_DIR/feed-raw-tables.json")" != "$(realpath -m "$ROOT/scripts/feed-raw-tables.json")" ]; then
  install -m 0644 "$SCRIPT_DIR/feed-raw-tables.json" "$ROOT/scripts/feed-raw-tables.json"
fi

# 3. Seed r2.env template if absent
R2_ENV="$ROOT/backup/r2.env"
if [ ! -f "$R2_ENV" ]; then
  install -m 0600 "$DEPLOY_DIR/backup/r2.env.example" "$R2_ENV"
  echo "install-r2-backup: created $R2_ENV (mode 0600). Fill in R2 credentials and passphrases, then re-run."
fi
chmod 0600 "$R2_ENV"

# 4. Install systemd units if systemd directory exists
SYSTEMD_SRC="$DEPLOY_DIR/systemd"
if [ -d /etc/systemd/system ] && [ -d "$SYSTEMD_SRC" ]; then
  for unit in yourtj-backup-r2.service yourtj-backup-r2.timer yourtj-backup-health.service yourtj-backup-health.timer; do
    install -m 0644 "$SYSTEMD_SRC/$unit" "/etc/systemd/system/$unit"
  done
  if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    if [ "$ENABLE_TIMERS" -eq 1 ]; then
      systemctl enable --now yourtj-backup-r2.timer yourtj-backup-health.timer
      echo "install-r2-backup: enabled yourtj-backup-r2.timer and yourtj-backup-health.timer"
    fi
  fi
fi

# 5. Render pgbackrest.conf when r2.env has no remaining REPLACE_* placeholders
if grep -q 'REPLACE_' "$R2_ENV"; then
  echo "install-r2-backup: $R2_ENV still contains REPLACE_* placeholders; skipping pgbackrest.conf render."
  echo "Next steps:"
  echo "  1. Edit $R2_ENV with dedicated Cloudflare R2 backup credentials & encryption passphrases."
  echo "  2. Build PG backup image: docker build -t yourtj-postgres-backup:16-alpine $REPO_ROOT/services/postgres-backup"
  echo "  3. Re-run: sudo bash $0 --enable-timers"
  exit 0
fi

# shellcheck disable=SC1090
source "$R2_ENV"

PG_USER="${POSTGRES_USER:-}"
if [ -z "$PG_USER" ] && [ -f "$ROOT/.env" ]; then
  PG_USER="$(grep -E '^POSTGRES_USER=' "$ROOT/.env" | head -1 | cut -d= -f2- || true)"
fi
PG_USER="${PG_USER:-yourtj}"

PGB_CONF="$ROOT/backup/pgbackrest/pgbackrest.conf"
cat > "$PGB_CONF" <<EOF
[global]
repo1-type=s3
repo1-s3-bucket=${R2_BUCKET}
repo1-s3-endpoint=${R2_ACCOUNT_ID}.r2.cloudflarestorage.com
repo1-s3-region=auto
repo1-s3-uri-style=path
repo1-s3-key=${R2_ACCESS_KEY_ID}
repo1-s3-key-secret=${R2_SECRET_ACCESS_KEY}
repo1-path=${R2_PGBACKREST_PATH:-/pgbackrest}
repo1-cipher-type=aes-256-cbc
repo1-cipher-pass=${PGBACKREST_CIPHER_PASS}
repo1-bundle=y
repo1-block=y
compress-type=lz4
process-max=1
archive-async=n
start-fast=y
repo1-retention-full=${PGBACKREST_RETENTION_FULL:-2}
repo1-retention-diff=${PGBACKREST_RETENTION_DIFF:-7}
repo1-retention-full-type=count
log-level-console=warn
log-level-file=info
log-path=/var/log/pgbackrest

[${PGBACKREST_STANZA:-main}]
pg1-path=/var/lib/postgresql/data
pg1-port=5432
pg1-user=${PG_USER}
EOF

PG_UID="70"
PG_GID="70"
if command -v docker >/dev/null 2>&1 && docker inspect "$PG_CONTAINER" >/dev/null 2>&1; then
  PG_UID="$(docker exec "$PG_CONTAINER" id -u postgres 2>/dev/null || echo 70)"
  PG_GID="$(docker exec "$PG_CONTAINER" id -g postgres 2>/dev/null || echo 70)"
fi
chown "$PG_UID:$PG_GID" "$ROOT/backup/pgbackrest" "$PGB_CONF"
chmod 0750 "$ROOT/backup/pgbackrest"
chmod 0600 "$PGB_CONF"

echo "install-r2-backup: rendered $PGB_CONF (owner ${PG_UID}:${PG_GID}, mode 0600)"
echo "Next activation steps (see docs/operations/backup-and-recovery.md):"
echo "  1. Set POSTGRES_IMAGE=yourtj-postgres-backup:16-alpine, PG_ARCHIVE_MODE=on,"
echo "     PG_ARCHIVE_COMMAND=/usr/local/bin/pgbackrest-archive-push.sh %p in $ROOT/.env"
echo "  2. docker compose --env-file $ROOT/.env -f $ROOT/docker-compose.yaml up -d postgres"
echo "  3. docker exec -u postgres $PG_CONTAINER pgbackrest --stanza=${PGBACKREST_STANZA:-main} stanza-create"
echo "  4. docker exec -u postgres $PG_CONTAINER pgbackrest --stanza=${PGBACKREST_STANZA:-main} check"
echo "  5. Initialize restic repo once, then run: $ROOT/scripts/backup-r2.sh --pg-type full"
