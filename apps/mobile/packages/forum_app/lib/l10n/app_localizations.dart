import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_de.dart';
import 'app_localizations_en.dart';
import 'app_localizations_ja.dart';
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
    Locale('de'),
    Locale('en'),
    Locale('ja'),
    Locale('zh'),
  ];

  /// No description provided for @myCourseReviewsTitle.
  ///
  /// In en, this message translates to:
  /// **'My course reviews'**
  String get myCourseReviewsTitle;

  /// No description provided for @myCourseReviewsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No reviews yet. Explore the course catalog to get started.'**
  String get myCourseReviewsEmpty;

  /// No description provided for @myCourseReviewHidden.
  ///
  /// In en, this message translates to:
  /// **'This review is hidden. You can delete it here.'**
  String get myCourseReviewHidden;

  /// No description provided for @myCourseReviewUnavailable.
  ///
  /// In en, this message translates to:
  /// **'This course is unavailable. You can still manage your review.'**
  String get myCourseReviewUnavailable;

  /// No description provided for @myCourseReviewOpen.
  ///
  /// In en, this message translates to:
  /// **'View details'**
  String get myCourseReviewOpen;

  /// No description provided for @courseReviewNetworkError.
  ///
  /// In en, this message translates to:
  /// **'Could not connect. Check your connection and retry; your review has been kept.'**
  String get courseReviewNetworkError;

  /// No description provided for @courseReviewUnknownError.
  ///
  /// In en, this message translates to:
  /// **'Your review could not be saved. The server did not provide a specific reason; please retry later.'**
  String get courseReviewUnknownError;

  /// No description provided for @commonHideKeyboard.
  ///
  /// In en, this message translates to:
  /// **'Hide keyboard'**
  String get commonHideKeyboard;

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'YourTJ'**
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

  /// No description provided for @authOidcLogin.
  ///
  /// In en, this message translates to:
  /// **'Sign in with yourtj'**
  String get authOidcLogin;

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

  /// No description provided for @authCacheClearFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to clear the previous account\'s offline data. Please retry.'**
  String get authCacheClearFailed;

  /// No description provided for @authSessionSaveFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to save the new session securely. Please retry.'**
  String get authSessionSaveFailed;

  /// No description provided for @loginWelcome.
  ///
  /// In en, this message translates to:
  /// **'Welcome to YourTJ'**
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

  /// No description provided for @topicReplyTargetUnavailable.
  ///
  /// In en, this message translates to:
  /// **'The original reply is unavailable'**
  String get topicReplyTargetUnavailable;

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

  /// No description provided for @topicWatch.
  ///
  /// In en, this message translates to:
  /// **'Watch topic replies'**
  String get topicWatch;

  /// No description provided for @topicUnwatch.
  ///
  /// In en, this message translates to:
  /// **'Stop watching topic replies'**
  String get topicUnwatch;

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

  /// No description provided for @settingsEmailOAuthReauthRequired.
  ///
  /// In en, this message translates to:
  /// **'The password could not be verified for this OAuth-linked account. Re-authenticate with your provider or contact an administrator.'**
  String get settingsEmailOAuthReauthRequired;

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
  /// **'External account connections'**
  String get settingsOAuth;

  /// No description provided for @settingsOAuthSub.
  ///
  /// In en, this message translates to:
  /// **'GitHub and Google login connections'**
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

  /// No description provided for @commonLoadFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to load'**
  String get commonLoadFailed;

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

  /// No description provided for @scheduleTitle.
  ///
  /// In en, this message translates to:
  /// **'Scheduler'**
  String get scheduleTitle;

  /// No description provided for @scheduleTabTimetable.
  ///
  /// In en, this message translates to:
  /// **'Plan preview'**
  String get scheduleTabTimetable;

  /// No description provided for @scheduleTabPick.
  ///
  /// In en, this message translates to:
  /// **'Pick courses'**
  String get scheduleTabPick;

  /// No description provided for @scheduleTerm.
  ///
  /// In en, this message translates to:
  /// **'Term'**
  String get scheduleTerm;

  /// No description provided for @scheduleGrade.
  ///
  /// In en, this message translates to:
  /// **'Grade'**
  String get scheduleGrade;

  /// No description provided for @scheduleMajor.
  ///
  /// In en, this message translates to:
  /// **'Major'**
  String get scheduleMajor;

  /// No description provided for @scheduleSyncLatest.
  ///
  /// In en, this message translates to:
  /// **'Sync latest'**
  String get scheduleSyncLatest;

  /// No description provided for @scheduleDataOutdated.
  ///
  /// In en, this message translates to:
  /// **'Course data updated, tap to sync'**
  String get scheduleDataOutdated;

  /// No description provided for @scheduleSyncedTo.
  ///
  /// In en, this message translates to:
  /// **'Synced to {date}'**
  String scheduleSyncedTo(String date);

  /// No description provided for @scheduleSyncConflictTitle.
  ///
  /// In en, this message translates to:
  /// **'Plan sync conflict'**
  String get scheduleSyncConflictTitle;

  /// No description provided for @scheduleSyncConflictBody.
  ///
  /// In en, this message translates to:
  /// **'Your local schedule plans differ from the cloud copy. Which one should be kept?'**
  String get scheduleSyncConflictBody;

  /// No description provided for @scheduleSyncUseCloud.
  ///
  /// In en, this message translates to:
  /// **'Use cloud'**
  String get scheduleSyncUseCloud;

  /// No description provided for @scheduleSyncKeepLocal.
  ///
  /// In en, this message translates to:
  /// **'Keep local'**
  String get scheduleSyncKeepLocal;

  /// No description provided for @scheduleWeekAll.
  ///
  /// In en, this message translates to:
  /// **'All weeks'**
  String get scheduleWeekAll;

  /// No description provided for @scheduleWeekN.
  ///
  /// In en, this message translates to:
  /// **'Week {week}'**
  String scheduleWeekN(int week);

  /// No description provided for @scheduleCurrentWeek.
  ///
  /// In en, this message translates to:
  /// **'Current week'**
  String get scheduleCurrentWeek;

  /// No description provided for @schedulePlanNew.
  ///
  /// In en, this message translates to:
  /// **'New plan'**
  String get schedulePlanNew;

  /// No description provided for @schedulePlanRename.
  ///
  /// In en, this message translates to:
  /// **'Rename plan'**
  String get schedulePlanRename;

  /// No description provided for @schedulePlanDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete plan'**
  String get schedulePlanDelete;

  /// No description provided for @schedulePlanClear.
  ///
  /// In en, this message translates to:
  /// **'Clear courses'**
  String get schedulePlanClear;

  /// No description provided for @schedulePlanN.
  ///
  /// In en, this message translates to:
  /// **'Plan {n}'**
  String schedulePlanN(int n);

  /// No description provided for @scheduleStatsCourses.
  ///
  /// In en, this message translates to:
  /// **'{count} courses'**
  String scheduleStatsCourses(int count);

  /// No description provided for @scheduleStatsCredits.
  ///
  /// In en, this message translates to:
  /// **'Credits'**
  String get scheduleStatsCredits;

  /// No description provided for @scheduleStatsHours.
  ///
  /// In en, this message translates to:
  /// **'Hours'**
  String get scheduleStatsHours;

  /// No description provided for @scheduleStatsConflicts.
  ///
  /// In en, this message translates to:
  /// **'Conflicts'**
  String get scheduleStatsConflicts;

  /// No description provided for @scheduleConflictBadge.
  ///
  /// In en, this message translates to:
  /// **'Conflict'**
  String get scheduleConflictBadge;

  /// No description provided for @scheduleConflictWith.
  ///
  /// In en, this message translates to:
  /// **'Conflicts with \"{course}\"'**
  String scheduleConflictWith(String course);

  /// No description provided for @scheduleConflictsWith.
  ///
  /// In en, this message translates to:
  /// **'Conflicts with \"{course}\" and {count} other courses'**
  String scheduleConflictsWith(String course, int count);

  /// No description provided for @scheduleConflictCanAdd.
  ///
  /// In en, this message translates to:
  /// **'Can still add'**
  String get scheduleConflictCanAdd;

  /// No description provided for @scheduleAddCustomEvent.
  ///
  /// In en, this message translates to:
  /// **'Add placeholder'**
  String get scheduleAddCustomEvent;

  /// No description provided for @scheduleCustomEventLabel.
  ///
  /// In en, this message translates to:
  /// **'Busy'**
  String get scheduleCustomEventLabel;

  /// No description provided for @scheduleExportPng.
  ///
  /// In en, this message translates to:
  /// **'Export image'**
  String get scheduleExportPng;

  /// No description provided for @scheduleExportCsv.
  ///
  /// In en, this message translates to:
  /// **'Export CSV'**
  String get scheduleExportCsv;

  /// No description provided for @scheduleDegraded.
  ///
  /// In en, this message translates to:
  /// **'Degraded search, results may be incomplete'**
  String get scheduleDegraded;

  /// No description provided for @scheduleNoReviewData.
  ///
  /// In en, this message translates to:
  /// **'No review data yet'**
  String get scheduleNoReviewData;

  /// No description provided for @schedulePickClass.
  ///
  /// In en, this message translates to:
  /// **'Pick class'**
  String get schedulePickClass;

  /// No description provided for @scheduleCompulsory.
  ///
  /// In en, this message translates to:
  /// **'Compulsory'**
  String get scheduleCompulsory;

  /// No description provided for @scheduleOptional.
  ///
  /// In en, this message translates to:
  /// **'Elective'**
  String get scheduleOptional;

  /// No description provided for @scheduleSearchHint.
  ///
  /// In en, this message translates to:
  /// **'Search name/code/teacher'**
  String get scheduleSearchHint;

  /// No description provided for @scheduleStaged.
  ///
  /// In en, this message translates to:
  /// **'Staged'**
  String get scheduleStaged;

  /// No description provided for @scheduleSelected.
  ///
  /// In en, this message translates to:
  /// **'Selected'**
  String get scheduleSelected;

  /// No description provided for @scheduleRemoveCourse.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get scheduleRemoveCourse;

  /// No description provided for @scheduleParityOdd.
  ///
  /// In en, this message translates to:
  /// **'odd weeks'**
  String get scheduleParityOdd;

  /// No description provided for @scheduleParityEven.
  ///
  /// In en, this message translates to:
  /// **'even weeks'**
  String get scheduleParityEven;

  /// No description provided for @scheduleWeeksN.
  ///
  /// In en, this message translates to:
  /// **'{range} weeks'**
  String scheduleWeeksN(String range);

  /// No description provided for @scheduleDayMon.
  ///
  /// In en, this message translates to:
  /// **'Mon'**
  String get scheduleDayMon;

  /// No description provided for @scheduleDayTue.
  ///
  /// In en, this message translates to:
  /// **'Tue'**
  String get scheduleDayTue;

  /// No description provided for @scheduleDayWed.
  ///
  /// In en, this message translates to:
  /// **'Wed'**
  String get scheduleDayWed;

  /// No description provided for @scheduleDayThu.
  ///
  /// In en, this message translates to:
  /// **'Thu'**
  String get scheduleDayThu;

  /// No description provided for @scheduleDayFri.
  ///
  /// In en, this message translates to:
  /// **'Fri'**
  String get scheduleDayFri;

  /// No description provided for @scheduleDaySat.
  ///
  /// In en, this message translates to:
  /// **'Sat'**
  String get scheduleDaySat;

  /// No description provided for @scheduleDaySun.
  ///
  /// In en, this message translates to:
  /// **'Sun'**
  String get scheduleDaySun;

  /// No description provided for @scheduleMorning.
  ///
  /// In en, this message translates to:
  /// **'Morning'**
  String get scheduleMorning;

  /// No description provided for @scheduleAfternoon.
  ///
  /// In en, this message translates to:
  /// **'Afternoon'**
  String get scheduleAfternoon;

  /// No description provided for @scheduleEvening.
  ///
  /// In en, this message translates to:
  /// **'Evening'**
  String get scheduleEvening;

  /// No description provided for @coursesTitle.
  ///
  /// In en, this message translates to:
  /// **'Courses'**
  String get coursesTitle;

  /// No description provided for @coursesSearchHint.
  ///
  /// In en, this message translates to:
  /// **'Search courses'**
  String get coursesSearchHint;

  /// No description provided for @coursesFilterDepartment.
  ///
  /// In en, this message translates to:
  /// **'Department'**
  String get coursesFilterDepartment;

  /// No description provided for @coursesFilterTerm.
  ///
  /// In en, this message translates to:
  /// **'Term'**
  String get coursesFilterTerm;

  /// No description provided for @coursesFilterCampus.
  ///
  /// In en, this message translates to:
  /// **'Campus'**
  String get coursesFilterCampus;

  /// No description provided for @coursesFilterInstructor.
  ///
  /// In en, this message translates to:
  /// **'Instructor'**
  String get coursesFilterInstructor;

  /// No description provided for @coursesOnlyWithReviews.
  ///
  /// In en, this message translates to:
  /// **'With reviews only'**
  String get coursesOnlyWithReviews;

  /// No description provided for @coursesRatingCount.
  ///
  /// In en, this message translates to:
  /// **'{count} reviews'**
  String coursesRatingCount(int count);

  /// No description provided for @coursesNoRating.
  ///
  /// In en, this message translates to:
  /// **'No rating'**
  String get coursesNoRating;

  /// No description provided for @courseDetailReviews.
  ///
  /// In en, this message translates to:
  /// **'Reviews'**
  String get courseDetailReviews;

  /// No description provided for @courseDetailOfferings.
  ///
  /// In en, this message translates to:
  /// **'Classes'**
  String get courseDetailOfferings;

  /// No description provided for @courseDetailLineage.
  ///
  /// In en, this message translates to:
  /// **'History'**
  String get courseDetailLineage;

  /// No description provided for @courseDetailRelated.
  ///
  /// In en, this message translates to:
  /// **'Related courses'**
  String get courseDetailRelated;

  /// No description provided for @courseDetailAiSummary.
  ///
  /// In en, this message translates to:
  /// **'AI summary'**
  String get courseDetailAiSummary;

  /// No description provided for @courseDetailAiSummaryEmpty.
  ///
  /// In en, this message translates to:
  /// **'No AI summary'**
  String get courseDetailAiSummaryEmpty;

  /// No description provided for @courseBookmark.
  ///
  /// In en, this message translates to:
  /// **'Bookmark'**
  String get courseBookmark;

  /// No description provided for @courseBookmarked.
  ///
  /// In en, this message translates to:
  /// **'Bookmarked'**
  String get courseBookmarked;

  /// No description provided for @courseWriteReview.
  ///
  /// In en, this message translates to:
  /// **'Write review'**
  String get courseWriteReview;

  /// No description provided for @reviewAnonymous.
  ///
  /// In en, this message translates to:
  /// **'Anonymous'**
  String get reviewAnonymous;

  /// No description provided for @reviewSubmit.
  ///
  /// In en, this message translates to:
  /// **'Submit'**
  String get reviewSubmit;

  /// No description provided for @reviewHelpful.
  ///
  /// In en, this message translates to:
  /// **'Helpful'**
  String get reviewHelpful;

  /// No description provided for @reviewsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No reviews yet'**
  String get reviewsEmpty;

  /// No description provided for @wikiTitle.
  ///
  /// In en, this message translates to:
  /// **'Wiki'**
  String get wikiTitle;

  /// No description provided for @wikiRecent.
  ///
  /// In en, this message translates to:
  /// **'Recently updated'**
  String get wikiRecent;

  /// No description provided for @wikiEditOnGithub.
  ///
  /// In en, this message translates to:
  /// **'Edit on GitHub'**
  String get wikiEditOnGithub;

  /// No description provided for @wikiToc.
  ///
  /// In en, this message translates to:
  /// **'Contents'**
  String get wikiToc;

  /// No description provided for @wikiNamespaces.
  ///
  /// In en, this message translates to:
  /// **'Namespaces'**
  String get wikiNamespaces;

  /// No description provided for @wikiViewCount.
  ///
  /// In en, this message translates to:
  /// **'{count} views'**
  String wikiViewCount(int count);

  /// No description provided for @settingsPush.
  ///
  /// In en, this message translates to:
  /// **'Push notifications'**
  String get settingsPush;

  /// No description provided for @settingsPushDenied.
  ///
  /// In en, this message translates to:
  /// **'Notification permission denied, tap to open Settings'**
  String get settingsPushDenied;

  /// No description provided for @settingsFollowSiteTheme.
  ///
  /// In en, this message translates to:
  /// **'Follow site theme'**
  String get settingsFollowSiteTheme;

  /// No description provided for @settingsFollowSiteThemeDesc.
  ///
  /// In en, this message translates to:
  /// **'Use server-issued site colors'**
  String get settingsFollowSiteThemeDesc;

  /// No description provided for @entryCourses.
  ///
  /// In en, this message translates to:
  /// **'Courses'**
  String get entryCourses;

  /// No description provided for @entrySchedule.
  ///
  /// In en, this message translates to:
  /// **'Schedule'**
  String get entrySchedule;

  /// No description provided for @entryWiki.
  ///
  /// In en, this message translates to:
  /// **'Wiki'**
  String get entryWiki;

  /// No description provided for @navCampus.
  ///
  /// In en, this message translates to:
  /// **'Campus'**
  String get navCampus;

  /// No description provided for @campusTitle.
  ///
  /// In en, this message translates to:
  /// **'Your campus, connected'**
  String get campusTitle;

  /// No description provided for @campusSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Find courses, plan your week and share campus knowledge.'**
  String get campusSubtitle;

  /// No description provided for @campusCoursesHint.
  ///
  /// In en, this message translates to:
  /// **'Explore courses and student reviews'**
  String get campusCoursesHint;

  /// No description provided for @campusScheduleHint.
  ///
  /// In en, this message translates to:
  /// **'Plan your week, check conflicts and export'**
  String get campusScheduleHint;

  /// No description provided for @campusWikiHint.
  ///
  /// In en, this message translates to:
  /// **'A campus guide built by the community'**
  String get campusWikiHint;

  /// No description provided for @publishMoment.
  ///
  /// In en, this message translates to:
  /// **'Moment'**
  String get publishMoment;

  /// No description provided for @publishQuestion.
  ///
  /// In en, this message translates to:
  /// **'Question'**
  String get publishQuestion;

  /// No description provided for @publishArticle.
  ///
  /// In en, this message translates to:
  /// **'Article'**
  String get publishArticle;

  /// No description provided for @publishNext.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get publishNext;

  /// No description provided for @publishGallery.
  ///
  /// In en, this message translates to:
  /// **'Choose photos, then tell your story'**
  String get publishGallery;

  /// No description provided for @publishGalleryHint.
  ///
  /// In en, this message translates to:
  /// **'Up to 9 photos. Hold and drag to reorder.'**
  String get publishGalleryHint;

  /// No description provided for @publishFormatting.
  ///
  /// In en, this message translates to:
  /// **'Formatting'**
  String get publishFormatting;

  /// No description provided for @publishClassification.
  ///
  /// In en, this message translates to:
  /// **'Choose categories'**
  String get publishClassification;

  /// No description provided for @publishLeaveTitle.
  ///
  /// In en, this message translates to:
  /// **'Keep your work?'**
  String get publishLeaveTitle;

  /// No description provided for @publishLeaveBody.
  ///
  /// In en, this message translates to:
  /// **'You have unsaved changes. Continue editing or discard them.'**
  String get publishLeaveBody;

  /// No description provided for @publishDiscard.
  ///
  /// In en, this message translates to:
  /// **'Discard changes'**
  String get publishDiscard;

  /// No description provided for @publishContinue.
  ///
  /// In en, this message translates to:
  /// **'Keep editing'**
  String get publishContinue;

  /// No description provided for @publishImageOnlyTitle.
  ///
  /// In en, this message translates to:
  /// **'A moment to share'**
  String get publishImageOnlyTitle;

  /// No description provided for @publishTypeLocked.
  ///
  /// In en, this message translates to:
  /// **'The original type is retained when editing'**
  String get publishTypeLocked;

  /// No description provided for @profileMore.
  ///
  /// In en, this message translates to:
  /// **'More options'**
  String get profileMore;

  /// No description provided for @profileContent.
  ///
  /// In en, this message translates to:
  /// **'Manage content'**
  String get profileContent;

  /// No description provided for @profileTrash.
  ///
  /// In en, this message translates to:
  /// **'Recycle bin'**
  String get profileTrash;

  /// No description provided for @profileSecurity.
  ///
  /// In en, this message translates to:
  /// **'Account, security & privacy'**
  String get profileSecurity;

  /// No description provided for @profileAdmin.
  ///
  /// In en, this message translates to:
  /// **'Admin workspace'**
  String get profileAdmin;

  /// No description provided for @fabDiscussion.
  ///
  /// In en, this message translates to:
  /// **'Jump to discussion'**
  String get fabDiscussion;

  /// No description provided for @fabRefresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh and return to top'**
  String get fabRefresh;

  /// No description provided for @contentRestore.
  ///
  /// In en, this message translates to:
  /// **'Restore'**
  String get contentRestore;

  /// No description provided for @contentDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get contentDelete;

  /// No description provided for @contentPurge.
  ///
  /// In en, this message translates to:
  /// **'Delete permanently'**
  String get contentPurge;

  /// No description provided for @contentPrivacyErase.
  ///
  /// In en, this message translates to:
  /// **'Erase personal content'**
  String get contentPrivacyErase;

  /// No description provided for @contentDeleteConfirm.
  ///
  /// In en, this message translates to:
  /// **'Content moves to the recycle bin and can be restored within 30 days when eligible.'**
  String get contentDeleteConfirm;

  /// No description provided for @contentPurgeConfirm.
  ///
  /// In en, this message translates to:
  /// **'This cannot be undone. Text and images will be permanently erased.'**
  String get contentPurgeConfirm;

  /// No description provided for @contentPassword.
  ///
  /// In en, this message translates to:
  /// **'Enter your current password to confirm'**
  String get contentPassword;

  /// No description provided for @contentSelected.
  ///
  /// In en, this message translates to:
  /// **'Selected'**
  String get contentSelected;

  /// No description provided for @contentSelectAll.
  ///
  /// In en, this message translates to:
  /// **'Select loaded items'**
  String get contentSelectAll;

  /// No description provided for @contentEmpty.
  ///
  /// In en, this message translates to:
  /// **'No content here yet'**
  String get contentEmpty;

  /// No description provided for @contentTrashHint.
  ///
  /// In en, this message translates to:
  /// **'Eligible content can be restored for 30 days. Moderated removals cannot be restored here.'**
  String get contentTrashHint;

  /// No description provided for @commonConfirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get commonConfirm;

  /// No description provided for @adminUnavailable.
  ///
  /// In en, this message translates to:
  /// **'The admin console is unavailable. Check your permissions, session and connection, then retry.'**
  String get adminUnavailable;

  /// No description provided for @adminDownloadFailed.
  ///
  /// In en, this message translates to:
  /// **'Could not download the export. Please try again.'**
  String get adminDownloadFailed;

  /// No description provided for @publishGalleryTooMany.
  ///
  /// In en, this message translates to:
  /// **'Content supports up to 9 images. Remove some images or continue with the article editor.'**
  String get publishGalleryTooMany;

  /// No description provided for @profileActivity.
  ///
  /// In en, this message translates to:
  /// **'Activity'**
  String get profileActivity;

  /// No description provided for @profileBookmarks.
  ///
  /// In en, this message translates to:
  /// **'Bookmarks'**
  String get profileBookmarks;

  /// No description provided for @profileModeration.
  ///
  /// In en, this message translates to:
  /// **'Moderation workspace'**
  String get profileModeration;

  /// No description provided for @settingsCloseAccount.
  ///
  /// In en, this message translates to:
  /// **'Close account'**
  String get settingsCloseAccount;

  /// No description provided for @settingsCloseAccountWarning.
  ///
  /// In en, this message translates to:
  /// **'Account closure is irreversible and signs out all devices. Keep historical content under an anonymized identity, or request deletion of your content. Retention rules may preserve some content. Enter your current password to confirm.'**
  String get settingsCloseAccountWarning;

  /// No description provided for @settingsCloseKeepContent.
  ///
  /// In en, this message translates to:
  /// **'Keep anonymized content'**
  String get settingsCloseKeepContent;

  /// No description provided for @settingsCloseDeleteContent.
  ///
  /// In en, this message translates to:
  /// **'Request content deletion'**
  String get settingsCloseDeleteContent;

  /// No description provided for @publishUndo.
  ///
  /// In en, this message translates to:
  /// **'Undo'**
  String get publishUndo;

  /// No description provided for @publishRedo.
  ///
  /// In en, this message translates to:
  /// **'Redo'**
  String get publishRedo;

  /// No description provided for @publishHeading.
  ///
  /// In en, this message translates to:
  /// **'Heading'**
  String get publishHeading;

  /// No description provided for @publishToolLink.
  ///
  /// In en, this message translates to:
  /// **'Insert link'**
  String get publishToolLink;

  /// No description provided for @publishLinkInvalid.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid http, https or email link.'**
  String get publishLinkInvalid;

  /// No description provided for @publishPhotoLibrary.
  ///
  /// In en, this message translates to:
  /// **'Choose from library'**
  String get publishPhotoLibrary;

  /// No description provided for @publishCamera.
  ///
  /// In en, this message translates to:
  /// **'Take a photo'**
  String get publishCamera;

  /// No description provided for @settingsWebsiteName.
  ///
  /// In en, this message translates to:
  /// **'Website name'**
  String get settingsWebsiteName;

  /// No description provided for @settingsWebsite.
  ///
  /// In en, this message translates to:
  /// **'Website'**
  String get settingsWebsite;

  /// No description provided for @settingsSocialLinks.
  ///
  /// In en, this message translates to:
  /// **'Social links'**
  String get settingsSocialLinks;

  /// No description provided for @settingsProfileLanguage.
  ///
  /// In en, this message translates to:
  /// **'Profile language'**
  String get settingsProfileLanguage;

  /// No description provided for @settingsInvalidLink.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid http or https link'**
  String get settingsInvalidLink;

  /// No description provided for @siteInfoTitle.
  ///
  /// In en, this message translates to:
  /// **'About the community'**
  String get siteInfoTitle;

  /// No description provided for @siteInfoLinks.
  ///
  /// In en, this message translates to:
  /// **'Community links'**
  String get siteInfoLinks;

  /// No description provided for @siteInfoSponsors.
  ///
  /// In en, this message translates to:
  /// **'Supporters'**
  String get siteInfoSponsors;

  /// No description provided for @siteInfoTerms.
  ///
  /// In en, this message translates to:
  /// **'Terms of service'**
  String get siteInfoTerms;

  /// No description provided for @siteInfoPrivacy.
  ///
  /// In en, this message translates to:
  /// **'Privacy policy'**
  String get siteInfoPrivacy;

  /// No description provided for @siteInfoEmpty.
  ///
  /// In en, this message translates to:
  /// **'No public content yet'**
  String get siteInfoEmpty;

  /// No description provided for @settingsUsernameHint.
  ///
  /// In en, this message translates to:
  /// **'This is your sign-in name. Site naming rules apply.'**
  String get settingsUsernameHint;

  /// No description provided for @settingsUsernameUpdated.
  ///
  /// In en, this message translates to:
  /// **'Username updated'**
  String get settingsUsernameUpdated;

  /// No description provided for @settingsPresetAvatar.
  ///
  /// In en, this message translates to:
  /// **'Choose a preset avatar'**
  String get settingsPresetAvatar;

  /// No description provided for @coursesManagement.
  ///
  /// In en, this message translates to:
  /// **'Manage courses'**
  String get coursesManagement;

  /// No description provided for @coursesReviewModeration.
  ///
  /// In en, this message translates to:
  /// **'Review moderation'**
  String get coursesReviewModeration;

  /// No description provided for @settingsCover.
  ///
  /// In en, this message translates to:
  /// **'Profile cover'**
  String get settingsCover;

  /// No description provided for @settingsCoverDescription.
  ///
  /// In en, this message translates to:
  /// **'Choose an image and adjust the crop'**
  String get settingsCoverDescription;

  /// No description provided for @settingsCoverRemove.
  ///
  /// In en, this message translates to:
  /// **'Remove cover'**
  String get settingsCoverRemove;

  /// No description provided for @settingsCoverRemoveConfirm.
  ///
  /// In en, this message translates to:
  /// **'Your profile will use the default background.'**
  String get settingsCoverRemoveConfirm;

  /// No description provided for @settingsCoverMinSize.
  ///
  /// In en, this message translates to:
  /// **'Choose an image at least 1200 × 240 pixels'**
  String get settingsCoverMinSize;

  /// No description provided for @settingsCoverSafeArea.
  ///
  /// In en, this message translates to:
  /// **'The bright center is the main mobile view. The full width is preserved for wider screens.'**
  String get settingsCoverSafeArea;

  /// No description provided for @settingsCropHint.
  ///
  /// In en, this message translates to:
  /// **'Drag to reposition. Pinch to zoom.'**
  String get settingsCropHint;

  /// No description provided for @settingsCropPreview.
  ///
  /// In en, this message translates to:
  /// **'Image crop preview'**
  String get settingsCropPreview;

  /// No description provided for @settingsCropZoom.
  ///
  /// In en, this message translates to:
  /// **'Zoom'**
  String get settingsCropZoom;

  /// No description provided for @settingsCropReset.
  ///
  /// In en, this message translates to:
  /// **'Reset position'**
  String get settingsCropReset;

  /// No description provided for @settingsImageSaved.
  ///
  /// In en, this message translates to:
  /// **'Image updated'**
  String get settingsImageSaved;

  /// No description provided for @settingsImageTooLarge.
  ///
  /// In en, this message translates to:
  /// **'Image must be no larger than {maxMb} MB'**
  String settingsImageTooLarge(int maxMb);

  /// No description provided for @settingsOAuthUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Not enabled on this site'**
  String get settingsOAuthUnavailable;

  /// No description provided for @settingsOAuthOpenBrowser.
  ///
  /// In en, this message translates to:
  /// **'Manage connections in browser'**
  String get settingsOAuthOpenBrowser;

  /// No description provided for @settingsOAuthBrowserHint.
  ///
  /// In en, this message translates to:
  /// **'Sign in as @{username} in the browser to connect an account. Connections refresh when you return to the app.'**
  String settingsOAuthBrowserHint(String username);

  /// No description provided for @commonRefresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get commonRefresh;

  /// No description provided for @topicEarliest.
  ///
  /// In en, this message translates to:
  /// **'Earliest'**
  String get topicEarliest;

  /// No description provided for @topicLatest.
  ///
  /// In en, this message translates to:
  /// **'Latest'**
  String get topicLatest;

  /// No description provided for @topicEarlierReplies.
  ///
  /// In en, this message translates to:
  /// **'Load earlier replies'**
  String get topicEarlierReplies;

  /// No description provided for @topicHistory.
  ///
  /// In en, this message translates to:
  /// **'Revision history'**
  String get topicHistory;

  /// No description provided for @topicHistoryUnavailable.
  ///
  /// In en, this message translates to:
  /// **'This version is unavailable'**
  String get topicHistoryUnavailable;

  /// No description provided for @topicHistoryEmpty.
  ///
  /// In en, this message translates to:
  /// **'No revisions yet'**
  String get topicHistoryEmpty;

  /// No description provided for @topicDeleteConfirm.
  ///
  /// In en, this message translates to:
  /// **'Delete this content? Recoverable items can be found in the recycle bin.'**
  String get topicDeleteConfirm;

  /// No description provided for @topicModerateBan.
  ///
  /// In en, this message translates to:
  /// **'Hide content'**
  String get topicModerateBan;

  /// No description provided for @topicModerateUnban.
  ///
  /// In en, this message translates to:
  /// **'Restore visibility'**
  String get topicModerateUnban;

  /// No description provided for @topicModerateConfirm.
  ///
  /// In en, this message translates to:
  /// **'Change the visibility of this content?'**
  String get topicModerateConfirm;

  /// No description provided for @topicShare.
  ///
  /// In en, this message translates to:
  /// **'Share'**
  String get topicShare;

  /// No description provided for @topicEditReply.
  ///
  /// In en, this message translates to:
  /// **'Edit reply'**
  String get topicEditReply;

  /// No description provided for @topicRemoved.
  ///
  /// In en, this message translates to:
  /// **'This content has been deleted or removed'**
  String get topicRemoved;

  /// No description provided for @topicBookmark.
  ///
  /// In en, this message translates to:
  /// **'Bookmark'**
  String get topicBookmark;

  /// No description provided for @topicBookmarked.
  ///
  /// In en, this message translates to:
  /// **'Remove bookmark'**
  String get topicBookmarked;

  /// No description provided for @topicLike.
  ///
  /// In en, this message translates to:
  /// **'Like'**
  String get topicLike;

  /// No description provided for @authEmailPrefix.
  ///
  /// In en, this message translates to:
  /// **'Email username'**
  String get authEmailPrefix;

  /// No description provided for @authEmailDomain.
  ///
  /// In en, this message translates to:
  /// **'Email domain'**
  String get authEmailDomain;

  /// No description provided for @authAgreePolicies.
  ///
  /// In en, this message translates to:
  /// **'I have read and agree to the published policies'**
  String get authAgreePolicies;

  /// No description provided for @authPasswordMismatch.
  ///
  /// In en, this message translates to:
  /// **'The passwords do not match'**
  String get authPasswordMismatch;

  /// No description provided for @schedulerWebTitle.
  ///
  /// In en, this message translates to:
  /// **'Explore the full scheduler on the Web'**
  String get schedulerWebTitle;

  /// No description provided for @schedulerWebAction.
  ///
  /// In en, this message translates to:
  /// **'Open f.yourtj.de'**
  String get schedulerWebAction;

  /// No description provided for @schedulerPlanDisclaimer.
  ///
  /// In en, this message translates to:
  /// **'This is a course plan. Final enrolment is determined by the university.'**
  String get schedulerPlanDisclaimer;

  /// No description provided for @campusExploreCourses.
  ///
  /// In en, this message translates to:
  /// **'Find your next course through student reviews'**
  String get campusExploreCourses;

  /// No description provided for @campusPlanTitle.
  ///
  /// In en, this message translates to:
  /// **'Turn your course shortlist into a plan'**
  String get campusPlanTitle;

  /// No description provided for @campusPlanDescription.
  ///
  /// In en, this message translates to:
  /// **'Compare offerings and check conflicts before choosing.'**
  String get campusPlanDescription;

  /// No description provided for @loginGoogle.
  ///
  /// In en, this message translates to:
  /// **'Continue with Google'**
  String get loginGoogle;

  /// No description provided for @loginGithub.
  ///
  /// In en, this message translates to:
  /// **'Continue with GitHub'**
  String get loginGithub;

  /// No description provided for @wikiSearchUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Search is unavailable. You can still browse the directory.'**
  String get wikiSearchUnavailable;

  /// No description provided for @wikiSearchHint.
  ///
  /// In en, this message translates to:
  /// **'Search campus knowledge'**
  String get wikiSearchHint;

  /// No description provided for @wikiExploreTitle.
  ///
  /// In en, this message translates to:
  /// **'Your campus companion'**
  String get wikiExploreTitle;

  /// No description provided for @notificationComment.
  ///
  /// In en, this message translates to:
  /// **'{actor} commented on your topic'**
  String notificationComment(String actor);

  /// No description provided for @notificationPostReply.
  ///
  /// In en, this message translates to:
  /// **'{actor} replied to you'**
  String notificationPostReply(String actor);

  /// No description provided for @notificationTopicPost.
  ///
  /// In en, this message translates to:
  /// **'{actor} posted in a topic you watch'**
  String notificationTopicPost(String actor);

  /// No description provided for @notificationFollow.
  ///
  /// In en, this message translates to:
  /// **'{actor} followed you'**
  String notificationFollow(String actor);

  /// No description provided for @notificationLike.
  ///
  /// In en, this message translates to:
  /// **'{actor} liked your reply'**
  String notificationLike(String actor);

  /// No description provided for @notificationWikiUpdated.
  ///
  /// In en, this message translates to:
  /// **'{actor} updated a wiki page you watch'**
  String notificationWikiUpdated(String actor);

  /// No description provided for @notificationBadge.
  ///
  /// In en, this message translates to:
  /// **'You earned the “{badge}” badge'**
  String notificationBadge(String badge);

  /// No description provided for @notificationNew.
  ///
  /// In en, this message translates to:
  /// **'New notification'**
  String get notificationNew;

  /// No description provided for @notificationSomeone.
  ///
  /// In en, this message translates to:
  /// **'Someone'**
  String get notificationSomeone;

  /// No description provided for @profileRoleAdmin.
  ///
  /// In en, this message translates to:
  /// **'Admin'**
  String get profileRoleAdmin;

  /// No description provided for @profileActionSignup.
  ///
  /// In en, this message translates to:
  /// **'Joined the community'**
  String get profileActionSignup;

  /// No description provided for @profileActionPost.
  ///
  /// In en, this message translates to:
  /// **'Published a topic'**
  String get profileActionPost;

  /// No description provided for @profileActionLike.
  ///
  /// In en, this message translates to:
  /// **'Liked'**
  String get profileActionLike;

  /// No description provided for @profileActionFollow.
  ///
  /// In en, this message translates to:
  /// **'Followed'**
  String get profileActionFollow;

  /// No description provided for @profileActionComment.
  ///
  /// In en, this message translates to:
  /// **'Replied'**
  String get profileActionComment;

  /// No description provided for @replyQuoteExpand.
  ///
  /// In en, this message translates to:
  /// **'Show full quote'**
  String get replyQuoteExpand;

  /// No description provided for @replyQuoteCollapse.
  ///
  /// In en, this message translates to:
  /// **'Collapse quote'**
  String get replyQuoteCollapse;

  /// No description provided for @notificationBadgeUnnamed.
  ///
  /// In en, this message translates to:
  /// **'You earned a new badge'**
  String get notificationBadgeUnnamed;

  /// No description provided for @settingsAppLanguage.
  ///
  /// In en, this message translates to:
  /// **'App language'**
  String get settingsAppLanguage;

  /// No description provided for @settingsLanguageSystem.
  ///
  /// In en, this message translates to:
  /// **'Follow system'**
  String get settingsLanguageSystem;

  /// No description provided for @scheduleGradeYear.
  ///
  /// In en, this message translates to:
  /// **'Class of {year}'**
  String scheduleGradeYear(String year);

  /// No description provided for @schedulePeriods.
  ///
  /// In en, this message translates to:
  /// **'Periods'**
  String get schedulePeriods;

  /// No description provided for @scheduleWeeksLabel.
  ///
  /// In en, this message translates to:
  /// **'Weeks'**
  String get scheduleWeeksLabel;

  /// No description provided for @schedulePeriodRange.
  ///
  /// In en, this message translates to:
  /// **'Periods {range}'**
  String schedulePeriodRange(String range);

  /// No description provided for @courseCopyCreditUnit.
  ///
  /// In en, this message translates to:
  /// **'Credits'**
  String get courseCopyCreditUnit;

  /// No description provided for @courseCopyNoTeacher.
  ///
  /// In en, this message translates to:
  /// **'No teacher'**
  String get courseCopyNoTeacher;

  /// No description provided for @courseCopyCatalogEmptyTitle.
  ///
  /// In en, this message translates to:
  /// **'No courses yet'**
  String get courseCopyCatalogEmptyTitle;

  /// No description provided for @courseCopyCatalogEmptyDescription.
  ///
  /// In en, this message translates to:
  /// **'The course catalog has not been imported yet.'**
  String get courseCopyCatalogEmptyDescription;

  /// No description provided for @courseCopyNoFilterResults.
  ///
  /// In en, this message translates to:
  /// **'No courses match these filters.'**
  String get courseCopyNoFilterResults;

  /// No description provided for @courseCopyNoFilterResultsDescription.
  ///
  /// In en, this message translates to:
  /// **'Try adjusting or clearing filters to see more courses.'**
  String get courseCopyNoFilterResultsDescription;

  /// No description provided for @courseCopyClearSearch.
  ///
  /// In en, this message translates to:
  /// **'Clear search'**
  String get courseCopyClearSearch;

  /// No description provided for @courseCopyDone.
  ///
  /// In en, this message translates to:
  /// **'Done'**
  String get courseCopyDone;

  /// No description provided for @courseCopyNoOptions.
  ///
  /// In en, this message translates to:
  /// **'No options available'**
  String get courseCopyNoOptions;

  /// No description provided for @courseCopySelectedCount.
  ///
  /// In en, this message translates to:
  /// **'{count} selected'**
  String courseCopySelectedCount(int count);

  /// No description provided for @courseCopyInstructorInputHint.
  ///
  /// In en, this message translates to:
  /// **'Enter an instructor name and press enter'**
  String get courseCopyInstructorInputHint;

  /// No description provided for @courseCopyInstructorAdd.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get courseCopyInstructorAdd;

  /// No description provided for @courseCopyInstructorEmptyHint.
  ///
  /// In en, this message translates to:
  /// **'Add instructors with the field above'**
  String get courseCopyInstructorEmptyHint;

  /// No description provided for @courseCopyAliasesLabel.
  ///
  /// In en, this message translates to:
  /// **'Aliases: '**
  String get courseCopyAliasesLabel;

  /// No description provided for @courseCopyLegacyNamesLabel.
  ///
  /// In en, this message translates to:
  /// **'Former name: '**
  String get courseCopyLegacyNamesLabel;

  /// No description provided for @courseCopyReviewScopeTeam.
  ///
  /// In en, this message translates to:
  /// **'Teaching Team'**
  String get courseCopyReviewScopeTeam;

  /// No description provided for @courseCopyReviewScopeCourse.
  ///
  /// In en, this message translates to:
  /// **'Course-level Reviews'**
  String get courseCopyReviewScopeCourse;

  /// No description provided for @courseCopyTeamInstructorsPrefix.
  ///
  /// In en, this message translates to:
  /// **'Teaching team · '**
  String get courseCopyTeamInstructorsPrefix;

  /// No description provided for @courseCopyTeamInstructorsSuffix.
  ///
  /// In en, this message translates to:
  /// **' ({count} teachers)'**
  String courseCopyTeamInstructorsSuffix(int count);

  /// No description provided for @courseCopyRatingTitle.
  ///
  /// In en, this message translates to:
  /// **'Course rating'**
  String get courseCopyRatingTitle;

  /// No description provided for @courseCopyRatingOutOf.
  ///
  /// In en, this message translates to:
  /// **'/ 5.0'**
  String get courseCopyRatingOutOf;

  /// No description provided for @courseCopyNoRatingQuiet.
  ///
  /// In en, this message translates to:
  /// **'No rating yet'**
  String get courseCopyNoRatingQuiet;

  /// No description provided for @courseCopyOfferingsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No offerings yet.'**
  String get courseCopyOfferingsEmpty;

  /// No description provided for @courseCopyOfferingFocusLabel.
  ///
  /// In en, this message translates to:
  /// **'Showing reviews for this class only'**
  String get courseCopyOfferingFocusLabel;

  /// No description provided for @courseCopyOfferingFocusClear.
  ///
  /// In en, this message translates to:
  /// **'Show all reviews'**
  String get courseCopyOfferingFocusClear;

  /// No description provided for @courseCopySummaryGenerated.
  ///
  /// In en, this message translates to:
  /// **'Generated'**
  String get courseCopySummaryGenerated;

  /// No description provided for @courseCopySummaryKeywords.
  ///
  /// In en, this message translates to:
  /// **'Keywords'**
  String get courseCopySummaryKeywords;

  /// No description provided for @courseCopySummaryPros.
  ///
  /// In en, this message translates to:
  /// **'Pros'**
  String get courseCopySummaryPros;

  /// No description provided for @courseCopySummaryCons.
  ///
  /// In en, this message translates to:
  /// **'Cons'**
  String get courseCopySummaryCons;

  /// No description provided for @courseCopySummaryRepresentativeReviews.
  ///
  /// In en, this message translates to:
  /// **'Representative reviews'**
  String get courseCopySummaryRepresentativeReviews;

  /// No description provided for @courseCopySummarySentimentPositive.
  ///
  /// In en, this message translates to:
  /// **'Positive'**
  String get courseCopySummarySentimentPositive;

  /// No description provided for @courseCopySummarySentimentNeutral.
  ///
  /// In en, this message translates to:
  /// **'Neutral'**
  String get courseCopySummarySentimentNeutral;

  /// No description provided for @courseCopySummarySentimentNegative.
  ///
  /// In en, this message translates to:
  /// **'Negative'**
  String get courseCopySummarySentimentNegative;

  /// No description provided for @courseCopySummaryRefresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get courseCopySummaryRefresh;

  /// No description provided for @courseCopySummaryExpand.
  ///
  /// In en, this message translates to:
  /// **'Expand'**
  String get courseCopySummaryExpand;

  /// No description provided for @courseCopySummaryCollapse.
  ///
  /// In en, this message translates to:
  /// **'Collapse'**
  String get courseCopySummaryCollapse;

  /// No description provided for @courseCopySummaryDisclaimer.
  ///
  /// In en, this message translates to:
  /// **'Generated by AI from student reviews. For reference only; not a course recommendation.'**
  String get courseCopySummaryDisclaimer;

  /// No description provided for @courseCopySummaryInsufficient.
  ///
  /// In en, this message translates to:
  /// **'Not enough reviews to generate an AI summary yet.'**
  String get courseCopySummaryInsufficient;

  /// No description provided for @courseCopySummaryLoadFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to generate the AI summary. Please try again later.'**
  String get courseCopySummaryLoadFailed;

  /// No description provided for @courseCopyWriteReviewTitle.
  ///
  /// In en, this message translates to:
  /// **'Write a review'**
  String get courseCopyWriteReviewTitle;

  /// No description provided for @courseCopyEditReviewTitle.
  ///
  /// In en, this message translates to:
  /// **'Edit review'**
  String get courseCopyEditReviewTitle;

  /// No description provided for @courseCopySelectOffering.
  ///
  /// In en, this message translates to:
  /// **'Select offering'**
  String get courseCopySelectOffering;

  /// No description provided for @courseCopyRatingLabel.
  ///
  /// In en, this message translates to:
  /// **'Rating'**
  String get courseCopyRatingLabel;

  /// No description provided for @courseCopyContentLabel.
  ///
  /// In en, this message translates to:
  /// **'Review'**
  String get courseCopyContentLabel;

  /// No description provided for @courseCopyContentPlaceholder.
  ///
  /// In en, this message translates to:
  /// **'Share your experience with the course, teaching quality, or grading…'**
  String get courseCopyContentPlaceholder;

  /// No description provided for @courseCopyRatingRequired.
  ///
  /// In en, this message translates to:
  /// **'Please choose a 1–5 star rating.'**
  String get courseCopyRatingRequired;

  /// No description provided for @courseCopyContentRequired.
  ///
  /// In en, this message translates to:
  /// **'Review content cannot be empty.'**
  String get courseCopyContentRequired;

  /// No description provided for @courseCopyAnonymousLabel.
  ///
  /// In en, this message translates to:
  /// **'Post anonymously (identity hidden from the public)'**
  String get courseCopyAnonymousLabel;

  /// No description provided for @courseCopySubmitSuccess.
  ///
  /// In en, this message translates to:
  /// **'Submitted'**
  String get courseCopySubmitSuccess;

  /// No description provided for @courseCopyUpdateSuccess.
  ///
  /// In en, this message translates to:
  /// **'Updated'**
  String get courseCopyUpdateSuccess;

  /// No description provided for @courseCopyDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get courseCopyDelete;

  /// No description provided for @courseCopyDeleteReviewTitle.
  ///
  /// In en, this message translates to:
  /// **'Delete review'**
  String get courseCopyDeleteReviewTitle;

  /// No description provided for @courseCopyConfirmDeleteReview.
  ///
  /// In en, this message translates to:
  /// **'Delete this review? This cannot be undone.'**
  String get courseCopyConfirmDeleteReview;

  /// No description provided for @courseCopyReviewDeleted.
  ///
  /// In en, this message translates to:
  /// **'Review deleted'**
  String get courseCopyReviewDeleted;

  /// No description provided for @courseCopyOperationFailed.
  ///
  /// In en, this message translates to:
  /// **'Operation failed. Please try again later.'**
  String get courseCopyOperationFailed;

  /// No description provided for @courseCopyReviewsLoadFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to load reviews. Please try again later.'**
  String get courseCopyReviewsLoadFailed;

  /// No description provided for @courseCopyAuthorAnonymousLabel.
  ///
  /// In en, this message translates to:
  /// **'Anonymous'**
  String get courseCopyAuthorAnonymousLabel;

  /// No description provided for @courseCopyAuthorLegacyLabel.
  ///
  /// In en, this message translates to:
  /// **'Legacy anonymous review'**
  String get courseCopyAuthorLegacyLabel;

  /// No description provided for @courseCopyRelatedTeacherCoursesTitle.
  ///
  /// In en, this message translates to:
  /// **'Other courses by the same teachers'**
  String get courseCopyRelatedTeacherCoursesTitle;

  /// No description provided for @courseCopyRelatedOtherTeachersTitle.
  ///
  /// In en, this message translates to:
  /// **'Other teachers of this course'**
  String get courseCopyRelatedOtherTeachersTitle;

  /// No description provided for @courseCopyRelatedEmpty.
  ///
  /// In en, this message translates to:
  /// **'No related content'**
  String get courseCopyRelatedEmpty;

  /// No description provided for @courseCopyRelationEquivalent.
  ///
  /// In en, this message translates to:
  /// **'Equivalent'**
  String get courseCopyRelationEquivalent;

  /// No description provided for @courseCopyRelationRenamed.
  ///
  /// In en, this message translates to:
  /// **'Renamed'**
  String get courseCopyRelationRenamed;

  /// No description provided for @courseCopyRelationSplit.
  ///
  /// In en, this message translates to:
  /// **'Split'**
  String get courseCopyRelationSplit;

  /// No description provided for @courseCopyRelationMerged.
  ///
  /// In en, this message translates to:
  /// **'Merged'**
  String get courseCopyRelationMerged;

  /// No description provided for @courseCopyRelationRelated.
  ///
  /// In en, this message translates to:
  /// **'Related'**
  String get courseCopyRelationRelated;

  /// No description provided for @courseCopySummaryConsensusStrongRecommend.
  ///
  /// In en, this message translates to:
  /// **'Strongly recommended'**
  String get courseCopySummaryConsensusStrongRecommend;

  /// No description provided for @courseCopySummaryConsensusRecommend.
  ///
  /// In en, this message translates to:
  /// **'Recommended'**
  String get courseCopySummaryConsensusRecommend;

  /// No description provided for @courseCopySummaryConsensusNeutral.
  ///
  /// In en, this message translates to:
  /// **'Mixed'**
  String get courseCopySummaryConsensusNeutral;

  /// No description provided for @courseCopySummaryConsensusCautious.
  ///
  /// In en, this message translates to:
  /// **'Caution advised'**
  String get courseCopySummaryConsensusCautious;

  /// No description provided for @courseCopySummaryConsensusNotRecommend.
  ///
  /// In en, this message translates to:
  /// **'Not recommended'**
  String get courseCopySummaryConsensusNotRecommend;

  /// No description provided for @courseCopySummaryConsensusTextStrongRecommend.
  ///
  /// In en, this message translates to:
  /// **'Most students strongly recommend this course.'**
  String get courseCopySummaryConsensusTextStrongRecommend;

  /// No description provided for @courseCopySummaryConsensusTextRecommend.
  ///
  /// In en, this message translates to:
  /// **'Most students recommend this course.'**
  String get courseCopySummaryConsensusTextRecommend;

  /// No description provided for @courseCopySummaryConsensusTextNeutral.
  ///
  /// In en, this message translates to:
  /// **'Students have mixed opinions about this course.'**
  String get courseCopySummaryConsensusTextNeutral;

  /// No description provided for @courseCopySummaryConsensusTextCautious.
  ///
  /// In en, this message translates to:
  /// **'Most students advise caution before choosing this course.'**
  String get courseCopySummaryConsensusTextCautious;

  /// No description provided for @courseCopySummaryConsensusTextNotRecommend.
  ///
  /// In en, this message translates to:
  /// **'Most students do not recommend this course.'**
  String get courseCopySummaryConsensusTextNotRecommend;

  /// No description provided for @topicJoinDiscussion.
  ///
  /// In en, this message translates to:
  /// **'Join discussion'**
  String get topicJoinDiscussion;

  /// No description provided for @updateCheck.
  ///
  /// In en, this message translates to:
  /// **'Check for updates'**
  String get updateCheck;

  /// No description provided for @updateAvailable.
  ///
  /// In en, this message translates to:
  /// **'Update available'**
  String get updateAvailable;

  /// No description provided for @updateLatest.
  ///
  /// In en, this message translates to:
  /// **'You’re up to date'**
  String get updateLatest;

  /// No description provided for @updateFailed.
  ///
  /// In en, this message translates to:
  /// **'Unable to check or download updates. Please try again later.'**
  String get updateFailed;

  /// No description provided for @updateDownload.
  ///
  /// In en, this message translates to:
  /// **'Download update'**
  String get updateDownload;

  /// No description provided for @updateSkip.
  ///
  /// In en, this message translates to:
  /// **'Skip this version'**
  String get updateSkip;

  /// No description provided for @updateLater.
  ///
  /// In en, this message translates to:
  /// **'Later'**
  String get updateLater;

  /// No description provided for @updatePreparing.
  ///
  /// In en, this message translates to:
  /// **'Choosing the fastest download source…'**
  String get updatePreparing;

  /// No description provided for @updateDownloading.
  ///
  /// In en, this message translates to:
  /// **'Downloading…'**
  String get updateDownloading;

  /// No description provided for @updateReady.
  ///
  /// In en, this message translates to:
  /// **'Update verified and ready to install.'**
  String get updateReady;

  /// No description provided for @updateInstall.
  ///
  /// In en, this message translates to:
  /// **'Install update'**
  String get updateInstall;

  /// No description provided for @updatePermission.
  ///
  /// In en, this message translates to:
  /// **'Allow YourTJ to install apps, then return and tap Install again.'**
  String get updatePermission;

  /// No description provided for @updateRetry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get updateRetry;
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
      <String>['de', 'en', 'ja', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'de':
      return AppLocalizationsDe();
    case 'en':
      return AppLocalizationsEn();
    case 'ja':
      return AppLocalizationsJa();
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
