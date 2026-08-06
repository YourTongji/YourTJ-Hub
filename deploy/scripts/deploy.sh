#!/usr/bin/env bash
# deploy.sh — 容器版部署: 构建镜像 → compose 更新 → 健康检查 → 失败回滚。
# usage: deploy.sh <instance> <new-binary> <image-tag> [health-port]
#   instance: main 或 dev
set -euo pipefail

INSTANCE="${1:?usage: deploy.sh <instance> <new-binary> <image-tag> [health-port]}"
NEW_BINARY="${2:?usage: deploy.sh <instance> <new-binary> <image-tag> [health-port]}"
IMAGE_TAG="${3:?usage: deploy.sh <instance> <new-binary> <image-tag> [health-port]}"
PORT="${4:-5234}"

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
BUILD_DIR="$ROOT/build"
ENV_FILE="$ROOT/.env"
COMPOSE_FILE="$ROOT/docker-compose.yaml"
TAG_VAR="$([ "$INSTANCE" = "main" ] && echo MAIN_TAG || echo DEV_TAG)"
IMAGE="yourtj-hub"

log() { echo "[deploy:$INSTANCE] $*"; }

[ -x "$NEW_BINARY" ] || { log "FATAL: new binary not executable: $NEW_BINARY"; exit 1; }
[ -f "$ENV_FILE" ] || { log "FATAL: $ENV_FILE missing (run init-server.sh first)"; exit 1; }
[ -f "$COMPOSE_FILE" ] || { log "FATAL: $COMPOSE_FILE missing (run init-server.sh first)"; exit 1; }
[ -f "$BUILD_DIR/Dockerfile" ] || { log "FATAL: $BUILD_DIR/Dockerfile missing (run init-server.sh first)"; exit 1; }

# 1. 记录当前 tag 用于回滚
OLD_TAG="$(grep -E "^$TAG_VAR=" "$ENV_FILE" | cut -d= -f2 || true)"
if [ -n "$OLD_TAG" ] && docker image inspect "$IMAGE:$OLD_TAG" >/dev/null 2>&1; then
  docker tag "$IMAGE:$OLD_TAG" "$IMAGE:prev" >/dev/null 2>&1 || true
  log "saved previous image tag: $OLD_TAG"
fi

# 2. 安装新二进制并构建镜像
install -m 0755 "$NEW_BINARY" "$BUILD_DIR/yourtj-hub"
docker build -q -t "$IMAGE:$IMAGE_TAG" "$BUILD_DIR"
log "built image $IMAGE:$IMAGE_TAG"

# 3. 更新 .env tag 并启动实例
sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$IMAGE_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d "$INSTANCE"
log "compose up $INSTANCE with $IMAGE_TAG"

# 4. 健康检查(覆盖启动 + AutoMigrate 大库迁移)
for ((i = 1; i <= 60; i++)); do
  if curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then
    log "health check passed"
    exit 0
  fi
  log "waiting for health ($i/60)..."
  sleep 3
done

# 5. 失败: 回滚到旧 tag
log "FATAL: health check failed, rolling back"
if [ -n "$OLD_TAG" ]; then
  sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$OLD_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d "$INSTANCE"
  log "rolled back to $OLD_TAG"
fi
exit 1
