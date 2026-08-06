#!/usr/bin/env bash
# backup-db.sh — 实例部署前数据库一致性备份(主库 + 文件库), 保留最近 KEEP 份。
# usage: backup-db.sh [instance]
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
INSTANCE="${1:-main}"
KEEP="${KEEP:-7}"
DB_DIR="$ROOT/$INSTANCE/storage/database"
BACKUP_DIR="$ROOT/snapshots/$INSTANCE"
TS="$(date +%Y%m%d_%H%M%S)"

if [ ! -d "$DB_DIR" ]; then
  echo "backup-db: $DB_DIR not found, skip"
  exit 0
fi

mkdir -p "$BACKUP_DIR"

backup_one() {
  local db="$1" label="$2"
  [ -f "$db" ] || { echo "backup-db: $db not found, skip"; return 0; }
  TMP="/tmp/backup-${label}-$$.db"
  sqlite3 "$db" ".backup '$TMP'" || { echo "backup-db: $label snapshot failed" >&2; rm -f "$TMP"; exit 1; }
  mv -f "$TMP" "$BACKUP_DIR/${label}-${TS}.db"
  echo "backup-db: $label backed up"
}

backup_one "$DB_DIR/sqlite.db" "sqlite"
backup_one "$DB_DIR/file.db" "file"

# 清理旧备份(任一标签), 每个标签保留 KEEP 份
for label in sqlite file; do
  ls -1t "$BACKUP_DIR"/${label}-*.db 2>/dev/null | tail -n +$((KEEP + 1)) | xargs -r rm -f
done
echo "backup-db: $INSTANCE backups done (keep $KEEP per label)"
