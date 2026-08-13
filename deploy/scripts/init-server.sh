#!/usr/bin/env bash
# init-server.sh — 服务器一次性初始化(在目标服务器上以 root 运行)。
# 用法: sudo bash init-server.sh [main-domain] [dev-domain]
#   例: sudo bash init-server.sh https://forum.yourtj.de https://dev.yourtj.de
set -euo pipefail

MAIN_DOMAIN="${1:-https://forum.yourtj.de}"
DEV_DOMAIN="${2:-https://dev.yourtj.de}"

ROOT=/opt/yourtj
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. 依赖: 部署脚本需要 sqlite3 CLI 做 .backup 一致性快照
if ! command -v sqlite3 >/dev/null 2>&1; then
  apt-get update -qq && apt-get install -y -qq sqlite3
fi

# 2. 目录结构
mkdir -p "$ROOT"/{build,scripts,snapshots,main/storage,dev/storage}

# 3. 复制 Dockerfile / compose / 配置模板 / 脚本(源=目标时跳过)
copy_if_diff() {
  local src="$1" dst="$2"
  [ "$(realpath "$src")" = "$(realpath "$dst")" ] && return 0
  cp "$src" "$dst"
}
copy_if_diff "$SCRIPT_DIR/../Dockerfile" "$ROOT/build/Dockerfile"
copy_if_diff "$SCRIPT_DIR/../build/wiki.Dockerfile" "$ROOT/build/wiki.Dockerfile"
copy_if_diff "$SCRIPT_DIR/../build/wiki.nginx.conf" "$ROOT/build/wiki.nginx.conf"
copy_if_diff "$SCRIPT_DIR/../docker-compose.yaml" "$ROOT/docker-compose.yaml"
copy_if_diff "$SCRIPT_DIR/../config.toml.example" "$ROOT/config.toml.example"
for f in "$SCRIPT_DIR"/*.sh; do
  copy_if_diff "$f" "$ROOT/scripts/$(basename "$f")"
done
chmod +x "$ROOT/scripts/"*.sh
# 4. 生成/补齐 .env:
#    - 不存在时整文件生成
#    - 已存在(存量服务器)时逐条追加缺失的 WIKI_* 变量, 保证已有部署
#      首次 wiki 部署拿到 WIKI_MAIN_TAG 等, 否则 deploy-wiki.sh 的
#      sed 匹配不到 tag 行 → compose 回退 latest → pull 失败阻断部署(review F2)
if [ ! -f "$ROOT/.env" ]; then
  cat > "$ROOT/.env" <<EOF
MAIN_PORT=5234
DEV_PORT=5235
MAIN_TAG=latest
DEV_TAG=latest
WIKI_MAIN_PORT=5284
WIKI_DEV_PORT=5285
WIKI_MAIN_TAG=latest
WIKI_DEV_TAG=latest
EOF
  echo "init: $ROOT/.env created"
else
  append_if_missing() {
    local key="$1" val="$2"
    if ! grep -q "^$key=" "$ROOT/.env"; then
      printf '%s=%s\n' "$key" "$val" >> "$ROOT/.env"
      echo "init: $ROOT/.env += $key=$val"
    fi
  }
  append_if_missing WIKI_MAIN_PORT 5284
  append_if_missing WIKI_DEV_PORT 5285
  append_if_missing WIKI_MAIN_TAG latest
  append_if_missing WIKI_DEV_TAG latest
fi


# 5. 生成 config.toml(随机 signingKey, 域名参数化)
GEN_KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
for inst in main dev; do
  if [ ! -f "$ROOT/$inst/config.toml" ]; then
    domain="$MAIN_DOMAIN"
    [ "$inst" = "dev" ] && domain="$DEV_DOMAIN"
    sed -e "s|REPLACE_SIGNING_KEY|$GEN_KEY|" \
        -e "s|REPLACE_SERVER_URL|$domain|" \
      "$ROOT/config.toml.example" > "$ROOT/$inst/config.toml"
    echo "init: $ROOT/$inst/config.toml created (domain: $domain)"
  fi
done

# 6. 权限: 宿主部署用户 yourtj uid=1000 与容器内 app uid=1000 一致,
#    整体交给 1000:1000, 保证 CI 可写、容器可写 storage
chown -R 1000:1000 "$ROOT"

echo ""
echo "=== INIT DONE ==="
echo "  main: $ROOT/main   (port $(grep '^MAIN_PORT=' "$ROOT/.env" | cut -d= -f2), domain $MAIN_DOMAIN)"
echo "  dev:  $ROOT/dev    (port $(grep '^DEV_PORT=' "$ROOT/.env" | cut -d= -f2), domain $DEV_DOMAIN)"
echo "  下一步: 推送 dev 分支触发 CI 部署(需已配置 GitHub Secrets VM_HOST/VM_USER/VM_SSH_KEY)"
