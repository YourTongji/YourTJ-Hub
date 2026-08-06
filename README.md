# YourTJ-Hub

同济大学校级论坛平台 monorepo（品牌：yourtj，Go + Vue + Flutter，统一认证 Casdoor，积分 credit 二期）。

## 文档

**[文档中心 → docs/README.md](docs/README.md)**（事实来源表 + 实现状态词）

- 产品：[愿景与原则](docs/product/vision-and-principles.md) · [当前能力与差距](docs/product/current-state.md) · [身份与登录](docs/product/identity-and-access.md) · [积分](docs/product/credit-and-escrow.md)
- 架构：[系统概览与域边界](docs/architecture/system-overview.md) · [契约与数据](docs/architecture/contracts-and-data.md)
- 开发：[开发入口](docs/development/README.md) · [本地环境](docs/development/local-development.md) · [测试](docs/development/testing.md) · [PR](docs/development/pull-requests.md) · [文档治理](docs/development/documentation.md)
- 运维：[部署与发布](docs/operations/deployment.md)

开发前必须阅读 [AGENTS.md](AGENTS.md) 和需求对应的文档。仓库级 `$yourtj-development` skill 位于 `.agents/skills/yourtj-development`。

## 结构

```
yourtj-hub/
├── apps/                  # 可独立部署的应用
│   ├── server/            # Go 论坛后端 API（web 产物 go:embed 进单二进制）
│   ├── web/               # Vue 3 Web 端源码（构建产物 → server/webdist）
│   └── mobile/            # Flutter（melos workspace：core/auth/ui_kit/forum_app，规划中）
├── packages/
│   └── api-contract/      # openapi.yaml 契约中心（swag 接入后生成）
├── services/              # 基础服务部署配置（casdoor / search / credit）
├── third_party/           # 第三方参考源码快照（gooseforum，不参与构建）
├── deploy/                # 分环境部署配置
└── docs/                  # 文档中心（product/architecture/development/operations）
```

## 参考上游（Reference）

本仓库**不 fork** [GooseForum](https://github.com/leancodebox/GooseForum)（MIT），以其为参考蓝本自研，
允许在数据库、搜索和结构上大改。上游源码快照保留在
[`third_party/gooseforum/`](third_party/gooseforum/)，供实现时对照借鉴（架构分层、数据模型、
三模渲染、契约组织等）。

- 上游仓库：https://github.com/leancodebox/GooseForum
- 快照版本：`63949f2d`（2026-08-04，feat(resource): extract reusable client SDK）
- 许可证：MIT（见 `third_party/gooseforum/LICENSE`）
- 更新方式：手动 rsync 更新快照并记录新的上游 commit（上游不活跃，按需更新）

> 注意：快照仅供参考，**不参与构建**，也不作为事实来源（见 [docs/README.md](docs/README.md) 事实来源表）。

## 快速开始

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

详见 [docs/development/local-development.md](docs/development/local-development.md)。

## 决策速览

| 主题 | 决策 | 记录 |
|---|---|---|
| 代码组织 | apps/server + apps/web 源码分目录，部署合并（go:embed 单二进制） | 决策记录见 note |
| 认证 | Casdoor 统一认证（OIDC，数字用户 ID，已实测） | docs/product/identity-and-access.md |
| 状态管理（移动端） | Riverpod | docs/architecture/system-overview.md |
| 数据库 / 搜索 | 待定（建议 PostgreSQL + Meilisearch） | 待决策，记录见 note |

## 当前状态

实现状态与差距见 [docs/product/current-state.md](docs/product/current-state.md)（用 `Current`/`Partial`/`Planned`/`Decision needed` 标注，不写时间线）。
