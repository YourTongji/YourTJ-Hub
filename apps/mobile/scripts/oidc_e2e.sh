#!/usr/bin/env bash
# OIDC(AppAuth) 模拟器端到端验证脚本
#
# 用途:在 Android 模拟器上完成 论坛内建 OIDC Provider → flutter_appauth 授权
#       → yourtj://callback 回跳 → 后端 POST /api/auth/oidc/exchange
#       → forum JWT 会话持久化 的完整链路验证。
#
# 前置条件:
#   - Android 模拟器已启动且 adb 可见(adb devices 有 device)
#   - adb reverse 可用:脚本会自动建立 模拟器 tcp:5234 → 宿主机 的后向映射,
#     使 App 内 http://localhost:5234 直达宿主机后端(见下方 issuer 说明)
#   - 后端已在跑(默认 http://localhost:5234,可覆盖)
#   - config.toml [oidc] 已启用,且 yourtj-mobile 客户端的
#     redirect_uris 白名单含 yourtj://callback
#   - 测试账号存在(默认 mobile_e2e / Test1234!,可覆盖)
#
# issuer 一致性说明:OIDC 客户端要求 issuer 与服务端 discovery 广告值完全相等,
# 而内建 Provider 的 http issuer 仅允许 loopback(localhost/127.0.0.1)。
# 因此 App 默认使用 http://localhost:5234/api/oauth(与配置模板一致),
# 并通过 adb reverse 访问宿主机;不能使用 10.0.2.2(非 loopback,无法作为 issuer)。
#
# 用法:
#   ./apps/mobile/scripts/oidc_e2e.sh check      # 只跑环境前置检查(CI 可用)
#   ./apps/mobile/scripts/oidc_e2e.sh run        # 构建安装 + 指引 + 服务端验证
#
# 退出码:0=链路验证通过;1=前置检查失败;2=构建/安装失败;3=服务端验证失败。
set -euo pipefail

# ---------------------------------------------------------------- 配置(可覆盖)
OIDC_ISSUER="${OIDC_ISSUER:-http://localhost:5234/api/oauth}"
API_URL="${API_URL:-http://localhost:5234}"
CLIENT_ID="${CLIENT_ID:-yourtj-mobile}"
REDIRECT_URI="${REDIRECT_URI:-yourtj://callback}"
TEST_USER="${TEST_USER:-mobile_e2e}"
TEST_PASS="${TEST_PASS:-Test1234!}"
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

  # 2. adb reverse:模拟器内 localhost:<port> → 宿主机 API_URL。
  #    App 用 loopback issuer(与服务端广告值完全一致),必须依赖该映射访问后端。
  local api_port
  api_port="$(printf '%s' "$API_URL" | sed -n 's#.*:\([0-9][0-9]*\)$#\1#p')"
  if [ -n "$api_port" ]; then
    adb reverse "tcp:$api_port" "tcp:$api_port" >/dev/null \
      || fail "adb reverse tcp:$api_port 失败(模拟器需支持 adb reverse)" 1
    ok "adb reverse tcp:$api_port → 宿主机 $API_URL"
  else
    fail "API_URL 无法解析端口(需要 http://localhost:<port> 形式): $API_URL" 1
  fi

  # 3. 后端
  curl -sf -o /dev/null --max-time 5 "$API_URL/" \
    || fail "后端不可达: $API_URL(请先启动 ./bin/yourtj-hub serve)" 1
  ok "后端可达: $API_URL"

  # 4. 内建 Provider discovery + issuer
  local discovery issuer token_ep
  discovery="$(curl -sf --max-time 5 "$OIDC_ISSUER/.well-known/openid-configuration" \
    || fail "内建 OIDC Provider 不可达: $OIDC_ISSUER（确认 config.toml [oidc].enabled=true）" 1)"
  issuer="$(printf '%s' "$discovery" | sed -n 's/.*"issuer"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  token_ep="$(printf '%s' "$discovery" | sed -n 's/.*"token_endpoint"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  [ -n "$issuer" ] && [ -n "$token_ep" ] || fail "OIDC discovery 响应异常" 1
  [ "$issuer" = "$OIDC_ISSUER" ] || fail "OIDC issuer 不匹配: 期望 $OIDC_ISSUER, 实际 $issuer" 1
  ok "内建 Provider discovery: issuer=$issuer"

  # 5. 客户端白名单由后端在 authorize/token 请求时强制校验。
  info "确认后端 config.toml [oidc]: client id=$CLIENT_ID, redirect_uris 含 $REDIRECT_URI"
}

build_install() {
  info "=== 构建并安装 forum_app(debug,含 dart-define)==="
  require_cmd flutter
  (cd "$FORUM_APP" && flutter build apk --debug \
      --dart-define=YOURTJ_OIDC_ISSUER="$OIDC_ISSUER" \
      --dart-define=YOURTJ_OIDC_CLIENT_ID="$CLIENT_ID" \
      --dart-define=YOURTJ_API_BASE_URL="$API_URL" \
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
  1. 打开 App → 「我的」→ 登录 → 「使用 yourtj 统一登录」。
  2. 在论坛登录页使用测试账号登录(见 TEST_USER/TEST_PASS,可用环境变量覆盖)。
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
