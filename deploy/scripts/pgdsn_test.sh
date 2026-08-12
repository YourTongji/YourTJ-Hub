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

# 断言 stderr 不包含给定子串(用于验证错误路径不回显明文密码)
assert_stderr_no() {
  local desc="$1" needle="$2"
  shift 2
  local err
  # 命令可能返回非零(错误路径), set -e 下必须显式容忍
  err="$("$@" 2>&1 >/dev/null)" || true
  if [[ "$err" != *"$needle"* ]]; then
    PASS=$((PASS + 1))
    echo "ok  - $desc (stderr 不含 [$needle])"
  else
    FAIL=$((FAIL + 1))
    echo "FAIL- $desc: stderr 含 [$needle]: $err"
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
assert_eq "URL 大写 scheme" "forum" "$(pg_dsn_dbname 'POSTGRES://u:p@h/forum')"
assert_eq "URL query-only dbname 无路径" "dbq" "$(pg_dsn_dbname 'postgres://u@h/?dbname=dbq')"
assert_eq "URL query-only dbname 无路径斜杠" "dbq" "$(pg_dsn_dbname 'postgres://u@h?dbname=dbq')"
assert_eq "URL %XX 编码 dbname" "my%20db" "$(pg_dsn_dbname 'postgres://u@h/my%20db')"

## --- review P1 回归: 单引号密码(旧 eval 实现下命令注入+解析损坏) ---
assert_eq "URL 密码含单引号(注入防护)" "db" "$(pg_dsn_dbname "postgres://user:pa'; touch /tmp/pwned2; echo 'ss@host/db")"
assert_eq "URL 密码含单引号 无路径仅 query" "dbq" "$(pg_dsn_dbname "postgres://user:pa'; touch /tmp/pwned2; echo 'ss@h?dbname=dbq")"
# 确认 PoC 的任意命令未被执行(注入防护生效)
if [ -e /tmp/pwned2 ]; then
  FAIL=$((FAIL + 1))
  echo "FAIL- PoC 命令注入执行: /tmp/pwned2 被创建"
else
  PASS=$((PASS + 1))
  echo "ok  - PoC 命令注入未执行(/tmp/pwned2 不存在)"
fi
# 密码含单引号时 normalize 不逃逸破坏
assert_eq "URL normalize 单引号密码脱敏" "postgres://user:***@host/db" \
  "$(pg_dsn_normalize "postgres://user:pa'; touch /tmp/pwned2; echo 'ss@host/db")"

## --- review P1 回归: 密码含裸 @(按最后一个 @ 切分 host) ---
assert_eq "URL 密码含裸 @ host 解析" "db" "$(pg_dsn_dbname 'postgres://user:pa@ss@host/db')"
assert_eq "URL 密码含裸 @ normalize" "postgres://user:***@host/db" \
  "$(pg_dsn_normalize 'postgres://user:pa@ss@host/db')"

## --- key=value 格式 ---
assert_eq "KV 基本格式" "yourtj_main" "$(pg_dsn_dbname 'host=postgres user=yourtj password=secret dbname=yourtj_main port=5432 sslmode=disable')"
assert_eq "KV dbname 在中间" "forum" "$(pg_dsn_dbname 'user=u dbname=forum host=h')"
assert_eq "KV dbname 在开头" "forum" "$(pg_dsn_dbname 'dbname=forum user=u')"
assert_eq "KV 双引号值" "forum" "$(pg_dsn_dbname 'host=h dbname="forum" user=u')"
assert_eq "KV 单引号值" "forum" "$(pg_dsn_dbname "host=h dbname='forum' user=u")"

## --- review P1 回归: KV 其他值含 dbname= 子串(按 token 解析, 取最后一个) ---
assert_eq "KV 密码含 dbname= 子串" "forum" "$(pg_dsn_dbname 'user=u password=xdbname=y dbname=forum')"
assert_eq "KV 密码含 dbname= 子串且 dbname 在前" "forum" "$(pg_dsn_dbname 'dbname=forum user=u password=xdbname=y')"
assert_eq "KV 多 dbname token 取最后一个" "last" "$(pg_dsn_dbname 'dbname=first dbname=last user=u')"

## --- 非法 DSN ---
assert_run "空 DSN 报错" 1 pg_dsn_dbname ""
assert_run "URL 缺数据库名报错" 1 pg_dsn_dbname "postgres://yourtj@postgres:5432/"
assert_run "URL 缺主机报错" 1 pg_dsn_dbname "postgres:///forum"
assert_run "URL 缺库名(仅 host)报错" 1 pg_dsn_dbname "postgres://yourtj@postgres"
assert_run "KV 缺 dbname 报错" 1 pg_dsn_dbname "host=h user=u password=p"
assert_run "KV dbname 为空报错" 1 pg_dsn_dbname "host=h dbname= user=u"
assert_run "无关字符串报错" 1 pg_dsn_dbname "this is not a dsn"

## --- 错误路径不回显明文密码 ---
assert_stderr_no "URL 缺库名错误不泄露密码" "SUPERSECRET" pg_dsn_dbname "postgres://yourtj:SUPERSECRET@/db"
assert_stderr_no "URL 缺主机错误不泄露密码" "SUPERSECRET" pg_dsn_dbname "postgres://yourtj:SUPERSECRET@/forum"
assert_stderr_no "KV 无法解析错误不泄露密码" "SUPERSECRET" pg_dsn_dbname "this is not a dsn password=SUPERSECRET"

## --- 归一化(脱敏) ---
assert_eq "URL 归一化脱敏密码" "postgres://yourtj:***@postgres:5432/yourtj_main" \
  "$(pg_dsn_normalize 'postgres://yourtj:secret@postgres:5432/yourtj_main?sslmode=disable')"
assert_eq "KV 归一化脱敏密码" "host=h user=u password=*** dbname=forum" \
  "$(pg_dsn_normalize 'host=h user=u password=secret dbname=forum')"
assert_eq "KV normalize 密码含 dbname= 子串" "user=u password=*** dbname=forum" \
  "$(pg_dsn_normalize 'user=u password=xdbname=y dbname=forum')"
assert_eq "URL normalize 大写 scheme" "postgres://u:***@h/forum" \
  "$(pg_dsn_normalize 'POSTGRES://u:p@h/forum')"

## --- review W1 回归: TOML 行尾内联注释不破坏 DSN 解析 ---
W1_CFG="$(mktemp)"
trap 'rm -f "$W1_CFG"' EXIT
cat > "$W1_CFG" <<'EOF'
[db.default]
url = "postgres://yourtj:secret@postgres:5432/yourtj_main"  # production main db

[db.file]
url = "sqlite:///tmp/file.db"
EOF
assert_eq "W1 URL+行尾注释 提取" "postgres://yourtj:secret@postgres:5432/yourtj_main" "$(pg_toml_url "$W1_CFG")"
assert_eq "W1 URL+行尾注释 dbname" "yourtj_main" "$(pg_dsn_dbname "$(pg_toml_url "$W1_CFG")")"

cat > "$W1_CFG" <<'EOF'
[db.default]
url = "host=postgres user=yourtj password=secret dbname=yourtj_kv"  # kv comment
EOF
assert_eq "W1 KV+行尾注释 提取" "host=postgres user=yourtj password=secret dbname=yourtj_kv" "$(pg_toml_url "$W1_CFG")"
assert_eq "W1 KV+行尾注释 dbname" "yourtj_kv" "$(pg_dsn_dbname "$(pg_toml_url "$W1_CFG")")"

cat > "$W1_CFG" <<'EOF'
[db.default]
url = "postgres://yourtj:pa#ss@postgres:5432/yourtj_main"
EOF
assert_eq "W1 引号内 # 不剥离" "postgres://yourtj:pa#ss@postgres:5432/yourtj_main" "$(pg_toml_url "$W1_CFG")"

cat > "$W1_CFG" <<'EOF'
[meilisearch]
url = "dbname=meili_wrong"

[db.default]
url = "postgres://u:p@h:5432/correct_main"
EOF
assert_eq "W1 [db.default] 区块限定" "correct_main" "$(pg_dsn_dbname "$(pg_toml_url "$W1_CFG")")"

echo
echo "pgdsn_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]