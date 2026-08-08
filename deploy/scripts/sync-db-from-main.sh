#!/usr/bin/env bash
# sync-db-from-main.sh — dev 实例从 main 同步一致性数据快照(单向)。
# 主库模式自动检测: SQLite(.backup) 或 PostgreSQL(pg_dump|psql, 重建 dev 库)。
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

db_mode() {
  local cfg="$1"
  [ -f "$cfg" ] || { echo sqlite; return; }
  if grep -E '^\s*connection\s*=\s*"postgres"' "$cfg" >/dev/null 2>&1; then
    echo postgres
  else
    echo sqlite
  fi
}

pg_dbname() {
  local cfg="$1"
  grep -E '^\s*url\s*=' "$cfg" | grep -oE 'dbname=[^ ]+' | cut -d= -f2 || true
}

MAIN_MODE="$(db_mode "$ROOT/main/config.toml")"
DEV_MODE="$(db_mode "$ROOT/dev/config.toml")"

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

if [ "$MAIN_MODE" = "postgres" ] && [ "$DEV_MODE" = "postgres" ]; then
  MAIN_PG="$(pg_dbname "$ROOT/main/config.toml")"
  DEV_PG="$(pg_dbname "$ROOT/dev/config.toml")"
  if [ -z "$MAIN_PG" ] || [ -z "$DEV_PG" ]; then
    echo "sync-db: cannot parse PG dbnames from configs" >&2
    exit 1
  fi
  echo "sync-db: PG mode ($MAIN_PG -> $DEV_PG)"
  if [ -f "$DEV_FILE_DB" ]; then
    mkdir -p "$ROOT/snapshots/dev-prev"
    cp -f "$DEV_FILE_DB" "$ROOT/snapshots/dev-prev/file-$(date +%Y%m%d_%H%M%S).db"
  fi
  docker exec yourtj-postgres psql -U yourtj -d postgres \
    -c "DROP DATABASE IF EXISTS \"$DEV_PG\";" -c "CREATE DATABASE \"$DEV_PG\";" >/dev/null
  docker exec yourtj-postgres sh -c "pg_dump -U yourtj -d \"$MAIN_PG\" | psql -U yourtj -d \"$DEV_PG\"" >/dev/null
  echo "sync-db: dev PG db synced from main"
  sync_one "$MAIN_FILE_DB" "$DEV_FILE_DB" "file"
  chown -R 1000:1000 "$ROOT/dev/storage" 2>/dev/null || true
  echo "sync-db: dev file db synced from main"
  exit 0
fi

# 保留 dev 旧库一份(排查用)
if [ -f "$DEV_DB" ]; then
  mkdir -p "$ROOT/snapshots/dev-prev"
  cp -f "$DEV_DB" "$ROOT/snapshots/dev-prev/sqlite-$(date +%Y%m%d_%H%M%S).db"
  [ -f "$DEV_FILE_DB" ] && cp -f "$DEV_FILE_DB" "$ROOT/snapshots/dev-prev/file-$(date +%Y%m%d_%H%M%S).db"
fi

sync_one "$MAIN_DB" "$DEV_DB" "sqlite"
sync_one "$MAIN_FILE_DB" "$DEV_FILE_DB" "file"

# 容器内 uid 1000 需要可写 storage
chown -R 1000:1000 "$ROOT/dev/storage" 2>/dev/null || true
echo "sync-db: dev dbs synced from main"
