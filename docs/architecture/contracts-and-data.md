# 契约、数据与派生投影

> 文档类型：架构
>
> 状态：Active（契约管线 `Partial`，swag 接入后完整）
>
> 负责人：Platform maintainers
>
> 最近核验：2026-08-06

## 契约管线

```
apps/server（Go 注解 swag）
   │  make gen
   ▼
packages/api-contract/openapi.yaml      ← CI 校验：生成结果无 diff
   │  gen-ts.sh                 │  gen-dart.sh
   ▼                           ▼
apps/web/src/api/*.ts          apps/mobile/packages/core/lib/src/gen/*.dart
```

- **后端是事实源**：Go struct + swag 注解 → openapi.yaml。
- **web/mobile 都是生成物**：类型永不同步失败；CI 校验"生成结果无 diff"防漂移。
- **fixtures 契约测试**：真实 API 响应样本，反序列化测试兜底运行时行为。
- 当前 openapi.yaml 为占位（`paths: {}`）；swag 接入后生成。

## 数据模型

- 数据库迁移：golang-migrate（SQL 文件版本化，append-only），迁移框架落地后执行。
- 状态机：业务生命周期用显式状态机（如 topic: draft/published/archived/deleted），
  不靠多个布尔字段拼接（产品原则 9）。
- 软删除/硬删除策略在数据建模时定，决策记录见项目 note。

## 派生投影

| 投影 | 来源 | 可重建 |
|---|---|---|
| 搜索索引 | Meilisearch | ✅ 全量重建 |
| 计数（回帖数/点赞数） | DB 聚合或 Redis 计数 | ✅ 重算 |
| 热榜/feed | 派生查询 | ✅ |
| 通知已读/未读 | 用户指针表 | ✅ |

原则：投影必须可从事实源重建；禁止把投影当唯一事实。

## 契约变更纪律

- 后端字段变更必须同 PR 更新：Go struct → openapi.yaml → TS/Dart 生成物 → fixture。
- CI 的 contract.yml 在 `apps/server/**`、`packages/**` 变更时校验生成无 diff。
- 文档状态词同步更新（docs/README.md）。
