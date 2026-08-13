#!/usr/bin/env bash
# deploy-wiki.sh — wiki 静态站部署: 解包 dist → 构建 nginx 镜像 → compose up
#                 → 健康检查 → 失败回滚。与 deploy.sh 同构(单脚本风格)。
# usage: deploy-wiki.sh <instance> <dist-tarball> <image-tag> [health-port]
#   instance: main 或 dev (对应 compose 服务 wiki-main / wiki-dev)
# 环境变量: IMAGE_KEEP_N — wiki 前缀镜像保留数(含当前), 默认 5
set -euo pipefail

INSTANCE="${1:?usage: deploy-wiki.sh <instance> <dist-tarball> <image-tag> [health-port]}"
DIST_TARBALL="${2:?usage: deploy-wiki.sh <instance> <dist-tarball> <image-tag> [health-port]}"
IMAGE_TAG="${3:?usage: deploy-wiki.sh <instance> <dist-tarball> <image-tag> [health-port]}"
PORT="${4:-5284}"

# instance 校验(review S2): 仅 main/dev 对应 compose 服务 wiki-main/wiki-dev,
# 任意其他值会落到 WIKI_DEV_TAG + 不存在的服务, 提前拒绝
case "$INSTANCE" in
  main|dev) ;;
  *) echo "deploy-wiki: FATAL: invalid instance '$INSTANCE' (must be main or dev)" >&2; exit 1 ;;
esac

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
BUILD_DIR="$ROOT/build"
ENV_FILE="$ROOT/.env"
COMPOSE_FILE="$ROOT/docker-compose.yaml"
TAG_VAR="$([ "$INSTANCE" = "main" ] && echo WIKI_MAIN_TAG || echo WIKI_DEV_TAG)"
SERVICE="wiki-$INSTANCE"
IMAGE="yourtj-wiki"

log() { echo "[deploy-wiki:$INSTANCE] $*"; }

# 清理旧 wiki 镜像(与 deploy.sh prune_old_images 同构, 前缀 wiki-main-*/wiki-dev-*)
prune_old_images() {
  local instance_prefix keep_n removed=0 keep_count=0
  local dangling_before=0 dangling_after=0
  instance_prefix="wiki-$INSTANCE-"
  keep_n="${IMAGE_KEEP_N:-5}"
  if ! [[ "$keep_n" =~ ^[0-9]+$ ]] || [ "$keep_n" -lt 1 ]; then
    log "WARNING: invalid IMAGE_KEEP_N='$keep_n', falling back to default 5"
    keep_n=5
  fi

  log "pruning old wiki images (keep newest $keep_n ${instance_prefix}* incl. current + prev)"
  while IFS='|' read -r tag _; do
    [ -z "$tag" ] && continue
    [ "$tag" = "prev" ] && continue
    keep_count=$((keep_count + 1))
    if [ "$keep_count" -le "$keep_n" ]; then
      continue
    fi
    if docker image rm "$IMAGE:$tag" >/dev/null 2>&1; then
      removed=$((removed + 1))
      log "removed old wiki image $IMAGE:$tag"
    fi
  done < <(docker images "$IMAGE" --format '{{.Tag}}|{{.CreatedAt}}' \
      | grep "^${instance_prefix}" \
      | sort -t'|' -k2 -r)

  dangling_before="$(docker images -q -f dangling=true 2>/dev/null | wc -l | tr -d ' ' || true)"
  docker image prune -f >/dev/null 2>&1 || true
  docker builder prune -f --filter "until=72h" >/dev/null 2>&1 || true
  dangling_after="$(docker images -q -f dangling=true 2>/dev/null | wc -l | tr -d ' ' || true)"
  log "prune done: removed $removed image tag(s); dangling $dangling_before -> $dangling_after"
}

[ -f "$DIST_TARBALL" ] || { log "FATAL: dist tarball not found: $DIST_TARBALL"; exit 1; }
[ -f "$ENV_FILE" ] || { log "FATAL: $ENV_FILE missing (run init-server.sh first)"; exit 1; }
[ -f "$COMPOSE_FILE" ] || { log "FATAL: $COMPOSE_FILE missing (run init-server.sh first)"; exit 1; }
[ -f "$BUILD_DIR/wiki.Dockerfile" ] || { log "FATAL: $BUILD_DIR/wiki.Dockerfile missing (run init-server.sh first)"; exit 1; }

# 1. 记录当前 tag 用于回滚
OLD_TAG="$(grep -E "^$TAG_VAR=" "$ENV_FILE" | cut -d= -f2 || true)"
if [ -n "$OLD_TAG" ] && docker image inspect "$IMAGE:$OLD_TAG" >/dev/null 2>&1; then
  docker tag "$IMAGE:$OLD_TAG" "$IMAGE:prev" >/dev/null 2>&1 || true
  log "saved previous wiki image tag: $OLD_TAG"
fi

# 2. 解包 dist 到构建上下文并构建 nginx 镜像
rm -rf "$BUILD_DIR/wiki-dist"
mkdir -p "$BUILD_DIR/wiki-dist"
tar -xzf "$DIST_TARBALL" -C "$BUILD_DIR/wiki-dist"
[ -f "$BUILD_DIR/wiki-dist/index.html" ] || { log "FATAL: tarball 缺少 index.html(非 VitePress dist?)"; exit 1; }
docker build -q -f "$BUILD_DIR/wiki.Dockerfile" -t "$IMAGE:$IMAGE_TAG" "$BUILD_DIR"
log "built wiki image $IMAGE:$IMAGE_TAG"

# 3. 更新 .env tag 并启动 wiki 实例
sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$IMAGE_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d "$SERVICE"
log "compose up $SERVICE with $IMAGE_TAG"

# 4. 健康检查: 首页 200
for ((i = 1; i <= 30; i++)); do
  if curl -fsS "http://127.0.0.1:$PORT/" >/dev/null 2>&1; then
    log "health check passed (HTTP 200)"
    prune_old_images || log "prune failed (non-fatal, deploy succeeded)"
    exit 0
  fi
  log "waiting for wiki health ($i/30)..."
  sleep 2
done

# 5. 失败: 回滚到旧 tag
log "FATAL: health check failed, rolling back"
if [ -n "$OLD_TAG" ]; then
  sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$OLD_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d "$SERVICE"
  log "rolled back to $OLD_TAG"
fi
exit 1
