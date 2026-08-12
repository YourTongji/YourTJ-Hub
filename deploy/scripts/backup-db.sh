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

# 共享 PostgreSQL DSN 解析库(支持 postgres:// URL 与 key=value 两种格式, issue #134)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pgdsn.sh
source "$SCRIPT_DIR/pgdsn.sh"

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

# 从 config.toml 提取 [db.default] url 值, 经 pg_dsn_dbname 解析数据库名。
# URL 与 key=value 两种格式均支持; 解析失败输出错误并返回非零(调用方须在
# 任何 DB 操作之前检查, 避免"先操作再报错"留下中间状态)。
pg_dbname() {
  local cfg="$ROOT/$INSTANCE/config.toml" url
  # 只匹配 [db.default] 区块内的 url 行, 避免误取 [db.file]/[meilisearch] 的 url
  # 注: 用 POSIX 字符类而非 \s, 兼容 BSD sed(macOS)
  url="$(sed -n '/^\[db\.default\]/,/^\[/p' "$cfg" | grep -E '^[[:space:]]*url[[:space:]]*=' | head -n1 | sed -E 's/^[[:space:]]*url[[:space:]]*=[[:space:]]*//' | tr -d '"')"
  [ -n "$url" ] || { echo "backup-db: $cfg 未配置 [db.default].url" >&2; return 1; }
  pg_dsn_dbname "$url"
}

if [ "$(db_mode)" = "postgres" ]; then
  # 解析失败时 pg_dbname 已输出错误并返回非零(set -e 下即终止),
  # 与 sync-db-from-main.sh 的 pg_dbname || exit 1 语义一致。
  PG_DB="$(pg_dbname)" || exit 1
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
  ls -1t "$BACKUP_DIR"/pg-"${PG_DB}"-*.sql 2>/dev/null | tail -n +$((KEEP + 1)) | xargs -r rm -f
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
  ls -1t "$BACKUP_DIR"/"${label}"-*.db 2>/dev/null | tail -n +$((KEEP + 1)) | xargs -r rm -f
done
echo "backup-db: $INSTANCE backups done (keep $KEEP per label)"
