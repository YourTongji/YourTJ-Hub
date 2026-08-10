# Issue #83 邮箱修改 re-auth 安全加固 — 设计文档

- 日期：2026-08-10
- 分支：`fix/issue-83-email-change-reauth`
- 关联 Issue：[#83](https://github.com/YourTongji/YourTJ-Hub/issues/83)

## 问题

`EditUserEmail`（`apps/gooseforum/app/http/controllers/api/userController.go:70`）修改邮箱时：

1. **无 re-auth**：仅做会话查询 + 域名白名单 + 唯一性检查，直接写库。对比 `TotpSetup`（`totpController.go:47`）与 `ChangePassword`（`userController.go:500`）都要求登录密码。
2. **无旧邮箱通知**：只 `SendActivationEmail` 发到新地址，旧邮箱毫不知情。
3. **重置令牌不校验激活状态**：`ForgotPassword`（`userController.go:523`）→ `GeneratePasswordResetToken`（`tokenservice/password_reset_token.go:17`）对任何用户签发，激活状态不构成门槛。
4. **路由无限流**：`set-user-email`（`route4api.go:140`）仅 `JWTAuthCheck` + `CheckWritableAccount`，对比 `forgot-password`/`upload` 都挂 RateLimit。

**攻击链**：会话 token 泄露 → 静默改绑邮箱 → 用新邮箱 `ForgotPassword` → 重置密码 → `SetPassword` 自增 `TokenVersion` 踢掉受害者所有会话 → 完全接管账号。

## 修复方案（对应 Issue 验收标准 4 项）

### 1. 修改邮箱需登录密码二次确认

- `EditUserEmailReq` 增加 `Password string \`json:"password" validate:"required"\``
- `EditUserEmail` 调用 `algorithm.VerifyEncryptPassword(userEntity.Password, req.Params.Password)`，失败返回 `MessageAuthOldPasswordInvalid`（与 `ChangePassword` 一致）
- 校验失败在触碰数据库前拒绝，保证 `email` 不被修改

### 2. 新旧邮箱都收到变更通知

- 新邮箱：维持现有 `SendActivationEmail`（激活邮件，`emailactivationservice/send.go`）
- 旧邮箱：新增 `email.changed` 通知邮件
  - 新模板 `mailservice/email-changed-email.gohtml`（仿 `password-reset-email.gohtml`）
  - 新发送器 `mailservice.SendEmailChangedEmail`（仿 `SendPasswordResetEmail`，subject 用 `email.changed.subject`）
  - i18n 键：en/zh/ja/it 四语言各加 `email.changed.*`
  - 通过现有 `mailservice.AddToQueue`（queue.go，`EmailTask.Type = "email_changed"`，`processEmailTask` 增加分发 case）
- 改动成功后在**触碰数据库之前**记录旧邮箱，写库成功后向旧邮箱入队通知

### 3. 变更冷静期：24 小时内新邮箱不能用于密码重置

- `users.EntityComplete` 新增 `EmailChangedAt *time.Time`（`gorm:"column:email_changed_at;"`，AutoMigrate-safe，PG 兼容）
- `EditUserEmail` 成功时写入 `EmailChangedAt = now`
- `ForgotPassword` 中：查用户后若 `EmailChangedAt` 距今 <24h，**静默返回成功但不入队重置令牌**——响应与"邮箱未注册"完全一致（`MessageAuthResetMailQueued`），无枚举差异
- 冷静期时长常量定义在用户服务或控制器层（常量 `emailChangeCooldown = 24 * time.Hour`）

### 4. 路由增加速率限制

- `ratelimit.json`（`defaultconfig/pageconfig/`）新增 action：
  `{ "action": "email.change", "windowSeconds": 3600, "limitPerIp": 10, "limitPerUser": 5 }`
- `middleware/rateLimit.go` 新增 `RateLimitEmailChange = "email.change"`
- `route4api.go` 的 `set-user-email` 路由挂 `middleware.RateLimit(middleware.RateLimitEmailChange)`
- 存量部署自动生效：`mergeDefaultRateLimitActions`（`config_cache.go:123`）会补入缺失 action

## 前端同步

### Web（`apps/gooseforum/resource`）

- `runtime/api.ts` `saveUserEmail(email: string, password: string)`，body `{ email, password }`
- `site/pages/SettingsPage.vue` 邮箱编辑表单（当前 L1406-1444）增加密码输入框，`saveEmail()`（L605-619）传 password

### Mobile（`apps/mobile`）

- `packages/core/lib/src/api/repositories/user_repository.dart` `setUserEmail(String email, String password)`，body `{ email, password }`
- `packages/forum_app/lib/src/pages/settings/settings_page.dart` `_changeEmail()`（L277-319）对话框增加密码输入框

## 回归测试

新增 `apps/gooseforum/app/http/routes/email_change_api_test.go`（仿 `totp_setup_api_test.go` 模式），覆盖接管链：

| 场景 | 断言 |
|---|---|
| 无密码提交 | 拒绝，email 未变 |
| 错误密码提交 | 拒绝（oldInvalid），email 未变 |
| 正确密码 + 新邮箱 | 成功，email 更新，IsActivated=ActivationPending |
| 成功双通知 | 队列含 2 个任务：activation→新邮箱、`email_changed`→旧邮箱（携带 NewEmail） |
| 冷静期内 ForgotPassword | 静默成功返回，但零重置令牌任务入队 |
| 冷静期过期 ForgotPassword | 重置邮件正常入队 |
| 路由注册 | `POST /api/set-user-email` 已注册且挂限流 |

邮件模板渲染测试追加到 `mailservice/emailTmpl_test.go`（`TestGenerateEmailChangedEmailBodyUsesLocale`）。

## 设计取舍

- **OAuth/OIDC 用户**密码为随机生成，将无法通过 API 改邮箱——有意为之（Issue 要求 re-auth）。管理员保留 `set-user-email` 控制台命令（`app/console/cmd/user.go`）。
- **仅密码 re-auth，不做 2FA 双轨**（用户已确认）：与 TOTP setup / ChangePassword 防线一致。

## 验证计划

1. `cd apps/gooseforum && go vet ./... && go test ./...`
2. PG 迁移测试（模型变更必跑）：
   `YOURTJ_TEST_PG_URL="..." go test ./app/migration/ -run 'TestSchema' -v`
3. `cd apps/gooseforum/resource && pnpm typecheck && pnpm test && pnpm build`
4. 手工 curl 验证接管链：无密码→拒绝、错误密码→拒绝、正确→双通知、冷静期内 forgot-password→零重置任务

## 范围外（本 PR 不做）

- 冷静期内**注册新账号**限制（Issue 未要求，仅要求密码重置）
- 2FA 双轨二次确认（用户选择仅密码）
- openapi 契约同步（`set-user-email` 不在 api-contract 覆盖范围）
