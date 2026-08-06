# yourtj-hub 文档中心

> 文档类型：文档索引
>
> 状态：Active
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

这里是 yourtj-hub 产品、架构、开发与运维规范的统一入口。文档只描述当前支持的行为模型；
已经失真的阶段计划、PR 交付清单和重复的 API/DDL 快照不在当前文档树中长期保留（Git 历史承担归档）。

## 如何判断事实来源

不同问题由不同来源负责，不建立一个会混淆"目标"和"现状"的总排序：

| 问题 | 权威来源 |
|---|---|
| 产品应该如何工作 | `docs/product/` |
| 安全、隐私、合规硬约束 | `AGENTS.md` 与 `docs/security/`（未建立前以 AGENTS.md 为准） |
| HTTP 请求和响应结构 | `packages/api-contract/openapi.yaml` |
| 已部署数据库结构 | `apps/server/migrations/` 中按编号追加的 migration |
| 当前代码实际行为 | 源码、自动化测试和部署版本 |
| 开发、测试和 PR 流程 | `docs/development/` |
| 部署与故障处置 | `docs/operations/` |

这些来源不一致时，不应选择一个方便的版本继续开发。应把差异视为缺陷，在同一个 PR 中
修正契约、实现、测试和相关文档，或明确记录为 `Partial`。

## 实现状态词

产品文档只使用以下四种实现状态：

- `Current`：用户可达的必要链路、后端约束和相应验证均存在。
- `Partial`：只有部分层完成；必须写明缺少 Web、API、schema、worker、运营流程或测试中的哪一层。
- `Planned`：目标业务规则已形成，但尚未交付，不能在界面或宣传中声称可用。
- `Decision needed`：数据模型、权限或产品方向仍需负责人决策；决策前不得擅自实现。

状态标注作用于具体、可验证的行为，不作用于整个大领域。一个领域可以同时有 `Current` 的后端
基础和 `Partial` 的端到端产品链路；前者不能被用来宣称整个功能已经完成。

文档自身使用 `Active`、`Draft`、`Deprecated` 表示生命周期。它与功能实现状态是两件事。
禁止使用合并后立即失真的 PR-relative "本次已交付/以后再做"标签作为长期状态。

## 目录

### 产品

- [产品愿景与原则](product/vision-and-principles.md)
- [当前能力与差距](product/current-state.md)
- [身份、登录与账号生命周期](product/identity-and-access.md)
- [积分与跨平台结算](product/credit-and-escrow.md)

### 架构

- [系统概览与域边界](architecture/system-overview.md)
- [契约、数据与派生投影](architecture/contracts-and-data.md)

### 开发

- [开发入口](development/README.md)
- [本地环境](development/local-development.md)
- [测试策略与命令](development/testing.md)
- [分支、提交与 Pull Request](development/pull-requests.md)
- [文档治理](development/documentation.md)

### 运维

- [部署与发布](operations/deployment.md)

### 决策记录

- 架构决策（ADR）保存在项目 note（yourtj-hub 架构决策记录），**不落盘 git**；
  新决策追加编号、只追加不改历史。
