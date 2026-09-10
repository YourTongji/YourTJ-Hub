# Wiki GitHub 唯一真实源 + PR 协作编辑

## Status
Accepted
Class: architecture

## Context and Problem Statement

ADR-0004 的「单一事件源 + 写即发布 + CAS 编辑锁」是单写者模型：CAS 冲突只能基于最新版本重编，无法并发多源同步编辑，也没有真正的多人评审与贡献者体系（`wiki_namespace_editors` 手工维护）。

用户决策（2026-08-15）：新开独立公开 GitHub 仓库专存 wiki Markdown 文件并作为唯一真实源；所有编辑走 GitHub PR（天然多人协作 + 审核 + contributors 自动生成）；前端只保留「编辑此页」跳转按钮；服务器每日定时 git 同步 + 管理员后台手动同步（+ webhook 即时触发）。

## Decision Drivers

- wiki 编辑需要真正的多人协作、评审与贡献记录。
- 论坛侧不得再承载内容写路径与审核状态机。
- 同步必须幂等、可观测、可手动触发。

## Considered Options

- GitHub 仓库作为唯一真实源，论坛降为只读投影 + 互动层（选定）。
- 维持站内修订模型并补多人评审（在 0004 上继续加审核状态机）。
- 自建 git 后端（bare repo + 自研 PR 流）。

## Decision Outcome

1. 仓库公开：`YourTongji/YourTJ-Wiki` 公开仓库；图片用仓库内相对路径（渲染为 raw 直链），无鉴权依赖；contributors 公开可见；branch protection 强制 PR + ≥1 review。
2. 同步三路齐全：webhook（PR merge = push 事件，HMAC-SHA256 验签，分钟级生效）+ 每日定时（默认 03:00，`[wiki.git].schedule` 可配）+ 管理端手动按钮；防重入锁；每次同步写 `wiki_sync_runs` 日志。
3. 空站起步：不做种子迁移；新仓库建初始结构（README/CONTRIBUTING/示例 index.md），首同步即空 namespace 初始化。
4. 论坛 = 只读投影 + 互动层：`wiki_pages` 增投影列（title/content/rendered_html/toc/content_hash/last_commit_sha/contributors_json），topics/posts 保留（topic_type=1 互动/搜索/隔离不变）；站内全部 wiki 写路径（站内编辑/审核/回滚/树管理/编辑者 ACL）整体退役。
5. 同步引擎：`git fetch` + `reset --hard`（不用 pull）→ 逐文件 sha256 diff（幂等）→ 新建/变更/删除 → 独立事务 goldmark 渲染 → 搜索增量 → `wiki_updated` 通知（节流）；软删页面在仓库重新出现时自动恢复（含 topic 生命周期）。
6. API 契约：删写 API（edit/review/rollback/diff/editors/tree 写），加 `GET /api/admin/wiki/sync/status`、`POST /api/admin/wiki/sync`、`GET /api/admin/wiki/sync/runs`、`POST /api/wiki/webhook`；读 API（tree/home/namespaces）保留，detail payload 增加 `editUrl/historyUrl`；openapi + 生成 TS + mobile Dart mirror 同 PR。
7. 运维：`config.toml [wiki.git]`（repo/branch/clone_dir/webhook_secret/schedule）；公开库 clone 无需 token；webhook 可选但推荐；服务器需 git + 出网 443；迁移版本 v21（含 v21 回填：legacy 修订复制到投影列）。

### Consequences

- Good: GitHub 承担内容/历史/审核/贡献者/回滚；论坛无任何 wiki 内容写 API，「编辑此页」直跳 GitHub。
- Good: 同步幂等（正文 hash 比对），重复同步零变更；仓库缺失页面 → 论坛软删（保留互动）。
- Trade-off: wiki 协作依赖 GitHub 账号与外网可达性；同步延迟（webhook 分钟级、定时小时级）。
- 旧 `wiki_page_revisions`/`wiki_namespace_editors` 表保留（迁移历史/contentdeleteservice 依赖），仅功能层退役。

## Pros and Cons of the Options

### GitHub SSoT（选定）
- Good: 评审/历史/贡献者/回滚全部复用 GitHub 原生能力，零自建。
- Bad: 编辑体验跳转外部；同步有延迟；依赖 GitHub 可达。

### 站内修订 + 多人评审
- Good: 体验站内闭环。
- Bad: 需要自建完整 PR/review/diff/贡献者体系，维护成本高且已被 0004 的复杂性教训证明。

### 自建 git 后端
- Good: 不依赖外部服务。
- Bad: 工程量巨大，收益不成比例。

## Links

- PR #256
- 被取代记录：[0004](0004-wiki-single-event-source-write-publish.md)
- 契约：packages/api-contract（wiki domain）
