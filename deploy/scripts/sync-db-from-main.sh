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

# 共享 PostgreSQL DSN 解析库(支持 postgres:// URL 与 key=value 两种格式, issue #134)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pgdsn.sh
source "$SCRIPT_DIR/pgdsn.sh"

db_mode() {
  local cfg="$1"
  [ -f "$cfg" ] || { echo sqlite; return; }
  if grep -E '^\s*connection\s*=\s*"postgres"' "$cfg" >/dev/null 2>&1; then
    echo postgres
  else
    echo sqlite
  fi
}

# 从实例 config.toml 提取 [db.default] url 并解析数据库名(URL / key=value 均可)。
# 解析失败输出错误并返回非零。
# 注: pg_toml_url 同时负责剥离 TOML 行尾内联注释(review W1)。
pg_dbname() {
  local cfg="$1" url
  url="$(pg_toml_url "$cfg")"
  [ -n "$url" ] || { echo "sync-db: $cfg 未配置 [db.default].url" >&2; return 1; }
  pg_dsn_dbname "$url"
}

MAIN_MODE="$(db_mode "$ROOT/main/config.toml")"
DEV_MODE="$(db_mode "$ROOT/dev/config.toml")"

# 关键顺序(issue #134): 任何 DB 操作(包括停 dev 容器)之前先解析 DSN。
# 若 PG 模式配置非法, 必须在此报错退出, 绝不能让服务停留在停止状态。
MAIN_PG=""
DEV_PG=""
if [ "$MAIN_MODE" = "postgres" ] || [ "$DEV_MODE" = "postgres" ]; then
  if [ "$MAIN_MODE" = "postgres" ]; then
    MAIN_PG="$(pg_dbname "$ROOT/main/config.toml")" || exit 1
  fi
  if [ "$DEV_MODE" = "postgres" ]; then
    DEV_PG="$(pg_dbname "$ROOT/dev/config.toml")" || exit 1
  fi
  # 仅单边为 postgres 属配置错误: 同步无法进行, 且不允许先停服务
  if [ "$MAIN_MODE" != "$DEV_MODE" ]; then
    echo "sync-db: main/dev 主库模式不一致 (main=$MAIN_MODE dev=$DEV_MODE), 配置错误, 中止" >&2
    exit 1
  fi
fi

# 停 dev 容器, 避免覆盖已打开数据库文件导致数据写丢。
# 注意: 此步骤必须在上面 DSN 解析成功之后, 解析失败时不能触碰服务状态。
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" stop dev >/dev/null 2>&1 || true

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
  echo "sync-db: PG mode ($MAIN_PG -> $DEV_PG)"
  if [ -f "$DEV_FILE_DB" ]; then
    mkdir -p "$ROOT/snapshots/dev-prev"
    cp -f "$DEV_FILE_DB" "$ROOT/snapshots/dev-prev/file-$(date +%Y%m%d_%H%M%S).db"
  fi
  # 参数化(review S3): dbname 经 psql -v 变量传递, :"devpg" 由 psql 做
  # SQL 标识符引用(内部引号转义), 消除 dbname 插值进 SQL 字符串的注入面。
  # 注(fix #163 回归): psql -c 传入的 SQL 不做 psql 变量插值(官方文档:
  # -c 内容必须可被服务端完整解析、不含 psql 特有特性), :"devpg" 会被
  # 原样发给服务端报 syntax error at or near ":"; SQL 必须经 stdin 喂给
  # psql(:var 插值仅在 stdin/-f/交互输入路径生效)。ON_ERROR_STOP=1 保证
  # 任一语句失败立即非零退出, 配合 set -e 阻断后续部署。
  docker exec -i yourtj-postgres psql -U yourtj -d postgres \
    -v devpg="$DEV_PG" -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
DROP DATABASE IF EXISTS :"devpg";
CREATE DATABASE :"devpg";
SQL
  # 命令注入防护(review S3): dbname 经环境变量(-e)传入容器, 内层 sh -c
  # 只做变量展开不做语法解析, 恶意 dbname(如 %22%3B touch /tmp/PWN %3B)
  # 无法逃逸引号执行任意命令; 管道两侧的 $MAIN_PG/$DEV_PG 均由容器内
  # shell 从环境变量读取, 而非拼接进命令字符串。
  docker exec -e MAIN_PG="$MAIN_PG" -e DEV_PG="$DEV_PG" yourtj-postgres sh -c \
    'pg_dump -U yourtj -d "$MAIN_PG" | psql -U yourtj -d "$DEV_PG"' >/dev/null
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
