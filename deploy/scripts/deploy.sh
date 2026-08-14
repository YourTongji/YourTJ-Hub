#!/usr/bin/env bash
# deploy.sh — GHCR 镜像流部署: 拉取镜像 → compose 更新 → 健康检查 → 失败回滚。
#   compose up 带 --remove-orphans: 不在当前 compose 文件中定义的服务容器
#   (如旧 VitePress wiki 的 yourtj-wiki-main/-dev) 会被停止并移除。
#   部署成功后自动清理本实例前缀的旧镜像, 防止磁盘无限膨胀。
# usage: deploy.sh <instance> <image-tag> [health-port]
#   instance: main 或 dev
# 环境变量:
#   IMAGE_REPO   — 镜像仓库(默认 ghcr.io/yourtongji/yourtj-hub, 公开镜像匿名 pull)
#   IMAGE_KEEP_N — 每个实例前缀保留的镜像 tag 数(含当前), 默认 5
set -euo pipefail

INSTANCE="${1:?usage: deploy.sh <instance> <image-tag> [health-port]}"
IMAGE_TAG="${2:?usage: deploy.sh <instance> <image-tag> [health-port]}"
PORT="${3:-5234}"

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
ENV_FILE="$ROOT/.env"
COMPOSE_FILE="$ROOT/docker-compose.yaml"
TAG_VAR="$([ "$INSTANCE" = "main" ] && echo MAIN_TAG || echo DEV_TAG)"
IMAGE="${IMAGE_REPO:-ghcr.io/yourtongji/yourtj-hub}"

log() { echo "[deploy:$INSTANCE] $*"; }

# 清理旧镜像, 防止磁盘无限膨胀:
#   - 保留该实例前缀(dev-*/main-*)最近 IMAGE_KEEP_N 个 tag(含当前) + prev 回滚 tag
#   - 清理 dangling images 与构建缓存
prune_old_images() {
  local instance_prefix keep_n removed=0 keep_count=0
  local dangling_before=0 dangling_after=0
  instance_prefix="$([ "$INSTANCE" = "main" ] && echo "main-" || echo "dev-")"
  keep_n="${IMAGE_KEEP_N:-5}"
  if ! [[ "$keep_n" =~ ^[0-9]+$ ]] || [ "$keep_n" -lt 1 ]; then
    log "WARNING: invalid IMAGE_KEEP_N='$keep_n', falling back to default 5"
    keep_n=5
  fi

  log "pruning old images (keep newest $keep_n ${instance_prefix}* incl. current + prev)"
  # 按创建时间倒序列出该实例前缀的 tag, 保留前 keep_n 个(含当前), 超出删除; prev 始终保留
  while IFS='|' read -r tag _; do
    [ -z "$tag" ] && continue
    [ "$tag" = "prev" ] && continue
    keep_count=$((keep_count + 1))
    if [ "$keep_count" -le "$keep_n" ]; then
      continue
    fi
    if docker image rm "$IMAGE:$tag" >/dev/null 2>&1; then
      removed=$((removed + 1))
      log "removed old image $IMAGE:$tag"
    fi
  done < <(docker images "$IMAGE" --format '{{.Tag}}|{{.CreatedAt}}' \
      | grep "^${instance_prefix}" \
      | sort -t'|' -k2 -r)

  # 清理 dangling 镜像(拉取/重打标签残留)与不再使用的构建缓存, 失败不影响部署结果
  dangling_before="$(docker images -q -f dangling=true 2>/dev/null | wc -l | tr -d ' ' || true)"
  docker image prune -f >/dev/null 2>&1 || true
  docker builder prune -f --filter "until=72h" >/dev/null 2>&1 || true
  dangling_after="$(docker images -q -f dangling=true 2>/dev/null | wc -l | tr -d ' ' || true)"
  log "prune done: removed $removed image tag(s); dangling $dangling_before -> $dangling_after"
}

[ -f "$ENV_FILE" ] || { log "FATAL: $ENV_FILE missing (run init-server.sh first)"; exit 1; }
[ -f "$COMPOSE_FILE" ] || { log "FATAL: $COMPOSE_FILE missing (run init-server.sh first)"; exit 1; }

# 1. 记录当前 tag 用于回滚
OLD_TAG="$(grep -E "^$TAG_VAR=" "$ENV_FILE" | cut -d= -f2 || true)"
if [ -n "$OLD_TAG" ] && docker image inspect "$IMAGE:$OLD_TAG" >/dev/null 2>&1; then
  docker tag "$IMAGE:$OLD_TAG" "$IMAGE:prev" >/dev/null 2>&1 || true
  log "saved previous image tag: $OLD_TAG"
fi

# 2. 拉取新镜像(GHCR 公开镜像, 匿名 pull)
docker pull "$IMAGE:$IMAGE_TAG"
log "pulled image $IMAGE:$IMAGE_TAG"

# 3. 更新 .env tag 并启动实例。
#    nginx 反代若在当前 compose 中定义则一并 up(新机首个实例部署时拉起;
#    旧机 compose 无 nginx 服务时不传, 保持向后兼容)
sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$IMAGE_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
SERVICES="$INSTANCE"
if docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" config --services 2>/dev/null | grep -qx nginx; then
  SERVICES="$SERVICES nginx"
fi
# shellcheck disable=SC2086
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans $SERVICES
log "compose up $SERVICES with $IMAGE_TAG (--remove-orphans)"

# 4. 健康检查(覆盖启动 + AutoMigrate 大库迁移)
for ((i = 1; i <= 60; i++)); do
  if curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then
    log "health check passed"
    prune_old_images || log "prune failed (non-fatal, deploy succeeded)"
    exit 0
  fi
  log "waiting for health ($i/60)..."
  sleep 3
done

# 5. 失败: 回滚到旧 tag
log "FATAL: health check failed, rolling back"
if [ -n "$OLD_TAG" ]; then
  sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$OLD_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans "$INSTANCE"
  log "rolled back to $OLD_TAG"
fi
exit 1
