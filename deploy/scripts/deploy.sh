#!/usr/bin/env bash
# deploy.sh — 通用部署: 替换二进制 → 重启 → 健康检查 → 失败自动回滚。
# usage: deploy.sh <instance> <new-binary-path> [health-port]
#   实例: main 或 dev; 端口默认 5234(main 生产按 config.toml 实际端口传)
set -euo pipefail

INSTANCE="${1:?usage: deploy.sh <instance> <new-binary> [health-port]}"
NEW_BINARY="${2:?usage: deploy.sh <instance> <new-binary> [health-port]}"
PORT="${3:-5234}"

ROOT="${YOURTJ_ROOT:-/srv/yourtj}"
DIR="$ROOT/$INSTANCE"
BIN_DIR="$DIR/bin"
SERVICE="yourtj-$INSTANCE.service"
HEALTH_URL="http://localhost:$PORT/health"
RETRIES="${RETRIES:-60}"   # 覆盖启动 + AutoMigrate 大库迁移的等待
INTERVAL="${INTERVAL:-3}"

log() { echo "[deploy:$INSTANCE] $*"; }

[ -x "$NEW_BINARY" ] || { log "FATAL: new binary not executable: $NEW_BINARY"; exit 1; }

# 1. 保存当前二进制(回滚用)
if [ -f "$BIN_DIR/yourtj-hub" ]; then
  cp -f "$BIN_DIR/yourtj-hub" "$BIN_DIR/yourtj-hub.prev"
  log "saved previous binary"
fi

# 2. 停止 → 替换 → 启动
systemctl stop "$SERVICE" || true
install -m 0755 "$NEW_BINARY" "$BIN_DIR/yourtj-hub"
systemctl start "$SERVICE"

# 3. 健康检查
for ((i = 1; i <= RETRIES; i++)); do
  if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
    log "health check passed ($HEALTH_URL)"
    exit 0
  fi
  log "waiting for health ($i/$RETRIES)..."
  sleep "$INTERVAL"
done

# 4. 失败: 回滚到上一版二进制
log "FATAL: health check failed, rolling back"
systemctl stop "$SERVICE" || true
if [ -f "$BIN_DIR/yourtj-hub.prev" ]; then
  install -m 0755 "$BIN_DIR/yourtj-hub.prev" "$BIN_DIR/yourtj-hub"
  systemctl start "$SERVICE"
  log "rolled back to previous binary"
else
  log "no previous binary to roll back to"
fi
exit 1
