// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get myCourseReviewsTitle => '我的课评';

  @override
  String get myCourseReviewsEmpty => '还没有发布课评，去课程目录看看吧。';

  @override
  String get myCourseReviewHidden => '这条评价已被隐藏，可在此删除。';

  @override
  String get myCourseReviewUnavailable => '课程暂不可访问，你仍可管理自己的评价。';

  @override
  String get myCourseReviewOpen => '查看详情';

  @override
  String get courseReviewNetworkError => '网络连接失败，请检查网络后重试，已保留你的评价。';

  @override
  String get courseReviewUnknownError => '评价未能保存，服务端没有提供具体原因，请稍后重试。';

  @override
  String get commonHideKeyboard => '收起键盘';

  @override
  String get appTitle => 'YourTJ';

  @override
  String get navHome => '首页';

  @override
  String get navSearch => '搜索';

  @override
  String get navPublish => '发布';

  @override
  String get navMessages => '消息';

  @override
  String get navProfile => '我的';

  @override
  String get commonCancel => '取消';

  @override
  String get commonSave => '保存';

  @override
  String get commonLoading => '加载中…';

  @override
  String get commonLoadMore => '加载更多';

  @override
  String get commonRetry => '重试';

  @override
  String get commonClose => '关闭';

  @override
  String get commonSend => '发送';

  @override
  String get commonSearch => '搜索';

  @override
  String get commonEdit => '编辑';

  @override
  String get commonCurrent => '当前';

  @override
  String get commonEmpty => '暂无内容';

  @override
  String get commonBack => '返回';

  @override
  String get commonBackToTop => '返回顶部';

  @override
  String get commonUseLightTheme => '切换浅色模式';

  @override
  String get commonUseDarkTheme => '切换深色模式';

  @override
  String get timeAgoJustNow => '刚刚';

  @override
  String timeAgoMinutes(int count) {
    return '$count 分钟前';
  }

  @override
  String timeAgoHours(int count) {
    return '$count 小时前';
  }

  @override
  String timeAgoDays(int count) {
    return '$count 天前';
  }

  @override
  String timeAgoWeeks(int count) {
    return '$count 周前';
  }

  @override
  String timeAgoMonths(int count) {
    return '$count 个月前';
  }

  @override
  String timeAgoYears(int count) {
    return '$count 年前';
  }

  @override
  String get authLoginTitle => '登录账号';

  @override
  String get authRegisterTitle => '创建新账号';

  @override
  String get authForgotTitle => '重置密码';

  @override
  String get authLoginSubtitle => '欢迎回来，继续你的讨论和创作。';

  @override
  String get authRegisterSubtitle => '创建账号，加入校园里的每一次讨论。';

  @override
  String get authForgotSubtitle => '输入邮箱，我们会发送一封重置密码邮件。';

  @override
  String get authUsernameOrEmail => '用户名或邮箱';

  @override
  String get authUsername => '用户名';

  @override
  String get authEmail => '邮箱';

  @override
  String get authPassword => '密码';

  @override
  String get authNewPassword => '新密码';

  @override
  String get authConfirmPassword => '确认密码';

  @override
  String get authCaptcha => '验证码';

  @override
  String get authForgotPassword => '忘记密码？';

  @override
  String get authCreateAccount => '创建账号';

  @override
  String get authSendResetEmail => '发送重置邮件';

  @override
  String get authBackToLogin => '返回登录';

  @override
  String get authTwoFactorTitle => '两步验证';

  @override
  String get authTwoFactorCode => 'TOTP 验证码';

  @override
  String get authVerify => '验证';

  @override
  String get authGetCode => '获取验证码';

  @override
  String get authOidcLogin => '使用 yourtj 统一登录';

  @override
  String get authRegisterSuccess => '注册成功,请登录';

  @override
  String get authResetEmailSent => '重置邮件已发送,请查收';

  @override
  String get authLoading => '处理中…';

  @override
  String get authCacheClearFailed => '清除上一账号离线数据失败,请重试';

  @override
  String get authSessionSaveFailed => '安全保存新会话失败,请重试';

  @override
  String get loginWelcome => '欢迎回到 YourTJ';

  @override
  String get loginModeLogin => '登录';

  @override
  String get loginModeRegister => '注册';

  @override
  String get loginModeForgot => '找回密码';

  @override
  String get publishTitle => '发布话题';

  @override
  String get publishEditTitle => '编辑话题';

  @override
  String get publishPublish => '发布';

  @override
  String get publishSaveDraft => '保存草稿';

  @override
  String get publishTitleField => '标题';

  @override
  String get publishTitleHint => '请输入标题(5-100 字)';

  @override
  String get publishBodyPlaceholder => '正文内容…';

  @override
  String get publishTitleRequired => '标题不能为空';

  @override
  String get publishContentRequired => '内容不能为空';

  @override
  String get publishSuccess => '发布成功';

  @override
  String get publishSavedDraft => '已保存为草稿';

  @override
  String publishFailed(String error) {
    return '发布失败:$error';
  }

  @override
  String publishImageFailed(String error) {
    return '图片上传失败:$error';
  }

  @override
  String get composePreview => '预览';

  @override
  String get composeEdit => '编辑';

  @override
  String get publishBodyField => '正文';

  @override
  String get publishCategoryRequired => '请至少选择一个分类';

  @override
  String get publishPreviewEmpty => '开始输入后，这里会实时显示排版效果';

  @override
  String publishLoadFailed(String error) {
    return '编辑器数据加载失败:$error';
  }

  @override
  String get publishToolBold => '粗体';

  @override
  String get publishToolItalic => '斜体';

  @override
  String get publishToolStrike => '删除线';

  @override
  String get publishToolQuote => '引用';

  @override
  String get publishToolCode => '行内代码';

  @override
  String get publishToolBulletList => '无序列表';

  @override
  String get publishToolOrderedList => '有序列表';

  @override
  String get publishToolImage => '添加图片';

  @override
  String get publishRemoveImage => '移除图片';

  @override
  String topicReplyTarget(String name) {
    return '回复 $name';
  }

  @override
  String get topicTitle => '话题';

  @override
  String get topicReply => '回复';

  @override
  String get topicReplySuccess => '回复成功';

  @override
  String topicReplyFailed(String error) {
    return '回复失败:$error';
  }

  @override
  String get topicReplyHint => '写下你的评论…';

  @override
  String get topicReplyTargetUnavailable => '原回复不可见';

  @override
  String get topicReplying => '回复中…(点击取消)';

  @override
  String get topicReport => '举报帖子';

  @override
  String get topicReportHint => '请描述举报原因';

  @override
  String get topicReportSubmit => '提交';

  @override
  String get topicReportSubmitted => '举报已提交';

  @override
  String topicReportFailed(String error) {
    return '举报失败:$error';
  }

  @override
  String get topicWatch => '关注话题回复';

  @override
  String get topicUnwatch => '取消关注话题回复';

  @override
  String topicReplies(int count) {
    return '$count 回复';
  }

  @override
  String get topicNoTitle => '无标题';

  @override
  String get profileTitle => '个人主页';

  @override
  String get profileFollow => '关注';

  @override
  String get profileFollowing => '已关注';

  @override
  String get profileTopics => '主题';

  @override
  String get profileReplies => '回复';

  @override
  String get profileLikes => '获赞';

  @override
  String get profileFollowers => '粉丝';

  @override
  String get profileFollowingCount => '关注';

  @override
  String get profileBadges => '徽章';

  @override
  String get profileNoBadges => '暂无徽章';

  @override
  String get profileEmptyActivity => '暂无动态';

  @override
  String get profileEmptyTopics => '暂无主题';

  @override
  String get profileEmptyLikes => '暂无点赞';

  @override
  String get profileEmptyBookmarks => '暂无收藏';

  @override
  String get profileEmptyFollowing => '暂无关注';

  @override
  String get profileEmptyFollowers => '暂无粉丝';

  @override
  String get profileNotLoggedIn => '未登录';

  @override
  String get messagesTitle => '消息';

  @override
  String get messagesEmpty => '还没有私信会话';

  @override
  String get messagesEmptyDescription => '从社区里找一个人开始聊聊。';

  @override
  String get messagesSearchConversations => '搜索会话';

  @override
  String get messagesConversation => '私信对话';

  @override
  String get messagesStartChat => '开始聊天';

  @override
  String messagesFirstMessageTo(String user) {
    return '给 $user 发出第一条消息。';
  }

  @override
  String get messagesNoMessagesYet => '还没有消息';

  @override
  String get messagesEmptyDetail => '暂无消息,说点什么吧';

  @override
  String get messagesInputHint => '输入消息…';

  @override
  String messagesSendFailed(String error) {
    return '发送失败:$error';
  }

  @override
  String get notificationsTitle => '通知';

  @override
  String get notificationsEmpty => '暂无通知';

  @override
  String get notificationsMarkAllRead => '全部已读';

  @override
  String get notificationsAll => '全部';

  @override
  String get notificationsUnread => '未读';

  @override
  String get searchTitle => '搜索';

  @override
  String get searchHint => '搜索帖子、用户、分类…';

  @override
  String get searchEmpty => '输入关键词开始搜索';

  @override
  String get searchNoUsers => '没有匹配的用户';

  @override
  String get searchNoCategories => '没有匹配的分类';

  @override
  String get searchUnavailable => '搜索暂不可用';

  @override
  String get searchAll => '全部';

  @override
  String get searchTopics => '帖子';

  @override
  String get searchUsers => '用户';

  @override
  String get searchCategories => '分类';

  @override
  String get categoryTitle => '分类';

  @override
  String get settingsTitle => '设置';

  @override
  String get settingsTabProfile => '资料';

  @override
  String get settingsTabAccount => '账户';

  @override
  String get settingsTabPrivacy => '隐私';

  @override
  String get settingsTabBinding => '绑定';

  @override
  String get settingsTabSecurity => '安全';

  @override
  String get settingsSectionProfile => '个人资料';

  @override
  String get settingsNickname => '昵称';

  @override
  String get settingsNicknameEdit => '编辑显示昵称';

  @override
  String get settingsBio => '个人简介';

  @override
  String get settingsBioEdit => '编辑简介与签名';

  @override
  String get settingsAvatar => '头像';

  @override
  String get settingsAvatarUpload => '上传头像(前端转 webp)';

  @override
  String get settingsAvatarUploading => '上传中…';

  @override
  String settingsAvatarUploadFailed(String error) {
    return '头像上传失败:$error';
  }

  @override
  String get settingsEmail => '邮箱';

  @override
  String get settingsEmailEdit => '修改绑定邮箱';

  @override
  String get settingsEmailUpdated => '邮箱已更新,请验证';

  @override
  String get settingsEmailOAuthReauthRequired =>
      '此 OAuth 关联账号无法通过当前密码验证，请通过 OAuth 重新认证或联系管理员。';

  @override
  String settingsEmailFailed(String error) {
    return '邮箱修改失败:$error';
  }

  @override
  String get settingsNewEmail => '新邮箱';

  @override
  String get settingsChangePassword => '修改密码';

  @override
  String get settingsChangePasswordSub => '更换登录密码';

  @override
  String get settingsCurrentPassword => '当前密码';

  @override
  String get settingsPasswordUpdated => '密码已更新';

  @override
  String settingsPasswordFailed(String error) {
    return '修改失败:$error';
  }

  @override
  String get settingsBadge => '徽章';

  @override
  String get settingsBadgeNone => '未佩戴';

  @override
  String settingsBadgeCurrent(String name) {
    return '当前:$name';
  }

  @override
  String get settingsBadgePick => '选择佩戴徽章';

  @override
  String get settingsBadgeNoOptions => '暂无可佩戴徽章';

  @override
  String get settingsBadgeUpdated => '已更新佩戴徽章';

  @override
  String settingsBadgeFailed(String error) {
    return '佩戴失败:$error';
  }

  @override
  String get settingsOAuth => '外部账号关联';

  @override
  String get settingsOAuthSub => 'GitHub 与 Google 登录关联';

  @override
  String get settingsOAuthManage => 'OAuth 绑定管理';

  @override
  String get settingsOAuthBindings => 'OAuth 绑定管理';

  @override
  String get settingsBound => '已绑定';

  @override
  String get settingsUnbound => '未绑定';

  @override
  String get settingsUnbind => '解绑';

  @override
  String get settingsUnboundDone => '已解绑';

  @override
  String settingsUnbindFailed(String error) {
    return '解绑失败:$error';
  }

  @override
  String settingsLoadBindingsFailed(String error) {
    return '加载绑定失败:$error';
  }

  @override
  String get settingsPrivacyDirect => '私信仅好友可见';

  @override
  String get settingsPrivacyLikes => '公开我的点赞';

  @override
  String get settingsSessions => '会话管理';

  @override
  String get settingsSessionsEmpty => '暂无会话';

  @override
  String get settingsRevokeAll => '吊销全部会话';

  @override
  String get settingsRevoked => '已吊销会话';

  @override
  String settingsRevokeFailed(String error) {
    return '吊销失败:$error';
  }

  @override
  String get settingsRevokeAllDone => '已吊销全部会话';

  @override
  String settingsOpFailed(String error) {
    return '操作失败:$error';
  }

  @override
  String get settingsAppearance => '外观';

  @override
  String get settingsDarkMode => '深色模式';

  @override
  String get settingsDarkCurrent => '当前:深色';

  @override
  String get settingsLightCurrent => '当前:浅色';

  @override
  String get settingsAbout => '关于';

  @override
  String get settingsAboutVersion => '版本 0.1.0 · 同济大学校园论坛';

  @override
  String get settingsEditProfile => '编辑资料';

  @override
  String get settingsSignature => '签名';

  @override
  String get settingsSaveInfo => '保存';

  @override
  String get settingsInfoSaved => '资料已更新';

  @override
  String settingsInfoFailed(String error) {
    return '资料更新失败:$error';
  }

  @override
  String get settingsUserDataLoading => '账户数据加载中,请稍后再试';

  @override
  String get settingsFillComplete => '请填写完整';

  @override
  String get settingsSecondPhase => '二期接入';

  @override
  String get settingsTotpTitle => '两步验证(TOTP)';

  @override
  String get settingsTotpEnable => '启用';

  @override
  String get settingsTotpDisable => '禁用';

  @override
  String get settingsTotpPasswordPrompt => '输入密码以管理 TOTP';

  @override
  String get settingsTotpSetupSecret => '在验证器应用中扫描或输入密钥';

  @override
  String get settingsTotpCode => '输入 6 位动态验证码';

  @override
  String get settingsTotpRecoveryCodes => '恢复码(请妥善保存):';

  @override
  String get settingsTotpEnabled => 'TOTP 已启用';

  @override
  String get settingsTotpDisabled => 'TOTP 已禁用';

  @override
  String settingsTotpFailed(String error) {
    return 'TOTP 操作失败:$error';
  }

  @override
  String get settingsTotpDisableTitle => '禁用 TOTP';

  @override
  String get settingsTotpEnableTitle => '启用 TOTP';

  @override
  String get settingsTotpPassword => '输入密码';

  @override
  String get settingsTotpNext => '下一步';

  @override
  String get settingsTotpScanSecret => '扫描或输入密钥';

  @override
  String get settingsTotpDone => '完成';

  @override
  String get settingsTotpUnavailable => 'TOTP 不可用';

  @override
  String get draftsTitle => '草稿箱';

  @override
  String get draftsEmpty => '暂无草稿';

  @override
  String get draftsNew => '新建草稿';

  @override
  String get draftsBlocked => '被屏蔽';

  @override
  String draftsMetaCreated(Object date) {
    return '创建于 $date';
  }

  @override
  String draftsMetaViews(Object count) {
    return '$count 次浏览';
  }

  @override
  String draftsMetaReplies(Object count) {
    return '$count 条回复';
  }

  @override
  String get messagesNew => '新私信';

  @override
  String get messagesSearchUsers => '搜索用户';

  @override
  String get messagesNoContactableUsers => '暂无可联系用户';

  @override
  String get settingsLogout => '退出登录';

  @override
  String get settingsLogoutConfirm => '确定退出 yourtj 吗?';

  @override
  String get commonParseFailed => '页面数据解析失败';

  @override
  String get commonLoadFailed => '加载失败';

  @override
  String get topicEmpty => '暂无话题';

  @override
  String settingsAvatarUploaded(String url) {
    return '头像已上传:$url';
  }

  @override
  String get settingsImageDecodeFailed => '图片解码失败';

  @override
  String dateMonthDayTime(int month, int day, String time) {
    return '$month月$day日 $time';
  }

  @override
  String dateYearMonthDayTime(int year, int month, int day, String time) {
    return '$year年$month月$day日 $time';
  }

  @override
  String topicFloorSelected(Object floor) {
    return '已跳转到 $floor 楼';
  }

  @override
  String get sortLatest => '最新';

  @override
  String get sortHot => '热门';

  @override
  String get sortPopular => '流行';

  @override
  String get topicFeedModeList => '列表';

  @override
  String get topicFeedModeCard => '卡片';

  @override
  String get topicNewTopic => '新建话题';

  @override
  String get scheduleTitle => '排课器';

  @override
  String get scheduleTabTimetable => '方案预览';

  @override
  String get scheduleTabPick => '选课';

  @override
  String get scheduleTerm => '学期';

  @override
  String get scheduleGrade => '年级';

  @override
  String get scheduleMajor => '专业';

  @override
  String get scheduleSyncLatest => '同步最新';

  @override
  String get scheduleDataOutdated => '教务数据已更新，点击同步';

  @override
  String scheduleSyncedTo(String date) {
    return '已同步至 $date';
  }

  @override
  String get scheduleSyncConflictTitle => '排课方案同步冲突';

  @override
  String get scheduleSyncConflictBody => '本地与云端的排课方案不一致，保留哪一份？';

  @override
  String get scheduleSyncUseCloud => '使用云端';

  @override
  String get scheduleSyncKeepLocal => '保留本地';

  @override
  String get scheduleWeekAll => '全部周次';

  @override
  String scheduleWeekN(int week) {
    return '第 $week 周';
  }

  @override
  String get scheduleCurrentWeek => '当前周次';

  @override
  String get schedulePlanNew => '新建方案';

  @override
  String get schedulePlanRename => '重命名方案';

  @override
  String get schedulePlanDelete => '删除方案';

  @override
  String get schedulePlanClear => '清空课程';

  @override
  String schedulePlanN(int n) {
    return '方案 $n';
  }

  @override
  String scheduleStatsCourses(int count) {
    return '$count 门';
  }

  @override
  String get scheduleStatsCredits => '学分';

  @override
  String get scheduleStatsHours => '学时';

  @override
  String get scheduleStatsConflicts => '冲突';

  @override
  String get scheduleConflictBadge => '冲突';

  @override
  String scheduleConflictWith(String course) {
    return '与「$course」时间冲突';
  }

  @override
  String scheduleConflictsWith(String course, int count) {
    return '与「$course」等$count门课程冲突';
  }

  @override
  String get scheduleConflictCanAdd => '仍可加入';

  @override
  String get scheduleAddCustomEvent => '添加占位';

  @override
  String get scheduleCustomEventLabel => '有事';

  @override
  String get scheduleExportPng => '导出图片';

  @override
  String get scheduleExportCsv => '导出 CSV';

  @override
  String get scheduleDegraded => '检索降级中，结果可能不全';

  @override
  String get scheduleNoReviewData => '暂无课评数据';

  @override
  String get schedulePickClass => '选择教学班';

  @override
  String get scheduleCompulsory => '必修';

  @override
  String get scheduleOptional => '选修';

  @override
  String get scheduleSearchHint => '搜索课名/课号/教师';

  @override
  String get scheduleStaged => '备选';

  @override
  String get scheduleSelected => '已选';

  @override
  String get scheduleRemoveCourse => '退课';

  @override
  String get scheduleParityOdd => '单周';

  @override
  String get scheduleParityEven => '双周';

  @override
  String scheduleWeeksN(String range) {
    return '共$range周';
  }

  @override
  String get scheduleDayMon => '周一';

  @override
  String get scheduleDayTue => '周二';

  @override
  String get scheduleDayWed => '周三';

  @override
  String get scheduleDayThu => '周四';

  @override
  String get scheduleDayFri => '周五';

  @override
  String get scheduleDaySat => '周六';

  @override
  String get scheduleDaySun => '周日';

  @override
  String get scheduleMorning => '上午';

  @override
  String get scheduleAfternoon => '下午';

  @override
  String get scheduleEvening => '晚上';

  @override
  String get coursesTitle => '课程目录';

  @override
  String get coursesSearchHint => '搜索课程';

  @override
  String get coursesFilterDepartment => '院系';

  @override
  String get coursesFilterTerm => '学期';

  @override
  String get coursesFilterCampus => '校区';

  @override
  String get coursesFilterInstructor => '教师';

  @override
  String get coursesOnlyWithReviews => '只看有评价';

  @override
  String coursesRatingCount(int count) {
    return '$count 条评价';
  }

  @override
  String get coursesNoRating => '暂无评分';

  @override
  String get courseDetailReviews => '课评';

  @override
  String get courseDetailOfferings => '开课班级';

  @override
  String get courseDetailLineage => '课程沿革';

  @override
  String get courseDetailRelated => '相关课程';

  @override
  String get courseDetailAiSummary => 'AI 总结';

  @override
  String get courseDetailAiSummaryEmpty => '暂无 AI 总结';

  @override
  String get courseBookmark => '收藏';

  @override
  String get courseBookmarked => '已收藏';

  @override
  String get courseWriteReview => '写课评';

  @override
  String get reviewAnonymous => '匿名评价';

  @override
  String get reviewSubmit => '发布评价';

  @override
  String get reviewHelpful => '有用';

  @override
  String get reviewsEmpty => '还没有评价';

  @override
  String get wikiTitle => 'Wiki';

  @override
  String get wikiRecent => '最近更新';

  @override
  String get wikiEditOnGithub => '在 GitHub 编辑';

  @override
  String get wikiToc => '目录';

  @override
  String get wikiNamespaces => '命名空间';

  @override
  String wikiViewCount(int count) {
    return '$count 次浏览';
  }

  @override
  String get settingsPush => '推送通知';

  @override
  String get settingsPushDenied => '系统通知权限未授予，点击前往设置';

  @override
  String get settingsFollowSiteTheme => '跟随站点主题';

  @override
  String get settingsFollowSiteThemeDesc => '使用服务器下发的站点配色';

  @override
  String get entryCourses => '课程';

  @override
  String get entrySchedule => '课表';

  @override
  String get entryWiki => 'Wiki';

  @override
  String get navCampus => '校园';

  @override
  String get campusTitle => '在同济，发现更多';

  @override
  String get campusSubtitle => '选好课，排好每一天，分享你的校园经验。';

  @override
  String get campusCoursesHint => '查课程、看评价，选课更有底气';

  @override
  String get campusScheduleHint => '多方案排课、冲突提示与课表导出';

  @override
  String get campusWikiHint => '校园指南，与大家一起完善的知识库';

  @override
  String get publishMoment => '瞬间';

  @override
  String get publishQuestion => '提问';

  @override
  String get publishArticle => '文章';

  @override
  String get publishNext => '下一步';

  @override
  String get publishGallery => '先选图片，再记录这一刻';

  @override
  String get publishGalleryHint => '最多 9 张，长按拖动排序';

  @override
  String get publishFormatting => '文字格式';

  @override
  String get publishClassification => '选择分区与标签';

  @override
  String get publishLeaveTitle => '保留这次创作？';

  @override
  String get publishLeaveBody => '尚有未保存的内容。返回编辑，或放弃本次修改。';

  @override
  String get publishDiscard => '放弃修改';

  @override
  String get publishContinue => '继续编辑';

  @override
  String get publishImageOnlyTitle => '分享此刻';

  @override
  String get publishTypeLocked => '编辑已有帖子时保留原类型';

  @override
  String get profileMore => '更多功能';

  @override
  String get profileContent => '内容管理';

  @override
  String get profileTrash => '回收站';

  @override
  String get profileSecurity => '账号、安全与隐私';

  @override
  String get profileAdmin => '管理工作台';

  @override
  String get fabDiscussion => '前往讨论区';

  @override
  String get fabRefresh => '刷新并回到顶部';

  @override
  String get contentRestore => '恢复';

  @override
  String get contentDelete => '删除';

  @override
  String get contentPurge => '永久删除';

  @override
  String get contentDeleteConfirm => '内容会移入回收站，符合恢复条件时可在 30 天内恢复。';

  @override
  String get contentPurgeConfirm => '此操作不可撤销。正文和图片将被永久清除。';

  @override
  String get contentPassword => '请输入当前密码以确认操作';

  @override
  String get contentSelected => '已选择';

  @override
  String get contentSelectAll => '全选当前列表';

  @override
  String get contentEmpty => '这里还没有内容';

  @override
  String get contentTrashHint => '可恢复的内容保留 30 天；审核移除的内容不能自行恢复。';

  @override
  String get commonConfirm => '确认';

  @override
  String get adminUnavailable => '管理后台暂不可用。请确认账号有管理权限、登录未过期，并检查网络后重试。';

  @override
  String get adminDownloadFailed => '导出下载失败，请稍后重试。';

  @override
  String get publishGalleryTooMany => '内容最多支持 9 张图片，请先减少图片或继续使用文章编辑器。';

  @override
  String get profileActivity => '动态';

  @override
  String get profileBookmarks => '收藏';

  @override
  String get profileModeration => '审核工作台';

  @override
  String get settingsCloseAccount => '注销账号';

  @override
  String get settingsCloseAccountWarning =>
      '注销不可撤销，所有设备都会退出登录。你可以保留匿名化后的历史内容，或请求删除自己的内容；受保留规则限制的内容可能仍会保留。请输入当前账号密码以确认。';

  @override
  String get settingsCloseKeepContent => '保留匿名内容';

  @override
  String get settingsCloseDeleteContent => '请求删除内容';

  @override
  String get publishUndo => '撤销';

  @override
  String get publishRedo => '重做';

  @override
  String get publishHeading => '标题';

  @override
  String get publishToolLink => '插入链接';

  @override
  String get publishLinkInvalid => '请输入有效的 http、https 或邮件链接。';

  @override
  String get publishPhotoLibrary => '从相册选择';

  @override
  String get publishCamera => '拍摄照片';

  @override
  String get settingsWebsiteName => '网站名称';

  @override
  String get settingsWebsite => '个人网站';

  @override
  String get settingsSocialLinks => '社交链接';

  @override
  String get settingsProfileLanguage => '资料语言';

  @override
  String get settingsInvalidLink => '请输入有效的 http 或 https 链接';

  @override
  String get siteInfoTitle => '关于社区';

  @override
  String get siteInfoLinks => '友情链接';

  @override
  String get siteInfoSponsors => '支持与赞助';

  @override
  String get siteInfoTerms => '服务条款';

  @override
  String get siteInfoPrivacy => '隐私政策';

  @override
  String get siteInfoEmpty => '暂无公开内容';

  @override
  String get settingsUsernameHint => '这是用于登录的用户名，需符合站点命名规则。';

  @override
  String get settingsUsernameUpdated => '用户名已更新';

  @override
  String get settingsPresetAvatar => '选择预设头像';

  @override
  String get coursesManagement => '课程管理';

  @override
  String get coursesReviewModeration => '课评审核';

  @override
  String get settingsCover => '主页封面';

  @override
  String get settingsCoverDescription => '选择图片并调整裁切范围';

  @override
  String get settingsCoverRemove => '移除封面';

  @override
  String get settingsCoverRemoveConfirm => '移除后将恢复默认主页背景。';

  @override
  String get settingsCoverMinSize => '请选择至少 1200 × 240 像素的图片';

  @override
  String get settingsCoverSafeArea => '中间明亮区域是手机上主要显示的部分；完整横图会保留给宽屏。';

  @override
  String get settingsCropHint => '拖动图片调整位置，双指缩放查看效果';

  @override
  String get settingsCropPreview => '图片裁切预览';

  @override
  String get settingsCropZoom => '缩放';

  @override
  String get settingsCropReset => '重置位置';

  @override
  String get settingsImageSaved => '图片已更新';

  @override
  String settingsImageTooLarge(int maxMb) {
    return '图片不能超过 $maxMb MB';
  }

  @override
  String get settingsOAuthUnavailable => '站点尚未启用';

  @override
  String get settingsOAuthOpenBrowser => '在浏览器中管理绑定';

  @override
  String settingsOAuthBrowserHint(String username) {
    return '请在浏览器中使用当前账号 @$username 登录并完成绑定，返回 App 后会刷新绑定状态。';
  }

  @override
  String get commonRefresh => '刷新';

  @override
  String get topicEarliest => '最早';

  @override
  String get topicLatest => '最新';

  @override
  String get topicEarlierReplies => '加载更早回复';

  @override
  String get topicHistory => '修订记录';

  @override
  String get topicHistoryUnavailable => '此版本的内容不可见';

  @override
  String get topicHistoryEmpty => '暂无修订记录';

  @override
  String get topicDeleteConfirm => '确定删除这条内容？可以在回收站中查看可恢复的内容。';

  @override
  String get topicModerateBan => '屏蔽内容';

  @override
  String get topicModerateUnban => '恢复显示';

  @override
  String get topicModerateConfirm => '确定更改这条内容的可见状态？';

  @override
  String get topicShare => '分享';

  @override
  String get topicEditReply => '编辑回复';

  @override
  String get topicRemoved => '这条内容已被删除或移除';

  @override
  String get topicBookmark => '收藏';

  @override
  String get topicBookmarked => '取消收藏';

  @override
  String get topicLike => '点赞';

  @override
  String get authEmailPrefix => '邮箱前缀';

  @override
  String get authEmailDomain => '邮箱域名';

  @override
  String get authAgreePolicies => '我已阅读并同意已发布的协议';

  @override
  String get authPasswordMismatch => '两次输入的密码不一致';

  @override
  String get schedulerWebTitle => '完整版排课器，请到网页端体验';

  @override
  String get schedulerWebAction => '前往 f.yourtj.de';

  @override
  String get schedulerPlanDisclaimer => '这是你的选课规划，最终选课结果以教务为准。';

  @override
  String get campusExploreCourses => '从同学的真实评价，发现适合你的课';

  @override
  String get campusPlanTitle => '把感兴趣的课，排成自己的方案';

  @override
  String get campusPlanDescription => '比较教学班、检查时间冲突，再决定怎么选。';

  @override
  String get loginGoogle => '使用 Google 登录';

  @override
  String get loginGithub => '使用 GitHub 登录';

  @override
  String get wikiSearchUnavailable => '搜索暂不可用，你仍可以返回目录浏览。';

  @override
  String get wikiSearchHint => '搜索校园知识';

  @override
  String get wikiExploreTitle => '你的校园生活指南';

  @override
  String notificationComment(String actor) {
    return '$actor 评论了你的主题';
  }

  @override
  String notificationPostReply(String actor) {
    return '$actor 回复了你';
  }

  @override
  String notificationTopicPost(String actor) {
    return '$actor 在你关注的主题中发表了回复';
  }

  @override
  String notificationFollow(String actor) {
    return '$actor 关注了你';
  }

  @override
  String notificationLike(String actor) {
    return '$actor 赞了你的回复';
  }

  @override
  String notificationWikiUpdated(String actor) {
    return '$actor 更新了你关注的 Wiki 页面';
  }

  @override
  String notificationBadge(String badge) {
    return '你获得了“$badge”徽章';
  }

  @override
  String get notificationNew => '新通知';

  @override
  String get notificationSomeone => '有人';

  @override
  String get profileRoleAdmin => '管理员';

  @override
  String get profileActionSignup => '加入社区';

  @override
  String get profileActionPost => '发布主题';

  @override
  String get profileActionLike => '点赞';

  @override
  String get profileActionFollow => '关注用户';

  @override
  String get profileActionComment => '发表回复';

  @override
  String get replyQuoteExpand => '展开引用';

  @override
  String get replyQuoteCollapse => '收起引用';

  @override
  String get notificationBadgeUnnamed => '你获得了一枚新徽章';

  @override
  String get settingsAppLanguage => '应用语言';

  @override
  String get settingsLanguageSystem => '跟随系统';

  @override
  String scheduleGradeYear(String year) {
    return '$year 级';
  }

  @override
  String get schedulePeriods => '节次';

  @override
  String get scheduleWeeksLabel => '周次';

  @override
  String schedulePeriodRange(String range) {
    return '第 $range 节';
  }

  @override
  String get courseCopyCreditUnit => '学分';

  @override
  String get courseCopyNoTeacher => '无教师';

  @override
  String get courseCopyCatalogEmptyTitle => '暂无课程';

  @override
  String get courseCopyCatalogEmptyDescription => '课程目录尚未导入。';

  @override
  String get courseCopyNoFilterResults => '没有找到符合筛选条件的课程。';

  @override
  String get courseCopyNoFilterResultsDescription => '试试调整或清除筛选，查看更广的课程。';

  @override
  String get courseCopyClearSearch => '清空搜索';

  @override
  String get courseCopyDone => '完成';

  @override
  String get courseCopyNoOptions => '暂无可选项';

  @override
  String courseCopySelectedCount(int count) {
    return '已选 $count 项';
  }

  @override
  String get courseCopyInstructorInputHint => '输入教师姓名，回车添加';

  @override
  String get courseCopyInstructorAdd => '添加';

  @override
  String get courseCopyInstructorEmptyHint => '通过上方输入框添加教师';

  @override
  String get courseCopyAliasesLabel => '别名：';

  @override
  String get courseCopyLegacyNamesLabel => '原名：';

  @override
  String get courseCopyReviewScopeTeam => '教学团队';

  @override
  String get courseCopyReviewScopeCourse => '课程级评价';

  @override
  String get courseCopyTeamInstructorsPrefix => '教学团队 · ';

  @override
  String courseCopyTeamInstructorsSuffix(int count) {
    return '等 $count 位教师';
  }

  @override
  String get courseCopyRatingTitle => '课程评分';

  @override
  String get courseCopyRatingOutOf => '/ 5.0';

  @override
  String get courseCopyNoRatingQuiet => '暂无评分';

  @override
  String get courseCopyOfferingsEmpty => '暂无开课记录。';

  @override
  String get courseCopyOfferingFocusLabel => '只看该教学班的评价';

  @override
  String get courseCopyOfferingFocusClear => '查看全部评价';

  @override
  String get courseCopySummaryGenerated => '已生成';

  @override
  String get courseCopySummaryKeywords => '关键词';

  @override
  String get courseCopySummaryPros => '优点';

  @override
  String get courseCopySummaryCons => '缺点';

  @override
  String get courseCopySummaryRepresentativeReviews => '代表性评价';

  @override
  String get courseCopySummarySentimentPositive => '好评';

  @override
  String get courseCopySummarySentimentNeutral => '中立';

  @override
  String get courseCopySummarySentimentNegative => '差评';

  @override
  String get courseCopySummaryRefresh => '刷新';

  @override
  String get courseCopySummaryExpand => '展开';

  @override
  String get courseCopySummaryCollapse => '收起';

  @override
  String get courseCopySummaryDisclaimer => '以上内容由 AI 基于学生评价自动生成，仅供参考，不构成选课建议。';

  @override
  String get courseCopySummaryInsufficient => '评价数量不足，暂无法生成 AI 总结。';

  @override
  String get courseCopySummaryLoadFailed => 'AI 总结生成失败，请稍后重试。';

  @override
  String get courseCopyWriteReviewTitle => '写一条评价';

  @override
  String get courseCopyEditReviewTitle => '编辑评价';

  @override
  String get courseCopySelectOffering => '选择开课实例';

  @override
  String get courseCopyRatingLabel => '评分';

  @override
  String get courseCopyContentLabel => '评价内容';

  @override
  String get courseCopyContentPlaceholder => '写下你的学习体验、课程质量或给分情况…';

  @override
  String get courseCopyRatingRequired => '请选择 1–5 星评分。';

  @override
  String get courseCopyContentRequired => '评价内容不能为空。';

  @override
  String get courseCopyAnonymousLabel => '匿名发布（对公众隐藏身份）';

  @override
  String get courseCopySubmitSuccess => '已提交';

  @override
  String get courseCopyUpdateSuccess => '已更新';

  @override
  String get courseCopyDelete => '删除';

  @override
  String get courseCopyDeleteReviewTitle => '删除评价';

  @override
  String get courseCopyConfirmDeleteReview => '确定删除这条评价吗？删除后不可恢复。';

  @override
  String get courseCopyReviewDeleted => '评价已删除';

  @override
  String get courseCopyOperationFailed => '操作失败，请稍后重试。';

  @override
  String get courseCopyReviewsLoadFailed => '评价加载失败，请稍后重试。';

  @override
  String get courseCopyAuthorAnonymousLabel => '匿名同学';

  @override
  String get courseCopyAuthorLegacyLabel => '历史匿名评价';

  @override
  String get courseCopyRelatedTeacherCoursesTitle => '同教师其他课程';

  @override
  String get courseCopyRelatedOtherTeachersTitle => '同课程其他教师';

  @override
  String get courseCopyRelatedEmpty => '暂无相关内容';

  @override
  String get courseCopyRelationEquivalent => '等价';

  @override
  String get courseCopyRelationRenamed => '改名';

  @override
  String get courseCopyRelationSplit => '拆分';

  @override
  String get courseCopyRelationMerged => '合并';

  @override
  String get courseCopyRelationRelated => '相关';

  @override
  String get courseCopySummaryConsensusStrongRecommend => '强烈推荐';

  @override
  String get courseCopySummaryConsensusRecommend => '推荐';

  @override
  String get courseCopySummaryConsensusNeutral => '褒贬不一';

  @override
  String get courseCopySummaryConsensusCautious => '谨慎选择';

  @override
  String get courseCopySummaryConsensusNotRecommend => '不推荐';

  @override
  String get courseCopySummaryConsensusTextStrongRecommend => '多数同学强烈推荐这门课程。';

  @override
  String get courseCopySummaryConsensusTextRecommend => '多数同学推荐这门课程。';

  @override
  String get courseCopySummaryConsensusTextNeutral => '同学们对这门课程的评价褒贬不一。';

  @override
  String get courseCopySummaryConsensusTextCautious => '多数同学建议谨慎选择这门课程。';

  @override
  String get courseCopySummaryConsensusTextNotRecommend => '多数同学不推荐这门课程。';

  @override
  String get topicJoinDiscussion => '参与讨论';

  @override
  String get updateCheck => '检查更新';

  @override
  String get updateAvailable => '发现新版本';

  @override
  String get updateLatest => '已是最新版本';

  @override
  String get updateFailed => '暂时无法检查或下载更新，请稍后重试。';

  @override
  String get updateDownload => '下载更新';

  @override
  String get updateSkip => '忽略此版本';

  @override
  String get updateLater => '稍后';

  @override
  String get updatePreparing => '正在选择最快的下载来源…';

  @override
  String get updateDownloading => '正在下载…';

  @override
  String get updateReady => '更新已验证，可以开始安装。';

  @override
  String get updateInstall => '安装更新';

  @override
  String get updatePermission => '请允许 YourTJ 安装应用，然后返回并再次点击安装。';

  @override
  String get updateRetry => '重试';
}
