#!/usr/bin/env bash
# backup-db.sh — 实例部署前数据库备份, 保留最近 KEEP 份(默认 7)。
# usage: backup-db.sh [instance]   (instance 默认 main)
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/srv/yourtj}"
INSTANCE="${1:-main}"
KEEP="${KEEP:-7}"

DB="$ROOT/$INSTANCE/storage/database/sqlite.db"
BACKUP_DIR="$ROOT/snapshots/$INSTANCE"

"$ROOT/scripts/snapshot-db.sh" "$DB" "$BACKUP_DIR/sqlite-$(date +%Y%m%d_%H%M%S).db"

# 清理旧备份, 保留 KEEP 份
if [ -d "$BACKUP_DIR" ]; then
  ls -1t "$BACKUP_DIR"/sqlite-*.db 2>/dev/null | tail -n +$((KEEP + 1)) | xargs -r rm -f
fi
echo "backup-db: $INSTANCE backups in $BACKUP_DIR (max $KEEP)"
