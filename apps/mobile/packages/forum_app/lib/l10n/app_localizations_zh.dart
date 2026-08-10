// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appTitle => 'yourtj';

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
  String get authCasdoorLogin => '统一身份登录(Casdoor)';

  @override
  String get authRegisterSuccess => '注册成功,请登录';

  @override
  String get authResetEmailSent => '重置邮件已发送,请查收';

  @override
  String get authLoading => '处理中…';

  @override
  String get loginWelcome => '欢迎回到 yourtj';

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
  String get settingsOAuth => 'Casdoor 统一身份';

  @override
  String get settingsOAuthSub => 'OIDC 登录绑定';

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
}
