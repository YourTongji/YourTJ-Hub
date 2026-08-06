#!/usr/bin/env bash
# sync-db-from-main.sh — dev 实例从 main 同步 SQLite 一致性快照(单向)。
# 同步主库(sqlite.db)+ 文件库(file.db, 媒体 BLOB), 保证 dev 有完整媒体数据。
# 先停止 dev 容器避免文件句柄冲突, 替换后由 deploy.sh 重新拉起。
# usage: sync-db-from-main.sh
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
MAIN_DB="$ROOT/main/storage/database/sqlite.db"
MAIN_FILE_DB="$ROOT/main/storage/database/file.db"
DEV_DB="$ROOT/dev/storage/database/sqlite.db"
DEV_FILE_DB="$ROOT/dev/storage/database/file.db"
ENV_FILE="$ROOT/.env"
COMPOSE_FILE="$ROOT/docker-compose.yaml"

# 停 dev 容器, 避免覆盖已打开数据库文件导致数据写丢
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" stop dev >/dev/null 2>&1 || true

# 保留 dev 旧库一份(排查用)
if [ -f "$DEV_DB" ]; then
  mkdir -p "$ROOT/snapshots/dev-prev"
  cp -f "$DEV_DB" "$ROOT/snapshots/dev-prev/sqlite-$(date +%Y%m%d_%H%M%S).db"
  [ -f "$DEV_FILE_DB" ] && cp -f "$DEV_FILE_DB" "$ROOT/snapshots/dev-prev/file-$(date +%Y%m%d_%H%M%S).db"
fi

sync_one() {
  local src="$1" dst="$2" label="$3"
  if [ ! -f "$src" ]; then
    echo "sync-db: $label source not found, skip"
    return 0
  fi
  TMP="/tmp/${label}-snapshot-$$.db"
  sqlite3 "$src" ".backup '$TMP'" || { echo "sync-db: $label snapshot failed" >&2; rm -f "$TMP"; exit 1; }
  mkdir -p "$(dirname "$dst")"
  install -m 0644 "$TMP" "$dst"
  rm -f "$TMP"
  echo "sync-db: $label synced from main"
}

sync_one "$MAIN_DB" "$DEV_DB" "sqlite"
sync_one "$MAIN_FILE_DB" "$DEV_FILE_DB" "file"

# 容器内 uid 1000 需要可写 storage
chown -R 1000:1000 "$ROOT/dev/storage" 2>/dev/null || true
echo "sync-db: dev dbs synced from main"
