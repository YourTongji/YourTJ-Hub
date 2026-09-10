#!/usr/bin/env bash
# verify-instance.sh — 实例 config 漂移守卫（配置治理）。
#
# 核对项（不打印 secret 值）:
#   1. config.toml 存在;
#   2. .config.sha256 marker 存在且与现网文件 SHA-256 一致
#      （不一致 = 人工 SSH 改动或未走 apply-config 的下发 → 告警）;
#   3. [server].url 与期望实例域名一致（默认 main=https://f.yourtj.de dev=https://dev.yourtj.de,
#      可用 EXPECT_SERVER_URL 覆盖，CI 从仓库 instances JSON 读入）;
#   4. [db.default].url 的 dbname 与期望一致（复用 pgdsn.sh 解析）;
#   5. 必需 secret 键非空（github 凭据 main 必填; dev 允许空——DB 站点设置无环境隔离）。
#
# usage: verify-instance.sh <instance>
# 环境变量: YOURTJ_ROOT(默认 /opt/yourtj), EXPECT_SERVER_URL, EXPECT_DBNAME
set -euo pipefail

INSTANCE="${1:?usage: verify-instance.sh <instance>}"
[ "$INSTANCE" = "main" ] || [ "$INSTANCE" = "dev" ] || { echo "verify-instance: instance 必须是 main|dev (got $INSTANCE)" >&2; exit 2; }

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
INST_DIR="$ROOT/$INSTANCE"
CFG="$INST_DIR/config.toml"
MARKER="$INST_DIR/.config.sha256"

EXPECT_URL="${EXPECT_SERVER_URL:-}"
[ -n "$EXPECT_URL" ] || EXPECT_URL="$([ "$INSTANCE" = "main" ] && echo https://f.yourtj.de || echo https://dev.yourtj.de)"
EXPECT_DB="${EXPECT_DBNAME:-}"
[ -n "$EXPECT_DB" ] || EXPECT_DB="yourtj_$INSTANCE"

# --- cfg_get <file> <parent.section.key> : 表感知取键值（只做非空/比对，输出不涉密） ---
cfg_get() {
  local file="$1" path="$2" sec key
  sec="${path%.*}"   # 父节路径, 如 db.default / app / wiki.git
  key="${path##*.}"
  awk -v sec="$sec" -v key="$key" '
    /^\[/ {
      line=$0; gsub(/^\[|\]$/, "", line)
      in_sec = (line == sec)
      next
    }
    in_sec && match($0, "^[ \t]*" key "[ \t]*=") {
      sub(/^[^=]*=[ \t]*/, "")
      gsub(/^"|"$/, "")
      print
      exit
    }
  ' "$file"
}

FAIL=0
warn() { echo "verify-instance[$INSTANCE]: FAIL - $*"; FAIL=1; }
ok() { echo "verify-instance[$INSTANCE]: ok - $*"; }

# --- 1. 存在性 ---
[ -f "$CFG" ] || { echo "verify-instance[$INSTANCE]: FAIL - $CFG 不存在"; exit 2; }

# --- 2. marker 漂移信号 ---
CUR_SHA="$(sha256sum "$CFG" 2>/dev/null | awk '{print $1}')"
if [ -f "$MARKER" ]; then
  RECORDED="$(tr -d '[:space:]' < "$MARKER")"
  if [ -n "$RECORDED" ] && [ "$RECORDED" = "$CUR_SHA" ]; then
    ok "config SHA-256 与 marker 一致 ($CUR_SHA)"
  else
    warn "config SHA-256 与 marker 不一致 (recorded=$RECORDED current=$CUR_SHA) → 人工改动或未同步下发"
  fi
else
  warn ".config.sha256 marker 不存在（尚无 apply-config 成功下发）"
fi

# --- 3. server.url ---
URL_VAL="$(awk '/^\[server\]/{f=1;next} f&&/^url =/{sub(/^url = "/,""); sub(/"$/,""); print; exit}' "$CFG")"
if [ -n "$URL_VAL" ] && [ "$URL_VAL" = "$EXPECT_URL" ]; then
  ok "server.url = $EXPECT_URL"
else
  warn "server.url = '${URL_VAL:-<空>}' ≠ 期望 '$EXPECT_URL'"
fi

# --- 4. dbname（复用 pgdsn.sh; 只打印库名, 不打印 DSN/密码） ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=pgdsn.sh
source "$SCRIPT_DIR/pgdsn.sh"
DSN_URL="$(pg_toml_url "$CFG" 2>/dev/null || true)"
if [ -n "$DSN_URL" ]; then
  DBNAME="$(pg_dsn_dbname "$DSN_URL" 2>/dev/null || true)"
  if [ -n "$DBNAME" ] && [ "$DBNAME" = "$EXPECT_DB" ]; then
    ok "db.default dbname = $EXPECT_DB"
  else
    warn "db.default dbname = '${DBNAME:-<解析失败>}' ≠ 期望 '$EXPECT_DB'"
  fi
else
  warn "无法解析 db.default url"
fi

# --- 5. 必需键非空（github 凭据 main 必填 / dev 允许空） ---
check_nonempty() {
  local path="$1" desc="$2"
  local v
  v="$(cfg_get "$CFG" "$path")"
  if [ -n "$v" ] && [ "$v" != '""' ]; then
    ok "$desc 非空"
  else
    if [ "$INSTANCE" = "dev" ] && { [ "$path" = "github.client_id" ] || [ "$path" = "github.client_secret" ]; }; then
      ok "$desc 为空（dev 已知边界: DB siteUrl 无环境隔离, github 凭据保持空）"
    else
      warn "$desc 为空（必需键未配置）"
    fi
  fi
}

check_nonempty "app.signingKey" "signingKey"
check_nonempty "meilisearch.masterkey" "meilisearch masterkey"
check_nonempty "wiki.git.webhook_secret" "wiki.git webhook_secret"
if [ "$INSTANCE" = "main" ]; then
  check_nonempty "github.client_id" "github client_id"
  check_nonempty "github.client_secret" "github client_secret"
fi

if [ "$FAIL" -eq 0 ]; then
  echo "verify-instance[$INSTANCE]: 全部通过"
  exit 0
else
  echo "verify-instance[$INSTANCE]: 存在漂移/配置问题（见上）"
  exit 1
fi
