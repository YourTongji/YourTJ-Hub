---
name: yourtj-code-review
description: Use when reviewing a yourtj-hub code change (own or others') before merge — checking contract impact, database migration safety, security boundaries, performance, test coverage, and maintainability against the repository's hard constraints. Not for running toolchain checks (use $yourtj-pre-push-checks) or implementing changes (use $yourtj-development).
---

# Reviewing YourTJ Hub Changes

按风险面分层评审；每层给出阻塞项（blocker，必须修）与建议项（should/nit，可以修）。证据优先：
读代码与测试，不凭记忆。

## Contract impact

- 改动是否触及 OpenAPI 覆盖的操作？是 → 检查 `packages/api-contract/openapi.yaml`、生成 TypeScript 输出
  （`apps/gooseforum/resource/packages/client/src/gen`）、fixture 契约测试是否同 PR 同步；未完全覆盖的操作
  还需同步 mobile Dart 镜像（`apps/mobile/packages/core/lib/src/gen/`）与手动维护的 web TS 类型
  （`apps/gooseforum/resource/packages/client/src/contracts/`）。
- 路由/请求/响应结构是否改变？与 `packages/api-contract/openapi.yaml` 或 `app/http/controllers` 实际行为核对。

## Database & migration safety

- 模型/迁移改动必须过 PG 门禁：`YOURTJ_TEST_PG_URL` 下跑 `TestSchema*`（`app/migration/migration_pg_test.go`）。
- 模型禁止 MySQL-only 类型标签（`bigint unsigned` / `datetime` / `tinyint`）。
- 迁移是否 append-only？backfill/数据迁移是否可重入、有游标/幂等？删除生命周期（`contentdeleteservice`）
  是否覆盖关联数据（附件、搜索索引、通知、审计）？

## Security boundaries

- auth/PII/隐私/保留/审计：会话（`jti` + `user_sessions`）、TOTP、OIDC Provider、GitHub OAuth 路径。
- 用户 ID 必须 numeric（uint64）；forum JWT 是会话凭证不是身份真相，不发给外部 OIDC 客户端。
- `config.toml` 含 signingKey —— 永不提交。权限白名单、越权（水平/垂直）、限流、文件上传类型与路径、
  搜索注入、外部动作（通知/webhook/CDN/Meilisearch 写入）的来源验证与幂等。

## Performance & resource risk

- 热路径（topic/post 列表、搜索、通知、事件处理）是否有 N+1、全表扫描、无界内存、锁竞争？
- 事件驱动搜索同步：幂等、重试、死信？后台 worker 并发与 backoff？

## Maintainability

- 分层边界：bundles → models → service → http/controllers；跨域访问走 owner 公共 API，无外域 SQL。
- 命名/重复/死代码/投机抽象；TODO 三档（`FIXME` 阻塞发布 / `TODO` 近期 / `XXX` 远期）。
- 与周围模式一致；新抽象是否降低净复杂度。

## Test coverage

- 测试要覆盖代码实际存在**所有可能的分支**。
- 边界与量级敏感点必须显式测。
- 失败路径可观测。
- 数据量要贴近真实。
- 通用示例与注解见本技能 `references/test-coverage-examples.md`。评审时对照示例理解规则意图，再在改动中找等价的实际用例。

## Output

- 每个发现：文件+行、严重级（blocker / should / nit）、理由、建议修法。
- blocker 必须修才能合；should 尽量修或明确记录；nit 可留。
- 汇总：整体就绪度（approve / needs changes）、剩余风险、需 reviewer 关注点。
