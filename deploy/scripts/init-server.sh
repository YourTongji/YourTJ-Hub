#!/usr/bin/env bash
# init-server.sh — 服务器一次性初始化(在目标服务器上以 root 运行)。
# 用法: sudo bash init-server.sh [main-domain] [dev-domain]
#   例: sudo bash init-server.sh https://f.yourtj.de https://dev.yourtj.de
set -euo pipefail

MAIN_DOMAIN="${1:-https://f.yourtj.de}"
DEV_DOMAIN="${2:-https://dev.yourtj.de}"

ROOT=/opt/yourtj
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. 依赖: 部署脚本需要 sqlite3 CLI 做 .backup 一致性快照
if ! command -v sqlite3 >/dev/null 2>&1; then
  apt-get update -qq && apt-get install -y -qq sqlite3
fi

# 2. 目录结构
mkdir -p "$ROOT"/{scripts,snapshots,main/storage,dev/storage}

# 3. 复制 compose / 配置模板 / 脚本(源=目标时跳过)
copy_if_diff() {
  local src="$1" dst="$2"
  [ "$(realpath "$src")" = "$(realpath "$dst")" ] && return 0
  cp "$src" "$dst"
}
copy_if_diff "$SCRIPT_DIR/../docker-compose.yaml" "$ROOT/docker-compose.yaml"
copy_if_diff "$SCRIPT_DIR/../config.toml.example" "$ROOT/config.toml.example"

for f in "$SCRIPT_DIR"/*.sh; do
  copy_if_diff "$f" "$ROOT/scripts/$(basename "$f")"
done
chmod +x "$ROOT/scripts/"*.sh
# 4. 生成/补齐 .env:
#    - 不存在时整文件生成
#    - 已存在(存量服务器)时逐条追加缺失的 POSTGRES_* 变量, 保证已有部署
#      POSTGRES_PASSWORD 由 init 生成随机值(compose 要求非空; 部署默认 PG 主库)
PG_PASS="$(openssl rand -hex 16)"
MEILI_KEY="$(openssl rand -hex 16)"
if [ ! -f "$ROOT/.env" ]; then
  cat > "$ROOT/.env" <<EOF
MAIN_PORT=5234
DEV_PORT=5235
MAIN_TAG=latest
DEV_TAG=latest
IMAGE_REPO=ghcr.io/yourtongji/yourtj-hub
POSTGRES_USER=yourtj
POSTGRES_PASSWORD=$PG_PASS
POSTGRES_DB=postgres
MEILI_MASTER_KEY=$MEILI_KEY
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
  append_if_missing MAIN_PORT 5234
  append_if_missing DEV_PORT 5235
  append_if_missing MAIN_TAG latest
  append_if_missing DEV_TAG latest
  append_if_missing IMAGE_REPO ghcr.io/yourtongji/yourtj-hub
  append_if_missing MEILI_MASTER_KEY "$MEILI_KEY"
  append_if_missing POSTGRES_USER yourtj
  if grep -q "^POSTGRES_PASSWORD=" "$ROOT/.env" && [ -z "$(grep '^POSTGRES_PASSWORD=' "$ROOT/.env" | cut -d= -f2-)" ]; then
    sed -i.bak -E "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$PG_PASS/" "$ROOT/.env" && rm -f "$ROOT/.env.bak"
    echo "init: $ROOT/.env POSTGRES_PASSWORD 已生成随机值"
  else
    append_if_missing POSTGRES_PASSWORD "$PG_PASS"
  fi
  append_if_missing POSTGRES_DB postgres
fi


# 5. 生成 config.toml(随机 signingKey, 域名参数化, PG DSN)
GEN_KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
PG_USER="$(grep '^POSTGRES_USER=' "$ROOT/.env" | cut -d= -f2-)"
PG_PASS="$(grep '^POSTGRES_PASSWORD=' "$ROOT/.env" | cut -d= -f2-)"
MEILI_KEY="$(grep '^MEILI_MASTER_KEY=' "$ROOT/.env" | cut -d= -f2-)"
for inst in main dev; do
  if [ ! -f "$ROOT/$inst/config.toml" ]; then
    domain="$MAIN_DOMAIN"
    [ "$inst" = "dev" ] && domain="$DEV_DOMAIN"
    pg_dsn="host=postgres user=$PG_USER password=$PG_PASS dbname=yourtj_$inst port=5432 sslmode=disable"
    sed -e "s|REPLACE_SIGNING_KEY|$GEN_KEY|" \
        -e "s|REPLACE_SERVER_URL|$domain|" \
        -e "s|REPLACE_POSTGRES_DSN|$pg_dsn|" \
        -e "s|REPLACE_MEILI_KEY|$MEILI_KEY|" \
      "$ROOT/config.toml.example" > "$ROOT/$inst/config.toml"
    echo "init: $ROOT/$inst/config.toml created (domain: $domain, db: yourtj_$inst)"
  fi
done

# 5.2 存量 config 同步 MEILI key: init 为 .env 补 MEILI_MASTER_KEY 后,
#     旧 config 的 masterkey 若为空/占位符, 搜索会 401; 仅替换空/占位符值
for inst in main dev; do
  if [ -f "$ROOT/$inst/config.toml" ] && grep -qE '^\s*masterkey\s*=\s*""|^\s*masterkey\s*=\s*"REPLACE_MEILI_KEY"' "$ROOT/$inst/config.toml"; then
    sed -i.bak -E "s|^(\s*masterkey\s*=\s*)\"\"|\1\"$MEILI_KEY\"|; s|^(\s*masterkey\s*=\s*)\"REPLACE_MEILI_KEY\"|\1\"$MEILI_KEY\"|" "$ROOT/$inst/config.toml" && rm -f "$ROOT/$inst/config.toml.bak"
    echo "init: $ROOT/$inst/config.toml masterkey 已同步 .env 的 MEILI_MASTER_KEY"
  fi
done

# 5.5 启动 postgres 容器并创建实例数据库(main/dev 隔离, 幂等; 供部署默认 PG 主库使用)
PG_USER="$(grep '^POSTGRES_USER=' "$ROOT/.env" | cut -d= -f2-)"
docker compose --env-file "$ROOT/.env" -f "$ROOT/docker-compose.yaml" up -d postgres meilisearch
for _ in $(seq 1 30); do
  docker exec yourtj-postgres pg_isready -U "$PG_USER" >/dev/null 2>&1 && break
  sleep 2
done
for pgdb in yourtj_main yourtj_dev; do
  if ! docker exec yourtj-postgres psql -U "$PG_USER" -d postgres -tAc \
    "SELECT 1 FROM pg_database WHERE datname='$pgdb'" | grep -q 1; then
    docker exec yourtj-postgres psql -U "$PG_USER" -d postgres -c "CREATE DATABASE $pgdb"
    echo "init: postgres database $pgdb created"
  else
    echo "init: postgres database $pgdb exists"
  fi
done



# 6. 权限: 宿主部署用户 yourtj uid=1000 与容器内 app uid=1000 一致,
#    整体交给 1000:1000, 保证 CI 可写、容器可写 storage
chown -R 1000:1000 "$ROOT"

echo ""
echo "=== INIT DONE ==="
echo "  main: $ROOT/main   (port $(grep '^MAIN_PORT=' "$ROOT/.env" | cut -d= -f2), domain $MAIN_DOMAIN)"
echo "  dev:  $ROOT/dev    (port $(grep '^DEV_PORT=' "$ROOT/.env" | cut -d= -f2), domain $DEV_DOMAIN)"
echo ""
echo "=== 配置治理: 需回填到 GitHub Environments secrets 的值（CI 渲染 config 用） ==="
echo "  production 环境: PG_DSN / SIGNING_KEY / MEILI_MASTER_KEY / WIKI_WEBHOOK_SECRET"
echo "                   VM_HOST / VM_USER / VM_SSH_KEY / VM_SSH_PORT"
echo "                   GH_CLIENT_ID / GH_CLIENT_SECRET（GitHub OAuth, 生产启用）"
echo "  dev 环境:       同 PG_DSN / SIGNING_KEY / MEILI_MASTER_KEY / WIKI_WEBHOOK_SECRET + VM_*;"
echo "                   GitHub 凭据不填（DB siteUrl 无环境隔离, dev 保持空）"
echo "  AI_API_KEY(可选): 两环境都可留空（AI 总结默认关闭）"
echo "  命令示例（在本地持 GH token 处执行, 值从服务器读取, 勿回显）:"
echo "    ssh root@<host> \"awk '/^\\\\[db.default\\\\]/{f=1} f&&/^url =/{print;exit}' /opt/yourtj/main/config.toml\" \\"
echo "      | gh secret set PG_DSN --env production"
echo ""
echo "  下一步: 推送 dev 分支触发 CI 部署（渲染 config 随镜像下发; apply-config.sh 做健康回滚）"
