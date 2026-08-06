# 本地环境

> 文档类型：开发指南
>
> 状态：Active
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

## 依赖

- Go 1.26+
- Node 24 + pnpm 11（**注意**：主目录 `/Users/yzxoi/pnpm-workspace.yaml` 会干扰 pnpm
  向上查找；仓库内 `pnpm-workspace.yaml` 必须存在，勿删）
- Docker + Compose（本地依赖服务）
- Flutter SDK（移动端规划中，当前未安装）

## 启动

```bash
# 1. 起本地依赖（postgres + meilisearch + mariadb + casdoor）
make dev

# 2. 起后端（:8080）
make server

# 3. 起前端 dev（:5173，代理 /api 到 :8080）
make web

# 4. 生产构建：web → webdist → server 单二进制
make build
```

## 服务地址

| 服务 | 地址 | 说明 |
|---|---|---|
| server | http://localhost:8080 | `/healthz`、`/api/ping` |
| web dev | http://localhost:5173 | vite，代理 /api |
| casdoor | http://localhost:8001 | 统一认证（admin/123，开发） |
| meilisearch | http://localhost:7700 | master key: `yourtj-dev-master-key` |
| postgres | localhost:5432 | yourtj/yourtj，库 yourtj |
| mariadb | localhost:13306 | casdoor 专用 |

## 移动端连后端

- iOS 模拟器：`http://localhost:8080` 直连
- Android 模拟器：`http://10.0.2.2:8080`
- 真机：局域网 IP（dart-define 注入 baseUrl，移动端落地时）

## 环境变量

复制 `deploy/env.example` 为 `deploy/.env` 并按环境修改。server 通过环境变量读配置
（`YOURTJ_ENV`/`YOURTJ_PORT`，后续扩展 DB/Meili/Casdoor）。

## 已知问题

- Go module 拉取：官方 proxy 偶发超时，用 `GOPROXY=https://goproxy.cn,direct`。
- pnpm `ERR_PNPM_IGNORED_BUILDS`：esbuild 已在 `pnpm-workspace.yaml` 放行；新增原生依赖时
  需同步更新 `allowBuilds`。
