// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'yourtj';

  @override
  String get navHome => 'Home';

  @override
  String get navSearch => 'Search';

  @override
  String get navPublish => 'Publish';

  @override
  String get navMessages => 'Messages';

  @override
  String get navProfile => 'Me';

  @override
  String get commonCancel => 'Cancel';

  @override
  String get commonSave => 'Save';

  @override
  String get commonLoading => 'Loading…';

  @override
  String get commonLoadMore => 'Load more';

  @override
  String get commonRetry => 'Retry';

  @override
  String get commonClose => 'Close';

  @override
  String get commonSend => 'Send';

  @override
  String get commonSearch => 'Search';

  @override
  String get commonEdit => 'Edit';

  @override
  String get commonCurrent => 'Current';

  @override
  String get commonEmpty => 'Nothing here';

  @override
  String get commonBack => 'Back';

  @override
  String get timeAgoJustNow => 'just now';

  @override
  String timeAgoMinutes(int count) {
    return '$count min ago';
  }

  @override
  String timeAgoHours(int count) {
    return '$count h ago';
  }

  @override
  String timeAgoDays(int count) {
    return '$count d ago';
  }

  @override
  String timeAgoWeeks(int count) {
    return '$count w ago';
  }

  @override
  String timeAgoMonths(int count) {
    return '$count mo ago';
  }

  @override
  String timeAgoYears(int count) {
    return '$count y ago';
  }

  @override
  String get authLoginTitle => 'Sign in';

  @override
  String get authRegisterTitle => 'Create account';

  @override
  String get authForgotTitle => 'Reset password';

  @override
  String get authLoginSubtitle =>
      'Welcome back. Continue your discussions and writing.';

  @override
  String get authRegisterSubtitle =>
      'Create an account and join the campus conversation.';

  @override
  String get authForgotSubtitle =>
      'Enter your email and we will send a reset link.';

  @override
  String get authUsernameOrEmail => 'Username or email';

  @override
  String get authUsername => 'Username';

  @override
  String get authEmail => 'Email';

  @override
  String get authPassword => 'Password';

  @override
  String get authNewPassword => 'New password';

  @override
  String get authConfirmPassword => 'Confirm password';

  @override
  String get authCaptcha => 'Captcha';

  @override
  String get authForgotPassword => 'Forgot password?';

  @override
  String get authCreateAccount => 'Create account';

  @override
  String get authSendResetEmail => 'Send reset email';

  @override
  String get authBackToLogin => 'Back to login';

  @override
  String get authTwoFactorTitle => 'Two-factor authentication';

  @override
  String get authTwoFactorCode => 'TOTP code';

  @override
  String get authVerify => 'Verify';

  @override
  String get authGetCode => 'Get code';

  @override
  String get authOidcLogin => 'Sign in with yourtj';

  @override
  String get authRegisterSuccess => 'Registered successfully, please sign in';

  @override
  String get authResetEmailSent => 'Reset email sent, please check your inbox';

  @override
  String get authLoading => 'Processing…';

  @override
  String get loginWelcome => 'Welcome back to yourtj';

  @override
  String get loginModeLogin => 'Sign in';

  @override
  String get loginModeRegister => 'Sign up';

  @override
  String get loginModeForgot => 'Forgot password';

  @override
  String get publishTitle => 'Publish topic';

  @override
  String get publishEditTitle => 'Edit topic';

  @override
  String get publishPublish => 'Publish';

  @override
  String get publishSaveDraft => 'Save draft';

  @override
  String get publishTitleField => 'Title';

  @override
  String get publishTitleHint => 'Enter title (5-100 chars)';

  @override
  String get publishBodyPlaceholder => 'Content…';

  @override
  String get publishTitleRequired => 'Title is required';

  @override
  String get publishContentRequired => 'Content is required';

  @override
  String get publishSuccess => 'Published successfully';

  @override
  String get publishSavedDraft => 'Saved as draft';

  @override
  String publishFailed(String error) {
    return 'Publish failed: $error';
  }

  @override
  String publishImageFailed(String error) {
    return 'Image upload failed: $error';
  }

  @override
  String get topicTitle => 'Topic';

  @override
  String get topicReply => 'Reply';

  @override
  String get topicReplySuccess => 'Reply posted';

  @override
  String topicReplyFailed(String error) {
    return 'Reply failed: $error';
  }

  @override
  String get topicReplyHint => 'Write a comment…';

  @override
  String get topicReplying => 'Replying… (tap to cancel)';

  @override
  String get topicReport => 'Report post';

  @override
  String get topicReportHint => 'Describe the reason';

  @override
  String get topicReportSubmit => 'Submit';

  @override
  String get topicReportSubmitted => 'Report submitted';

  @override
  String topicReportFailed(String error) {
    return 'Report failed: $error';
  }

  @override
  String topicReplies(int count) {
    return '$count replies';
  }

  @override
  String get topicNoTitle => 'Untitled';

  @override
  String get profileTitle => 'Profile';

  @override
  String get profileFollow => 'Follow';

  @override
  String get profileFollowing => 'Following';

  @override
  String get profileTopics => 'Topics';

  @override
  String get profileReplies => 'Replies';

  @override
  String get profileLikes => 'Likes';

  @override
  String get profileFollowers => 'Followers';

  @override
  String get profileFollowingCount => 'Following';

  @override
  String get profileBadges => 'Badges';

  @override
  String get profileNoBadges => 'No badges';

  @override
  String get profileEmptyActivity => 'No activity yet';

  @override
  String get profileEmptyTopics => 'No topics yet';

  @override
  String get profileEmptyLikes => 'No likes yet';

  @override
  String get profileEmptyBookmarks => 'No bookmarks yet';

  @override
  String get profileEmptyFollowing => 'Not following anyone';

  @override
  String get profileEmptyFollowers => 'No followers yet';

  @override
  String get profileNotLoggedIn => 'Not signed in';

  @override
  String get messagesTitle => 'Messages';

  @override
  String get messagesEmpty => 'No conversations';

  @override
  String get messagesEmptyDetail => 'No messages yet, say hi!';

  @override
  String get messagesInputHint => 'Type a message…';

  @override
  String messagesSendFailed(String error) {
    return 'Send failed: $error';
  }

  @override
  String get notificationsTitle => 'Notifications';

  @override
  String get notificationsEmpty => 'No notifications';

  @override
  String get notificationsMarkAllRead => 'Mark all read';

  @override
  String get notificationsAll => 'All';

  @override
  String get notificationsUnread => 'Unread';

  @override
  String get searchTitle => 'Search';

  @override
  String get searchHint => 'Search topics, users, categories…';

  @override
  String get searchEmpty => 'Enter keywords to search';

  @override
  String get searchNoUsers => 'No matching users';

  @override
  String get searchNoCategories => 'No matching categories';

  @override
  String get searchUnavailable => 'Search unavailable';

  @override
  String get searchAll => 'All';

  @override
  String get searchTopics => 'Topics';

  @override
  String get searchUsers => 'Users';

  @override
  String get searchCategories => 'Categories';

  @override
  String get categoryTitle => 'Category';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get settingsTabProfile => 'Profile';

  @override
  String get settingsTabAccount => 'Account';

  @override
  String get settingsTabPrivacy => 'Privacy';

  @override
  String get settingsTabBinding => 'Bindings';

  @override
  String get settingsTabSecurity => 'Security';

  @override
  String get settingsSectionProfile => 'Personal info';

  @override
  String get settingsNickname => 'Nickname';

  @override
  String get settingsNicknameEdit => 'Edit display nickname';

  @override
  String get settingsBio => 'Bio';

  @override
  String get settingsBioEdit => 'Edit bio and signature';

  @override
  String get settingsAvatar => 'Avatar';

  @override
  String get settingsAvatarUpload => 'Upload avatar (converted to webp)';

  @override
  String get settingsAvatarUploading => 'Uploading…';

  @override
  String settingsAvatarUploadFailed(String error) {
    return 'Avatar upload failed: $error';
  }

  @override
  String get settingsEmail => 'Email';

  @override
  String get settingsEmailEdit => 'Change bound email';

  @override
  String get settingsEmailUpdated => 'Email updated, please verify';

  @override
  String settingsEmailFailed(String error) {
    return 'Email update failed: $error';
  }

  @override
  String get settingsNewEmail => 'New email';

  @override
  String get settingsChangePassword => 'Change password';

  @override
  String get settingsChangePasswordSub => 'Change login password';

  @override
  String get settingsCurrentPassword => 'Current password';

  @override
  String get settingsPasswordUpdated => 'Password updated';

  @override
  String settingsPasswordFailed(String error) {
    return 'Password change failed: $error';
  }

  @override
  String get settingsBadge => 'Badge';

  @override
  String get settingsBadgeNone => 'Not wearing';

  @override
  String settingsBadgeCurrent(String name) {
    return 'Current: $name';
  }

  @override
  String get settingsBadgePick => 'Choose badge to wear';

  @override
  String get settingsBadgeNoOptions => 'No wearable badges';

  @override
  String get settingsBadgeUpdated => 'Badge updated';

  @override
  String settingsBadgeFailed(String error) {
    return 'Badge update failed: $error';
  }

  @override
  String get settingsOAuth => 'External account connections';

  @override
  String get settingsOAuthSub => 'GitHub and Google login connections';

  @override
  String get settingsOAuthManage => 'Manage OAuth bindings';

  @override
  String get settingsOAuthBindings => 'OAuth bindings';

  @override
  String get settingsBound => 'Bound';

  @override
  String get settingsUnbound => 'Not bound';

  @override
  String get settingsUnbind => 'Unbind';

  @override
  String get settingsUnboundDone => 'Unbound';

  @override
  String settingsUnbindFailed(String error) {
    return 'Unbind failed: $error';
  }

  @override
  String settingsLoadBindingsFailed(String error) {
    return 'Failed to load bindings: $error';
  }

  @override
  String get settingsPrivacyDirect => 'Messages visible to friends only';

  @override
  String get settingsPrivacyLikes => 'Show my likes publicly';

  @override
  String get settingsSessions => 'Session management';

  @override
  String get settingsSessionsEmpty => 'No sessions';

  @override
  String get settingsRevokeAll => 'Revoke all sessions';

  @override
  String get settingsRevoked => 'Session revoked';

  @override
  String settingsRevokeFailed(String error) {
    return 'Revoke failed: $error';
  }

  @override
  String get settingsRevokeAllDone => 'All sessions revoked';

  @override
  String settingsOpFailed(String error) {
    return 'Operation failed: $error';
  }

  @override
  String get settingsAppearance => 'Appearance';

  @override
  String get settingsDarkMode => 'Dark mode';

  @override
  String get settingsDarkCurrent => 'Current: dark';

  @override
  String get settingsLightCurrent => 'Current: light';

  @override
  String get settingsAbout => 'About';

  @override
  String get settingsAboutVersion => 'v0.1.0 · Tongji campus forum';

  @override
  String get settingsEditProfile => 'Edit profile';

  @override
  String get settingsSignature => 'Signature';

  @override
  String get settingsSaveInfo => 'Save';

  @override
  String get settingsInfoSaved => 'Profile updated';

  @override
  String settingsInfoFailed(String error) {
    return 'Profile update failed: $error';
  }

  @override
  String get settingsUserDataLoading => 'Account data loading…';

  @override
  String get settingsFillComplete => 'Please fill in all fields';

  @override
  String get settingsSecondPhase => 'Coming in phase 2';

  @override
  String get settingsTotpTitle => 'Two-factor auth (TOTP)';

  @override
  String get settingsTotpEnable => 'Enable';

  @override
  String get settingsTotpDisable => 'Disable';

  @override
  String get settingsTotpPasswordPrompt => 'Enter password to manage TOTP';

  @override
  String get settingsTotpSetupSecret =>
      'Scan or enter the secret in your authenticator app';

  @override
  String get settingsTotpCode => 'Enter the 6-digit code';

  @override
  String get settingsTotpRecoveryCodes => 'Recovery codes (save them safely):';

  @override
  String get settingsTotpEnabled => 'TOTP enabled';

  @override
  String get settingsTotpDisabled => 'TOTP disabled';

  @override
  String settingsTotpFailed(String error) {
    return 'TOTP operation failed: $error';
  }

  @override
  String get settingsTotpDisableTitle => 'Disable TOTP';

  @override
  String get settingsTotpEnableTitle => 'Enable TOTP';

  @override
  String get settingsTotpPassword => 'Password';

  @override
  String get settingsTotpNext => 'Next';

  @override
  String get settingsTotpScanSecret => 'Scan or enter the secret';

  @override
  String get settingsTotpDone => 'Done';

  @override
  String get settingsTotpUnavailable => 'TOTP unavailable';

  @override
  String get draftsTitle => 'Drafts';

  @override
  String get draftsEmpty => 'No drafts';

  @override
  String get draftsNew => 'New draft';

  @override
  String get draftsBlocked => 'Blocked';

  @override
  String draftsMetaCreated(Object date) {
    return 'Created $date';
  }

  @override
  String draftsMetaViews(Object count) {
    return '$count views';
  }

  @override
  String draftsMetaReplies(Object count) {
    return '$count replies';
  }

  @override
  String get messagesNew => 'New message';

  @override
  String get messagesSearchUsers => 'Search users';

  @override
  String get messagesNoContactableUsers => 'No contactable users';

  @override
  String get settingsLogout => 'Log out';

  @override
  String get settingsLogoutConfirm => 'Log out of yourtj?';

  @override
  String get commonParseFailed => 'Failed to parse page data';

  @override
  String get topicEmpty => 'No topics yet';

  @override
  String settingsAvatarUploaded(String url) {
    return 'Avatar uploaded: $url';
  }

  @override
  String get settingsImageDecodeFailed => 'Failed to decode image';

  @override
  String dateMonthDayTime(int month, int day, String time) {
    return '$month/$day $time';
  }

  @override
  String dateYearMonthDayTime(int year, int month, int day, String time) {
    return '$year/$month/$day $time';
  }

  @override
  String topicFloorSelected(Object floor) {
    return 'Jumped to floor $floor';
  }

  @override
  String get sortLatest => 'Latest';

  @override
  String get sortHot => 'Hot';

  @override
  String get sortPopular => 'Popular';

  @override
  String get topicFeedModeList => 'List';

  @override
  String get topicFeedModeCard => 'Cards';

  @override
  String get topicNewTopic => 'New topic';
}
