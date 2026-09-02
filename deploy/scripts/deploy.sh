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
#   CONFIG_FILE  — (可选) 渲染好的 config.toml 产物绝对路径; 携带时与镜像同事务
#                  原子替换实例 config(先备份 .prev, 健康失败一并恢复)。
set -euo pipefail

INSTANCE="${1:?usage: deploy.sh <instance> <image-tag> [health-port]}"
IMAGE_TAG="${2:?usage: deploy.sh <instance> <image-tag> [health-port]}"
PORT="${3:-5234}"

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
ENV_FILE="$ROOT/.env"
COMPOSE_FILE="$ROOT/docker-compose.yaml"
TAG_VAR="$([ "$INSTANCE" = "main" ] && echo MAIN_TAG || echo DEV_TAG)"

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

# IMAGE_REPO 优先取 .env(与 compose 一致), 未设置时用默认 GHCR 公开仓库
IMAGE="$(grep -E '^IMAGE_REPO=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2- || true)"
IMAGE="${IMAGE:-ghcr.io/yourtongji/yourtj-hub}"
log "image repo: $IMAGE"

# 1. 记录当前 tag 用于回滚。
#    prev 仅代表"本脚本最近一次成功部署的镜像": 先清掉旧的 prev 引用,
#    避免首次部署/镜像被 prune 后, 失败回滚时 prev 指向无关或已不存在的镜像;
#    没有可回滚的 prev 时回滚步骤自然跳过(只改 .env, 容器保持旧镜像)。
OLD_TAG="$(grep -E "^$TAG_VAR=" "$ENV_FILE" | cut -d= -f2 || true)"
if [ -n "$OLD_TAG" ] && docker image inspect "$IMAGE:$OLD_TAG" >/dev/null 2>&1; then
  docker image rm "$IMAGE:prev" >/dev/null 2>&1 || true
  docker tag "$IMAGE:$OLD_TAG" "$IMAGE:prev" >/dev/null 2>&1 || true
  log "saved previous image tag: $OLD_TAG (as prev)"
else
  log "no previous local image for $OLD_TAG; rollback will only revert .env tag"
fi
# 2. 拉取新镜像(GHCR 公开镜像, 匿名 pull)
docker pull "$IMAGE:$IMAGE_TAG"
log "pulled image $IMAGE:$IMAGE_TAG"

# 2.5 可选: 携带渲染 config 时原子替换(与镜像同一回滚事务)。
#     单文件 bind-mount 持有旧 inode, 容器需在 config 变更时重建才重挂新文件;
#     下方步骤 3 的 compose up 因新镜像 tag 变更会 recreate 容器, 若 tag 未变
#     而 config 已替换, 则显式 force-recreate。
CONFIG_APPLIED=""
if [ -n "${CONFIG_FILE:-}" ]; then
  [ -f "$CONFIG_FILE" ] || { log "FATAL: CONFIG_FILE=$CONFIG_FILE 不存在"; exit 1; }
  if grep -q '{{' "$CONFIG_FILE"; then
    log "FATAL: CONFIG_FILE 含未替换占位符 {{, 拒绝应用"; exit 1
  fi
  INST_DIR="$ROOT/$INSTANCE"
  CFG="$INST_DIR/config.toml"
  PREV="$CFG.prev"
  CFG_MARKER="$INST_DIR/.config.sha256"
  NEW_SHA="$(sha256sum "$CONFIG_FILE" | awk '{print $1}')"
  CUR_SHA="$(sha256sum "$CFG" 2>/dev/null | awk '{print $1}' || true)"
  if [ -n "$CUR_SHA" ] && [ "$CUR_SHA" = "$NEW_SHA" ]; then
    log "config 与现网一致(sha=$NEW_SHA), 跳过替换"
  else
    [ -e "$PREV" ] && { log "FATAL: $PREV 已存在(并发 apply 或残留), 拒绝覆盖回滚点"; exit 1; }
    cp -f "$CFG" "$PREV" || { log "FATAL: 备份 config 失败"; exit 1; }
    cp -f "$CONFIG_FILE" "$CFG.new" || { log "FATAL: 写入 $CFG.new 失败"; rm -f "$CFG.new"; exit 1; }
    mv -f "$CFG.new" "$CFG"
    CONFIG_APPLIED=1
    log "config 已原子替换(prev 已备份); sha=$NEW_SHA"
  fi
fi

# 3. 更新 .env tag 并启动实例。
#    反代由 1Panel 负责(公网 TLS 终止 + 回源宿主机端口), 本 compose 不含 nginx 服务。
sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$IMAGE_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
if [ -n "$CONFIG_APPLIED" ]; then
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans --force-recreate --no-deps "$INSTANCE"
  log "compose recreate $INSTANCE with $IMAGE_TAG (--remove-orphans --force-recreate for new config)"
else
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans "$INSTANCE"
  log "compose up $INSTANCE with $IMAGE_TAG (--remove-orphans)"
fi

# 4. 健康检查(覆盖启动 + AutoMigrate 大库迁移)
health_ok=0
for ((i = 1; i <= 60; i++)); do
  if curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then
    health_ok=1
    break
  fi
  log "waiting for health ($i/60)..."
  sleep 3
done

if [ "$health_ok" -eq 1 ]; then
  log "health check passed"
  # 若本次替换过 config, 成功即记录 marker 并释放 prev 单槽
  if [ -n "$CONFIG_APPLIED" ]; then
    printf '%s\n' "$NEW_SHA" > "$CFG_MARKER"
    rm -f "$PREV"
    log "config marker 已记录 $NEW_SHA; prev 已清理"
  fi
  prune_old_images || log "prune failed (non-fatal, deploy succeeded)"
  exit 0
fi

# 5. 失败: 回滚到旧 tag(prev 不可用时仅改 .env, 容器保持旧镜像)
log "FATAL: health check failed, rolling back"

# 5.0 若本次替换过 config, 先恢复(与镜像同一回滚事务)
if [ -n "$CONFIG_APPLIED" ] && [ -f "$PREV" ]; then
  mv -f "$PREV" "$CFG"
  log "restored previous config.toml from config.toml.prev"
fi

# 5.1 若 main 部署前备份了 compose(main workflow 更新共享 compose 时),
#     一并恢复旧 compose, 避免新的 mounts/env/网络配置残留(PR review #4)
if [ -f "$ROOT/docker-compose.yaml.prev" ]; then
  cp -f "$ROOT/docker-compose.yaml.prev" "$ROOT/docker-compose.yaml"
  log "restored previous compose file from docker-compose.yaml.prev"
fi
if [ -n "$OLD_TAG" ]; then
  if docker image inspect "$IMAGE:$OLD_TAG" >/dev/null 2>&1; then
    sed -i.bak -E "s/^$TAG_VAR=.*/$TAG_VAR=$OLD_TAG/" "$ENV_FILE" && rm -f "$ENV_FILE.bak"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans "$INSTANCE"
    log "rolled back to $OLD_TAG"
  else
    log "WARNING: $IMAGE:$OLD_TAG not present locally, cannot roll back container; .env still points at new tag $IMAGE_TAG"
  fi
else
  log "WARNING: no previous tag recorded, cannot roll back"
fi

# 5.3 成功回滚镜像后, 若 config 也被恢复过, 容器需重建以重挂旧 config
if [ -n "$CONFIG_APPLIED" ]; then
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --remove-orphans --force-recreate --no-deps "$INSTANCE" >/dev/null 2>&1 || true
  log "recreated $INSTANCE to re-mount restored config"
fi
exit 1
