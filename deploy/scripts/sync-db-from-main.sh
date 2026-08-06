#!/usr/bin/env bash
# sync-db-from-main.sh — dev 实例从 main 同步 SQLite 一致性快照(单向)。
# 先停止 dev 容器避免文件句柄冲突, 替换后由 deploy.sh 重新拉起。
# usage: sync-db-from-main.sh
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
MAIN_DB="$ROOT/main/storage/database/sqlite.db"
DEV_DB="$ROOT/dev/storage/database/sqlite.db"
ENV_FILE="$ROOT/.env"
COMPOSE_FILE="$ROOT/docker-compose.yaml"

# 停 dev 容器, 避免覆盖已打开数据库文件导致数据写丢
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" stop dev >/dev/null 2>&1 || true

if [ ! -f "$MAIN_DB" ]; then
  echo "sync-db: main db not found, skipping sync"
  exit 0
fi

TMP="/tmp/main-snapshot-$$.db"
sqlite3 "$MAIN_DB" ".backup '$TMP'" || { echo "sync-db: snapshot failed" >&2; rm -f "$TMP"; exit 1; }

# 保留 dev 旧库一份(排查用)
if [ -f "$DEV_DB" ]; then
  mkdir -p "$ROOT/snapshots/dev-prev"
  cp -f "$DEV_DB" "$ROOT/snapshots/dev-prev/sqlite-$(date +%Y%m%d_%H%M%S).db"
fi

mkdir -p "$(dirname "$DEV_DB")"
install -m 0644 "$TMP" "$DEV_DB"
rm -f "$TMP"
# 容器内 uid 1000 需要可写 storage
chown -R 1000:1000 "$ROOT/dev/storage" 2>/dev/null || true
echo "sync-db: dev db synced from main"
