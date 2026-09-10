# 聚合搜索：三域分组展示 + scope tabs + 拼音

## Status
Accepted
Class: feature

## Context and Problem Statement

- 搜索此前仅覆盖帖子（topics 索引），搜索页只有帖子列表；用户/分类无法搜索（issue #22）。
- 兄弟仓库 YourTJ-Platform 采用联邦搜索模式：Meilisearch 只产出候选 ID，展示数据回 PostgreSQL 重构并强制可见性。
- Meilisearch v1.11 起默认禁用 pinyin-normalization，拼音/首字母搜索需自建字段。

## Decision Drivers

- 用户/分类必须可搜，且拼音/首字母可用（中文校园场景）。
- 展示数据（头像/图标/颜色）必须始终与 DB 一致，索引不得承载展示真相。
- Meilisearch 不可用时搜索可整体降级，论坛主体不受影响。

## Considered Options

- 候选 ID 索引 + DB 重构展示 + 自建拼音字段 + multi-index 分组（选定）。
- Meilisearch federation 混合排序。
- 展示字段全量入索引（避免回表）。
- PostgreSQL 全文检索替代 Meilisearch。

## Decision Outcome

1. 索引只存候选 ID + 可搜字段，展示数据回 DB 重构：users/categories 索引只放公开可搜字段（含拼音辅助字段），头像/图标/颜色等展示字段一律回 DB 取最新，展示字段变更不触发索引同步。
2. 拼音字段自建：引入 `github.com/mozillazg/go-pinyin`（零依赖、纯 Go），生成全拼 + 首字母字段（username/nickname/分类名），写入索引文档并加入 searchableAttributes。
3. multi-index 分组展示 + scope tabs：一次 `MultiSearch` 请求三域（topics/users/categories），前端分组展示 + 全部/帖子/用户/分类 tabs；不启用 federation 混合排序，架构预留 `federation:{}` 升级路径。
4. 部分降级：multi-search fail-fast 时逐索引回退，失败域记入 `failedScopes`；Meilisearch 整体不可用保持 `searchUnavailable` 全页降级。
5. 冻结/未激活用户不过滤（用户决策，简化处理），索引不设可见性过滤。
6. 迁移 v13 自动建索引：存量部署升级后首次启动自动构建 users + categories 索引（Meilisearch 不可用 → Skipped 照常推进版本；构建失败 → 不推进下次重试）；`rebuild-search-index` 扩展为重建三索引。
7. 控制器层发布事件避免 import cycle：`eventhandlers` 已 import `userservice`，反向在 `userservice.SaveUser` 内发布会成环；改为在控制器层相关调用点发布（EditUserInfo/EditUsername/分类增删），注册复用 `UserSignUpEvent` 覆盖 password/OAuth/OIDC 三注册路径。

### Consequences

- Good: 展示数据始终与 DB 一致（头像/图标变更即时生效）；索引是可重建投影，`rebuild-search-index` 兜底对账。
- Trade-off: 每次搜索多一次 DB 批量查询（校园论坛规模可接受）。
- 用户/分类区一期无分页（展示前 N 条），后续可加 cursor 分页；标签域一期不做（当前无 tag 模型）。

## Pros and Cons of the Options

### 候选 ID + DB 重构（选定）
- Good: 展示字段永不过期；与 YourTJ-Platform 模式同构，跨仓库心智一致。
- Bad: 每次搜索一次回表。

### federation 混合排序
- Good: 单请求单结果集。
- Bad: 排序权重跨域不可控，一期需求只需分组展示。

### 展示字段入索引
- Good: 省回表。
- Bad: 头像/改名等展示变更需要同步索引，投影语义被破坏。

### PG 全文检索
- Good: 少一个外部依赖。
- Bad: 拼音/首字母与中文分词需大量自建，且搜索已是可选增强服务。

## Links

- issue #22
- 索引可重建性：docs/architecture/contracts-and-data.md
