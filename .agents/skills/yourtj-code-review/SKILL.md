---
name: yourtj-code-review
description: Review yourtj-hub changes against user needs, research, acceptance evidence and repository engineering constraints. Use for self-review or PR review, including requirement gaps, behavior, contracts, migrations, security and test value. Use yourtj-pre-push-checks for running publication checks and yourtj-development for implementation.
---

# Reviewing YourTJ Hub Changes

按风险面分层评审；每层给出阻塞项（blocker，必须修）与建议项（should/nit，可以修）。证据优先：
读代码与测试，不凭记忆。

## Review scope and product intent

- 先确认评审对象、base/head 与实际 diff；工作区自评包含未提交改动。阅读相关 Issue、PR、所属文档
  和必要的上下文。纯评审不改代码、不发布评论或提交 GitHub 审批，除非用户已授权相应动作。
- 按 [Issue/PR 标准](../../../docs/development/pull-requests.md) 评估现状与问题证据、外部调研的
  可信性/适用性、目标用户及不同受影响角色的故事。修复和维护按该标准简写，不强求竞品或虚构角色。
- 把 `US-*` / 缺陷场景、`AC-*`、实现和证据对应起来。检查被动受影响者、角色权限冲突、
  不在范围内的扩张及尚未解决的产品决定；不能只验证“代码实现了作者想写的东西”。
- 保留 PR 的摘要、行为变更、验证、文档与契约影响、已知缺口；产品材料补充这些工程信息。
  若已有 Issue 提供详情，概括与链接足够，不要求重复整篇需求。
- 对 UI 改动检查相关平台的入口、状态、反馈、可访问性和恢复路径，核对操作/视觉证据。
  无法实测时明确限制，不把代码阅读写成体验验收通过。

## Contract impact

- 改动是否触及 OpenAPI 覆盖的操作？是 → 检查 `packages/api-contract/openapi.yaml`、生成 TypeScript 输出
  （`apps/gooseforum/resource/packages/client/src/gen`）、fixture 契约测试是否同 PR 同步；未完全覆盖的操作
  还需同步 mobile Dart 镜像（`apps/mobile/packages/core/lib/src/gen/`）与手动维护的 web TS 类型
  （`apps/gooseforum/resource/packages/client/src/contracts/`）。
- 路由/请求/响应结构是否改变？与 `packages/api-contract/openapi.yaml` 或 `app/http/controllers` 实际行为核对。
- 新路由是否同步 routes snapshot、coverage matrix，且有 operation 或明确 exclusion？相关 token、
  serverMessages 等跨端镜像是否按 `AGENTS.md` 同步？

## Database & migration safety

- 模型/迁移改动必须过 PG 门禁：`YOURTJ_TEST_PG_URL` 下跑 `PostgreSQL|Postgres`（覆盖 `app/migration/` 全部 PG 用例；完整 CI 命令见 testing.md）。
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

- 以验收标准和真实回归风险选证据，遵守 [testing.md](../../../docs/development/testing.md)。
  覆盖改变的关键行为、权限、边界与失败恢复；不要求穷举所有实现分支，不把覆盖率作为通过条件。
- Bug 修复应有先红后绿证据（机械改动豁免）；测试应在行为被破坏时失败，不只复述实现。
- 在最低但足够真实的边界证明行为；只有跨层风险需要额外集成验证，避免多层重复同一断言。
- 量级敏感时使用能越过阈值的最小数据；SQL 方言风险使用真实 PostgreSQL。跳过的 PG 测试不算通过。
- 仅在涉及批处理/回填/数据库失败恢复时读取
  [test-coverage-examples.md](references/test-coverage-examples.md)，示例不是所有变更的强制清单。
- `AC-*` 的未验证/失败项与 CI 状态必须如实记录；文档和模板改动通常使用语义核对与现有校验即可。

## Documentation and delivery

核对所属文档、状态词和 PR 描述是否反映最终实现；持久取舍按
[MADR 规则](../repo-decisions/SKILL.md) 记录。按实际影响检查迁移、兼容、部署/恢复步骤及补充材料。
用 [repo-review](../repo-review/SKILL.md) 的通用语义和来源检查补足本技能的项目检查；不重复报告同一发现。

## Output

- 先给高影响发现：位置（代码文件+行，或 Issue/PR 栏目/AC 编号）、严重级、触发条件、用户/系统后果、
  支持证据和最小修正方向。需求/证据缺口与代码缺陷分开；不把疑问包装成已证实缺陷。
- 严重级按 [Review contract](../../../docs/development/pull-requests.md#review-contract)：
  blocker 必须解决才能就绪，should 为有依据的非阻塞改进，nit 为可选表述/风格。
  不能只因少一个标题或个人偏好而阻塞。
- 汇总已覆盖的需求/风险、实际执行的检查及未验证限制，给出就绪/需修改/证据不足的结论。
  无发现则明确说明，无须凑数。本地评审结论不等于 GitHub approval 或 CI 通过。
