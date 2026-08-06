# 当前能力与差距

> 文档类型：实现盘点
>
> 状态：Active
>
> 负责人：Product owner、Platform maintainers
>
> 最近核验：2026-08-06

本盘点以当前源码、OpenAPI（占位）和骨架为基线，说明已经存在什么、哪里只有骨架、
哪些界面承诺与实际行为不一致。后续 PR 改变这些结论时，必须在同一 PR 同步更新本文件。

## 已有的坚实基础

- Go 1.26 + Gin 后端骨架（`apps/server`），健康检查 `/healthz`、探活 `/api/ping` 可运行。
- Web 产物 go:embed 进 server 单二进制，SPA history fallback 已实测。
- Vue 3 + Vite + TS Web 骨架（`apps/web`），pnpm workspace 配置完成，typecheck/build 通过。
- monorepo 结构（apps/packages/services/deploy/docs）+ CI（server/web/contract 三 workflow）。
- docker-compose 本地依赖定义（postgres/meilisearch/mariadb/casdoor），Casdoor 数字 ID 链路
  已在调研阶段实测通过（sub=数字 ID，Incremental 规则 + 显式数字 id 双路径验证）。
- 文档中心（docs/README 事实来源表 + 实现状态词）、AGENTS.md。

## 当前关键差距

| 领域 | 状态 | 已验证问题 |
|---|---|---|
| 骨架与构建 | `Current` | 单二进制构建/运行/SPA fallback 已实测；`make build` 全流程通过 |
| 数据库 | `Decision needed` | 建议 PostgreSQL 15+ + golang-migrate；未落地，无迁移、无 repository 实现 |
| 认证 | `Planned` | Casdoor 数字 ID 链路已验证，但 server 尚未集成（无 exchange 端点、无 JWT 签发） |
| 论坛核心 | `Planned` | 无板块/主题/评论/通知/私信 API，无 domain/service/repository 实现 |
| 契约 | `Partial` | openapi.yaml 为占位（`paths: {}`）；swag 注解、gen-ts/gen-dart、fixture 契约测试均未接入 |
| Web 页面 | `Partial` | 仅有 Home 占位页；无列表/详情/发帖/登录/管理页面 |
| 移动端 | `Planned` | `apps/mobile` 仅目录占位；Flutter/melos/Riverpod 均未搭建 |
| 搜索 | `Planned` | Meilisearch 服务定义在 compose 中，未接入 server 索引同步与搜索 API |
| 积分 | `Planned` | services/credit 仅 README 占位；明确二期，本期不实现 |
| 媒体 | `Planned` | 无上传、OSS/存储、asset 管理实现 |
| 通知 | `Planned` | 无 outbox、无通知 API、无推送通道 |
| 治理 | `Planned` | 无 RBAC/审核/审计/管理后台实现 |
| 测试 | `Partial` | server 无业务测试（仅骨架）；web 有 typecheck/build；无契约测试、无 E2E |

## 正确性优先

在铺开新功能前，先闭合以下基线（避免在错误地基上扩展）：

1. 数据库选型决策并落地迁移框架（避免在 SQLite 上先写业务再迁移）。
2. 认证链路闭环（Casdoor → exchange → JWT），数字 ID 约束在 server 侧强校验。
3. 契约管线（swag → openapi.yaml → TS/Dart 生成）先于业务 API 铺开，防止契约漂移。
