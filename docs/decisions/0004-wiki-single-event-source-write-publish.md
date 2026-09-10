# Wiki 修订单一事件源 + 写即发布 + 管理员回滚

## Status
Superseded by [0005](0005-wiki-github-ssot-pr-collaboration.md)
Class: architecture

## Context and Problem Statement

PR #219（wiki 分站原生化）review 暴露三个结构性问题：① 内容重复存储（同一内容同时落 revision / posts#1 / topics 三处，只修一处永远不一致）；② 双审核状态机（revision.status 与 post.ProcessStatus 两套流转互相干扰，审核队列出现幽灵项）；③ 树不变量（path 层级 / parent_id / sort_order）分散在多处，排序、移动、重命名都存在未覆盖的缺陷。

同时用户决策（2026-08-14）：拥有 L1 权限的人编辑后自动通过审核直接发布，不需要 L2 审核；仅需要管理员回滚到某个版本的能力，回滚不可撤销，直接把某个版本号之后的都永久丢弃；回滚在管理页面操作，并且需要 diff 视图、能对比任意两个版本的差异。

## Decision Drivers

- 内容必须只有一处事实源，其余全部是可重建投影。
- L1 编辑写即发布，无审核队列。
- 管理员回滚不可撤销且可对比任意两版本。

## Considered Options

- `wiki_page_revisions` 单一事件源 + 写即发布 + CAS 编辑锁 + 物理删除回滚（本记录，后被 0005 取代）。
- 保留双状态机并修补同步逻辑。
- 引入独立内容版本库（后续 0005 的 GitHub 仓库方向）。

## Decision Outcome

1. 单一事件源：`wiki_page_revisions` 是唯一事实源（append-only，revision_no 单调递增）；`wiki_pages.published_revision_no` 是唯一跨表写点；posts/topics（及搜索索引）是物化视图，携带 `wiki_synced_revision_no` 水印，编辑/回滚时同事务重同步。
2. 写即发布：任何 L1 编辑者（创建者 / namespace 贡献者 / PageManager）编辑后立即追加一条 `approved` 修订并前移指针，无 pending/rejected/superseded 流转；敏感词写时拦截（无审核兜底）。
3. 编辑锁 = CAS：`Edit` 携带 `baseRevisionNo`（打开编辑器时的 published_revision_no）；不匹配返回 `wiki.revision.conflict`（409 语义），前端提示并基于最新版本重编，编辑内容不丢失。
4. 回滚（仅 PageManager，管理页面内操作）：`POST /api/admin/wiki/pages/:pageId/rollback {toRevisionNo}` 物理删除（Unscoped）目标之后的全部修订（不可撤销，永久丢弃）、指针回退、物化视图重同步、通知 watcher；UI 二次确认。
5. diff 视图：`GET /api/admin/wiki/pages/:pageId/diff?from=N&to=M` 返回两侧 markdown 原文（from 可缺省 = 创建前空基线）；版本历史由 `GET /api/admin/wiki/revisions?pageId=` 全量分页提供。
6. 删除/恢复对齐：wiki_pages 与 wiki_page_revisions 引入软删除（DeletedAt），论坛删除/恢复路径级联一致。
7. 树修复：sort 按目标位置重排兄弟（1..N）、move 级联重写 path 与后代、rename 前缀冲突预检、软删页面排除出 pending 审核子查询、索引配置启动重试。
8. 通知节流：同页面 `wiki_updated` 通知 10 分钟窗口内只发首条。

### Consequences

- 无审核队列：管理端"审核队列 tab"改为"版本历史 + diff + 回滚"。
- 回滚不可逆：UI 必须强确认；版本号无空洞。
- 本模型是单写者模型：CAS 冲突只能基于最新版本重编，无并发多源同步编辑，也无真正多人评审体系——被 0005 的 GitHub SSoT 模型整体取代（旧 `wiki_page_revisions` / `wiki_namespace_editors` 表保留供迁移历史与 contentdeleteservice 依赖）。

## Pros and Cons of the Options

### 单一事件源 + 写即发布（本记录，已被取代）
- Good: 内容单源、一致性由水印保证；落地快。
- Bad: 单写者模型，无多人评审与并发协作，贡献者体系靠手工 ACL。

### 修补双状态机
- Good: 改动最小。
- Bad: 双状态机是问题根源，修补不消除幽灵项与不一致。

### GitHub 版本库（0005 方向）
- 见 0005 的权衡。

## Links

- PR #219
- 取代记录：[0005](0005-wiki-github-ssot-pr-collaboration.md)
