#!/usr/bin/env bash
# bootstrap-wiki-assets.sh — 幂等补齐 wiki 部署所需的服务器资产, 由 CI 在每次
# deploy-dev / deploy-main 部署时调用。存量服务器因此无需手动重跑 init-server.sh。
# 调用前 CI 需已 scp(strip_components: 1)上传:
#   deploy/build/wiki.Dockerfile  -> /tmp/build/wiki.Dockerfile
#   deploy/build/wiki.nginx.conf  -> /tmp/build/wiki.nginx.conf
#   deploy/docker-compose.yaml    -> /tmp/docker-compose.yaml
# 新服务器(尚无 /opt/yourtj 目录结构)仍以 init-server.sh 为准, 本脚本会 FATAL。
set -euo pipefail

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
[ -d "$ROOT" ] || { echo "bootstrap-wiki: FATAL: $ROOT missing (新服务器请先运行 init-server.sh)" >&2; exit 1; }

mkdir -p "$ROOT/build"

# 1. nginx 镜像构建资产(仓库为唯一事实源, 每次覆盖)
[ -f /tmp/build/wiki.Dockerfile ] || { echo "bootstrap-wiki: FATAL: /tmp/build/wiki.Dockerfile 未上传" >&2; exit 1; }
[ -f /tmp/build/wiki.nginx.conf ] || { echo "bootstrap-wiki: FATAL: /tmp/build/wiki.nginx.conf 未上传" >&2; exit 1; }
install -m 0644 /tmp/build/wiki.Dockerfile "$ROOT/build/wiki.Dockerfile"
install -m 0644 /tmp/build/wiki.nginx.conf "$ROOT/build/wiki.nginx.conf"
echo "bootstrap-wiki: build/wiki.Dockerfile + wiki.nginx.conf 已安装"

# 2. compose: 仅当缺少 wiki 服务时替换(保护运维在服务器侧对 compose 的本地修改)
if [ -f /tmp/docker-compose.yaml ] && ! grep -q '^  wiki-dev:' "$ROOT/docker-compose.yaml" 2>/dev/null; then
  install -m 0644 /tmp/docker-compose.yaml "$ROOT/docker-compose.yaml"
  echo "bootstrap-wiki: $ROOT/docker-compose.yaml 已替换(补齐 wiki-main/wiki-dev 服务)"
fi

# 3. .env: 幂等追加缺失的 WIKI_* 变量(deploy-wiki.sh 的 N2 断言要求 tag 行存在)
ENV_FILE="$ROOT/.env"
[ -f "$ENV_FILE" ] || { echo "bootstrap-wiki: FATAL: $ENV_FILE missing" >&2; exit 1; }
append_if_missing() {
  local key="$1" val="$2"
  if ! grep -q "^$key=" "$ENV_FILE"; then
    printf '%s=%s\n' "$key" "$val" >> "$ENV_FILE"
    echo "bootstrap-wiki: $ENV_FILE += $key=$val"
  fi
}
append_if_missing WIKI_MAIN_PORT 5284
append_if_missing WIKI_DEV_PORT 5285
append_if_missing WIKI_MAIN_TAG latest
append_if_missing WIKI_DEV_TAG latest

echo "bootstrap-wiki: done"
