# Wiki 页面评论匿名发布（匿名楼层）

## Status
Accepted
Class: feature

## Context and Problem Statement

Wiki 页面评论区（`/wiki/*path`）复用论坛楼层回复（`PostStream`，postNo>1），但论坛楼层没有匿名能力。issue #524 要求 wiki 评论支持「匿名发布」，对齐既有课评匿名体验：对公众隐藏用户名/头像/主页链接，作者本人仍可管理自己的评论，治理侧可审计（举报不泄露作者，仅 Admin 填理由可揭示并留痕）。

## Decision Drivers

- 匿名只应覆盖 wiki 页面评论区（用户可见、有实名价值的对话形态）；普通论坛主题正文不发匿名能力（scope 收敛）。
- 匿名是**展示层掩码**，不是身份销毁：`user_id` 恒存真实作者，才能支撑作者管理自己的评论、举报治理与 Admin 揭示。
- 治理侧（举报/版主）不得向普通用户泄露匿名作者；仅 Admin 填理由可揭示，理由与揭示动作必须可审计。
- 复用课评匿名机制（`is_anonymous` 布尔列 + 零泄漏公开 DTO + Admin 揭示）作为参考模型，不新建独立身份模型。

## Considered Options

- **统一掩码（Uniform masking）**：匿名楼层的 `PostPayload.Author` 对所有查看者（含作者本人）都输出匿名占位（id=0、空头像、占位名），作者经 `isOwnPost` 管理。参考课评 `buildReviewPayload`（匿名作者也只见「匿名同学」）。
- **查看者相关掩码（Viewer-relative）**：作者本人看到真实身份，他人看到匿名占位。
- **通知：抑制 vs 掩码 Actor**：匿名楼层不产生以匿名者为 Actor 的通知，还是存储/回填一个掩码 Actor。
- **匿名进入 `topic_user_stat`/Posters vs 排除**：匿名作者是否出现在「参与过的话题」/参与者列表。

## Decision Outcome

1. **数据**：`posts` 表新增 `is_anonymous` 布尔列；`user_id` 恒存真实作者。写入路径（`POST /api/forum/posts/create`）接受 `isAnonymous`，仅当目标话题 `topic_type=TopicTypeWiki` 且为回复（`replyToPostId>0`，postNo>1）时放行，否则业务失败 `comment.anonymousNotAllowed`。
2. **统一掩码**：`buildPostPayloads` 中 `IsAnonymous` 楼层的 `Author` 输出 `{id:0, username:"匿名同学", avatarUrl:""}`；`PostPayload.IsAnonymous`/`ReplyTargetPayload.IsAnonymous` 供前端渲染占位（无 `/u/:id` 链接、无用户卡）。作者经 `isOwnPost`（`currentUserID==post.UserId`）管理自己的楼层。`buildReplyTargetPayload` 对匿名父楼层同样掩码 `replyToUsername`。
3. **身份表面排除**：匿名作者不进 `topic_user_stat`（`SyncTopicPostStats`/`RebuildTopicPostStats` 跳过），因此不出现于「参与过的话题」、话题 `Posters`、详情页 `Participants`；`buildTopicDetailProps` 的 `userMap` 排除匿名作者。JSON-LD 评论与 no-js SSR 因读取已掩码的 `PostPayload.Author` 自动覆盖。
4. **通知/Webhook 抑制**：匿名楼层 `CommentCreatedEvent.IsAnonymous=true` 时，`handleCommentCreated` 与 `handleHttpNotifyCommentCreated` 直接返回，不产生任何 in-app 通知或 webhook（避免读取时按 `ActorId` 回填真实用户名泄露身份）。他人回复匿名作者的楼层走独立事件（Actor 为真实回复者）正常通知。
5. **治理不泄露 + Admin 揭示**：举报证据快照（`buildReportEvidenceSnapshot`）与 `ViewDeletedContent` 对匿名楼层不定格/不返回真实作者。新增 `POST /api/forum/moderation/post-reveal`（`ModerationPostReveal`）：仅 Admin（控制器内 `IsAdmin`），理由必填，返回 `{postId, authorUserId, username, nickname, isAnonymous}`，并经 `optlogger.UserOptCode(RevealPostAuthor, ...)` 写 `opt_record` 审计（`OptEnum.RevealPostAuthor` 追加末尾、`TargetTypeEnum.Post=9` 追加，不重排既有值）。
6. **范围**：仅 Web 端。移动端 Wiki 保持只读（Route A），评论区与匿名能力不进入本期。

### Consequences

- Good: 匿名零泄漏——公开 DTO、参与者/参与话题、通知/Webhook、举报快照、版本历史编辑者、审核/删除审计日志、个人动态时间线、楼层 lastEditor 都不含真实作者；作者仍可管理；治理可揭示可审计。
- Trade-off: 匿名作者在公共视角与参与话题中完全不可见（`topic_user_stat` 不计数其参与）；同一楼层流内多条匿名评论无法区分归属（与课评一致）。
- 匿名仅限 wiki 页面评论区；普通论坛正文不提供匿名，避免匿名能力越界扩散。
- Security review（2026-09-07，合并前对抗性复核）：修订历史 / 审计日志 / 动态时间线 / 审批重发事件 / lastEditor / 揭示限流六处泄露面在评审后修复并补契约测试（匿名楼层版本编辑者恒为匿名占位、审核与删除日志快照不定格作者、匿名回复不进动态时间线、审批重发携带 IsAnonymous、post.reveal 默认限流接线）。揭示端点权限维持「控制器内 IsAdmin + 200 业务失败信封」（与 course-review-reveal 同构，契约已文档化 permission.denied），不加路由层 403 中间件以免破坏既有契约语义。

## Pros and Cons of the Options

### 统一掩码（选定）
- Good: 单一掩码分支覆盖 JSON-LD、参与者、SSR 全部读取路径，无「漏分支即泄露」风险；多条匿名评论无归属区分问题。
- Bad: 作者在公共载荷也看不到自己的真实身份（管理经 `isOwnPost`）。

### 查看者相关掩码
- Good: 作者体验更友好。
- Bad: 需把 viewer 身份穿入每条读取路径并保持分支一致，任一漏分支即真实泄露；且「作者能看到自己」会在反应交互上暴露归属，破坏匿名幻觉。

### 通知：抑制（选定）
- Good: 一个事件源守卫即阻断全部泄露面（ActorName 读取时按 ActorId 回填），无存储标志/读取时回查。
- Bad: 匿名发布不通知主题作者/关注者（匿名作者本人仍会收到他人回复其楼层的通知）。

### 通知：掩码 Actor
- Bad: 需存储匿名标志或读取时查楼层归属，且 `BuildNotificationPayload` 原始 payload 回显仍含真实 ActorId，两处易漏。

### 匿名进 topic_user_stat（未选）
- Bad: 匿名作者出现在「参与过的话题」/参与者即泄露参与身份。

## Links

- issue #524
- 课评匿名参考实现：`apps/gooseforum/app/models/forum/course/review.go`、`app/service/courseservice/review.go`、`app/http/controllers/forum/course_review.go`（`ModerationCourseReviewReveal`）
- 治理揭示审计机制：[0002](0002-jwt-session-revocation-user-sessions.md) 之外操作审计存 `opt_record`（`app/service/optlogger`）
