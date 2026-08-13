# 容器部署（main/dev 接入）

YourTJ wiki 已接入现有 deploy 链路：与论坛后端同构，通过 GitHub Actions
构建静态产物 → 上传服务器 → 构建 nginx 镜像 → compose 部署，失败自动回滚。

## 触发方式

| 环境 | 触发 | 端口 |
|---|---|---|
| **dev**（`wiki-dev`）| push dev 分支（deploy-dev.yml 自动）| `127.0.0.1:5285` |
| **main**（`wiki-main`）| release-to-main → deploy-main.yml（手动）| `127.0.0.1:5284` |

## 部署流程

```
CI wiki-build job: node20 + pnpm → cd wiki && pnpm build → tar 打包 dist
        ↓ scp 上传 wiki-dist.tar.gz 到服务器 /tmp
服务器 deploy-wiki.sh <instance> <tarball> <tag> <port>:
  1. 解包到 /opt/yourtj/build/wiki-dist
  2. docker build（nginx:alpine + dist + wiki.nginx.conf）→ yourtj-wiki:<tag>
  3. .env 更新 WIKI_<INSTANCE>_TAG → compose up wiki-<instance>
  4. 首页 HTTP 200 健康检查（30×2s）→ 失败回滚 prev tag
  5. IMAGE_KEEP_N 清理旧 wiki 镜像（前缀 wiki-main-*/wiki-dev-*）
```

## 镜像与配置

- **`deploy/build/wiki.Dockerfile`**：`FROM nginx:1.27-alpine`，COPY dist + nginx.conf
- **`deploy/build/wiki.nginx.conf`**：
  - `try_files $uri $uri/ /index.html`（VitePress SPA 路由回退）
  - `/assets/` 30 天 immutable 长缓存（内容 hash）
  - `/pagefind/` 1 小时缓存（离线搜索索引随内容更新，新版本部署后索引延迟可见）
  - 其余 html `no-cache`（内容更新即时可见）+ gzip
- **docker-compose.yaml**：`wiki-main` / `wiki-dev` 服务，127.0.0.1 暴露，
  wget 容器内 80 端口健康检查
- **环境变量**（服务器 `/opt/yourtj/.env`，init-server.sh 自动生成）：
  `WIKI_MAIN_TAG` / `WIKI_DEV_TAG`（默认 latest）、`WIKI_MAIN_PORT=5284` / `WIKI_DEV_PORT=5285`

## 评论（Waline）注入

CI wiki-build job 构建时注入 `VITE_WALINE_SERVER_URL`（GitHub secret
`WIKI_WALINE_SERVER_URL`）；secret 未配置时为空 → 构建不失败、站点无评论。
Waline 服务端部署见 [Waline 评论服务](./waline)。

## ⚠️ 上线前提（仓库外操作，需运维配合）

1. **服务器 nginx 反代**（宿主机配置，非容器内）：
   - `wiki.yourtj.de` → `127.0.0.1:5284`（main）
   - `dev-wiki.yourtj.de` → `127.0.0.1:5285`（dev）
2. **GitHub Secrets**：`WIKI_WALINE_SERVER_URL`（可选，评论服务地址）
3. **服务器 wiki 资产**：无需手动操作。deploy-dev/main 每次部署都会 scp 上传
   `wiki.Dockerfile`、`wiki.nginx.conf` 与 `docker-compose.yaml`，并由
   `bootstrap-wiki-assets.sh` 幂等安装（compose 仅在缺 wiki 服务时替换、
   `.env` 逐条追加缺失的 `WIKI_*` 变量），首次 wiki 部署即可自动创建
   `wiki-main`/`wiki-dev` 容器。

## 验证清单（部署后）

- [ ] `curl -I http://127.0.0.1:5284/` → 200，`/pagefind/pagefind.js` 200
- [ ] SPA 路由：`/guide/yourtj-wiki` → 200（回退 index.html）
- [ ] `/assets/` 响应头含 `Cache-Control: immutable`
- [ ] 反代域名访问正常 + 评论（若配置 Waline）
