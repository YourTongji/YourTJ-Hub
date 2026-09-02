#!/usr/bin/env bash
# apply-config.sh — 原子替换实例 config.toml（配置治理 CI 下发入口）。
#
# 语义（与 deploy.sh 同一回滚体系）:
#   1. 校验渲染产物（存在、无未替换占位符）;
#   2. 计算现网 config 当前 SHA-256，与 .config.sha256 marker 比较:
#      内容未变 → 幂等跳过（不重启，并刷新 marker）;
#   3. 备份现网 → config.toml.prev（单槽; 已存在 = 并发/残留 → 失败拒绝，
#      防止两个下发互相覆盖回滚点）;
#   4. 原子替换（先写同目录 config.toml.new 再 mv -f，rename 原子）;
#   5. --force-recreate 重建容器（关键! 单文件 bind-mount 持有旧 inode,
#      restart 不会重挂新文件; 重建容器才会重新解析挂载并完整重读 config,
#      OAuth provider / DB 连接等在进程启动时初始化）;
#   6. 健康检查（端口取 .env; 60×3s 与 deploy.sh 一致）;
#   7. 成功 → 写 .config.sha256 marker，并清理 prev（释放单槽）;
#      失败 → 恢复 config.toml.prev（mv 即消费 prev）+ 重建 + exit 1
#      （不回写 marker: 若恢复出的文件与上次成功值不同, drift-check 会告警）。
#
# usage: apply-config.sh <instance> <rendered-file>
#   instance: main|dev      rendered-file: CI 渲染产物绝对路径
# 环境变量:
#   YOURTJ_ROOT — 部署根（默认 /opt/yourtj）
set -euo pipefail

INSTANCE="${1:?usage: apply-config.sh <instance> <rendered-file>}"
RENDERED="${2:?usage: apply-config.sh <instance> <rendered-file>}"
[ "$INSTANCE" = "main" ] || [ "$INSTANCE" = "dev" ] || { echo "apply-config: instance 必须是 main|dev (got $INSTANCE)" >&2; exit 1; }

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
ENV_FILE="$ROOT/.env"
COMPOSE_FILE="$ROOT/docker-compose.yaml"
INST_DIR="$ROOT/$INSTANCE"
CFG="$INST_DIR/config.toml"
MARKER="$INST_DIR/.config.sha256"
PREV="$CFG.prev"

PORT_VAR="$([ "$INSTANCE" = "main" ] && echo MAIN_PORT || echo DEV_PORT)"
PORT="5234"
[ "$INSTANCE" = "dev" ] && PORT="5235"
if [ -f "$ENV_FILE" ] && grep -qE "^$PORT_VAR=" "$ENV_FILE"; then
  PORT="$(grep -E "^$PORT_VAR=" "$ENV_FILE" | head -1 | cut -d= -f2-)"
fi

compose_cmd() {
  # --force-recreate --no-deps: 单文件 bind-mount 替换后必须重建容器才重挂新文件;
  # --no-deps 避免触碰 postgres/meilisearch。
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --force-recreate --no-deps "$INSTANCE"
}

log() { echo "[apply-config:$INSTANCE] $*"; }
sha() { sha256sum "$1" 2>/dev/null | awk '{print $1}'; }

# --- 前置校验（fail-closed） ---
[ -f "$RENDERED" ] || { log "FATAL: 渲染产物不存在: $RENDERED"; exit 1; }
if grep -q '{{' "$RENDERED"; then
  log "FATAL: 渲染产物含未替换占位符，拒绝应用"; exit 1
fi
[ -f "$CFG" ] || { log "FATAL: $CFG 不存在（未初始化?）"; exit 1; }
[ -f "$ENV_FILE" ] || { log "FATAL: $ENV_FILE 缺失（run init-server.sh 先）"; exit 1; }
[ -f "$COMPOSE_FILE" ] || { log "FATAL: $COMPOSE_FILE 缺失（run init-server.sh 先）"; exit 1; }

# --- 幂等: 内容与现网一致则跳过 ---
CUR_SHA="$(sha "$CFG")"
NEW_SHA="$(sha "$RENDERED")"
if [ -n "$CUR_SHA" ] && [ "$CUR_SHA" = "$NEW_SHA" ]; then
  log "内容与现网一致（sha=$CUR_SHA），幂等跳过"
  printf '%s\n' "$CUR_SHA" > "$MARKER"
  exit 0
fi

# --- 备份（单槽 prev; 并发/残留防护） ---
if [ -e "$PREV" ]; then
  log "FATAL: $PREV 已存在（并发 apply 或上次失败残留），拒绝覆盖回滚点"
  log "       确认无并发后: rm -f $PREV 再重试"
  exit 1
fi
cp -f "$CFG" "$PREV" || { log "FATAL: 备份现网 config 失败"; exit 1; }
log "已备份现网 config → $PREV"

# --- 原子替换（同目录 .new + mv = rename 原子） ---
cp -f "$RENDERED" "$CFG.new" || { log "FATAL: 写入 $CFG.new 失败"; rm -f "$CFG.new"; exit 1; }
mv -f "$CFG.new" "$CFG" || { log "FATAL: mv 替换 config 失败"; rm -f "$CFG.new"; exit 1; }
log "已原子替换 config.toml"

# --- 重建容器（单文件 bind-mount: 必须 force-recreate 才重挂新文件 + 进程级重读） ---
if ! compose_cmd; then
  log "FATAL: compose 重建失败; 尝试恢复 prev"
  if [ -f "$PREV" ]; then
    mv -f "$PREV" "$CFG"
    compose_cmd >/dev/null 2>&1 || true
    log "已恢复 prev 并尝试重建"
  fi
  exit 1
fi
log "已重建 $INSTANCE 容器（force-recreate，重挂新 config）"

# --- 健康检查 ---
health_ok=0
for ((i = 1; i <= 60; i++)); do
  if curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then
    health_ok=1
    break
  fi
  log "等待健康 ($i/60)..."
  sleep 3
done

if [ "$health_ok" -eq 1 ]; then
  printf '%s\n' "$NEW_SHA" > "$MARKER"
  rm -f "$PREV"
  log "健康检查通过; marker=$NEW_SHA; prev 已清理"
  exit 0
fi

# --- 失败恢复（mv 消费 prev） ---
log "FATAL: 健康检查失败，恢复上一份 config"
if [ -f "$PREV" ]; then
  mv -f "$PREV" "$CFG"
  compose_cmd >/dev/null 2>&1 || true
  log "已恢复 prev → config.toml 并重建容器"
  for ((i = 1; i <= 20; i++)); do
    curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1 && { log "恢复后健康检查通过"; break; }
    sleep 3
  done
else
  log "FATAL: 无 prev 可恢复; 实例可能处于未健康状态，需人工介入"
fi
exit 1
