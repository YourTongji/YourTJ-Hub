#!/usr/bin/env bash
# backup-db.sh — 实例部署前数据库一致性备份, 保留最近 KEEP 份(默认 7)。
# usage: backup-db.sh [instance]
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
INSTANCE="${1:-main}"
KEEP="${KEEP:-7}"
DB="$ROOT/$INSTANCE/storage/database/sqlite.db"
BACKUP_DIR="$ROOT/snapshots/$INSTANCE"

if [ ! -f "$DB" ]; then
  echo "backup-db: $DB not found, skip"
  exit 0
fi

mkdir -p "$BACKUP_DIR"
TMP="/tmp/backup-$$.db"
sqlite3 "$DB" ".backup '$TMP'" || { echo "backup-db: snapshot failed" >&2; rm -f "$TMP"; exit 1; }
mv -f "$TMP" "$BACKUP_DIR/sqlite-$(date +%Y%m%d_%H%M%S).db"

# 清理旧备份, 保留 KEEP 份
ls -1t "$BACKUP_DIR"/sqlite-*.db 2>/dev/null | tail -n +$((KEEP + 1)) | xargs -r rm -f
echo "backup-db: $INSTANCE backup done (keep $KEEP)"
