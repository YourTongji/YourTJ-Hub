package component

// MessageCode is a stable, frontend-facing identifier for i18n messages.
// Backend responses expose messageCode and params only; clients translate them locally.
type MessageCode string

// MessageParams contains dynamic values used by frontend translations.
type MessageParams map[string]any

// MessageError carries a stable code plus fallback text through service helpers.
type MessageError struct {
	Code     MessageCode
	Fallback string
	Params   MessageParams
}

func (err MessageError) Error() string {
	return err.Fallback
}

func NewMessageError(code MessageCode, fallback string, params MessageParams) error {
	return MessageError{
		Code:     code,
		Fallback: fallback,
		Params:   params,
	}
}

const (
	MessageRequestInvalidFormat MessageCode = "common.request.invalidFormat" // 请求体或参数格式无法解析。
	MessageRequestInvalidParams MessageCode = "common.request.invalidParams" // 请求参数未通过业务校验。
	MessageRequestParseFailed   MessageCode = "common.request.parseFailed"   // 参数绑定失败（400；不返回原始解析错误）。
	MessageOperationSuccess     MessageCode = "common.operation.success"     // 通用操作成功。
	MessageOperationFailed      MessageCode = "common.operation.failed"      // 通用操作失败。
	MessageRateLimited          MessageCode = "common.rateLimited"           // 操作过于频繁，params.action/retryAfterSeconds。
	MessageCaptchaRequired      MessageCode = "common.captchaRequired"       // 需要完成验证码才能继续，params.action。
	MessagePageNotFound         MessageCode = "page.notFound"                // 页面不存在。
	MessageRouteNotFound        MessageCode = "route.notFound"               // 路由未定义。
	MessageUserFetchFailed      MessageCode = "user.fetchFailed"             // 当前用户信息读取失败。
	MessageUserNotFound         MessageCode = "user.notFound"                // 用户不存在。
	MessageUserUpdateFailed     MessageCode = "user.updateFailed"            // 用户信息保存失败。
	MessageUserUpdateSuccess    MessageCode = "user.updateSuccess"           // 用户信息保存成功。
)

const (
	MessageAuthRequired                  MessageCode = "auth.required"                   // 需要登录后才能继续操作。
	MessageAuthSignupDisabled            MessageCode = "auth.signupDisabled"             // 当前站点关闭了注册。
	MessageAuthEmailDomainInvalid        MessageCode = "auth.emailDomain.invalid"        // 邮箱格式不正确或无法提取域名。
	MessageAuthEmailDomainNotAllowed     MessageCode = "auth.emailDomain.notAllowed"     // 邮箱域名不在注册白名单。
	MessageAuthUsernameInvalid           MessageCode = "auth.username.invalid"           // 用户名格式不符合规则。
	MessageAuthUsernameExists            MessageCode = "auth.username.exists"            // 用户名已存在。
	MessageAuthEmailExists               MessageCode = "auth.email.exists"               // 邮箱已被使用。
	MessageAuthPasswordTooShort          MessageCode = "auth.password.tooShort"          // 密码过短，params.minLength 表示最小长度。
	MessageAuthPasswordTooLong           MessageCode = "auth.password.tooLong"           // 密码过长。
	MessageAuthPasswordNeedsLetterNumber MessageCode = "auth.password.needsLetterNumber" // 密码必须包含字母和数字。
	MessageAuthCaptchaInvalid            MessageCode = "auth.captcha.invalid"            // 验证码错误或已过期。
	MessageAuthRegisterFailed            MessageCode = "auth.register.failed"            // 注册失败。
	MessageAuthRegisterRetryLogin        MessageCode = "auth.register.retryLogin"        // 注册成功但自动登录失败，建议手动登录。
	MessageAuthRegisterEmailVerify       MessageCode = "auth.register.emailVerify"       // 注册成功，需要邮箱验证。
	MessageAuthLoginSuccess              MessageCode = "auth.login.success"              // 登录成功。
	MessageAuthLoginInvalidRequest       MessageCode = "auth.login.invalidRequest"       // 登录请求无效，通常需要刷新页面重试。
	MessageAuthPasswordInvalidFormat     MessageCode = "auth.password.invalidFormat"     // 登录密码格式不正确。
	MessageAuthInvalidCredentials        MessageCode = "auth.credentials.invalid"        // 用户名、邮箱或密码错误。
	MessageAuthAccountFrozen             MessageCode = "auth.account.frozen"             // 账号被冻结。
	MessageAuthEmailUnverified           MessageCode = "auth.email.unverified"           // 邮箱未验证。
	MessageAuthLoginFailed               MessageCode = "auth.login.failed"               // 登录异常。
	MessageAuthOldPasswordInvalid        MessageCode = "auth.password.oldInvalid"        // 原密码错误。
	MessageAuthPasswordOAuthRequired     MessageCode = "auth.password.oauthRequired"     // 无邮箱的 OAuth 账号密码校验失败，需通过 OAuth 恢复。
	MessageAuthPasswordUpdateFailed      MessageCode = "auth.password.updateFailed"      // 修改密码失败。
	MessageAuthPasswordUpdateSuccess     MessageCode = "auth.password.updateSuccess"     // 修改密码成功。
	MessageAuthResetMailQueued           MessageCode = "auth.passwordReset.mailQueued"   // 如邮箱存在，将收到密码重置邮件。
	MessageAuthResetTokenCreateFailed    MessageCode = "auth.passwordReset.tokenFailed"  // 生成重置令牌失败。
	MessageAuthResetMailSendFailed       MessageCode = "auth.passwordReset.mailFailed"   // 发送重置邮件失败。
	MessageAuthResetTokenInvalid         MessageCode = "auth.passwordReset.tokenInvalid" // 重置链接过期或无效。
	MessageAuthResetFailed               MessageCode = "auth.passwordReset.failed"       // 重置密码失败。
	MessageAuthResetSuccess              MessageCode = "auth.passwordReset.success"      // 重置密码成功。
	MessageAuthActivationResendSuccess   MessageCode = "auth.activation.resendSuccess"   // 验证邮件已重新发送。
	MessageAuthActivationAlreadyVerified MessageCode = "auth.activation.alreadyVerified" // 当前账号已完成邮箱验证。
	MessageAuthActivationDisabled        MessageCode = "auth.activation.disabled"        // 当前站点未启用邮箱验证。
	MessageAuthActivationResendCooldown  MessageCode = "auth.activation.resendCooldown"  // 验证邮件发送过于频繁，params.retryAfterSeconds。
	MessageAuthActivationResendDaily     MessageCode = "auth.activation.resendDaily"     // 验证邮件达到当天重发上限，params.limit。
	MessageAuthActivationResendFailed    MessageCode = "auth.activation.resendFailed"    // 验证邮件重新发送失败。
)

const (
	MessageAuthTotpRequired   MessageCode = "auth.totp.required"  // 需要完成两步验证。
	MessageTotpCodeInvalid    MessageCode = "totp.code.invalid"   // 两步验证码错误。
	MessageTotpAlreadyEnabled MessageCode = "totp.alreadyEnabled" // 两步验证已启用。
	MessageTotpNotEnabled     MessageCode = "totp.notEnabled"     // 两步验证未启用。
	MessageTotpRateLimited    MessageCode = "totp.rateLimited"    // 两步验证尝试过于频繁。
	MessageTotpSetupFailed    MessageCode = "totp.setupFailed"    // 生成两步验证密钥失败，params.error 可带原始错误。
	MessageTotpEnableFailed   MessageCode = "totp.enableFailed"   // 启用两步验证失败，params.error 可带原始错误。
	MessageTotpDisableFailed  MessageCode = "totp.disableFailed"  // 关闭两步验证失败，params.error 可带原始错误。
)

const (
	MessagePermissionResolveFailed MessageCode = "permission.resolveFailed" // 权限信息读取失败。
	MessagePermissionDenied        MessageCode = "permission.denied"        // 当前用户没有执行该操作的权限。
	MessagePermissionUserFrozen    MessageCode = "permission.userFrozen"    // 用户已被冻结，params.action 表示操作名称。
	MessagePermissionEmailRequired MessageCode = "permission.emailRequired" // 需要先完成邮箱验证，params.action 表示操作名称。
)

const (
	MessageUploadAttachmentDisabled MessageCode = "upload.attachment.disabled"   // 站点关闭了附件上传。
	MessageUploadCooldown           MessageCode = "upload.cooldown"              // 新用户上传冷却中，params.minutes/availableAt。
	MessageUploadDailyLimit         MessageCode = "upload.dailyLimit"            // 达到每日上传限制，params.count。
	MessageUploadDailyLimitAvatar   MessageCode = "upload.dailyLimit.avatar"     // 头像上传将超过每日限制，params.count/fileCount。
	MessageUploadFileMissing        MessageCode = "upload.file.missing"          // 未获取到上传文件。
	MessageUploadFilenameRequired   MessageCode = "upload.filename.required"     // 文件名为空。
	MessageUploadFileTooLarge       MessageCode = "upload.file.tooLarge"         // 文件超过大小限制，params.maxSizeKb。
	MessageUploadUnsupportedExt     MessageCode = "upload.extension.unsupported" // 文件扩展名不允许，params.extensions。
	MessageUploadUnsupportedImage   MessageCode = "upload.image.unsupported"     // 图片格式不支持。
	MessageUploadReadFailed         MessageCode = "upload.readFailed"            // 文件读取失败。
	MessageUploadOpenFailed         MessageCode = "upload.openFailed"            // 文件打开失败。
	MessageUploadInvalidImage       MessageCode = "upload.image.invalidContent"  // 文件内容不是有效图片。
	MessageUploadContentReadFailed  MessageCode = "upload.contentReadFailed"     // 文件内容读取失败。
	MessageUploadSaveFailed         MessageCode = "upload.saveFailed"            // 文件保存失败，params.error 可带原始错误。
	MessageUploadSuccess            MessageCode = "upload.success"               // 上传成功。
)

const (
	MessageTopicNotFound            MessageCode = "topic.notFound"            // 主题不存在。
	MessageTopicOwnerMismatch       MessageCode = "topic.ownerMismatch"       // 不能修改或删除他人的主题。
	MessageTopicOperationDenied     MessageCode = "topic.operationDenied"     // 当前主题不可操作。
	MessageTopicSaveFailed          MessageCode = "topic.saveFailed"          // 主题保存失败。
	MessageTopicDailyLimit          MessageCode = "topic.dailyLimit"          // 当天发布过多。
	MessageTopicTitleTooShort       MessageCode = "topic.title.tooShort"      // 标题过短，params.minLength。
	MessageTopicTitleTooLong        MessageCode = "topic.title.tooLong"       // 标题过长，params.maxLength。
	MessageTopicContentTooShort     MessageCode = "topic.content.tooShort"    // 正文过短，params.minLength。
	MessageTopicContentTooLong      MessageCode = "topic.content.tooLong"     // 正文过长，params.maxLength。
	MessageTopicPostCooldown        MessageCode = "topic.post.cooldown"       // 新用户发帖冷却中，params.minutes/availableAt。
	MessageCommentContentTooShort   MessageCode = "comment.content.tooShort"  // 评论过短，params.minLength。
	MessageCommentContentTooLong    MessageCode = "comment.content.tooLong"   // 评论过长，params.maxLength。
	MessageCommentPostCooldown      MessageCode = "comment.post.cooldown"     // 新用户评论冷却中，params.minutes/availableAt。
	MessageCommentParentPostMissing MessageCode = "comment.parentPostMissing" // 父 post 不存在。
	MessageCommentCreateFailed      MessageCode = "comment.createFailed"      // 评论创建失败，params.error 可带原始错误。
	MessagePostNotFound             MessageCode = "post.notFound"             // post 不存在。
	MessagePostUpdateFailed         MessageCode = "post.updateFailed"         // post 更新失败，params.error 可带原始错误。
	MessageReportNotFound           MessageCode = "report.notFound"           // 举报不存在。
	MessageReportTargetInvalid      MessageCode = "report.targetInvalid"      // 举报对象无效。
	MessageReportOwnContent         MessageCode = "report.ownContent"         // 不能举报自己的内容。
	MessageReportDuplicate          MessageCode = "report.duplicate"          // 已举报，等待处理。
	MessageReportCreateFailed       MessageCode = "report.createFailed"       // 举报提交失败。

	// 课评（course review）
	MessageReviewNotFound         MessageCode = "review.notFound"              // 评价不存在或不可见。
	MessageReviewNotOwned         MessageCode = "review.notOwned"              // 不能修改/删除他人的评价。
	MessageReviewDuplicate        MessageCode = "review.duplicate"             // 已评价过该开课实例。
	MessageReviewOfferingNotFound MessageCode = "review.offeringNotFound"      // 开课实例不存在或不可见。
	MessageReviewRatingInvalid    MessageCode = "review.rating.invalid"        // 评分必须为 1..5 的整数。
	MessageReviewContentEmpty     MessageCode = "review.content.empty"         // 评价内容不能为空。
	MessageReviewContentTooLong   MessageCode = "review.content.tooLong"       // 评价内容过长，params.maxLength。
	MessageReviewCreateFailed     MessageCode = "review.createFailed"          // 评价提交失败，params.error 可带原始错误。
	MessageReviewUpdateFailed     MessageCode = "review.updateFailed"          // 评价更新失败，params.error 可带原始错误。
	MessageReviewDeleteFailed     MessageCode = "review.deleteFailed"          // 评价删除失败，params.error 可带原始错误。
	MessageReviewListFailed       MessageCode = "review.listFailed"            // 评价列表读取失败。
	MessageReviewHelpfulFailed    MessageCode = "review.helpful.failed"        // 标记 helpful 失败。
	MessageReviewReportFailed     MessageCode = "review.report.failed"         // 举报评价失败。
	MessageReviewRevealReasonReq  MessageCode = "review.reveal.reasonRequired" // 查看匿名作者必须填写理由。

	// wiki 分站
	MessageWikiNamespaceNotFound     MessageCode = "wiki.namespace.notFound"     // namespace 不存在。
	MessageWikiNamespaceHasPages     MessageCode = "wiki.namespace.hasPages"     // namespace 下存在页面，无法删除。
	MessageWikiNamespaceNameInvalid  MessageCode = "wiki.namespace.nameInvalid"  // namespace 名称非法。
	MessageWikiPathInvalid           MessageCode = "wiki.path.invalid"           // wiki 路径非法。
	MessageWikiPageNotFound          MessageCode = "wiki.page.notFound"          // wiki 页面不存在。
	MessageWikiForbidden             MessageCode = "wiki.forbidden"              // 无 wiki 操作权限。
	MessageWikiRevisionNotFound      MessageCode = "wiki.revision.notFound"      // 修订不存在。
	MessageWikiPageHasChildren       MessageCode = "wiki.page.hasChildren"       // 页面存在子页面，无法删除。
	MessageWikiSaveFailed            MessageCode = "wiki.saveFailed"             // wiki 保存失败。
	MessageWikiNamespaceNameConflict MessageCode = "wiki.namespace.nameConflict" // namespace 名称已存在（契约 409 语义）。
	MessageWikiPathConflict          MessageCode = "wiki.page.pathConflict"      // wiki 路径已存在（契约 409 语义）。
	MessageWikiRevisionConflict      MessageCode = "wiki.revision.conflict"      // 版本 CAS 冲突：页面已被他人更新，需基于最新版本重编（409 语义）。
	// 课程管理（管理端课程/评价管理）
	MessageCourseNotFound           MessageCode = "course.notFound"           // 课程不存在或已删除。
	MessageCourseCodeRequired       MessageCode = "course.codeRequired"       // 主课号不能为空。
	MessageCourseNameRequired       MessageCode = "course.nameRequired"       // 课程名不能为空。
	MessageCourseCodeConflict       MessageCode = "course.codeConflict"       // 主课号已被其它课程占用。
	MessageCourseCreditInvalid      MessageCode = "course.creditInvalid"      // 学分格式不正确。
	MessageCourseListFailed         MessageCode = "course.listFailed"         // 课程列表读取失败。
	MessageCourseStatsRebuildQueued MessageCode = "course.statsRebuildQueued" // 课程统计重建任务已入队。
	MessageCourseStatsRebuildFailed MessageCode = "course.statsRebuildFailed" // 课程统计重建任务入队失败。
	MessageCourseSummaryFailed      MessageCode = "course.summary.failed"     // AI 总结生成失败（LLM 超时/输出非法等，不影响课程页主流程）。
)

const (
	MessageContentDeleteFailed         MessageCode = "content.delete.failed"               // 删除失败。
	MessageContentRestoreFailed        MessageCode = "content.restore.failed"              // 恢复失败。
	MessageContentRestoreSuccess       MessageCode = "content.restore.success"             // 内容已恢复。
	MessageContentPurgeFailed          MessageCode = "content.purge.failed"                // 永久删除失败。
	MessageContentPurgeSuccess         MessageCode = "content.purge.success"               // 内容已永久删除。
	MessageContentRecoveryExpired      MessageCode = "content.recovery.expired"            // 已超出恢复窗口，无法恢复。
	MessageContentNotRecoverable       MessageCode = "content.notRecoverable"              // 该内容不可由作者恢复。
	MessageContentPrivacyErased        MessageCode = "content.privacy.erased"              // 隐私内容已彻底删除。
	MessageContentBatchConfirmRequired MessageCode = "content.batchDelete.confirmRequired" // 短时间内删除过多，需要二次确认，params.count。
)

const (
	MessageNotificationMarkReadFailed  MessageCode = "notification.markRead.failed"     // 标记单条通知已读失败。
	MessageNotificationMarkReadSuccess MessageCode = "notification.markRead.success"    // 标记单条通知已读成功。
	MessageNotificationMarkAllFailed   MessageCode = "notification.markAllRead.failed"  // 标记全部通知已读失败。
	MessageNotificationMarkAllSuccess  MessageCode = "notification.markAllRead.success" // 标记全部通知已读成功。
	MessageOAuthUnbindFailed           MessageCode = "oauth.unbind.failed"              // 解绑第三方账号失败，params.error 可带原始错误。
	MessageOAuthUnbindSuccess          MessageCode = "oauth.unbind.success"             // 解绑第三方账号成功。
	MessageOAuthCallbackFailed         MessageCode = "oauth.callback.failed"            // OAuth 认证回调失败。
	MessageOAuthProcessFailed          MessageCode = "oauth.process.failed"             // OAuth 登录处理失败。
	MessageOAuthAccountFrozen          MessageCode = "oauth.account.frozen"             // OAuth 登录账号被冻结。
	MessageOAuthActivationUpdateFailed MessageCode = "oauth.activation.updateFailed"    // OAuth 用户激活状态更新失败。
	MessageOAuthTokenFailed            MessageCode = "oauth.token.failed"               // OAuth 登录 token 生成失败。
	MessageOidcStartFailed             MessageCode = "oidc.start.failed"                // OIDC 登录发起失败。
	MessageOidcCallbackFailed          MessageCode = "oidc.callback.failed"             // OIDC 登录回调失败。
	MessageChatSendFailed              MessageCode = "chat.send.failed"                 // 私信发送失败，params.error 可带原始错误。
	MessageChatGetMessagesFailed       MessageCode = "chat.messages.failed"             // 获取私信列表失败。
	MessageChatMarkReadFailed          MessageCode = "chat.markRead.failed"             // 标记私信已读失败。
	MessageSessionListFailed           MessageCode = "session.list.failed"              // 获取登录会话列表失败。
	MessageSessionRevokeFailed         MessageCode = "session.revoke.failed"            // 吊销会话失败。
	MessageSessionRevokeSuccess        MessageCode = "session.revoke.success"           // 会话已吊销。
	MessageSessionRevokeAllSuccess     MessageCode = "session.revokeAll.success"        // 已退出所有设备。
	MessageSessionCurrentNotRevocable  MessageCode = "session.current.notRevocable"     // 当前会话不可吊销。
	MessageSessionNotFound             MessageCode = "session.notFound"                 // 会话不存在。
)

const (
	MessageAdminStatsFetchFailed       MessageCode = "admin.stats.fetchFailed"         // 管理后台统计数据读取失败。
	MessageAdminBadgeNameRequired      MessageCode = "admin.badge.nameRequired"        // 徽章名称不能为空。
	MessageAdminBadgeTypeInvalid       MessageCode = "admin.badge.typeInvalid"         // 徽章类型不合法。
	MessageAdminBadgeCodeRequired      MessageCode = "admin.badge.codeRequired"        // 徽章编码不能为空。
	MessageAdminBadgeGrantModeInvalid  MessageCode = "admin.badge.grantModeInvalid"    // 徽章授予方式不合法。
	MessageAdminBadgeSystemNotFound    MessageCode = "admin.badge.systemNotFound"      // 系统徽章不存在。
	MessageAdminBadgeSaveFailed        MessageCode = "admin.badge.saveFailed"          // 保存徽章失败。
	MessageAdminBadgeSystemDeleteBlock MessageCode = "admin.badge.systemDeleteBlocked" // 系统默认徽章不可删除。
	MessageAdminBadgeDeleteFailed      MessageCode = "admin.badge.deleteFailed"        // 删除徽章失败。
	MessageAdminTargetUserFetchFailed  MessageCode = "admin.user.targetFetchFailed"    // 目标用户查询失败。
	MessageAdminCategoryRequired       MessageCode = "admin.category.nameRequired"     // 分类名称不能为空。
	MessageAdminCategoryNotFound       MessageCode = "admin.category.notFound"         // 分类不存在。
	MessageAdminCategoryDataNotFound   MessageCode = "admin.category.dataNotFound"     // 分类数据不存在。
	MessageAdminCategoryKeepOne        MessageCode = "admin.category.keepOne"          // 至少保留一个分类。
	MessageAdminCategoryHasTopics      MessageCode = "admin.category.hasTopics"        // 分类下存在有效主题。
	MessageAdminModeratorUserRequired  MessageCode = "admin.moderator.userRequired"    // 版主用户不能为空。
	MessageAdminModeratorUserNotFound  MessageCode = "admin.moderator.userNotFound"    // 版主用户不存在。
	MessageAdminModeratorNotFound      MessageCode = "admin.moderator.notFound"        // 版主记录不存在。
	MessageAdminTopicCategoryRequired  MessageCode = "admin.topic.categoryRequired"    // 主题至少需要一个分类。
	MessageAdminTopicCategoryTooMany   MessageCode = "admin.topic.categoryTooMany"     // 主题最多选择三个分类。
	MessageAdminTopicDeleteFailed      MessageCode = "admin.topic.deleteFailed"        // 删除主题失败。
	MessageAdminRoleNotFound           MessageCode = "admin.role.notFound"             // 角色不存在。
	MessageAdminTestEmailRequired      MessageCode = "admin.mail.testEmailRequired"    // 测试邮箱不能为空。
	MessageAdminTestEmailFailed        MessageCode = "admin.mail.testFailed"           // 邮件配置测试失败，params.error 可带原始错误。
	MessageAdminTestEmailSuccess       MessageCode = "admin.mail.testSuccess"          // 邮件配置测试成功，params.email 表示测试邮箱。
)

const (
	// 审核策略（保留/禁用用户名、敏感词）
	MessageAuthUsernameReserved    MessageCode = "auth.username.reserved"          // 用户名被保留，不可使用。
	MessageAuthUsernameBanned      MessageCode = "auth.username.banned"            // 用户名被禁用，不可使用。
	MessageContentSensitiveBlocked MessageCode = "content.sensitive.blocked"       // 内容包含敏感词，已被拦截。
	MessageContentSensitiveReview  MessageCode = "content.sensitive.pendingReview" // 内容包含敏感词，已转入人工审核。
	MessageChatSensitiveBlocked    MessageCode = "chat.sensitive.blocked"          // 私信内容包含敏感词，已被拦截。

	// 存储设置
	MessageAdminStorageSaveFailed             MessageCode = "admin.storage.saveFailed"             // 存储设置保存失败，params.error 可带原始错误。
	MessageAdminStorageTestFailed             MessageCode = "admin.storage.testFailed"             // 存储连接测试失败，params.error 可带原始错误。
	MessageAdminStorageTestSuccess            MessageCode = "admin.storage.testSuccess"            // 存储连接测试成功。
	MessageAdminStorageMigrateFailed          MessageCode = "admin.storage.migrateFailed"          // 文件迁移任务创建失败，params.error 可带原始错误。
	MessageAdminStorageMigrateInvalidProvider MessageCode = "admin.storage.migrateInvalidProvider" // 文件迁移仅支持对象存储（S3 兼容）配置。

	// 数据导入导出
	MessageAdminDataExportFailed        MessageCode = "admin.data.exportFailed"        // 导出任务创建失败，params.error 可带原始错误。
	MessageAdminDataImportFailed        MessageCode = "admin.data.importFailed"        // 导入失败，params.error 可带原始错误。
	MessageAdminDataImportInvalidFormat MessageCode = "admin.data.importInvalidFormat" // 导入仅支持 JSON 文件。
	MessageAdminDataTaskNotFound        MessageCode = "admin.data.taskNotFound"        // 导出任务不存在。
	MessageAdminDataTaskNotReady        MessageCode = "admin.data.taskNotReady"        // 导出任务尚未完成。
	MessageAdminDataDownloadDenied      MessageCode = "admin.data.downloadDenied"      // 导出文件不可下载。

	// 审核队列
	MessageAdminReviewTargetInvalid MessageCode = "admin.review.targetInvalid" // 审核对象无效。
	MessageAdminReviewNotFound      MessageCode = "admin.review.notFound"      // 审核对象不存在。
	MessageAdminReviewProcessed     MessageCode = "admin.review.processed"     // 审核对象已处理。
	MessageAdminReviewFailed        MessageCode = "admin.review.failed"        // 审核操作失败，params.error 可带原始错误。
	// Agent（机器人账号）管理
	MessageAdminAgentUsernameInvalid MessageCode = "admin.agent.usernameInvalid" // 用户名格式不符合规则。
	MessageAdminAgentUsernameExists  MessageCode = "admin.agent.usernameExists"  // 用户名已存在。
	MessageAdminAgentWebhookInvalid  MessageCode = "admin.agent.webhookInvalid"  // Webhook 端点必须是合法的 http(s) URL。
	MessageAdminAgentCreateFailed    MessageCode = "admin.agent.createFailed"    // 创建 Agent 失败，params.error 可带原始错误。
	MessageAdminAgentUpdateFailed    MessageCode = "admin.agent.updateFailed"    // 更新 Agent 失败，params.error 可带原始错误。
	MessageAdminAgentNotFound        MessageCode = "admin.agent.notFound"        // Agent 不存在。
	MessageAdminAgentRotateFailed    MessageCode = "admin.agent.rotateFailed"    // 轮换令牌失败，params.error 可带原始错误。
	MessageAdminAgentDisableFailed   MessageCode = "admin.agent.disableFailed"   // 禁用 Agent 失败，params.error 可带原始错误。
	MessageAdminAgentNeedsRotate     MessageCode = "admin.agent.needsRotate"     // 该 Agent 的令牌已被吊销，重新启用前必须先轮换。
	MessageAdminAgentRoleNotAllowed  MessageCode = "admin.agent.roleNotAllowed"  // 机器人账号不允许被授予角色。
	MessageAdminAgentRotateConflict  MessageCode = "admin.agent.rotateConflict"  // 并发轮换冲突，请重试。
)
