# 删除终态数据保留与管理员取证（R4 降级）

## Status
Accepted
Class: feature

## Context and Problem Statement

删除生命周期（删除 → 30 天恢复窗口 → PURGED）的终态此前是**内容销毁型**：`posts.MarkPurged`/`MarkPurgedOwned` 清空 `content`/`rendered_html`，`topics.MarkPurged` 清空标题/摘要/图片引用，附件经 `PurgeTargetFiles`/`ExpireRecoveringFiles` 物理删除文件本体。版主取证视图 `view-deleted-content` 同时拒绝 PURGED 目标。结果是：举报纠纷或治理取证在内容过期/永久删除后无法还原全文与附件，取证能力随删除一起消失。

issue #555 记录了项目组的三项决策：删除终态改为「用户不可恢复 + 数据保留」；有权限的管理员保持取证可见；用户恢复窗口内的主动永久删除（R4）与过期路径同样降级。用户侧隐私擦除通道（原文不留库）已由 PR #577 移除，其决策理由（破坏举报取证与治理审计）与本次方向一致。

## Decision Drivers

- **取证优先**：治理与举报需要事后还原被删内容全文与附件；销毁型终态留下不可弥补的取证缺口。
- **用户侧体验不变**：删除必须立即全渠道消失；用户的信任模型是「删除 = 不可见 + 不可恢复」，不是「删除 = 字节销毁」。数据保留不得产生任何用户可见的残留。
- **不引入新终态**：复用 PURGED 态则无 schema 变更、不动契约路径；「恢复窗口过期冻结」与「用户主动永久删除」靠 `visibility_status`/`delete_reason` 区分即可。
- **附件授权模型不变**：fileUsage 引用状态机（ACTIVE/RECOVERING/PURGED）已按 ACTIVE 引用判定公开下载权；保留字节只需停止物理删除，不需要新的授权语义。
- **最小权限取证**：沿用 `view-deleted-content` 的版主作用域鉴权 + 理由必填 + `EvidenceViewed`/`moderation_deleted_content_viewed` 双审计，不新造权限模型。

## Considered Options

- **a) 复用 PURGED 态：停止清空内容、停止物理删除附件，取证视图放开 PURGED**（选定）。
- **b) 删除时生成不可篡改全文快照归档进审计体系**：更符合取证惯例，但需要新表、迁移与 PG 门禁。
- **拆分终态**：为「过期冻结」新增独立状态与 PURGED 区分，语义最清晰但需要迁移、契约与前端全链路跟进。

## Decision Outcome

1. **状态翻转即终态**：`posts.MarkPurged`/`MarkPurgedOwned`、`topics.MarkPurged` 只做 `RECOVERABLE → PURGED` 状态翻转并置 `deleted_at`；正文、渲染 HTML、版本快照、标题、摘要与图片引用全部保留。`MarkPrivacyErased`（#492 无楼话题联动下架的后端标记）同样保留数据，仅状态下架。**PURGED 是 sink state**：`MarkUserDeleted`/`MarkModeratorRemoved` 拒绝改写 PURGED 行，治理删除→恢复通道无法复活终态内容；`MarkPurgedOwned` 对 ACTIVE 自回帖同时把 visibility 翻转为 `USER_DELETED`，保证 PURGED 行不留在任何 ACTIVE 可见性读路径上。
2. **附件退役不销毁**：`PurgeTargetFiles` 更名 `RetireTargetFiles`——引用置 PURGED、不再物理删除文件本体；`ExpireRecoveringFiles` 只退役引用。公开下载授权继续由 ACTIVE 引用判定（`/file/img/*` 控制器）；取证视图当前仅返回文本正文，附件字节的取证回显是后续增强——留存不等于现有读路径可访问。
3. **取证视图放开 PURGED**：`view-deleted-content` 对 PURGED 目标返回保留的标题与正文；鉴权、理由必填与双审计保持不变。仍拒绝的只有 ACTIVE 且未 PURGED（未删除）与不存在的目标。
4. **用户侧不可见由既有读路径保证**：公开列表/详情/搜索投影/LLMS 导出/Agent API 均按 `visibility_status=ACTIVE` 过滤；「最近删除」列表按 `retention_status != PURGED` 过滤。新增任何读路径必须显式过滤 PURGED（评审红线）。
5. **契约描述同步**：`purgeContent` 与 `viewDeletedContent` 的契约描述改为数据保留语义；不新增/删除路径，生成的 TS 类型无结构变化。
6. **存量明确接受**：切换上线前已被销毁的 PURGED 行原文不可追回，取证视野仅覆盖切换后的删除。

### Consequences

- Good: 取证闭环（文本经取证视图的理由+审计通道可还原；附件字节留存待回显能力）；零 schema 变更；删除的即时性与用户信任模型不变；#577 移除隐私擦除后不再有绕过生命周期的销毁通道，R4 降级在安全上自洽。
- Trade-off: 数据库与附件存储随删除增长，不再收缩；数据暴露面从「审计日志」扩大到「保留的原文行」——任何绕过 visibility 过滤的新读路径都可能泄露 PURGED 原文（见决策 4 的评审红线）；「过期冻结」与「主动清除」共用 PURGED 态，语义靠 `visibility_status`/`delete_reason` 区分。
- 合规口径：产品对用户的删除承诺从「彻底删除」变为「不可恢复地移除（数据按治理需要留存）」，服务条款/隐私政策为管理端可配置内容，随内容运营同步修订。
- 后续增强（非本期）：路线 b 的不可篡改快照归档、取证附件回显、长期保留策略与存储成本评估。

## Pros and Cons of the Options

### 复用 PURGED（选定）
- Good: 零迁移、契约路径不动、审计链连续（现有 `content_delete_events`/moderation_log 继续覆盖）；实现与回滚成本最低。
- Bad: 两种终态语义同态，需靠 `delete_reason` 区分；PURGED 行留在主表中，读路径过滤纪律从「可选优化」升级为「安全红线」。

### 快照归档（路线 b）
- Good: 不可篡改、符合取证惯例；原文行仍可按更激进策略清理。
- Bad: 新表 + 迁移 + PG 门禁 + 快照与行的一致性维护，成本高；查看审计已由现有通道覆盖，边际收益有限。

### 拆分独立终态
- Good: 「过期冻结」与「主动清除」语义最清晰。
- Bad: 迁移 + 契约 + 前端全链路跟进，而当前没有任何需要对两者做不同处理（展示/权限/恢复）的需求，属超前设计。

## Links

- issue #555（决策差距清单与拍板）、PR #577（用户侧隐私擦除通道移除，R4 降级的安全前提）
- issue #492 / PR #526（无楼话题联动下架，cascade 挂钩点）
- `paths/user-content.yaml` `purgeContent`、`paths/forum-moderation.yaml` `viewDeletedContent`（契约描述同步处）
