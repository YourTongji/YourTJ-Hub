#!/usr/bin/env bash
# init-server.sh — 服务器一次性初始化(在目标服务器上以 root 运行)。
# 用法: sudo bash init-server.sh [main-domain] [dev-domain]
#   例: sudo bash init-server.sh https://forum.yourtj.de https://dev.yourtj.de
set -euo pipefail

MAIN_DOMAIN="${1:-https://forum.yourtj.de}"
DEV_DOMAIN="${2:-https://dev.yourtj.de}"

ROOT=/opt/yourtj
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. 目录结构
mkdir -p "$ROOT"/{build,scripts,snapshots,main/storage,dev/storage}

# 2. 复制 Dockerfile / compose / 配置模板 / 脚本
cp "$SCRIPT_DIR/../Dockerfile" "$ROOT/build/Dockerfile"
cp "$SCRIPT_DIR/../docker-compose.yaml" "$ROOT/docker-compose.yaml"
cp "$SCRIPT_DIR/../config.toml.example" "$ROOT/config.toml.example"
cp "$SCRIPT_DIR"/*.sh "$ROOT/scripts/"
chmod +x "$ROOT/scripts/"*.sh

# 3. 生成 .env(不存在时)
if [ ! -f "$ROOT/.env" ]; then
  cat > "$ROOT/.env" <<'EOF'
MAIN_PORT=5234
DEV_PORT=5235
MAIN_TAG=latest
DEV_TAG=latest
EOF
  echo "init: $ROOT/.env created"
fi

# 4. 生成 config.toml(随机 signingKey, 域名参数化)
GEN_KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
for inst in main dev; do
  if [ ! -f "$ROOT/$inst/config.toml" ]; then
    domain="$MAIN_DOMAIN"
    [ "$inst" = "dev" ] && domain="$DEV_DOMAIN"
    sed -e "s|REPLACE_SIGNING_KEY|$GEN_KEY|" -e "s|REPLACE_SERVER_URL|$domain|" \
      "$ROOT/config.toml.example" > "$ROOT/$inst/config.toml"
    echo "init: $ROOT/$inst/config.toml created (domain: $domain)"
  fi
done

# 5. 权限: 容器内进程 uid 1000, 需要可写 storage
chown -R 1000:1000 "$ROOT/main/storage" "$ROOT/dev/storage" 2>/dev/null || true

echo ""
echo "=== INIT DONE ==="
echo "  main: $ROOT/main   (port $MAIN_PORT, domain $MAIN_DOMAIN)"
echo "  dev:  $ROOT/dev    (port $DEV_PORT, domain $DEV_DOMAIN)"
echo "  下一步: 在 GitHub 配置 Secrets (VM_HOST/VM_USER/VM_SSH_KEY), 推送 dev 分支触发部署"
