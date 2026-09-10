# dev 实例 DB 站点设置无环境隔离（已知边界）

## Status
Accepted
Class: architecture

## Context and Problem Statement

dev 每次部署前从 main 同步一致性 DB 快照（预演迁移机制），`page_config.siteSettings` 等站点设置随之携带 main 的值（siteUrl = f.yourtj.de、SMTP 密码等）。因此 dev 若启用 GitHub OAuth 凭据，回调会指回生产 f.yourtj.de，两端会话/绑定语义串扰；SMTP 等生产密钥也在 dev 生效。

## Decision Drivers

- 快照机制是迁移预演的核心价值，不可为隔离而放弃。
- 环境隔离牵扯管理端可配置项模型与快照机制改造，非本期能承载。

## Considered Options

- 本期不迁移 DB 站点设置到环境隔离，记录为已知边界 + 门控放行（选定）。
- 立即实现站点设置的 per-instance 覆盖。
- dev 停用全部生产敏感键。

## Decision Outcome

本期不迁移 DB 站点设置到环境隔离（产品语义变更，牵扯管理端可配置项与快照机制）；dev 的 `[github]` 凭据保持空（渲染 allow-empty），文档标注。后续单独立项：siteUrl 类键改为读 `[server].url` 或支持环境覆盖，SMTP/对象存储等密钥支持 per-instance 覆盖。

### Consequences

- Good: 快照预演机制不受影响；改造推迟到有独立立项承载。
- Bad（已知边界）: dev 实例无 GitHub 登录入口（按钮存在但后端无 provider → 400 no provider），属预期行为。
- drift-check 对 dev 的 github.client_id/secret 空值放行（已知边界检查项）。

## Pros and Cons of the Options

### 记录边界 + 门控放行（选定）
- Good: 诚实现状，检查项覆盖，不阻塞主线。
- Bad: dev 与生产的站点设置语义不隔离，新增敏感键时需逐个评估。

### 立即实现 per-instance 覆盖
- Good: 隔离彻底。
- Bad: 管理端配置模型 + 快照机制双重改造，无立项承载仓促上马。

### dev 停用生产敏感键
- Good: 立即消除串扰。
- Bad: SMTP 等键停用会让 dev 侧邮件类功能失去预演能力，快照价值受损。

## Links

- 配置治理主线：[0006](0006-deploy-config-as-artifact.md)
- 部署文档：docs/operations/deployment.md
