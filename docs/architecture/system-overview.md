# 系统概览与域边界

> 文档类型：架构
>
> 状态：Active
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

## 系统形态

```
                        ┌─────────────────────┐
                        │     Casdoor (OIDC)   │  统一认证（数字用户 ID，已实测）
                        └──────────┬──────────┘
              OIDC/PKCE            │            OIDC 浏览器流
       ┌───────────────────────────┼───────────────────────────┐
       │                           │                           │
┌──────▼──────┐           ┌────────▼────────┐          ┌───────▼───────┐
│ apps/mobile │           │   apps/web      │          │ services/     │
│  Flutter    │           │   Vue 3         │          │ credit(二期)   │
└──────┬──────┘           └────────┬────────┘          └───────────────┘
       │                           │
       └──────────┬────────────────┘
                  │ JSON API (JWT Bearer)
           ┌──────▼──────┐     ┌──────────────┐
           │ apps/server  │────▶│ services/    │  Meilisearch 索引同步
           │ Go 后端      │     │ search       │
           └──────┬──────┘     └──────────────┘
                  │
           ┌──────▼──────┐
           │ PostgreSQL  │（选型待定，建议 PG15+）
           └─────────────┘
```

## 部署形态

- **单二进制**：web 构建产物 go:embed 进 server。开发时 vite 代理同源联调，
  生产零 CORS/nginx/CDN。
- 依赖服务（Casdoor/Meilisearch/PostgreSQL/Redis）以 docker-compose 编排，`services/` 只放
  部署配置不放源码。

## 域边界（apps/server internal 分层）

| 层 | 职责 | 依赖 |
|---|---|---|
| `http` | handlers、middleware、路由（gin） | service |
| `service` | 业务逻辑、事务编排、领域事件 | repository、domain |
| `repository` | 数据访问（DB 抽象，可换库） | domain |
| `domain` | 纯领域模型与业务规则（无 IO） | 无 |
| `auth` | Casdoor 验签、JWT 签发/校验 | config |
| `search` | Meilisearch 索引同步/查询代理 | config |
| `config` | 环境配置加载 | 无 |

**Boundary rules**
- 只有 `repository` 触碰 DB；禁止 service/http 直接 SQL。
- 跨域（如论坛→通知）走 owner 的公开 service API，禁止 foreign SQL。
- Web/Mobile 消费 api-contract 生成类型，禁止手写重复 DTO。

## 关键流程

### 认证（规划中）

- Web：标准 OIDC 授权码（浏览器重定向）→ server 换 token → 验签 → 本地用户 upsert → JWT。
- Mobile：appauth + PKCE → id_token → `POST /api/auth/oidc/exchange` → 论坛 JWT。
- 数字 ID 约束：`sub` 必须 uint64；server 侧强校验（见 identity-and-access.md）。

### 搜索（规划中）

- server 写帖/回帖时异步同步索引到 Meilisearch（topic/posts）。
- 搜索 API 由 server 代理（统一鉴权），web/mobile 共用；索引可全量重建。

### 积分（二期）

- credit 是 OIDC 客户端 + 独立账本；论坛作为商户调分发 API（见 credit-and-escrow.md）。

## 一致性原则

- PostgreSQL（或选定 DB）是业务事实源；搜索、缓存、计数、热榜、feed 都是可重建投影。
- 关键副作用（通知、索引同步、积分分发）幂等、可重试、可观测。
- 契约变更同 PR 更新 Go struct → openapi.yaml → 生成物 → fixture 测试。
