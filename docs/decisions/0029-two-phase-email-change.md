# 换绑邮箱两阶段切换（pending_email 暂存）

## Status
Accepted
Class: feature

## Context and Problem Statement

issue #678 记录了换绑邮箱的释放语义缺陷：`set-user-email`（EditUserEmail）保存即生效——新邮箱覆写 `email` 并重置激活态，旧邮箱即刻释放；而注册仅通过 `ExistEmail` 检查当前占用，无任何历史限制。两者叠加形成「换绑释放旧邮箱 → 旧邮箱再注册新账号」的养号循环：持有一个校园邮箱的攻击者可无限换绑再注册，批量制造账号（白名单只约束域名，改变不了这个时序）。附带地，旧语义还有两个次生问题：已激活账号发起换绑即被重置为 pending，在邮箱验证开启的站点会立刻丢失写权限（re-pending 副作用）；24 小时找回冷静期从「发起换绑」起算，而非真正的邮箱切换时刻。

维护者决策（issue #678 Oryn 报告的推荐项）：采用两阶段换绑——暂存 `pending_email`，激活链接验证通过后原子切换；pending 期旧邮箱保留可登录/找回。

## Decision Drivers

- **封死养号循环**：旧邮箱在切换真正完成前必须保持占用（登录、找回、注册查重三处都不可释放），这是修复的核心目标。
- **不引入新表**：复用现有激活令牌（HMAC JWT 绑定目标邮箱）与邮件队列，暂存字段落在 `users` 行内，避免「已用邮箱历史表」方案的迁移与保留期/隐私边界设计。
- **受害者自救优先**：换绑是低频敏感操作，攻击者拿到会话发起换绑时，旧邮箱所有者必须仍能通过旧邮箱登录、找回密码、收到变更通知并止损。
- **不惩罚正常用户**：已激活账号发起换绑不应丢写权限（修复 re-pending 副作用）；pending 期账号功能完全可用。
- **并发正确**：确认链接点击、重复点击、过期链接、抢注注册之间必须互斥，切换必须原子。

## Considered Options

- **a) 两阶段换绑：`pending_email` 暂存 + 激活链接确认后原子切换**（选定，issue #678 推荐项）。
- **b) 保留即时切换 + 已用邮箱历史表**：换绑行为不变，被释放邮箱在保留期内禁止再注册。需要新表/迁移/PG 门禁与保留期、隐私边界定义；且旧邮箱在 pending 期仍被释放，登录/找回语义不变，受害者自救窗口不因换绑而延长。
- **c) 维持现状**：依赖换绑限流、注册验证码与 `MaxDailySignups` 承载滥用面。全站配额不绑定邮箱/身份，挡不住定向循环。

## Decision Outcome

1. **第一阶段只暂存**：验证开关开启时，`set-user-email` 仅写入 `pending_email`/`pending_email_at`（7 天占用窗口，不自发起顺延），不动 `email`、激活态与 `EmailChangedAt`。向新邮箱发确认邮件（令牌绑定暂存邮箱），向旧邮箱发「换绑申请」通知（新邮件类型 `email_change_pending`）。验证开关关闭时保留旧即时切换语义（邮箱该模式下不是验证通道）。
2. **第二阶段原子切换**：激活链接解析出的目标命中新鲜暂存时，单条条件 UPDATE（约束暂存值 + 新鲜度）完成 `email ← pending_email`、清暂存、置已激活、`EmailChangedAt = now`（24h 找回冷静期从真正切换起算）。未命中按无效链接处理；切换完成后向旧邮箱发最终「已变更」通知。
3. **占用查重扩暂存**：注册与换绑暂存在事务内取得同一邮箱的 PostgreSQL advisory lock，复查占用后写入并持有锁直到提交；SQLite 依赖写事务互斥。注册与换绑自查重统一走 `ExistEmailOrFreshPending`（当前 email ∪ 窗口内其他账号的 pending，换绑自查排除本人）。窗口外过期暂存视为放弃：不占用、不可确认，重新发起前定向清理（部分唯一索引 `uniq_users_pending_email_nonempty` 不区分新旧，需先清过期行）。
4. **取消与互斥路径**：点击发往当前邮箱的旧激活链接 = 放弃换绑（普通激活并清暂存）；密码重置成功 = 账号找回完成（清暂存）；管理员 CLI 直改邮箱 = 旁路验证（清暂存）。已激活账号发起换绑不再重置激活态（re-pending 副作用随两阶段语义自然消失）。
5. **resend 支持双目标**：窗口内存在暂存时，`resend-activation-email` 重发暂存邮箱的确认邮件（已激活账号也走此路径）；无暂存且已激活仍返回已验证错误。
6. **契约与镜像同步**：`setUserEmail`/`resendActivationEmail`/`register` 的契约描述更新；设置页 SSR props（`UserDetailedVo`/`SettingsUserPayload`）新增 `pendingEmail`（仅暴露窗口内暂存），web TS 与 mobile Dart 镜像同 PR 更新。

### Consequences

- Good: 养号循环被封死（旧邮箱占用到切换完成）；pending 期旧邮箱可登录/找回，受害者自救窗口完整；re-pending 副作用消除；冷静期起点修正为切换时刻；无新表。
- Good: 与既有激活门禁（issue #404）正交——pending 账号的恢复路径豁免不变，set-user-email 仍可在 pending 期发起（暂存语义下相当于重定向验证目标）。
- Trade-off: 换绑从「一步生效」变为「两步确认」，多一封确认邮件；窗口内新邮箱被暂存占用（他人不可注册），最长 7 天；`users` 表新增两列与一个部分唯一索引（AutoMigrate 增量，过 PG 门禁）。
- Trade-off: 白名单为空时任意后缀仍可换绑/注册——这是文档化的管理端行为（「留空表示不限制」），本次不改变；两阶段封住的是释放-再注册循环，不是后缀准入。
- 注意：`sub`/身份语义不变（OIDC `sub` = users.id），换绑不影响外部 OIDC 客户端。

## Pros and Cons of the Options

### 两阶段换绑（选定）
- Good: 复用激活令牌与邮件队列，无新表；核心循环（释放→再注册）被时序切断；受害者体验改善。
- Bad: 契约语义变化（success 响应含义从「已切换」变为「已暂存」）；前端需要 pending 横幅；窗口内邮箱被占用可能引发「我的新邮箱为什么注册不了」咨询。

### 即时切换 + 已用邮箱历史
- Good: 换绑交互不变，一步生效。
- Bad: 新表 + 迁移 + PG 门禁 + 保留期/隐私边界设计；登录/找回语义不变意味着受害者自救窗口不延长；养号循环只是被限流不是被切断。

### 维持现状
- Good: 零成本。
- Bad: 定向滥用无防线，与反馈（QQ 用户报告）直接冲突。

## Links

- issue #678（缺陷报告与维护者决策：两阶段换绑为推荐项）
- issue #404 / PR（激活写权限门禁，pending 恢复路径豁免的既有语义）
- issue #106（重置令牌绑定 token_version，本决策第 4 条的找回互斥基础）
- `packages/api-contract/paths/user-account.yaml` `setUserEmail`/`resendActivationEmail`、`paths/auth-account-recovery.yaml` `register`（契约描述同步处）
