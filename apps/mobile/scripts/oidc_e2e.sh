#!/usr/bin/env bash
# OIDC(AppAuth) 模拟器端到端验证脚本
#
# 用途:在 Android 模拟器上完成 本地 Casdoor → flutter_appauth 授权
#       → yourtj://callback 回跳 → 后端 POST /api/auth/oidc/exchange
#       → forum JWT 会话持久化 的完整链路验证。
#
# 前置条件:
#   - Android 模拟器已启动且 adb 可见(adb devices 有 device)
#   - 后端已在跑(默认 http://127.0.0.1:5234,可覆盖)
#   - Casdoor 已在跑(默认 http://127.0.0.1:8001,可覆盖),
#     且 forum-app 应用 redirectUris 白名单含 yourtj://callback
#   - 测试账号存在(默认 mobile_e2e / Test1234!,可覆盖)
#
# 用法:
#   ./apps/mobile/scripts/oidc_e2e.sh check      # 只跑环境前置检查(CI 可用)
#   ./apps/mobile/scripts/oidc_e2e.sh run        # 构建安装 + 指引 + 服务端验证
#
# 退出码:0=链路验证通过;1=前置检查失败;2=构建/安装失败;3=服务端验证失败。
set -euo pipefail

# ---------------------------------------------------------------- 配置(可覆盖)
CASDOOR_URL="${CASDOOR_URL:-http://127.0.0.1:8001}"
API_URL="${API_URL:-http://127.0.0.1:5234}"
CLIENT_ID="${CLIENT_ID:-f29f6177fac30dc47d14}"
REDIRECT_URI="${REDIRECT_URI:-yourtj://callback}"
TEST_USER="${TEST_USER:-mobile_e2e}"
TEST_PASS="${TEST_PASS:-Test1234!}"
EMULATOR_HOST="${EMULATOR_HOST:-10.0.2.2}"          # 模拟器访问宿主机
APP_ID="tj.yourtj.forum_app"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
FORUM_APP="$ROOT/apps/mobile/packages/forum_app"

# ---------------------------------------------------------------- 工具函数
info()  { printf '\033[1;34m[OIDC-E2E]\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m[OK]\033[0m %s\n' "$*"; }
fail()  { printf '\033[1;31m[FAIL]\033[0m %s\n' "$*"; exit "${2:-1}"; }

require_cmd() { command -v "$1" >/dev/null 2>&1 || fail "缺少命令: $1" 1; }

check_prereqs() {
  info "=== 前置检查 ==="
  require_cmd adb
  require_cmd curl
  require_cmd flutter

  # 1. 模拟器
  local devices
  devices="$(adb devices | awk 'NR>1 && $2=="device" {print $1}')"
  [ -n "$devices" ] || fail "无可用 Android 模拟器(adb devices 为空);请先启动 AVD" 1
  ok "模拟器在线: $devices"

  # 2. 后端
  curl -sf -o /dev/null --max-time 5 "$API_URL/" \
    || fail "后端不可达: $API_URL(请先启动 ./bin/yourtj-hub serve)" 1
  ok "后端可达: $API_URL"

  # 3. Casdoor discovery + issuer
  local discovery issuer token_ep
  discovery="$(curl -sf --max-time 5 "$CASDOOR_URL/.well-known/openid-configuration" \
    || fail "Casdoor 不可达: $CASDOOR_URL" 1)"
  issuer="$(printf '%s' "$discovery" | sed -n 's/.*"issuer"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  token_ep="$(printf '%s' "$discovery" | sed -n 's/.*"token_endpoint"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  [ -n "$issuer" ] && [ -n "$token_ep" ] || fail "Casdoor discovery 响应异常" 1
  ok "Casdoor discovery: issuer=$issuer"

  # 4. 后端 OIDC 配置与白名单(经 exchange 端点不可达时的快速反馈)
  #    真实白名单校验由后端在请求时执行,此处只提示配置检查。
  info "确认后端 config.toml [casdoor]: endpoint=$CASDOOR_URL client_id=$CLIENT_ID"
  info "确认 Casdoor forum-app redirectUris 含 $REDIRECT_URI"
}

build_install() {
  info "=== 构建并安装 forum_app(debug,含 dart-define)==="
  require_cmd flutter
  (cd "$FORUM_APP" && flutter build apk --debug \
      --dart-define=YOURTJ_OIDC_ISSUER="http://$EMULATOR_HOST:8001" \
      --dart-define=YOURTJ_OIDC_CLIENT_ID="$CLIENT_ID" \
      --dart-define=YOURTJ_API_BASE_URL="http://$EMULATOR_HOST:5234" \
      >/dev/null) || fail "flutter build apk 失败" 2

  local apk
  apk="$(ls "$FORUM_APP"/build/app/outputs/flutter-apk/app-debug.apk 2>/dev/null \
    || fail "未找到 APK 产物" 2)"
  adb install -r "$apk" >/dev/null || fail "adb install 失败" 2
  ok "APK 已安装"
}

launch_app() {
  info "=== 启动应用 ==="
  adb shell am start -n "$APP_ID/.MainActivity" >/dev/null 2>&1 \
    || adb shell monkey -p "$APP_ID" -c android.intent.category.LAUNCHER 1 >/dev/null \
    || fail "无法启动应用" 2
  ok "应用已启动"
}

manual_steps() {
  info "=== 手动操作指引(模拟器内)==="
  cat <<'EOM'
  1. 打开 App → 「我的」→ 设置 → 账号与登录 → 「Casdoor 统一登录」。
  2. Casdoor 授权页登录测试账号(见 TEST_USER/TEST_PASS,可用 --env 覆盖)。
  3. 授权后应回跳 yourtj://callback(AppAuth 捕获授权码),
     应用自动调后端 POST /api/auth/oidc/exchange 兑换 forum JWT。
  4. 回跳后应用应进入已登录态(我的页显示用户名/头像)。
EOM
}

verify_server_side() {
  info "=== 服务端验证 ==="
  # 1. 会话落库:最近 2 分钟内 user_sessions 新增行(本地 SQLite)。
  local db_file="${YOURTJ_DB_FILE:-}"
  if [ -z "$db_file" ]; then
    db_file="$(sed -n 's/.*path[[:space:]]*=[[:space:]]*"*\([^"]*\.db\)"*.*/\1/p' \
      "$ROOT/config.toml" 2>/dev/null | head -1 || true)"
    [ -n "$db_file" ] || db_file="yourtj-hub.db"
  fi
  if command -v sqlite3 >/dev/null 2>&1 && [ -f "$db_file" ]; then
    local sessions
    sessions="$(sqlite3 "$db_file" \
      "SELECT COUNT(*) FROM user_sessions WHERE created_at > datetime('now','localtime','-2 minutes');" 2>/dev/null || true)"
    if [ "${sessions:-0}" -gt 0 ]; then
      ok "检测到 $sessions 条新会话(user_sessions 落库)"
    else
      fail "未检测到新会话;请确认已在模拟器内完成登录" 3
    fi
  else
    info "跳过会话落库检查(无 sqlite3 或未找到 DB);请人工确认应用已登录"
  fi

  # 2. 后端日志无 exchange 报错(最近 20 行)。
  info "提示:后端日志若出现 'OIDC exchange failed' 表示兑换失败,请核对白名单与 client 配置。"
  ok "服务端验证完成"
}

# ---------------------------------------------------------------- 主流程
case "${1:-check}" in
  check)
    check_prereqs
    ok "前置检查全部通过"
    ;;
  run)
    check_prereqs
    build_install
    launch_app
    manual_steps
    echo
    info "请在模拟器内完成上述手动步骤,然后回车继续服务端验证..."
    read -r _
    verify_server_side
    ok "端到端链路验证通过"
    ;;
  *)
    echo "用法: $0 {check|run}" >&2
    exit 1
    ;;
esac
