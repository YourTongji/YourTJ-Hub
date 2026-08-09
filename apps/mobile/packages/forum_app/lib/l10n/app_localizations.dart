import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('zh'),
  ];

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'yourtj'**
  String get appTitle;

  /// No description provided for @navHome.
  ///
  /// In en, this message translates to:
  /// **'Home'**
  String get navHome;

  /// No description provided for @navSearch.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get navSearch;

  /// No description provided for @navPublish.
  ///
  /// In en, this message translates to:
  /// **'Publish'**
  String get navPublish;

  /// No description provided for @navMessages.
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get navMessages;

  /// No description provided for @navProfile.
  ///
  /// In en, this message translates to:
  /// **'Me'**
  String get navProfile;

  /// No description provided for @commonCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get commonCancel;

  /// No description provided for @commonSave.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get commonSave;

  /// No description provided for @commonLoading.
  ///
  /// In en, this message translates to:
  /// **'Loading…'**
  String get commonLoading;

  /// No description provided for @commonLoadMore.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get commonLoadMore;

  /// No description provided for @commonRetry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get commonRetry;

  /// No description provided for @commonClose.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get commonClose;

  /// No description provided for @commonSend.
  ///
  /// In en, this message translates to:
  /// **'Send'**
  String get commonSend;

  /// No description provided for @commonSearch.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get commonSearch;

  /// No description provided for @commonEdit.
  ///
  /// In en, this message translates to:
  /// **'Edit'**
  String get commonEdit;

  /// No description provided for @commonCurrent.
  ///
  /// In en, this message translates to:
  /// **'Current'**
  String get commonCurrent;

  /// No description provided for @commonEmpty.
  ///
  /// In en, this message translates to:
  /// **'Nothing here'**
  String get commonEmpty;

  /// No description provided for @commonBack.
  ///
  /// In en, this message translates to:
  /// **'Back'**
  String get commonBack;

  /// No description provided for @commonBackToTop.
  ///
  /// In en, this message translates to:
  /// **'Back to top'**
  String get commonBackToTop;

  /// No description provided for @commonUseLightTheme.
  ///
  /// In en, this message translates to:
  /// **'Use light theme'**
  String get commonUseLightTheme;

  /// No description provided for @commonUseDarkTheme.
  ///
  /// In en, this message translates to:
  /// **'Use dark theme'**
  String get commonUseDarkTheme;

  /// No description provided for @timeAgoJustNow.
  ///
  /// In en, this message translates to:
  /// **'just now'**
  String get timeAgoJustNow;

  /// No description provided for @timeAgoMinutes.
  ///
  /// In en, this message translates to:
  /// **'{count} min ago'**
  String timeAgoMinutes(int count);

  /// No description provided for @timeAgoHours.
  ///
  /// In en, this message translates to:
  /// **'{count} h ago'**
  String timeAgoHours(int count);

  /// No description provided for @timeAgoDays.
  ///
  /// In en, this message translates to:
  /// **'{count} d ago'**
  String timeAgoDays(int count);

  /// No description provided for @timeAgoWeeks.
  ///
  /// In en, this message translates to:
  /// **'{count} w ago'**
  String timeAgoWeeks(int count);

  /// No description provided for @timeAgoMonths.
  ///
  /// In en, this message translates to:
  /// **'{count} mo ago'**
  String timeAgoMonths(int count);

  /// No description provided for @timeAgoYears.
  ///
  /// In en, this message translates to:
  /// **'{count} y ago'**
  String timeAgoYears(int count);

  /// No description provided for @authLoginTitle.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get authLoginTitle;

  /// No description provided for @authRegisterTitle.
  ///
  /// In en, this message translates to:
  /// **'Create account'**
  String get authRegisterTitle;

  /// No description provided for @authForgotTitle.
  ///
  /// In en, this message translates to:
  /// **'Reset password'**
  String get authForgotTitle;

  /// No description provided for @authLoginSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Welcome back. Continue your discussions and writing.'**
  String get authLoginSubtitle;

  /// No description provided for @authRegisterSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Create an account and join the campus conversation.'**
  String get authRegisterSubtitle;

  /// No description provided for @authForgotSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Enter your email and we will send a reset link.'**
  String get authForgotSubtitle;

  /// No description provided for @authUsernameOrEmail.
  ///
  /// In en, this message translates to:
  /// **'Username or email'**
  String get authUsernameOrEmail;

  /// No description provided for @authUsername.
  ///
  /// In en, this message translates to:
  /// **'Username'**
  String get authUsername;

  /// No description provided for @authEmail.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get authEmail;

  /// No description provided for @authPassword.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get authPassword;

  /// No description provided for @authNewPassword.
  ///
  /// In en, this message translates to:
  /// **'New password'**
  String get authNewPassword;

  /// No description provided for @authConfirmPassword.
  ///
  /// In en, this message translates to:
  /// **'Confirm password'**
  String get authConfirmPassword;

  /// No description provided for @authCaptcha.
  ///
  /// In en, this message translates to:
  /// **'Captcha'**
  String get authCaptcha;

  /// No description provided for @authForgotPassword.
  ///
  /// In en, this message translates to:
  /// **'Forgot password?'**
  String get authForgotPassword;

  /// No description provided for @authCreateAccount.
  ///
  /// In en, this message translates to:
  /// **'Create account'**
  String get authCreateAccount;

  /// No description provided for @authSendResetEmail.
  ///
  /// In en, this message translates to:
  /// **'Send reset email'**
  String get authSendResetEmail;

  /// No description provided for @authBackToLogin.
  ///
  /// In en, this message translates to:
  /// **'Back to login'**
  String get authBackToLogin;

  /// No description provided for @authTwoFactorTitle.
  ///
  /// In en, this message translates to:
  /// **'Two-factor authentication'**
  String get authTwoFactorTitle;

  /// No description provided for @authTwoFactorCode.
  ///
  /// In en, this message translates to:
  /// **'TOTP code'**
  String get authTwoFactorCode;

  /// No description provided for @authVerify.
  ///
  /// In en, this message translates to:
  /// **'Verify'**
  String get authVerify;

  /// No description provided for @authGetCode.
  ///
  /// In en, this message translates to:
  /// **'Get code'**
  String get authGetCode;

  /// No description provided for @authCasdoorLogin.
  ///
  /// In en, this message translates to:
  /// **'Sign in with Casdoor'**
  String get authCasdoorLogin;

  /// No description provided for @authRegisterSuccess.
  ///
  /// In en, this message translates to:
  /// **'Registered successfully, please sign in'**
  String get authRegisterSuccess;

  /// No description provided for @authResetEmailSent.
  ///
  /// In en, this message translates to:
  /// **'Reset email sent, please check your inbox'**
  String get authResetEmailSent;

  /// No description provided for @authLoading.
  ///
  /// In en, this message translates to:
  /// **'Processing…'**
  String get authLoading;

  /// No description provided for @loginWelcome.
  ///
  /// In en, this message translates to:
  /// **'Welcome back to yourtj'**
  String get loginWelcome;

  /// No description provided for @loginModeLogin.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get loginModeLogin;

  /// No description provided for @loginModeRegister.
  ///
  /// In en, this message translates to:
  /// **'Sign up'**
  String get loginModeRegister;

  /// No description provided for @loginModeForgot.
  ///
  /// In en, this message translates to:
  /// **'Forgot password'**
  String get loginModeForgot;

  /// No description provided for @publishTitle.
  ///
  /// In en, this message translates to:
  /// **'Publish topic'**
  String get publishTitle;

  /// No description provided for @publishEditTitle.
  ///
  /// In en, this message translates to:
  /// **'Edit topic'**
  String get publishEditTitle;

  /// No description provided for @publishPublish.
  ///
  /// In en, this message translates to:
  /// **'Publish'**
  String get publishPublish;

  /// No description provided for @publishSaveDraft.
  ///
  /// In en, this message translates to:
  /// **'Save draft'**
  String get publishSaveDraft;

  /// No description provided for @publishTitleField.
  ///
  /// In en, this message translates to:
  /// **'Title'**
  String get publishTitleField;

  /// No description provided for @publishTitleHint.
  ///
  /// In en, this message translates to:
  /// **'Enter title (5-100 chars)'**
  String get publishTitleHint;

  /// No description provided for @publishBodyPlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Content…'**
  String get publishBodyPlaceholder;

  /// No description provided for @publishTitleRequired.
  ///
  /// In en, this message translates to:
  /// **'Title is required'**
  String get publishTitleRequired;

  /// No description provided for @publishContentRequired.
  ///
  /// In en, this message translates to:
  /// **'Content is required'**
  String get publishContentRequired;

  /// No description provided for @publishSuccess.
  ///
  /// In en, this message translates to:
  /// **'Published successfully'**
  String get publishSuccess;

  /// No description provided for @publishSavedDraft.
  ///
  /// In en, this message translates to:
  /// **'Saved as draft'**
  String get publishSavedDraft;

  /// No description provided for @publishFailed.
  ///
  /// In en, this message translates to:
  /// **'Publish failed: {error}'**
  String publishFailed(String error);

  /// No description provided for @publishImageFailed.
  ///
  /// In en, this message translates to:
  /// **'Image upload failed: {error}'**
  String publishImageFailed(String error);

  /// No description provided for @composePreview.
  ///
  /// In en, this message translates to:
  /// **'Preview'**
  String get composePreview;

  /// No description provided for @composeEdit.
  ///
  /// In en, this message translates to:
  /// **'Edit'**
  String get composeEdit;

  /// No description provided for @publishBodyField.
  ///
  /// In en, this message translates to:
  /// **'Body'**
  String get publishBodyField;

  /// No description provided for @publishCategoryRequired.
  ///
  /// In en, this message translates to:
  /// **'Select at least one category'**
  String get publishCategoryRequired;

  /// No description provided for @publishPreviewEmpty.
  ///
  /// In en, this message translates to:
  /// **'Start writing to see the formatted preview here'**
  String get publishPreviewEmpty;

  /// No description provided for @publishLoadFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to load editor data: {error}'**
  String publishLoadFailed(String error);

  /// No description provided for @publishToolBold.
  ///
  /// In en, this message translates to:
  /// **'Bold'**
  String get publishToolBold;

  /// No description provided for @publishToolItalic.
  ///
  /// In en, this message translates to:
  /// **'Italic'**
  String get publishToolItalic;

  /// No description provided for @publishToolStrike.
  ///
  /// In en, this message translates to:
  /// **'Strikethrough'**
  String get publishToolStrike;

  /// No description provided for @publishToolQuote.
  ///
  /// In en, this message translates to:
  /// **'Quote'**
  String get publishToolQuote;

  /// No description provided for @publishToolCode.
  ///
  /// In en, this message translates to:
  /// **'Inline code'**
  String get publishToolCode;

  /// No description provided for @publishToolBulletList.
  ///
  /// In en, this message translates to:
  /// **'Bulleted list'**
  String get publishToolBulletList;

  /// No description provided for @publishToolOrderedList.
  ///
  /// In en, this message translates to:
  /// **'Numbered list'**
  String get publishToolOrderedList;

  /// No description provided for @publishToolImage.
  ///
  /// In en, this message translates to:
  /// **'Add image'**
  String get publishToolImage;

  /// No description provided for @publishRemoveImage.
  ///
  /// In en, this message translates to:
  /// **'Remove image'**
  String get publishRemoveImage;

  /// No description provided for @topicReplyTarget.
  ///
  /// In en, this message translates to:
  /// **'Replying to {name}'**
  String topicReplyTarget(String name);

  /// No description provided for @topicTitle.
  ///
  /// In en, this message translates to:
  /// **'Topic'**
  String get topicTitle;

  /// No description provided for @topicReply.
  ///
  /// In en, this message translates to:
  /// **'Reply'**
  String get topicReply;

  /// No description provided for @topicReplySuccess.
  ///
  /// In en, this message translates to:
  /// **'Reply posted'**
  String get topicReplySuccess;

  /// No description provided for @topicReplyFailed.
  ///
  /// In en, this message translates to:
  /// **'Reply failed: {error}'**
  String topicReplyFailed(String error);

  /// No description provided for @topicReplyHint.
  ///
  /// In en, this message translates to:
  /// **'Write a comment…'**
  String get topicReplyHint;

  /// No description provided for @topicReplying.
  ///
  /// In en, this message translates to:
  /// **'Replying… (tap to cancel)'**
  String get topicReplying;

  /// No description provided for @topicReport.
  ///
  /// In en, this message translates to:
  /// **'Report post'**
  String get topicReport;

  /// No description provided for @topicReportHint.
  ///
  /// In en, this message translates to:
  /// **'Describe the reason'**
  String get topicReportHint;

  /// No description provided for @topicReportSubmit.
  ///
  /// In en, this message translates to:
  /// **'Submit'**
  String get topicReportSubmit;

  /// No description provided for @topicReportSubmitted.
  ///
  /// In en, this message translates to:
  /// **'Report submitted'**
  String get topicReportSubmitted;

  /// No description provided for @topicReportFailed.
  ///
  /// In en, this message translates to:
  /// **'Report failed: {error}'**
  String topicReportFailed(String error);

  /// No description provided for @topicReplies.
  ///
  /// In en, this message translates to:
  /// **'{count} replies'**
  String topicReplies(int count);

  /// No description provided for @topicNoTitle.
  ///
  /// In en, this message translates to:
  /// **'Untitled'**
  String get topicNoTitle;

  /// No description provided for @profileTitle.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get profileTitle;

  /// No description provided for @profileFollow.
  ///
  /// In en, this message translates to:
  /// **'Follow'**
  String get profileFollow;

  /// No description provided for @profileFollowing.
  ///
  /// In en, this message translates to:
  /// **'Following'**
  String get profileFollowing;

  /// No description provided for @profileTopics.
  ///
  /// In en, this message translates to:
  /// **'Topics'**
  String get profileTopics;

  /// No description provided for @profileReplies.
  ///
  /// In en, this message translates to:
  /// **'Replies'**
  String get profileReplies;

  /// No description provided for @profileLikes.
  ///
  /// In en, this message translates to:
  /// **'Likes'**
  String get profileLikes;

  /// No description provided for @profileFollowers.
  ///
  /// In en, this message translates to:
  /// **'Followers'**
  String get profileFollowers;

  /// No description provided for @profileFollowingCount.
  ///
  /// In en, this message translates to:
  /// **'Following'**
  String get profileFollowingCount;

  /// No description provided for @profileBadges.
  ///
  /// In en, this message translates to:
  /// **'Badges'**
  String get profileBadges;

  /// No description provided for @profileNoBadges.
  ///
  /// In en, this message translates to:
  /// **'No badges'**
  String get profileNoBadges;

  /// No description provided for @profileEmptyActivity.
  ///
  /// In en, this message translates to:
  /// **'No activity yet'**
  String get profileEmptyActivity;

  /// No description provided for @profileEmptyTopics.
  ///
  /// In en, this message translates to:
  /// **'No topics yet'**
  String get profileEmptyTopics;

  /// No description provided for @profileEmptyLikes.
  ///
  /// In en, this message translates to:
  /// **'No likes yet'**
  String get profileEmptyLikes;

  /// No description provided for @profileEmptyBookmarks.
  ///
  /// In en, this message translates to:
  /// **'No bookmarks yet'**
  String get profileEmptyBookmarks;

  /// No description provided for @profileEmptyFollowing.
  ///
  /// In en, this message translates to:
  /// **'Not following anyone'**
  String get profileEmptyFollowing;

  /// No description provided for @profileEmptyFollowers.
  ///
  /// In en, this message translates to:
  /// **'No followers yet'**
  String get profileEmptyFollowers;

  /// No description provided for @profileNotLoggedIn.
  ///
  /// In en, this message translates to:
  /// **'Not signed in'**
  String get profileNotLoggedIn;

  /// No description provided for @messagesTitle.
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get messagesTitle;

  /// No description provided for @messagesEmpty.
  ///
  /// In en, this message translates to:
  /// **'No conversations yet'**
  String get messagesEmpty;

  /// No description provided for @messagesEmptyDescription.
  ///
  /// In en, this message translates to:
  /// **'Start a conversation with someone from the community.'**
  String get messagesEmptyDescription;

  /// No description provided for @messagesSearchConversations.
  ///
  /// In en, this message translates to:
  /// **'Search conversations'**
  String get messagesSearchConversations;

  /// No description provided for @messagesConversation.
  ///
  /// In en, this message translates to:
  /// **'Private conversation'**
  String get messagesConversation;

  /// No description provided for @messagesStartChat.
  ///
  /// In en, this message translates to:
  /// **'Start a conversation'**
  String get messagesStartChat;

  /// No description provided for @messagesFirstMessageTo.
  ///
  /// In en, this message translates to:
  /// **'Send the first message to {user}.'**
  String messagesFirstMessageTo(String user);

  /// No description provided for @messagesNoMessagesYet.
  ///
  /// In en, this message translates to:
  /// **'No messages yet'**
  String get messagesNoMessagesYet;

  /// No description provided for @messagesEmptyDetail.
  ///
  /// In en, this message translates to:
  /// **'No messages yet, say hi!'**
  String get messagesEmptyDetail;

  /// No description provided for @messagesInputHint.
  ///
  /// In en, this message translates to:
  /// **'Type a message…'**
  String get messagesInputHint;

  /// No description provided for @messagesSendFailed.
  ///
  /// In en, this message translates to:
  /// **'Send failed: {error}'**
  String messagesSendFailed(String error);

  /// No description provided for @notificationsTitle.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get notificationsTitle;

  /// No description provided for @notificationsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No notifications'**
  String get notificationsEmpty;

  /// No description provided for @notificationsMarkAllRead.
  ///
  /// In en, this message translates to:
  /// **'Mark all read'**
  String get notificationsMarkAllRead;

  /// No description provided for @notificationsAll.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get notificationsAll;

  /// No description provided for @notificationsUnread.
  ///
  /// In en, this message translates to:
  /// **'Unread'**
  String get notificationsUnread;

  /// No description provided for @searchTitle.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get searchTitle;

  /// No description provided for @searchHint.
  ///
  /// In en, this message translates to:
  /// **'Search topics, users, categories…'**
  String get searchHint;

  /// No description provided for @searchEmpty.
  ///
  /// In en, this message translates to:
  /// **'Enter keywords to search'**
  String get searchEmpty;

  /// No description provided for @searchNoUsers.
  ///
  /// In en, this message translates to:
  /// **'No matching users'**
  String get searchNoUsers;

  /// No description provided for @searchNoCategories.
  ///
  /// In en, this message translates to:
  /// **'No matching categories'**
  String get searchNoCategories;

  /// No description provided for @searchUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Search unavailable'**
  String get searchUnavailable;

  /// No description provided for @searchAll.
  ///
  /// In en, this message translates to:
  /// **'All'**
  String get searchAll;

  /// No description provided for @searchTopics.
  ///
  /// In en, this message translates to:
  /// **'Topics'**
  String get searchTopics;

  /// No description provided for @searchUsers.
  ///
  /// In en, this message translates to:
  /// **'Users'**
  String get searchUsers;

  /// No description provided for @searchCategories.
  ///
  /// In en, this message translates to:
  /// **'Categories'**
  String get searchCategories;

  /// No description provided for @categoryTitle.
  ///
  /// In en, this message translates to:
  /// **'Category'**
  String get categoryTitle;

  /// No description provided for @settingsTitle.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settingsTitle;

  /// No description provided for @settingsTabProfile.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get settingsTabProfile;

  /// No description provided for @settingsTabAccount.
  ///
  /// In en, this message translates to:
  /// **'Account'**
  String get settingsTabAccount;

  /// No description provided for @settingsTabPrivacy.
  ///
  /// In en, this message translates to:
  /// **'Privacy'**
  String get settingsTabPrivacy;

  /// No description provided for @settingsTabBinding.
  ///
  /// In en, this message translates to:
  /// **'Bindings'**
  String get settingsTabBinding;

  /// No description provided for @settingsTabSecurity.
  ///
  /// In en, this message translates to:
  /// **'Security'**
  String get settingsTabSecurity;

  /// No description provided for @settingsSectionProfile.
  ///
  /// In en, this message translates to:
  /// **'Personal info'**
  String get settingsSectionProfile;

  /// No description provided for @settingsNickname.
  ///
  /// In en, this message translates to:
  /// **'Nickname'**
  String get settingsNickname;

  /// No description provided for @settingsNicknameEdit.
  ///
  /// In en, this message translates to:
  /// **'Edit display nickname'**
  String get settingsNicknameEdit;

  /// No description provided for @settingsBio.
  ///
  /// In en, this message translates to:
  /// **'Bio'**
  String get settingsBio;

  /// No description provided for @settingsBioEdit.
  ///
  /// In en, this message translates to:
  /// **'Edit bio and signature'**
  String get settingsBioEdit;

  /// No description provided for @settingsAvatar.
  ///
  /// In en, this message translates to:
  /// **'Avatar'**
  String get settingsAvatar;

  /// No description provided for @settingsAvatarUpload.
  ///
  /// In en, this message translates to:
  /// **'Upload avatar (converted to webp)'**
  String get settingsAvatarUpload;

  /// No description provided for @settingsAvatarUploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading…'**
  String get settingsAvatarUploading;

  /// No description provided for @settingsAvatarUploadFailed.
  ///
  /// In en, this message translates to:
  /// **'Avatar upload failed: {error}'**
  String settingsAvatarUploadFailed(String error);

  /// No description provided for @settingsEmail.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get settingsEmail;

  /// No description provided for @settingsEmailEdit.
  ///
  /// In en, this message translates to:
  /// **'Change bound email'**
  String get settingsEmailEdit;

  /// No description provided for @settingsEmailUpdated.
  ///
  /// In en, this message translates to:
  /// **'Email updated, please verify'**
  String get settingsEmailUpdated;

  /// No description provided for @settingsEmailFailed.
  ///
  /// In en, this message translates to:
  /// **'Email update failed: {error}'**
  String settingsEmailFailed(String error);

  /// No description provided for @settingsNewEmail.
  ///
  /// In en, this message translates to:
  /// **'New email'**
  String get settingsNewEmail;

  /// No description provided for @settingsChangePassword.
  ///
  /// In en, this message translates to:
  /// **'Change password'**
  String get settingsChangePassword;

  /// No description provided for @settingsChangePasswordSub.
  ///
  /// In en, this message translates to:
  /// **'Change login password'**
  String get settingsChangePasswordSub;

  /// No description provided for @settingsCurrentPassword.
  ///
  /// In en, this message translates to:
  /// **'Current password'**
  String get settingsCurrentPassword;

  /// No description provided for @settingsPasswordUpdated.
  ///
  /// In en, this message translates to:
  /// **'Password updated'**
  String get settingsPasswordUpdated;

  /// No description provided for @settingsPasswordFailed.
  ///
  /// In en, this message translates to:
  /// **'Password change failed: {error}'**
  String settingsPasswordFailed(String error);

  /// No description provided for @settingsBadge.
  ///
  /// In en, this message translates to:
  /// **'Badge'**
  String get settingsBadge;

  /// No description provided for @settingsBadgeNone.
  ///
  /// In en, this message translates to:
  /// **'Not wearing'**
  String get settingsBadgeNone;

  /// No description provided for @settingsBadgeCurrent.
  ///
  /// In en, this message translates to:
  /// **'Current: {name}'**
  String settingsBadgeCurrent(String name);

  /// No description provided for @settingsBadgePick.
  ///
  /// In en, this message translates to:
  /// **'Choose badge to wear'**
  String get settingsBadgePick;

  /// No description provided for @settingsBadgeNoOptions.
  ///
  /// In en, this message translates to:
  /// **'No wearable badges'**
  String get settingsBadgeNoOptions;

  /// No description provided for @settingsBadgeUpdated.
  ///
  /// In en, this message translates to:
  /// **'Badge updated'**
  String get settingsBadgeUpdated;

  /// No description provided for @settingsBadgeFailed.
  ///
  /// In en, this message translates to:
  /// **'Badge update failed: {error}'**
  String settingsBadgeFailed(String error);

  /// No description provided for @settingsOAuth.
  ///
  /// In en, this message translates to:
  /// **'Casdoor unified identity'**
  String get settingsOAuth;

  /// No description provided for @settingsOAuthSub.
  ///
  /// In en, this message translates to:
  /// **'OIDC login bindings'**
  String get settingsOAuthSub;

  /// No description provided for @settingsOAuthManage.
  ///
  /// In en, this message translates to:
  /// **'Manage OAuth bindings'**
  String get settingsOAuthManage;

  /// No description provided for @settingsOAuthBindings.
  ///
  /// In en, this message translates to:
  /// **'OAuth bindings'**
  String get settingsOAuthBindings;

  /// No description provided for @settingsBound.
  ///
  /// In en, this message translates to:
  /// **'Bound'**
  String get settingsBound;

  /// No description provided for @settingsUnbound.
  ///
  /// In en, this message translates to:
  /// **'Not bound'**
  String get settingsUnbound;

  /// No description provided for @settingsUnbind.
  ///
  /// In en, this message translates to:
  /// **'Unbind'**
  String get settingsUnbind;

  /// No description provided for @settingsUnboundDone.
  ///
  /// In en, this message translates to:
  /// **'Unbound'**
  String get settingsUnboundDone;

  /// No description provided for @settingsUnbindFailed.
  ///
  /// In en, this message translates to:
  /// **'Unbind failed: {error}'**
  String settingsUnbindFailed(String error);

  /// No description provided for @settingsLoadBindingsFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to load bindings: {error}'**
  String settingsLoadBindingsFailed(String error);

  /// No description provided for @settingsPrivacyDirect.
  ///
  /// In en, this message translates to:
  /// **'Messages visible to friends only'**
  String get settingsPrivacyDirect;

  /// No description provided for @settingsPrivacyLikes.
  ///
  /// In en, this message translates to:
  /// **'Show my likes publicly'**
  String get settingsPrivacyLikes;

  /// No description provided for @settingsSessions.
  ///
  /// In en, this message translates to:
  /// **'Session management'**
  String get settingsSessions;

  /// No description provided for @settingsSessionsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No sessions'**
  String get settingsSessionsEmpty;

  /// No description provided for @settingsRevokeAll.
  ///
  /// In en, this message translates to:
  /// **'Revoke all sessions'**
  String get settingsRevokeAll;

  /// No description provided for @settingsRevoked.
  ///
  /// In en, this message translates to:
  /// **'Session revoked'**
  String get settingsRevoked;

  /// No description provided for @settingsRevokeFailed.
  ///
  /// In en, this message translates to:
  /// **'Revoke failed: {error}'**
  String settingsRevokeFailed(String error);

  /// No description provided for @settingsRevokeAllDone.
  ///
  /// In en, this message translates to:
  /// **'All sessions revoked'**
  String get settingsRevokeAllDone;

  /// No description provided for @settingsOpFailed.
  ///
  /// In en, this message translates to:
  /// **'Operation failed: {error}'**
  String settingsOpFailed(String error);

  /// No description provided for @settingsAppearance.
  ///
  /// In en, this message translates to:
  /// **'Appearance'**
  String get settingsAppearance;

  /// No description provided for @settingsDarkMode.
  ///
  /// In en, this message translates to:
  /// **'Dark mode'**
  String get settingsDarkMode;

  /// No description provided for @settingsDarkCurrent.
  ///
  /// In en, this message translates to:
  /// **'Current: dark'**
  String get settingsDarkCurrent;

  /// No description provided for @settingsLightCurrent.
  ///
  /// In en, this message translates to:
  /// **'Current: light'**
  String get settingsLightCurrent;

  /// No description provided for @settingsAbout.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get settingsAbout;

  /// No description provided for @settingsAboutVersion.
  ///
  /// In en, this message translates to:
  /// **'v0.1.0 · Tongji campus forum'**
  String get settingsAboutVersion;

  /// No description provided for @settingsEditProfile.
  ///
  /// In en, this message translates to:
  /// **'Edit profile'**
  String get settingsEditProfile;

  /// No description provided for @settingsSignature.
  ///
  /// In en, this message translates to:
  /// **'Signature'**
  String get settingsSignature;

  /// No description provided for @settingsSaveInfo.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get settingsSaveInfo;

  /// No description provided for @settingsInfoSaved.
  ///
  /// In en, this message translates to:
  /// **'Profile updated'**
  String get settingsInfoSaved;

  /// No description provided for @settingsInfoFailed.
  ///
  /// In en, this message translates to:
  /// **'Profile update failed: {error}'**
  String settingsInfoFailed(String error);

  /// No description provided for @settingsUserDataLoading.
  ///
  /// In en, this message translates to:
  /// **'Account data loading…'**
  String get settingsUserDataLoading;

  /// No description provided for @settingsFillComplete.
  ///
  /// In en, this message translates to:
  /// **'Please fill in all fields'**
  String get settingsFillComplete;

  /// No description provided for @settingsSecondPhase.
  ///
  /// In en, this message translates to:
  /// **'Coming in phase 2'**
  String get settingsSecondPhase;

  /// No description provided for @settingsTotpTitle.
  ///
  /// In en, this message translates to:
  /// **'Two-factor auth (TOTP)'**
  String get settingsTotpTitle;

  /// No description provided for @settingsTotpEnable.
  ///
  /// In en, this message translates to:
  /// **'Enable'**
  String get settingsTotpEnable;

  /// No description provided for @settingsTotpDisable.
  ///
  /// In en, this message translates to:
  /// **'Disable'**
  String get settingsTotpDisable;

  /// No description provided for @settingsTotpPasswordPrompt.
  ///
  /// In en, this message translates to:
  /// **'Enter password to manage TOTP'**
  String get settingsTotpPasswordPrompt;

  /// No description provided for @settingsTotpSetupSecret.
  ///
  /// In en, this message translates to:
  /// **'Scan or enter the secret in your authenticator app'**
  String get settingsTotpSetupSecret;

  /// No description provided for @settingsTotpCode.
  ///
  /// In en, this message translates to:
  /// **'Enter the 6-digit code'**
  String get settingsTotpCode;

  /// No description provided for @settingsTotpRecoveryCodes.
  ///
  /// In en, this message translates to:
  /// **'Recovery codes (save them safely):'**
  String get settingsTotpRecoveryCodes;

  /// No description provided for @settingsTotpEnabled.
  ///
  /// In en, this message translates to:
  /// **'TOTP enabled'**
  String get settingsTotpEnabled;

  /// No description provided for @settingsTotpDisabled.
  ///
  /// In en, this message translates to:
  /// **'TOTP disabled'**
  String get settingsTotpDisabled;

  /// No description provided for @settingsTotpFailed.
  ///
  /// In en, this message translates to:
  /// **'TOTP operation failed: {error}'**
  String settingsTotpFailed(String error);

  /// No description provided for @settingsTotpDisableTitle.
  ///
  /// In en, this message translates to:
  /// **'Disable TOTP'**
  String get settingsTotpDisableTitle;

  /// No description provided for @settingsTotpEnableTitle.
  ///
  /// In en, this message translates to:
  /// **'Enable TOTP'**
  String get settingsTotpEnableTitle;

  /// No description provided for @settingsTotpPassword.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get settingsTotpPassword;

  /// No description provided for @settingsTotpNext.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get settingsTotpNext;

  /// No description provided for @settingsTotpScanSecret.
  ///
  /// In en, this message translates to:
  /// **'Scan or enter the secret'**
  String get settingsTotpScanSecret;

  /// No description provided for @settingsTotpDone.
  ///
  /// In en, this message translates to:
  /// **'Done'**
  String get settingsTotpDone;

  /// No description provided for @settingsTotpUnavailable.
  ///
  /// In en, this message translates to:
  /// **'TOTP unavailable'**
  String get settingsTotpUnavailable;

  /// No description provided for @draftsTitle.
  ///
  /// In en, this message translates to:
  /// **'Drafts'**
  String get draftsTitle;

  /// No description provided for @draftsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No drafts'**
  String get draftsEmpty;

  /// No description provided for @draftsNew.
  ///
  /// In en, this message translates to:
  /// **'New draft'**
  String get draftsNew;

  /// No description provided for @draftsBlocked.
  ///
  /// In en, this message translates to:
  /// **'Blocked'**
  String get draftsBlocked;

  /// No description provided for @draftsMetaCreated.
  ///
  /// In en, this message translates to:
  /// **'Created {date}'**
  String draftsMetaCreated(Object date);

  /// No description provided for @draftsMetaViews.
  ///
  /// In en, this message translates to:
  /// **'{count} views'**
  String draftsMetaViews(Object count);

  /// No description provided for @draftsMetaReplies.
  ///
  /// In en, this message translates to:
  /// **'{count} replies'**
  String draftsMetaReplies(Object count);

  /// No description provided for @messagesNew.
  ///
  /// In en, this message translates to:
  /// **'New message'**
  String get messagesNew;

  /// No description provided for @messagesSearchUsers.
  ///
  /// In en, this message translates to:
  /// **'Search users'**
  String get messagesSearchUsers;

  /// No description provided for @messagesNoContactableUsers.
  ///
  /// In en, this message translates to:
  /// **'No contactable users'**
  String get messagesNoContactableUsers;

  /// No description provided for @settingsLogout.
  ///
  /// In en, this message translates to:
  /// **'Log out'**
  String get settingsLogout;

  /// No description provided for @settingsLogoutConfirm.
  ///
  /// In en, this message translates to:
  /// **'Log out of yourtj?'**
  String get settingsLogoutConfirm;

  /// No description provided for @commonParseFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to parse page data'**
  String get commonParseFailed;

  /// No description provided for @topicEmpty.
  ///
  /// In en, this message translates to:
  /// **'No topics yet'**
  String get topicEmpty;

  /// No description provided for @settingsAvatarUploaded.
  ///
  /// In en, this message translates to:
  /// **'Avatar uploaded: {url}'**
  String settingsAvatarUploaded(String url);

  /// No description provided for @settingsImageDecodeFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to decode image'**
  String get settingsImageDecodeFailed;

  /// No description provided for @dateMonthDayTime.
  ///
  /// In en, this message translates to:
  /// **'{month}/{day} {time}'**
  String dateMonthDayTime(int month, int day, String time);

  /// No description provided for @dateYearMonthDayTime.
  ///
  /// In en, this message translates to:
  /// **'{year}/{month}/{day} {time}'**
  String dateYearMonthDayTime(int year, int month, int day, String time);

  /// No description provided for @topicFloorSelected.
  ///
  /// In en, this message translates to:
  /// **'Jumped to floor {floor}'**
  String topicFloorSelected(Object floor);

  /// No description provided for @sortLatest.
  ///
  /// In en, this message translates to:
  /// **'Latest'**
  String get sortLatest;

  /// No description provided for @sortHot.
  ///
  /// In en, this message translates to:
  /// **'Hot'**
  String get sortHot;

  /// No description provided for @sortPopular.
  ///
  /// In en, this message translates to:
  /// **'Popular'**
  String get sortPopular;

  /// No description provided for @topicFeedModeList.
  ///
  /// In en, this message translates to:
  /// **'List'**
  String get topicFeedModeList;

  /// No description provided for @topicFeedModeCard.
  ///
  /// In en, this message translates to:
  /// **'Cards'**
  String get topicFeedModeCard;

  /// No description provided for @topicNewTopic.
  ///
  /// In en, this message translates to:
  /// **'New topic'**
  String get topicNewTopic;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
