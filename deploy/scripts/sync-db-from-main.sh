#!/usr/bin/env bash
# sync-db-from-main.sh — dev 实例从 main 实例同步数据库一致性快照。
# 单向同步: dev 上的测试数据不会回传 main。调用前应已停止 youtj-dev 服务。
# usage: sync-db-from-main.sh
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/srv/yourtj}"
MAIN_DB="$ROOT/main/storage/database/sqlite.db"
DEV_DB="$ROOT/dev/storage/database/sqlite.db"
SNAPSHOT_DIR="$ROOT/snapshots"

"$ROOT/scripts/snapshot-db.sh" "$MAIN_DB" "/tmp/main-snapshot.db"

if [ ! -f /tmp/main-snapshot.db ]; then
  echo "sync-db: main db missing, keeping dev db as-is"
  exit 0
fi

# 保留 dev 旧库一份(排查用)
if [ -f "$DEV_DB" ]; then
  mkdir -p "$SNAPSHOT_DIR/dev-prev"
  cp -f "$DEV_DB" "$SNAPSHOT_DIR/dev-prev/sqlite-$(date +%Y%m%d_%H%M%S).db"
fi

mkdir -p "$(dirname "$DEV_DB")"
install -m 0644 /tmp/main-snapshot.db "$DEV_DB"
rm -f /tmp/main-snapshot.db
echo "sync-db: dev db synced from main"
