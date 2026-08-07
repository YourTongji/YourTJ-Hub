#!/usr/bin/env bash
# backup-db.sh — 实例部署前数据库一致性备份, 保留最近 KEEP 份。
# 主库模式自动检测: SQLite(.backup) 或 PostgreSQL(pg_dump)。
# usage: backup-db.sh [instance]
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
INSTANCE="${1:-main}"
KEEP="${KEEP:-7}"
DB_DIR="$ROOT/$INSTANCE/storage/database"
BACKUP_DIR="$ROOT/snapshots/$INSTANCE"
TS="$(date +%Y%m%d_%H%M%S)"

mkdir -p "$BACKUP_DIR"

# 检测主库模式: 读 config.toml [db.default] connection
db_mode() {
  local cfg="$ROOT/$INSTANCE/config.toml"
  [ -f "$cfg" ] || { echo sqlite; return; }
  if grep -E '^\s*connection\s*=\s*"postgres"' "$cfg" >/dev/null 2>&1; then
    echo postgres
  else
    echo sqlite
  fi
}

pg_dbname() {
  grep -E '^\s*url\s*=' "$ROOT/$INSTANCE/config.toml" | grep -oE 'dbname=[^ ]+' | cut -d= -f2 || true
}

if [ "$(db_mode)" = "postgres" ]; then
  PG_DB="$(pg_dbname)"
  if [ -z "$PG_DB" ]; then
    echo "backup-db: cannot parse dbname from $ROOT/$INSTANCE/config.toml" >&2
    exit 1
  fi
  TMP="/tmp/backup-${PG_DB}-$$.sql"
  if docker exec yourtj-postgres pg_dump -U yourtj -d "$PG_DB" > "$TMP"; then
    mv -f "$TMP" "$BACKUP_DIR/pg-${PG_DB}-${TS}.sql"
    echo "backup-db: $INSTANCE pg dump ($PG_DB) backed up"
  else
    echo "backup-db: pg_dump failed for $PG_DB" >&2
    rm -f "$TMP"
    exit 1
  fi
  # 清理旧 PG 备份(每库保留 KEEP 份)
  ls -1t "$BACKUP_DIR"/pg-${PG_DB}-*.sql 2>/dev/null | tail -n +$((KEEP + 1)) | xargs -r rm -f
  echo "backup-db: $INSTANCE pg backups done (keep $KEEP)"
  exit 0
fi

if [ ! -d "$DB_DIR" ]; then
  echo "backup-db: $DB_DIR not found, skip"
  exit 0
fi

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
