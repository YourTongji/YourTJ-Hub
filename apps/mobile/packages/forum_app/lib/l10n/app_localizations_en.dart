// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get myCourseReviewsTitle => 'My course reviews';

  @override
  String get myCourseReviewsEmpty =>
      'No reviews yet. Explore the course catalog to get started.';

  @override
  String get myCourseReviewHidden =>
      'This review is hidden. You can delete it here.';

  @override
  String get myCourseReviewUnavailable =>
      'This course is unavailable. You can still manage your review.';

  @override
  String get myCourseReviewOpen => 'View details';

  @override
  String get courseReviewNetworkError =>
      'Could not connect. Check your connection and retry; your review has been kept.';

  @override
  String get courseReviewUnknownError =>
      'Your review could not be saved. The server did not provide a specific reason; please retry later.';

  @override
  String get commonHideKeyboard => 'Hide keyboard';

  @override
  String get appTitle => 'YourTJ';

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
  String get commonBackToTop => 'Back to top';

  @override
  String get commonUseLightTheme => 'Use light theme';

  @override
  String get commonUseDarkTheme => 'Use dark theme';

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
  String get authCacheClearFailed =>
      'Failed to clear the previous account\'s offline data. Please retry.';

  @override
  String get authSessionSaveFailed =>
      'Failed to save the new session securely. Please retry.';

  @override
  String get loginWelcome => 'Welcome to YourTJ';

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
  String get composePreview => 'Preview';

  @override
  String get composeEdit => 'Edit';

  @override
  String get publishBodyField => 'Body';

  @override
  String get publishCategoryRequired => 'Select at least one category';

  @override
  String get publishPreviewEmpty =>
      'Start writing to see the formatted preview here';

  @override
  String publishLoadFailed(String error) {
    return 'Failed to load editor data: $error';
  }

  @override
  String get publishToolBold => 'Bold';

  @override
  String get publishToolItalic => 'Italic';

  @override
  String get publishToolStrike => 'Strikethrough';

  @override
  String get publishToolQuote => 'Quote';

  @override
  String get publishToolCode => 'Inline code';

  @override
  String get publishToolBulletList => 'Bulleted list';

  @override
  String get publishToolOrderedList => 'Numbered list';

  @override
  String get publishToolImage => 'Add image';

  @override
  String get publishRemoveImage => 'Remove image';

  @override
  String topicReplyTarget(String name) {
    return 'Replying to $name';
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
  String get topicReplyTargetUnavailable => 'The original reply is unavailable';

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
  String get topicWatch => 'Watch topic replies';

  @override
  String get topicUnwatch => 'Stop watching topic replies';

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
  String get messagesEmpty => 'No conversations yet';

  @override
  String get messagesEmptyDescription =>
      'Start a conversation with someone from the community.';

  @override
  String get messagesSearchConversations => 'Search conversations';

  @override
  String get messagesConversation => 'Private conversation';

  @override
  String get messagesStartChat => 'Start a conversation';

  @override
  String messagesFirstMessageTo(String user) {
    return 'Send the first message to $user.';
  }

  @override
  String get messagesNoMessagesYet => 'No messages yet';

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
  String get settingsEmailOAuthReauthRequired =>
      'The password could not be verified for this OAuth-linked account. Re-authenticate with your provider or contact an administrator.';

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
  String get commonLoadFailed => 'Failed to load';

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
  String get commentSortAsc => 'Oldest first';

  @override
  String get commentSortDesc => 'Newest first';

  @override
  String get commentSortOnlyOp => 'OP only';

  @override
  String get topicOpRepliesPending =>
      'No author replies in the loaded window. Load more to continue.';

  @override
  String get topicOpRepliesEmpty => 'No replies from the author yet';

  @override
  String get topicLaterReplies => 'Load newer replies';

  @override
  String get topicFeedModeList => 'List';

  @override
  String get topicFeedModeCard => 'Cards';

  @override
  String get topicNewTopic => 'New topic';

  @override
  String get scheduleTitle => 'Scheduler';

  @override
  String get scheduleTabTimetable => 'Plan preview';

  @override
  String get scheduleTabPick => 'Pick courses';

  @override
  String get scheduleTerm => 'Term';

  @override
  String get scheduleGrade => 'Grade';

  @override
  String get scheduleMajor => 'Major';

  @override
  String get scheduleSyncLatest => 'Sync latest';

  @override
  String get scheduleDataOutdated => 'Course data updated, tap to sync';

  @override
  String scheduleSyncedTo(String date) {
    return 'Synced to $date';
  }

  @override
  String get scheduleSyncConflictTitle => 'Plan sync conflict';

  @override
  String get scheduleSyncConflictBody =>
      'Your local schedule plans differ from the cloud copy. Which one should be kept?';

  @override
  String get scheduleSyncUseCloud => 'Use cloud';

  @override
  String get scheduleSyncKeepLocal => 'Keep local';

  @override
  String get scheduleWeekAll => 'All weeks';

  @override
  String scheduleWeekN(int week) {
    return 'Week $week';
  }

  @override
  String get scheduleCurrentWeek => 'Current week';

  @override
  String get schedulePlanNew => 'New plan';

  @override
  String get schedulePlanRename => 'Rename plan';

  @override
  String get schedulePlanDelete => 'Delete plan';

  @override
  String get schedulePlanClear => 'Clear courses';

  @override
  String schedulePlanN(int n) {
    return 'Plan $n';
  }

  @override
  String scheduleStatsCourses(int count) {
    return '$count courses';
  }

  @override
  String get scheduleStatsCredits => 'Credits';

  @override
  String get scheduleStatsHours => 'Hours';

  @override
  String get scheduleStatsConflicts => 'Conflicts';

  @override
  String get scheduleConflictBadge => 'Conflict';

  @override
  String scheduleConflictWith(String course) {
    return 'Conflicts with \"$course\"';
  }

  @override
  String scheduleConflictsWith(String course, int count) {
    return 'Conflicts with \"$course\" and $count other courses';
  }

  @override
  String get scheduleConflictCanAdd => 'Can still add';

  @override
  String get scheduleAddCustomEvent => 'Add placeholder';

  @override
  String get scheduleCustomEventLabel => 'Busy';

  @override
  String get scheduleExportPng => 'Export image';

  @override
  String get scheduleExportCsv => 'Export CSV';

  @override
  String get scheduleDegraded => 'Degraded search, results may be incomplete';

  @override
  String get scheduleNoReviewData => 'No review data yet';

  @override
  String get schedulePickClass => 'Pick class';

  @override
  String get scheduleCompulsory => 'Compulsory';

  @override
  String get scheduleOptional => 'Elective';

  @override
  String get scheduleSearchHint => 'Search name/code/teacher';

  @override
  String get scheduleStaged => 'Staged';

  @override
  String get scheduleSelected => 'Selected';

  @override
  String get scheduleRemoveCourse => 'Remove';

  @override
  String get scheduleParityOdd => 'odd weeks';

  @override
  String get scheduleParityEven => 'even weeks';

  @override
  String scheduleWeeksN(String range) {
    return '$range weeks';
  }

  @override
  String get scheduleDayMon => 'Mon';

  @override
  String get scheduleDayTue => 'Tue';

  @override
  String get scheduleDayWed => 'Wed';

  @override
  String get scheduleDayThu => 'Thu';

  @override
  String get scheduleDayFri => 'Fri';

  @override
  String get scheduleDaySat => 'Sat';

  @override
  String get scheduleDaySun => 'Sun';

  @override
  String get scheduleMorning => 'Morning';

  @override
  String get scheduleAfternoon => 'Afternoon';

  @override
  String get scheduleEvening => 'Evening';

  @override
  String get coursesTitle => 'Courses';

  @override
  String get coursesSearchHint => 'Search courses';

  @override
  String get coursesFilterDepartment => 'Department';

  @override
  String get coursesFilterTerm => 'Term';

  @override
  String get coursesFilterCampus => 'Campus';

  @override
  String get coursesFilterInstructor => 'Instructor';

  @override
  String get coursesOnlyWithReviews => 'With reviews only';

  @override
  String coursesRatingCount(int count) {
    return '$count reviews';
  }

  @override
  String get coursesNoRating => 'No rating';

  @override
  String get courseDetailReviews => 'Reviews';

  @override
  String get courseDetailOfferings => 'Classes';

  @override
  String get courseDetailLineage => 'History';

  @override
  String get courseDetailRelated => 'Related courses';

  @override
  String get courseDetailAiSummary => 'AI summary';

  @override
  String get courseDetailAiSummaryEmpty => 'No AI summary';

  @override
  String get courseBookmark => 'Bookmark';

  @override
  String get courseBookmarked => 'Bookmarked';

  @override
  String get courseWriteReview => 'Write review';

  @override
  String get reviewAnonymous => 'Anonymous';

  @override
  String get reviewSubmit => 'Submit';

  @override
  String get reviewHelpful => 'Helpful';

  @override
  String get reviewsEmpty => 'No reviews yet';

  @override
  String get wikiTitle => 'Wiki';

  @override
  String get wikiRecent => 'Recently updated';

  @override
  String get wikiEditOnGithub => 'Edit on GitHub';

  @override
  String get wikiToc => 'Contents';

  @override
  String get wikiNamespaces => 'Namespaces';

  @override
  String wikiViewCount(int count) {
    return '$count views';
  }

  @override
  String get settingsPush => 'Push notifications';

  @override
  String get settingsPushDenied =>
      'Notification permission denied, tap to open Settings';

  @override
  String get settingsFollowSiteTheme => 'Follow site theme';

  @override
  String get settingsFollowSiteThemeDesc => 'Use server-issued site colors';

  @override
  String get entryCourses => 'Courses';

  @override
  String get entrySchedule => 'Schedule';

  @override
  String get entryWiki => 'Wiki';

  @override
  String get navCampus => 'Campus';

  @override
  String get campusTitle => 'Your campus, connected';

  @override
  String get campusSubtitle =>
      'Find courses, plan your week and share campus knowledge.';

  @override
  String get campusCoursesHint => 'Explore courses and student reviews';

  @override
  String get campusScheduleHint => 'Plan your week, check conflicts and export';

  @override
  String get campusWikiHint => 'A campus guide built by the community';

  @override
  String get publishMoment => 'Moment';

  @override
  String get publishQuestion => 'Question';

  @override
  String get publishArticle => 'Article';

  @override
  String get publishNext => 'Next';

  @override
  String get publishGallery => 'Choose photos, then tell your story';

  @override
  String get publishGalleryHint => 'Up to 9 photos. Hold and drag to reorder.';

  @override
  String get publishFormatting => 'Formatting';

  @override
  String get publishClassification => 'Choose categories';

  @override
  String get publishLeaveTitle => 'Keep your work?';

  @override
  String get publishLeaveBody =>
      'You have unsaved changes. Continue editing or discard them.';

  @override
  String get publishDiscard => 'Discard changes';

  @override
  String get publishContinue => 'Keep editing';

  @override
  String get publishImageOnlyTitle => 'A moment to share';

  @override
  String get publishTypeLocked => 'The original type is retained when editing';

  @override
  String get profileMore => 'More options';

  @override
  String get profileContent => 'Manage content';

  @override
  String get profileTrash => 'Recycle bin';

  @override
  String get profileSecurity => 'Account, security & privacy';

  @override
  String get profileAdmin => 'Admin workspace';

  @override
  String get fabDiscussion => 'Jump to discussion';

  @override
  String get fabRefresh => 'Refresh and return to top';

  @override
  String get contentRestore => 'Restore';

  @override
  String get contentDelete => 'Delete';

  @override
  String get contentPurge => 'Delete permanently';

  @override
  String get contentDeleteConfirm =>
      'Content moves to the recycle bin and can be restored within 30 days when eligible.';

  @override
  String get contentPurgeConfirm =>
      'This cannot be undone. Text and images will be permanently erased.';

  @override
  String get contentPassword => 'Enter your current password to confirm';

  @override
  String get contentSelected => 'Selected';

  @override
  String get contentSelectAll => 'Select loaded items';

  @override
  String get contentEmpty => 'No content here yet';

  @override
  String get contentTrashHint =>
      'Eligible content can be restored for 30 days. Moderated removals cannot be restored here.';

  @override
  String get commonConfirm => 'Confirm';

  @override
  String get adminUnavailable =>
      'The admin console is unavailable. Check your permissions, session and connection, then retry.';

  @override
  String get adminDownloadFailed =>
      'Could not download the export. Please try again.';

  @override
  String get publishGalleryTooMany =>
      'Content supports up to 9 images. Remove some images or continue with the article editor.';

  @override
  String get profileActivity => 'Activity';

  @override
  String get profileBookmarks => 'Bookmarks';

  @override
  String get profileModeration => 'Moderation workspace';

  @override
  String get settingsCloseAccount => 'Close account';

  @override
  String get settingsCloseAccountWarning =>
      'Account closure is irreversible and signs out all devices. Keep historical content under an anonymized identity, or request deletion of your content. Retention rules may preserve some content. Enter your current password to confirm.';

  @override
  String get settingsCloseKeepContent => 'Keep anonymized content';

  @override
  String get settingsCloseDeleteContent => 'Request content deletion';

  @override
  String get publishUndo => 'Undo';

  @override
  String get publishRedo => 'Redo';

  @override
  String get publishHeading => 'Heading · hold to pick level';

  @override
  String get publishHeadingLevel1 => 'Heading 1';

  @override
  String get publishHeadingLevel2 => 'Heading 2';

  @override
  String get publishHeadingLevel3 => 'Heading 3';

  @override
  String get publishToolLink => 'Insert link';

  @override
  String get publishLinkInvalid => 'Enter a valid http, https or email link.';

  @override
  String get publishPhotoLibrary => 'Choose from library';

  @override
  String get publishCamera => 'Take a photo';

  @override
  String get settingsWebsiteName => 'Website name';

  @override
  String get settingsWebsite => 'Website';

  @override
  String get settingsSocialLinks => 'Social links';

  @override
  String get settingsProfileLanguage => 'Profile language';

  @override
  String get settingsInvalidLink => 'Enter a valid http or https link';

  @override
  String get siteInfoTitle => 'About the community';

  @override
  String get siteInfoLinks => 'Community links';

  @override
  String get siteInfoSponsors => 'Supporters';

  @override
  String get siteInfoTerms => 'Terms of service';

  @override
  String get siteInfoPrivacy => 'Privacy policy';

  @override
  String get siteInfoEmpty => 'No public content yet';

  @override
  String get settingsUsernameHint =>
      'This is your sign-in name. Site naming rules apply.';

  @override
  String get settingsUsernameUpdated => 'Username updated';

  @override
  String get settingsPresetAvatar => 'Choose a preset avatar';

  @override
  String get coursesManagement => 'Manage courses';

  @override
  String get coursesReviewModeration => 'Review moderation';

  @override
  String get settingsCover => 'Profile cover';

  @override
  String get settingsCoverDescription => 'Choose an image and adjust the crop';

  @override
  String get settingsCoverRemove => 'Remove cover';

  @override
  String get settingsCoverRemoveConfirm =>
      'Your profile will use the default background.';

  @override
  String get settingsCoverMinSize =>
      'Choose an image at least 1200 × 240 pixels';

  @override
  String get settingsCoverSafeArea =>
      'The bright center is the main mobile view. The full width is preserved for wider screens.';

  @override
  String get settingsCropHint => 'Drag to reposition. Pinch to zoom.';

  @override
  String get settingsCropPreview => 'Image crop preview';

  @override
  String get settingsCropZoom => 'Zoom';

  @override
  String get settingsCropReset => 'Reset position';

  @override
  String get settingsImageSaved => 'Image updated';

  @override
  String settingsImageTooLarge(int maxMb) {
    return 'Image must be no larger than $maxMb MB';
  }

  @override
  String get settingsOAuthUnavailable => 'Not enabled on this site';

  @override
  String get settingsOAuthOpenBrowser => 'Manage connections in browser';

  @override
  String settingsOAuthBrowserHint(String username) {
    return 'Sign in as @$username in the browser to connect an account. Connections refresh when you return to the app.';
  }

  @override
  String get commonRefresh => 'Refresh';

  @override
  String get topicEarliest => 'Earliest';

  @override
  String get topicLatest => 'Latest';

  @override
  String get topicEarlierReplies => 'Load earlier replies';

  @override
  String get topicHistory => 'Revision history';

  @override
  String get topicHistoryUnavailable => 'This version is unavailable';

  @override
  String get topicHistoryEmpty => 'No revisions yet';

  @override
  String get topicDeleteConfirm =>
      'Delete this content? Recoverable items can be found in the recycle bin.';

  @override
  String get topicModerateBan => 'Hide content';

  @override
  String get topicModerateUnban => 'Restore visibility';

  @override
  String get topicModerateConfirm => 'Change the visibility of this content?';

  @override
  String get topicShare => 'Share';

  @override
  String get topicEditReply => 'Edit reply';

  @override
  String get topicRemoved => 'This content has been deleted or removed';

  @override
  String get topicBookmark => 'Bookmark';

  @override
  String get topicBookmarked => 'Remove bookmark';

  @override
  String get topicLike => 'Like';

  @override
  String get authEmailPrefix => 'Email username';

  @override
  String get authEmailDomain => 'Email domain';

  @override
  String get authAgreePolicies =>
      'I have read and agree to the published policies';

  @override
  String get authPasswordMismatch => 'The passwords do not match';

  @override
  String get schedulerWebTitle => 'Explore the full scheduler on the Web';

  @override
  String get schedulerWebAction => 'Open f.yourtj.de';

  @override
  String get schedulerPlanDisclaimer =>
      'This is a course plan. Final enrolment is determined by the university.';

  @override
  String get campusExploreCourses =>
      'Find your next course through student reviews';

  @override
  String get campusPlanTitle => 'Turn your course shortlist into a plan';

  @override
  String get campusPlanDescription =>
      'Compare offerings and check conflicts before choosing.';

  @override
  String get loginGoogle => 'Continue with Google';

  @override
  String get loginGithub => 'Continue with GitHub';

  @override
  String get wikiSearchUnavailable =>
      'Search is unavailable. You can still browse the directory.';

  @override
  String get wikiSearchHint => 'Search campus knowledge';

  @override
  String get wikiExploreTitle => 'Your campus companion';

  @override
  String notificationComment(String actor) {
    return '$actor commented on your topic';
  }

  @override
  String notificationMention(String actor) {
    return '$actor mentioned you';
  }

  @override
  String get mentionListboxLabel => 'Mention users';

  @override
  String get mentionLoading => 'Searching users…';

  @override
  String get mentionNoResults => 'No matching users';

  @override
  String get mentionSearchFailed => 'User search failed, keep typing';

  @override
  String get mentionKeepTyping => 'Keep typing to search users';

  @override
  String get mentionTagReplyTarget => 'Replying to';

  @override
  String get mentionTagTopicAuthor => 'Topic author';

  @override
  String get mentionTagParticipant => 'Participant';

  @override
  String notificationPostReply(String actor) {
    return '$actor replied to you';
  }

  @override
  String notificationTopicPost(String actor) {
    return '$actor posted in a topic you watch';
  }

  @override
  String notificationFollow(String actor) {
    return '$actor followed you';
  }

  @override
  String notificationLike(String actor) {
    return '$actor liked your reply';
  }

  @override
  String notificationWikiUpdated(String actor) {
    return '$actor updated a wiki page you watch';
  }

  @override
  String notificationBadge(String badge) {
    return 'You earned the “$badge” badge';
  }

  @override
  String get notificationNew => 'New notification';

  @override
  String get notificationSomeone => 'Someone';

  @override
  String get profileRoleAdmin => 'Admin';

  @override
  String get profileActionSignup => 'Joined the community';

  @override
  String get profileActionPost => 'Published a topic';

  @override
  String get profileActionLike => 'Liked';

  @override
  String get profileActionFollow => 'Followed';

  @override
  String get profileActionComment => 'Replied';

  @override
  String get replyQuoteExpand => 'Show full quote';

  @override
  String get replyQuoteCollapse => 'Collapse quote';

  @override
  String get notificationBadgeUnnamed => 'You earned a new badge';

  @override
  String get settingsAppLanguage => 'App language';

  @override
  String get settingsLanguageSystem => 'Follow system';

  @override
  String scheduleGradeYear(String year) {
    return 'Class of $year';
  }

  @override
  String get schedulePeriods => 'Periods';

  @override
  String get scheduleWeeksLabel => 'Weeks';

  @override
  String schedulePeriodRange(String range) {
    return 'Periods $range';
  }

  @override
  String get courseCopyCreditUnit => 'Credits';

  @override
  String get courseCopyNoTeacher => 'No teacher';

  @override
  String get courseCopyCatalogEmptyTitle => 'No courses yet';

  @override
  String get courseCopyCatalogEmptyDescription =>
      'The course catalog has not been imported yet.';

  @override
  String get courseCopyNoFilterResults => 'No courses match these filters.';

  @override
  String get courseCopyNoFilterResultsDescription =>
      'Try adjusting or clearing filters to see more courses.';

  @override
  String get courseCopyClearSearch => 'Clear search';

  @override
  String get courseCopyDone => 'Done';

  @override
  String get courseCopyNoOptions => 'No options available';

  @override
  String courseCopySelectedCount(int count) {
    return '$count selected';
  }

  @override
  String get courseCopyInstructorInputHint =>
      'Enter an instructor name and press enter';

  @override
  String get courseCopyInstructorAdd => 'Add';

  @override
  String get courseCopyInstructorEmptyHint =>
      'Add instructors with the field above';

  @override
  String get courseCopyAliasesLabel => 'Aliases: ';

  @override
  String get courseCopyLegacyNamesLabel => 'Former name: ';

  @override
  String get courseCopyReviewScopeTeam => 'Teaching Team';

  @override
  String get courseCopyReviewScopeCourse => 'Course-level Reviews';

  @override
  String get courseCopyTeamInstructorsPrefix => 'Teaching team · ';

  @override
  String courseCopyTeamInstructorsSuffix(int count) {
    return ' ($count teachers)';
  }

  @override
  String get courseCopyRatingTitle => 'Course rating';

  @override
  String get courseCopyRatingOutOf => '/ 5.0';

  @override
  String get courseCopyNoRatingQuiet => 'No rating yet';

  @override
  String get courseCopyOfferingsEmpty => 'No offerings yet.';

  @override
  String get courseCopyOfferingFocusLabel =>
      'Showing reviews for this class only';

  @override
  String get courseCopyOfferingFocusClear => 'Show all reviews';

  @override
  String get courseCopySummaryGenerated => 'Generated';

  @override
  String get courseCopySummaryKeywords => 'Keywords';

  @override
  String get courseCopySummaryPros => 'Pros';

  @override
  String get courseCopySummaryCons => 'Cons';

  @override
  String get courseCopySummaryRepresentativeReviews => 'Representative reviews';

  @override
  String get courseCopySummarySentimentPositive => 'Positive';

  @override
  String get courseCopySummarySentimentNeutral => 'Neutral';

  @override
  String get courseCopySummarySentimentNegative => 'Negative';

  @override
  String get courseCopySummaryRefresh => 'Refresh';

  @override
  String get courseCopySummaryExpand => 'Expand';

  @override
  String get courseCopySummaryCollapse => 'Collapse';

  @override
  String get courseCopySummaryDisclaimer =>
      'Generated by AI from student reviews. For reference only; not a course recommendation.';

  @override
  String get courseCopySummaryInsufficient =>
      'Not enough reviews to generate an AI summary yet.';

  @override
  String get courseCopySummaryLoadFailed =>
      'Failed to generate the AI summary. Please try again later.';

  @override
  String get courseCopyWriteReviewTitle => 'Write a review';

  @override
  String get courseCopyEditReviewTitle => 'Edit review';

  @override
  String get courseCopySelectOffering => 'Select offering';

  @override
  String get courseCopyRatingLabel => 'Rating';

  @override
  String get courseCopyContentLabel => 'Review';

  @override
  String get courseCopyContentPlaceholder =>
      'Share your experience with the course, teaching quality, or grading…';

  @override
  String get courseCopyRatingRequired => 'Please choose a 1–5 star rating.';

  @override
  String get courseCopyContentRequired => 'Review content cannot be empty.';

  @override
  String get courseCopyAnonymousLabel =>
      'Post anonymously (identity hidden from the public)';

  @override
  String get courseCopySubmitSuccess => 'Submitted';

  @override
  String get courseCopyUpdateSuccess => 'Updated';

  @override
  String get courseCopyDelete => 'Delete';

  @override
  String get courseCopyDeleteReviewTitle => 'Delete review';

  @override
  String get courseCopyConfirmDeleteReview =>
      'Delete this review? This cannot be undone.';

  @override
  String get courseCopyReviewDeleted => 'Review deleted';

  @override
  String get courseCopyOperationFailed =>
      'Operation failed. Please try again later.';

  @override
  String get courseCopyReviewsLoadFailed =>
      'Failed to load reviews. Please try again later.';

  @override
  String get courseCopyAuthorAnonymousLabel => 'Anonymous';

  @override
  String get courseCopyAuthorLegacyLabel => 'Legacy anonymous review';

  @override
  String get courseCopyRelatedTeacherCoursesTitle =>
      'Other courses by the same teachers';

  @override
  String get courseCopyRelatedOtherTeachersTitle =>
      'Other teachers of this course';

  @override
  String get courseCopyRelatedEmpty => 'No related content';

  @override
  String get courseCopyRelationEquivalent => 'Equivalent';

  @override
  String get courseCopyRelationRenamed => 'Renamed';

  @override
  String get courseCopyRelationSplit => 'Split';

  @override
  String get courseCopyRelationMerged => 'Merged';

  @override
  String get courseCopyRelationRelated => 'Related';

  @override
  String get courseCopySummaryConsensusStrongRecommend =>
      'Strongly recommended';

  @override
  String get courseCopySummaryConsensusRecommend => 'Recommended';

  @override
  String get courseCopySummaryConsensusNeutral => 'Mixed';

  @override
  String get courseCopySummaryConsensusCautious => 'Caution advised';

  @override
  String get courseCopySummaryConsensusNotRecommend => 'Not recommended';

  @override
  String get courseCopySummaryConsensusTextStrongRecommend =>
      'Most students strongly recommend this course.';

  @override
  String get courseCopySummaryConsensusTextRecommend =>
      'Most students recommend this course.';

  @override
  String get courseCopySummaryConsensusTextNeutral =>
      'Students have mixed opinions about this course.';

  @override
  String get courseCopySummaryConsensusTextCautious =>
      'Most students advise caution before choosing this course.';

  @override
  String get courseCopySummaryConsensusTextNotRecommend =>
      'Most students do not recommend this course.';

  @override
  String get topicJoinDiscussion => 'Join discussion';

  @override
  String get updateCheck => 'Check for updates';

  @override
  String get updateAvailable => 'Update available';

  @override
  String get updateLatest => 'You’re up to date';

  @override
  String get updateFailed =>
      'Unable to check or download updates. Please try again later.';

  @override
  String get updateDownload => 'Download update';

  @override
  String get updateSkip => 'Skip this version';

  @override
  String get updateLater => 'Later';

  @override
  String get updatePreparing => 'Choosing the fastest download source…';

  @override
  String get updateDownloading => 'Downloading…';

  @override
  String get updateReady => 'Update verified and ready to install.';

  @override
  String get updateInstall => 'Install update';

  @override
  String get updatePermission =>
      'Allow YourTJ to install apps, then return and tap Install again.';

  @override
  String get updateRetry => 'Retry';

  @override
  String get settingsPushConsent =>
      'Enable system notifications via Apple (iOS) or JPush and device vendors (Android). These services process push identifiers and notification content.';

  @override
  String get settingsPushUnsupported => 'Push is not configured in this build';

  @override
  String get settingsPushServerDisabled =>
      'The server has not enabled this delivery channel. Tap to retry.';

  @override
  String get settingsPushFailed =>
      'Push registration failed. Check your connection and tap to retry.';

  @override
  String get settingsPushPrivacy => 'JPush privacy policy (Android)';
}
