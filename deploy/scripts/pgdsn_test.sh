#!/usr/bin/env bash
# pgdsn_test.sh — pgdsn.sh 共享 DSN 解析库的单元测试。
# 覆盖: URL 格式、key=value 格式、非法 DSN 报错、归一化脱敏。
# 运行: bash pgdsn_test.sh (无需外部依赖)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pgdsn.sh
source "$SCRIPT_DIR/pgdsn.sh"

PASS=0
FAIL=0

assert_eq() {
  local desc="$1" want="$2" got="$3"
  if [ "$want" = "$got" ]; then
    PASS=$((PASS + 1))
    echo "ok  - $desc"
  else
    FAIL=$((FAIL + 1))
    echo "FAIL- $desc: want [$want], got [$got]"
  fi
}

# 断言命令以给定退出码执行(成功码默认 0)
assert_run() {
  local desc="$1" want_code="$2"
  shift 2
  local got_code=0
  "$@" >/dev/null 2>&1 || got_code=$?
  if [ "$got_code" -eq "$want_code" ]; then
    PASS=$((PASS + 1))
    echo "ok  - $desc (exit $got_code)"
  else
    FAIL=$((FAIL + 1))
    echo "FAIL- $desc: want exit $want_code, got $got_code"
  fi
}

## --- URL 格式 ---
assert_eq "URL 基本格式" "yourtj_main" "$(pg_dsn_dbname 'postgres://yourtj:secret@postgres:5432/yourtj_main?sslmode=disable')"
assert_eq "URL postgresql:// scheme" "yourtj_dev" "$(pg_dsn_dbname 'postgresql://yourtj:secret@postgres/yourtj_dev')"
assert_eq "URL 无密码" "forum" "$(pg_dsn_dbname 'postgres://yourtj@db.example.com:5433/forum')"
assert_eq "URL 无端口" "forum" "$(pg_dsn_dbname 'postgres://yourtj@db.example.com/forum')"
assert_eq "URL 无端口无密码" "forum" "$(pg_dsn_dbname 'postgres://db.example.com/forum')"
assert_eq "URL query 参数 dbname 优先于路径段" "override" "$(pg_dsn_dbname 'postgres://u@h/dbname_in_path?dbname=override&sslmode=disable')"
assert_eq "URL query 多参数" "yourtj" "$(pg_dsn_dbname 'postgres://u:p@h:5432/yourtj?sslmode=require&connect_timeout=5')"

## --- key=value 格式 ---
assert_eq "KV 基本格式" "yourtj_main" "$(pg_dsn_dbname 'host=postgres user=yourtj password=secret dbname=yourtj_main port=5432 sslmode=disable')"
assert_eq "KV dbname 在中间" "forum" "$(pg_dsn_dbname 'user=u dbname=forum host=h')"
assert_eq "KV dbname 在开头" "forum" "$(pg_dsn_dbname 'dbname=forum user=u')"
assert_eq "KV 双引号值" "forum" "$(pg_dsn_dbname 'host=h dbname="forum" user=u')"
assert_eq "KV 单引号值" "forum" "$(pg_dsn_dbname "host=h dbname='forum' user=u")"

## --- 非法 DSN ---
assert_run "空 DSN 报错" 1 pg_dsn_dbname ""
assert_run "URL 缺数据库名报错" 1 pg_dsn_dbname "postgres://yourtj@postgres:5432/"
assert_run "URL 缺主机报错" 1 pg_dsn_dbname "postgres:///forum"
assert_run "URL 缺库名(仅 host)报错" 1 pg_dsn_dbname "postgres://yourtj@postgres"
assert_run "KV 缺 dbname 报错" 1 pg_dsn_dbname "host=h user=u password=p"
assert_run "KV dbname 为空报错" 1 pg_dsn_dbname "host=h dbname= user=u"
assert_run "无关字符串报错" 1 pg_dsn_dbname "this is not a dsn"

## --- 归一化(脱敏) ---
assert_eq "URL 归一化脱敏密码" "postgres://yourtj:***@postgres:5432/yourtj_main" \
  "$(pg_dsn_normalize 'postgres://yourtj:secret@postgres:5432/yourtj_main?sslmode=disable')"
assert_eq "KV 归一化脱敏密码" "host=h user=u password=*** dbname=forum" \
  "$(pg_dsn_normalize 'host=h user=u password=secret dbname=forum')"

echo
echo "pgdsn_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
