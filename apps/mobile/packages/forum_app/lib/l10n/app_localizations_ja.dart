// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Japanese (`ja`).
class AppLocalizationsJa extends AppLocalizations {
  AppLocalizationsJa([String locale = 'ja']) : super(locale);

  @override
  String get myCourseReviewsTitle => '自分の授業評価';

  @override
  String get myCourseReviewsEmpty => 'まだ評価がありません。科目一覧から探してみましょう。';

  @override
  String get myCourseReviewHidden => 'この評価は非表示です。ここで削除できます。';

  @override
  String get myCourseReviewUnavailable => 'この科目は現在表示できませんが、自分の評価は管理できます。';

  @override
  String get myCourseReviewOpen => '詳細を見る';

  @override
  String get courseReviewNetworkError =>
      '接続できませんでした。通信環境を確認して再試行してください。評価内容は保持されています。';

  @override
  String get courseReviewUnknownError =>
      '評価を保存できませんでした。サーバーから具体的な理由が返されませんでした。後でもう一度お試しください。';

  @override
  String get commonHideKeyboard => 'キーボードを閉じる';

  @override
  String get appTitle => 'YourTJ';

  @override
  String get navHome => 'ホームへ';

  @override
  String get navSearch => '検索';

  @override
  String get navPublish => '投稿';

  @override
  String get navMessages => 'メッセージ';

  @override
  String get navProfile => 'マイページ';

  @override
  String get commonCancel => 'キャンセル';

  @override
  String get commonSave => '保存';

  @override
  String get commonLoading => '読み込み中…';

  @override
  String get commonLoadMore => 'さらに読み込む';

  @override
  String get commonRetry => '再試行';

  @override
  String get commonClose => '閉じる';

  @override
  String get commonSend => '送信';

  @override
  String get commonSearch => '検索';

  @override
  String get commonEdit => '編集';

  @override
  String get commonCurrent => '現在';

  @override
  String get commonEmpty => 'トピックはありません';

  @override
  String get commonBack => '戻る';

  @override
  String get commonBackToTop => 'トップへ戻る';

  @override
  String get commonUseLightTheme => 'ライトモードにする';

  @override
  String get commonUseDarkTheme => 'ダークモードにする';

  @override
  String get timeAgoJustNow => 'たった今';

  @override
  String timeAgoMinutes(int count) {
    return '$count 分前';
  }

  @override
  String timeAgoHours(int count) {
    return '$count 時間前';
  }

  @override
  String timeAgoDays(int count) {
    return '$count 日前';
  }

  @override
  String timeAgoWeeks(int count) {
    return '$count週間前';
  }

  @override
  String timeAgoMonths(int count) {
    return '$countか月前';
  }

  @override
  String timeAgoYears(int count) {
    return '$count年前';
  }

  @override
  String get authLoginTitle => 'アカウントにログイン';

  @override
  String get authRegisterTitle => 'アカウント作成';

  @override
  String get authForgotTitle => 'パスワード再設定';

  @override
  String get authLoginSubtitle => 'おかえりなさい。議論と投稿を続けましょう。';

  @override
  String get authRegisterSubtitle => 'アカウントを作成して、キャンパスの会話に参加しましょう。';

  @override
  String get authForgotSubtitle => 'メールアドレスを入力すると、再設定メールを送信します。';

  @override
  String get authUsernameOrEmail => 'ユーザー名またはメール';

  @override
  String get authUsername => 'ユーザー名';

  @override
  String get authEmail => 'メール';

  @override
  String get authPassword => 'パスワード';

  @override
  String get authNewPassword => '新しいパスワード';

  @override
  String get authConfirmPassword => 'パスワード確認';

  @override
  String get authCaptcha => '認証コード';

  @override
  String get authForgotPassword => 'パスワードを忘れましたか？';

  @override
  String get authCreateAccount => 'アカウント作成';

  @override
  String get authSendResetEmail => '再設定メールを送信';

  @override
  String get authBackToLogin => 'ログインへ戻る';

  @override
  String get authTwoFactorTitle => '二段階認証';

  @override
  String get authTwoFactorCode => 'TOTPコード';

  @override
  String get authVerify => '確認';

  @override
  String get authGetCode => 'コードを取得';

  @override
  String get authOidcLogin => 'YourTJでログイン';

  @override
  String get authRegisterSuccess => '登録しました。ログインしてください';

  @override
  String get authResetEmailSent => '再設定メールを送信しました。受信箱をご確認ください';

  @override
  String get authLoading => '処理中…';

  @override
  String get authCacheClearFailed => '前のアカウントのオフラインデータを削除できませんでした。再試行してください。';

  @override
  String get authSessionSaveFailed => '新しいセッションを安全に保存できませんでした。再試行してください。';

  @override
  String get loginWelcome => 'YourTJへようこそ';

  @override
  String get loginModeLogin => 'ログイン';

  @override
  String get loginModeRegister => '登録';

  @override
  String get loginModeForgot => 'パスワードを忘れた場合';

  @override
  String get publishTitle => 'トピックを投稿';

  @override
  String get publishEditTitle => 'トピックを編集';

  @override
  String get publishPublish => '投稿';

  @override
  String get publishSaveDraft => '下書きを保存';

  @override
  String get publishTitleField => 'タイトル';

  @override
  String get publishTitleHint => 'タイトルを入力（5～100文字）';

  @override
  String get publishBodyPlaceholder => '本文…';

  @override
  String get publishTitleRequired => 'タイトルを入力してください';

  @override
  String get publishContentRequired => '本文を入力してください';

  @override
  String get publishSuccess => '投稿しました';

  @override
  String get publishSavedDraft => '下書きに保存しました';

  @override
  String publishFailed(String error) {
    return '投稿できませんでした：$error';
  }

  @override
  String publishImageFailed(String error) {
    return '画像をアップロードできませんでした：$error';
  }

  @override
  String get composePreview => 'プレビュー';

  @override
  String get composeEdit => '編集';

  @override
  String get publishBodyField => '本文';

  @override
  String get publishCategoryRequired => 'カテゴリーを少なくとも1つ選択してください';

  @override
  String get publishPreviewEmpty => '入力すると、書式付きのプレビューがここに表示されます';

  @override
  String publishLoadFailed(String error) {
    return '編集データを読み込めませんでした：$error';
  }

  @override
  String get publishToolBold => '太字';

  @override
  String get publishToolItalic => '斜体';

  @override
  String get publishToolStrike => '取り消し線';

  @override
  String get publishToolQuote => '引用';

  @override
  String get publishToolCode => 'インラインコード';

  @override
  String get publishToolBulletList => '箇条書き';

  @override
  String get publishToolOrderedList => '番号付きリスト';

  @override
  String get publishToolImage => '画像を追加';

  @override
  String get publishRemoveImage => '画像を削除';

  @override
  String topicReplyTarget(String name) {
    return '$nameに返信';
  }

  @override
  String get topicTitle => 'トピック';

  @override
  String get topicReply => '返信';

  @override
  String get topicReplySuccess => '返信しました';

  @override
  String topicReplyFailed(String error) {
    return '返信できませんでした：$error';
  }

  @override
  String get topicReplyHint => 'コメントを書く…';

  @override
  String get topicReplyTargetUnavailable => '元の返信は表示できません';

  @override
  String get topicReplying => '返信中…（タップでキャンセル）';

  @override
  String get topicReport => '投稿を報告';

  @override
  String get topicReportHint => '理由を入力してください';

  @override
  String get topicReportSubmit => '送信';

  @override
  String get topicReportSubmitted => '報告を送信しました';

  @override
  String topicReportFailed(String error) {
    return '報告できませんでした：$error';
  }

  @override
  String get topicWatch => 'このトピックの返信を通知';

  @override
  String get topicUnwatch => 'このトピックの通知を解除';

  @override
  String topicReplies(int count) {
    return '$count 件の返信';
  }

  @override
  String get topicNoTitle => '無題';

  @override
  String get profileTitle => 'プロフィール';

  @override
  String get profileFollow => 'フォロー';

  @override
  String get profileFollowing => 'フォロー中';

  @override
  String get profileTopics => 'トピック';

  @override
  String get profileReplies => '返信';

  @override
  String get profileLikes => 'いいね';

  @override
  String get profileFollowers => 'フォロワー';

  @override
  String get profileFollowingCount => 'フォロー中';

  @override
  String get profileBadges => 'バッジ';

  @override
  String get profileNoBadges => 'バッジはありません';

  @override
  String get profileEmptyActivity => '操作ログはありません';

  @override
  String get profileEmptyTopics => 'まだトピックはありません';

  @override
  String get profileEmptyLikes => 'まだ「いいね」はありません';

  @override
  String get profileEmptyBookmarks => 'まだブックマークはありません';

  @override
  String get profileEmptyFollowing => 'まだ誰もフォローしていません';

  @override
  String get profileEmptyFollowers => 'まだフォロワーはいません';

  @override
  String get profileNotLoggedIn => 'ログインしていません';

  @override
  String get messagesTitle => 'メッセージ';

  @override
  String get messagesEmpty => 'まだ会話はありません';

  @override
  String get messagesEmptyDescription => 'コミュニティのメンバーと会話を始めましょう。';

  @override
  String get messagesSearchConversations => '会話を検索';

  @override
  String get messagesConversation => 'プライベート会話';

  @override
  String get messagesStartChat => 'チャットを開始';

  @override
  String messagesFirstMessageTo(String user) {
    return '$user さんに最初のメッセージを送ります。';
  }

  @override
  String get messagesNoMessagesYet => 'まだメッセージはありません';

  @override
  String get messagesEmptyDetail => 'まだメッセージはありません。挨拶してみましょう！';

  @override
  String get messagesInputHint => 'メッセージを入力…';

  @override
  String messagesSendFailed(String error) {
    return '送信できませんでした：$error';
  }

  @override
  String get notificationsTitle => '通知';

  @override
  String get notificationsEmpty => '通知はありません';

  @override
  String get notificationsMarkAllRead => 'すべて既読';

  @override
  String get notificationsAll => 'すべて';

  @override
  String get notificationsUnread => '未読';

  @override
  String get searchTitle => '検索';

  @override
  String get searchHint => 'トピック、ユーザー、カテゴリを検索…';

  @override
  String get searchEmpty => 'キーワードを入力して検索';

  @override
  String get searchNoUsers => '該当するユーザーはいません';

  @override
  String get searchNoCategories => '該当するカテゴリはありません';

  @override
  String get searchUnavailable => '検索サービスを利用できません';

  @override
  String get searchAll => 'すべて';

  @override
  String get searchTopics => 'トピック';

  @override
  String get searchUsers => 'ユーザー';

  @override
  String get searchCategories => 'カテゴリ';

  @override
  String get categoryTitle => 'カテゴリ';

  @override
  String get settingsTitle => '設定';

  @override
  String get settingsTabProfile => 'プロフィール';

  @override
  String get settingsTabAccount => 'アカウント';

  @override
  String get settingsTabPrivacy => 'プライバシー';

  @override
  String get settingsTabBinding => '連携';

  @override
  String get settingsTabSecurity => 'セキュリティ';

  @override
  String get settingsSectionProfile => 'プロフィール情報';

  @override
  String get settingsNickname => 'ニックネーム';

  @override
  String get settingsNicknameEdit => '表示名を編集';

  @override
  String get settingsBio => '自己紹介';

  @override
  String get settingsBioEdit => '自己紹介と署名を編集';

  @override
  String get settingsAvatar => 'アバター';

  @override
  String get settingsAvatarUpload => 'アバターをアップロード（WebPに変換）';

  @override
  String get settingsAvatarUploading => 'アップロード中…';

  @override
  String settingsAvatarUploadFailed(String error) {
    return 'アバターをアップロードできませんでした：$error';
  }

  @override
  String get settingsEmail => 'メール';

  @override
  String get settingsEmailEdit => '登録メールアドレスを変更';

  @override
  String get settingsEmailUpdated => 'メールアドレスを更新しました。確認を完了してください';

  @override
  String get settingsEmailOAuthReauthRequired =>
      'この OAuth 専用アカウントは、メールアドレスを設定する前に OAuth で再認証する必要があります。必要に応じて管理者に連絡してください。';

  @override
  String settingsEmailFailed(String error) {
    return 'メールアドレスを更新できませんでした：$error';
  }

  @override
  String get settingsNewEmail => '新しいメールアドレス';

  @override
  String get settingsChangePassword => 'パスワードを変更';

  @override
  String get settingsChangePasswordSub => 'ログインパスワードを変更';

  @override
  String get settingsCurrentPassword => '現在のパスワード';

  @override
  String get settingsPasswordUpdated => 'パスワードを更新しました';

  @override
  String settingsPasswordFailed(String error) {
    return 'パスワードを変更できませんでした：$error';
  }

  @override
  String get settingsBadge => 'バッジ';

  @override
  String get settingsBadgeNone => '未着用';

  @override
  String settingsBadgeCurrent(String name) {
    return '着用中：$name';
  }

  @override
  String get settingsBadgePick => '着用するバッジを選択';

  @override
  String get settingsBadgeNoOptions => '着用できるバッジはありません';

  @override
  String get settingsBadgeUpdated => 'バッジを更新しました';

  @override
  String settingsBadgeFailed(String error) {
    return 'バッジを更新できませんでした：$error';
  }

  @override
  String get settingsOAuth => '外部アカウント連携';

  @override
  String get settingsOAuthSub => 'GitHubとGoogleのログイン連携';

  @override
  String get settingsOAuthManage => 'アカウント連携を管理';

  @override
  String get settingsOAuthBindings => 'アカウント連携';

  @override
  String get settingsBound => '連携済み';

  @override
  String get settingsUnbound => '未連携';

  @override
  String get settingsUnbind => '連携を解除';

  @override
  String get settingsUnboundDone => '連携を解除しました';

  @override
  String settingsUnbindFailed(String error) {
    return '連携を解除できませんでした：$error';
  }

  @override
  String settingsLoadBindingsFailed(String error) {
    return '連携情報を読み込めませんでした：$error';
  }

  @override
  String get settingsPrivacyDirect => '友達のみにメッセージを公開';

  @override
  String get settingsPrivacyLikes => '自分の「いいね」を公開';

  @override
  String get settingsSessions => 'セッション管理';

  @override
  String get settingsSessionsEmpty => '会話はありません';

  @override
  String get settingsRevokeAll => 'すべてのセッションを終了';

  @override
  String get settingsRevoked => 'セッションを失効しました';

  @override
  String settingsRevokeFailed(String error) {
    return 'セッションを終了できませんでした：$error';
  }

  @override
  String get settingsRevokeAllDone => 'すべてのセッションを終了しました';

  @override
  String settingsOpFailed(String error) {
    return '操作に失敗しました：$error';
  }

  @override
  String get settingsAppearance => '外観';

  @override
  String get settingsDarkMode => 'ダークモード';

  @override
  String get settingsDarkCurrent => '現在：ダーク';

  @override
  String get settingsLightCurrent => '現在：ライト';

  @override
  String get settingsAbout => '情報';

  @override
  String get settingsAboutVersion => 'v0.1.0 · 同済大学キャンパスフォーラム';

  @override
  String get settingsEditProfile => 'プロフィール編集';

  @override
  String get settingsSignature => '署名';

  @override
  String get settingsSaveInfo => '保存';

  @override
  String get settingsInfoSaved => 'プロフィールを更新しました';

  @override
  String settingsInfoFailed(String error) {
    return 'プロフィールを更新できませんでした：$error';
  }

  @override
  String get settingsUserDataLoading => 'アカウント情報を読み込み中…';

  @override
  String get settingsFillComplete => 'すべての項目を入力してください';

  @override
  String get settingsSecondPhase => '第2フェーズで対応予定';

  @override
  String get settingsTotpTitle => '2段階認証（TOTP）';

  @override
  String get settingsTotpEnable => '有効化';

  @override
  String get settingsTotpDisable => '無効にする';

  @override
  String get settingsTotpPasswordPrompt => 'TOTPの管理にはパスワードを入力してください';

  @override
  String get settingsTotpSetupSecret => '認証アプリでスキャンするか、秘密鍵を入力してください';

  @override
  String get settingsTotpCode => '6桁のコードを入力';

  @override
  String get settingsTotpRecoveryCodes => 'リカバリーコード（安全な場所に保存してください）：';

  @override
  String get settingsTotpEnabled => 'TOTPを有効にしました';

  @override
  String get settingsTotpDisabled => 'TOTPを無効にしました';

  @override
  String settingsTotpFailed(String error) {
    return 'TOTPの操作に失敗しました：$error';
  }

  @override
  String get settingsTotpDisableTitle => 'TOTPを無効にする';

  @override
  String get settingsTotpEnableTitle => 'TOTPを有効にする';

  @override
  String get settingsTotpPassword => 'パスワード';

  @override
  String get settingsTotpNext => '次へ';

  @override
  String get settingsTotpScanSecret => 'スキャンするか秘密鍵を入力';

  @override
  String get settingsTotpDone => '完了';

  @override
  String get settingsTotpUnavailable => 'TOTPを利用できません';

  @override
  String get draftsTitle => '下書き';

  @override
  String get draftsEmpty => '下書きはありません';

  @override
  String get draftsNew => '下書きを作成';

  @override
  String get draftsBlocked => 'ブロック済み';

  @override
  String draftsMetaCreated(Object date) {
    return '$dateに作成';
  }

  @override
  String draftsMetaViews(Object count) {
    return '$count回閲覧';
  }

  @override
  String draftsMetaReplies(Object count) {
    return '$count 件の返信';
  }

  @override
  String get messagesNew => '新規メッセージ';

  @override
  String get messagesSearchUsers => 'ユーザーを検索';

  @override
  String get messagesNoContactableUsers => 'メッセージを送信できるユーザーはいません';

  @override
  String get settingsLogout => 'ログアウト';

  @override
  String get settingsLogoutConfirm => 'YourTJからログアウトしますか？';

  @override
  String get commonParseFailed => 'ページデータを解析できませんでした';

  @override
  String get commonLoadFailed => '読み込みに失敗しました';

  @override
  String get topicEmpty => 'まだトピックはありません';

  @override
  String settingsAvatarUploaded(String url) {
    return 'アバターをアップロードしました：$url';
  }

  @override
  String get settingsImageDecodeFailed => '画像を読み込めませんでした';

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
    return '$floor階へ移動しました';
  }

  @override
  String get sortLatest => '最新';

  @override
  String get sortHot => '人気';

  @override
  String get sortPopular => '注目';

  @override
  String get topicFeedModeList => 'リスト';

  @override
  String get topicFeedModeCard => 'カード';

  @override
  String get topicNewTopic => '新規トピック';

  @override
  String get scheduleTitle => '時間割プランナー';

  @override
  String get scheduleTabTimetable => 'プランのプレビュー';

  @override
  String get scheduleTabPick => '授業を選ぶ';

  @override
  String get scheduleTerm => '学期';

  @override
  String get scheduleGrade => '学年';

  @override
  String get scheduleMajor => '専攻';

  @override
  String get scheduleSyncLatest => '最新に同期';

  @override
  String get scheduleDataOutdated => '授業データが更新されました。タップで同期';

  @override
  String scheduleSyncedTo(String date) {
    return '$dateまで同期済み';
  }

  @override
  String get scheduleSyncConflictTitle => 'プラン同期の競合';

  @override
  String get scheduleSyncConflictBody => 'クラウドとローカルのプランが一致しません。残す方を選択してください。';

  @override
  String get scheduleSyncUseCloud => 'クラウドを使用';

  @override
  String get scheduleSyncKeepLocal => 'ローカルを保持';

  @override
  String get scheduleWeekAll => '全週';

  @override
  String scheduleWeekN(int week) {
    return '第$week週';
  }

  @override
  String get scheduleCurrentWeek => '現在の週';

  @override
  String get schedulePlanNew => '新しいプラン';

  @override
  String get schedulePlanRename => 'プラン名を変更';

  @override
  String get schedulePlanDelete => 'プランを削除';

  @override
  String get schedulePlanClear => '授業をクリア';

  @override
  String schedulePlanN(int n) {
    return 'プラン $n';
  }

  @override
  String scheduleStatsCourses(int count) {
    return '$count 件のコース';
  }

  @override
  String get scheduleStatsCredits => '単位';

  @override
  String get scheduleStatsHours => '合計時限';

  @override
  String get scheduleStatsConflicts => '重複';

  @override
  String get scheduleConflictBadge => '時間重複';

  @override
  String get scheduleAddCustomEvent => '予定枠を追加';

  @override
  String get scheduleCustomEventLabel => '予定あり';

  @override
  String get scheduleExportPng => '画像を出力';

  @override
  String get scheduleExportCsv => 'CSVに書き出す';

  @override
  String get scheduleDegraded => '簡易検索中です。結果が不完全な場合があります';

  @override
  String get scheduleNoReviewData => 'まだ評価はありません';

  @override
  String get schedulePickClass => 'クラスを選択';

  @override
  String get scheduleCompulsory => '必修';

  @override
  String get scheduleOptional => '選択';

  @override
  String get scheduleSearchHint => '授業名・コード・教員名で検索';

  @override
  String get scheduleStaged => '候補';

  @override
  String get scheduleSelected => '選択済み';

  @override
  String get scheduleRemoveCourse => '削除';

  @override
  String get scheduleParityOdd => '奇数週';

  @override
  String get scheduleParityEven => '偶数週';

  @override
  String scheduleWeeksN(String range) {
    return '第$range週';
  }

  @override
  String get scheduleDayMon => '月';

  @override
  String get scheduleDayTue => '火';

  @override
  String get scheduleDayWed => '水';

  @override
  String get scheduleDayThu => '木';

  @override
  String get scheduleDayFri => '金';

  @override
  String get scheduleDaySat => '土';

  @override
  String get scheduleDaySun => '日';

  @override
  String get scheduleMorning => '午前';

  @override
  String get scheduleAfternoon => '午後';

  @override
  String get scheduleEvening => '夜間';

  @override
  String get coursesTitle => 'コースカタログ';

  @override
  String get coursesSearchHint => '授業を検索';

  @override
  String get coursesFilterDepartment => '学部・学科';

  @override
  String get coursesFilterTerm => '学期';

  @override
  String get coursesFilterCampus => 'キャンパス';

  @override
  String get coursesFilterInstructor => '教員';

  @override
  String get coursesOnlyWithReviews => '評価のある授業のみ';

  @override
  String coursesRatingCount(int count) {
    return '$count件の評価';
  }

  @override
  String get coursesNoRating => '評価なし';

  @override
  String get courseDetailReviews => '評価';

  @override
  String get courseDetailOfferings => 'クラス';

  @override
  String get courseDetailLineage => '履歴';

  @override
  String get courseDetailRelated => '関連コース';

  @override
  String get courseDetailAiSummary => 'AIによる要約';

  @override
  String get courseDetailAiSummaryEmpty => 'AI要約はありません';

  @override
  String get courseBookmark => 'ブックマーク';

  @override
  String get courseBookmarked => 'ブックマーク済み';

  @override
  String get courseWriteReview => '評価を書く';

  @override
  String get reviewAnonymous => '匿名';

  @override
  String get reviewSubmit => '送信';

  @override
  String get reviewHelpful => '参考になった';

  @override
  String get reviewsEmpty => 'まだ評価はありません';

  @override
  String get wikiTitle => 'Wiki';

  @override
  String get wikiRecent => '最近の更新';

  @override
  String get wikiEditOnGithub => 'GitHubで編集';

  @override
  String get wikiToc => '目次';

  @override
  String get wikiNamespaces => '名前空間';

  @override
  String wikiViewCount(int count) {
    return '$count回閲覧';
  }

  @override
  String get settingsPush => 'プッシュ通知';

  @override
  String get settingsPushDenied => '通知が許可されていません。タップして設定を開く';

  @override
  String get settingsFollowSiteTheme => 'サイトのテーマに合わせる';

  @override
  String get settingsFollowSiteThemeDesc => 'サイトで設定された配色を使用';

  @override
  String get entryCourses => 'コース';

  @override
  String get entrySchedule => '時間割';

  @override
  String get entryWiki => 'Wiki';

  @override
  String get navCampus => 'キャンパス';

  @override
  String get campusTitle => 'キャンパスとつながる';

  @override
  String get campusSubtitle => '授業を探し、時間割を組み、キャンパスの知識を共有しましょう。';

  @override
  String get campusCoursesHint => '授業と学生の評価を見る';

  @override
  String get campusScheduleHint => '時間割を作成し、重複を確認して書き出す';

  @override
  String get campusWikiHint => 'コミュニティで作るキャンパスガイド';

  @override
  String get publishMoment => 'モーメント';

  @override
  String get publishQuestion => '質問';

  @override
  String get publishArticle => '記事';

  @override
  String get publishNext => '次へ';

  @override
  String get publishGallery => '写真を選んで、ストーリーを添えましょう';

  @override
  String get publishGalleryHint => '最大9枚。長押しして並べ替えられます。';

  @override
  String get publishFormatting => '書式設定';

  @override
  String get publishClassification => 'カテゴリを選択';

  @override
  String get publishLeaveTitle => '編集内容を残しますか？';

  @override
  String get publishLeaveBody => '未保存の変更があります。編集を続けるか、破棄してください。';

  @override
  String get publishDiscard => '変更を破棄';

  @override
  String get publishContinue => '編集を続ける';

  @override
  String get publishImageOnlyTitle => '共有したい瞬間';

  @override
  String get publishTypeLocked => '編集時は元の投稿形式が維持されます';

  @override
  String get profileMore => 'その他の操作';

  @override
  String get profileContent => 'コンテンツ管理';

  @override
  String get profileTrash => 'ごみ箱';

  @override
  String get profileSecurity => 'アカウント・セキュリティ・プライバシー';

  @override
  String get profileAdmin => '管理ワークスペース';

  @override
  String get fabDiscussion => 'ディスカッションへ';

  @override
  String get fabRefresh => '更新してトップへ戻る';

  @override
  String get contentRestore => '復元';

  @override
  String get contentDelete => '削除';

  @override
  String get contentPurge => '完全削除';

  @override
  String get contentDeleteConfirm => 'ごみ箱に移動します。対象のコンテンツは30日以内であれば復元できます。';

  @override
  String get contentPurgeConfirm => '取り消せません。文章と画像は完全に消去されます。';

  @override
  String get contentPassword => '確認のため現在のパスワードを入力';

  @override
  String get contentSelected => '選択済み';

  @override
  String get contentSelectAll => '読み込み済みの項目を選択';

  @override
  String get contentEmpty => 'まだコンテンツはありません';

  @override
  String get contentTrashHint =>
      '対象のコンテンツは30日間復元できます。モデレーターが削除したものはここでは復元できません。';

  @override
  String get commonConfirm => '確認';

  @override
  String get adminUnavailable => '管理画面を利用できません。権限、セッション、接続を確認して再試行してください。';

  @override
  String get adminDownloadFailed => 'エクスポートをダウンロードできませんでした。再試行してください。';

  @override
  String get publishGalleryTooMany => '画像は最大9枚です。一部を削除するか、記事エディターを使用してください。';

  @override
  String get profileActivity => 'アクティビティ';

  @override
  String get profileBookmarks => 'ブックマーク';

  @override
  String get profileModeration => 'モデレーション';

  @override
  String get settingsCloseAccount => 'アカウントを削除';

  @override
  String get settingsCloseAccountWarning =>
      'アカウントの閉鎖は取り消せず、すべての端末からログアウトします。過去の投稿を匿名で残すか、削除を申請できます。保存規則により一部の内容が残る場合があります。確認のため現在のパスワードを入力してください。';

  @override
  String get settingsCloseKeepContent => '匿名化した投稿を残す';

  @override
  String get settingsCloseDeleteContent => '投稿の削除を申請';

  @override
  String get publishUndo => '元に戻す';

  @override
  String get publishRedo => 'やり直し';

  @override
  String get publishHeading => '見出し';

  @override
  String get publishToolLink => 'リンクを挿入';

  @override
  String get publishLinkInvalid => '有効なHTTP・HTTPSまたはメールリンクを入力してください。';

  @override
  String get publishPhotoLibrary => 'ライブラリから選択';

  @override
  String get publishCamera => '写真を撮る';

  @override
  String get settingsWebsiteName => 'サイト名';

  @override
  String get settingsWebsite => '個人サイト';

  @override
  String get settingsSocialLinks => 'ソーシャルリンク';

  @override
  String get settingsProfileLanguage => 'プロフィールの言語';

  @override
  String get settingsInvalidLink => '有効なHTTPまたはHTTPSリンクを入力してください';

  @override
  String get siteInfoTitle => 'コミュニティについて';

  @override
  String get siteInfoLinks => 'コミュニティリンク';

  @override
  String get siteInfoSponsors => '支援者';

  @override
  String get siteInfoTerms => '利用規約';

  @override
  String get siteInfoPrivacy => 'プライバシーポリシー';

  @override
  String get siteInfoEmpty => '公開コンテンツはありません';

  @override
  String get settingsUsernameHint => 'ログインに使用する名前です。サイトの命名規則が適用されます。';

  @override
  String get settingsUsernameUpdated => 'ユーザー名を更新しました';

  @override
  String get settingsPresetAvatar => 'プリセットアバターを選択';

  @override
  String get coursesManagement => '授業を管理';

  @override
  String get coursesReviewModeration => '授業評価の管理';

  @override
  String get settingsCover => 'プロフィールのカバー';

  @override
  String get settingsCoverDescription => '画像を選び、切り抜きを調整';

  @override
  String get settingsCoverRemove => 'カバーを削除';

  @override
  String get settingsCoverRemoveConfirm => 'プロフィールに標準の背景が表示されます。';

  @override
  String get settingsCoverMinSize => '1200 × 240ピクセル以上の画像を選択してください';

  @override
  String get settingsCoverSafeArea => '明るい中央部分がモバイルでの主な表示範囲です。広い画面では全幅が表示されます。';

  @override
  String get settingsCropHint => 'ドラッグで移動、ピンチで拡大縮小';

  @override
  String get settingsCropPreview => '切り抜きのプレビュー';

  @override
  String get settingsCropZoom => 'ズーム';

  @override
  String get settingsCropReset => '位置をリセット';

  @override
  String get settingsImageSaved => '画像を更新しました';

  @override
  String settingsImageTooLarge(int maxMb) {
    return '画像は$maxMb MB以下にしてください';
  }

  @override
  String get settingsOAuthUnavailable => 'このサイトでは利用できません';

  @override
  String get settingsOAuthOpenBrowser => 'ブラウザーで連携を管理';

  @override
  String settingsOAuthBrowserHint(String username) {
    return 'ブラウザーで@$usernameとしてログインして連携してください。アプリに戻ると連携情報を更新します。';
  }

  @override
  String get commonRefresh => '更新';

  @override
  String get topicEarliest => '最初の投稿';

  @override
  String get topicLatest => '最新';

  @override
  String get topicEarlierReplies => '以前の返信を読み込む';

  @override
  String get topicHistory => '編集履歴';

  @override
  String get topicHistoryUnavailable => 'この版は表示できません';

  @override
  String get topicHistoryEmpty => 'まだ編集履歴はありません';

  @override
  String get topicDeleteConfirm => 'この内容を削除しますか？復元可能な項目はごみ箱にあります。';

  @override
  String get topicModerateBan => '内容を非表示';

  @override
  String get topicModerateUnban => '再表示';

  @override
  String get topicModerateConfirm => 'この内容の公開状態を変更しますか？';

  @override
  String get topicShare => '共有';

  @override
  String get topicEditReply => '返信を編集';

  @override
  String get topicRemoved => 'この内容は削除または非公開にされました';

  @override
  String get topicBookmark => 'ブックマーク';

  @override
  String get topicBookmarked => 'ブックマークを解除';

  @override
  String get topicLike => 'いいね';

  @override
  String get authEmailPrefix => '@より前の部分';

  @override
  String get authEmailDomain => 'メールドメイン';

  @override
  String get authAgreePolicies => '公開されている規約を読み、同意します';

  @override
  String get authPasswordMismatch => '2つのパスワードが一致しません';

  @override
  String get schedulerWebTitle => 'Webで時間割プランナーの全機能を使う';

  @override
  String get schedulerWebAction => 'f.yourtj.deを開く';

  @override
  String get schedulerPlanDisclaimer => 'これは履修計画です。正式な履修登録は大学の手続きに従ってください。';

  @override
  String get campusExploreCourses => '学生の評価から次の授業を見つける';

  @override
  String get campusPlanTitle => '気になる授業を時間割にまとめる';

  @override
  String get campusPlanDescription => 'クラスを比較して、選択前に重複を確認しましょう。';

  @override
  String get loginGoogle => 'Googleで続行';

  @override
  String get loginGithub => 'GitHubで続行';

  @override
  String get wikiSearchUnavailable => '検索を利用できません。目次から引き続き閲覧できます。';

  @override
  String get wikiSearchHint => 'キャンパスの知識を検索';

  @override
  String get wikiExploreTitle => 'キャンパスの案内役';

  @override
  String notificationComment(String actor) {
    return '$actorがあなたのトピックにコメントしました';
  }

  @override
  String notificationPostReply(String actor) {
    return '$actorがあなたに返信しました';
  }

  @override
  String notificationTopicPost(String actor) {
    return '$actorがフォロー中のトピックに投稿しました';
  }

  @override
  String notificationFollow(String actor) {
    return '$actor さんがあなたをフォローしました';
  }

  @override
  String notificationLike(String actor) {
    return '$actorがあなたの返信に「いいね」しました';
  }

  @override
  String notificationWikiUpdated(String actor) {
    return '$actorがフォロー中のWikiページを更新しました';
  }

  @override
  String notificationBadge(String badge) {
    return '「$badge」バッジを獲得しました';
  }

  @override
  String get notificationNew => '新しい通知があります';

  @override
  String get notificationSomeone => '誰か';

  @override
  String get profileRoleAdmin => '管理画面';

  @override
  String get profileActionSignup => 'コミュニティに参加';

  @override
  String get profileActionPost => 'トピックを投稿しました';

  @override
  String get profileActionLike => 'いいね済み';

  @override
  String get profileActionFollow => 'ユーザーをフォローしました';

  @override
  String get profileActionComment => '返信';

  @override
  String get replyQuoteExpand => '引用をすべて表示';

  @override
  String get replyQuoteCollapse => '引用を折りたたむ';

  @override
  String get notificationBadgeUnnamed => '新しいバッジを獲得しました';

  @override
  String get settingsAppLanguage => 'アプリの言語';

  @override
  String get settingsLanguageSystem => 'システムに合わせる';

  @override
  String scheduleGradeYear(String year) {
    return '$year年度入学';
  }

  @override
  String get schedulePeriods => '時限';

  @override
  String get scheduleWeeksLabel => '週';

  @override
  String schedulePeriodRange(String range) {
    return '$range限目';
  }

  @override
  String get courseCopyCreditUnit => '単位';

  @override
  String get courseCopyNoTeacher => '教員なし';

  @override
  String get courseCopyCatalogEmptyTitle => 'コースはまだありません';

  @override
  String get courseCopyCatalogEmptyDescription => 'コースカタログはまだインポートされていません。';

  @override
  String get courseCopyNoFilterResults => '絞り込み条件に一致するコースが見つかりません。';

  @override
  String get courseCopyNoFilterResultsDescription =>
      '絞り込みを調整または解除して、より広い範囲のコースを表示してください。';

  @override
  String get courseCopyClearSearch => '検索をクリア';

  @override
  String get courseCopyDone => '完了';

  @override
  String get courseCopyNoOptions => '選択肢はありません';

  @override
  String courseCopySelectedCount(int count) {
    return '$count 件選択中';
  }

  @override
  String get courseCopyInstructorInputHint => '教員名を入力し、Enter で追加';

  @override
  String get courseCopyInstructorAdd => '追加';

  @override
  String get courseCopyInstructorEmptyHint => '上の入力欄から教員を追加';

  @override
  String get courseCopyAliasesLabel => '別名：';

  @override
  String get courseCopyLegacyNamesLabel => '旧名称：';

  @override
  String get courseCopyReviewScopeTeam => 'チーム授業';

  @override
  String get courseCopyReviewScopeCourse => 'コース単位の評価';

  @override
  String get courseCopyTeamInstructorsPrefix => 'チーム授業 · ';

  @override
  String courseCopyTeamInstructorsSuffix(int count) {
    return ' など $count 名の先生';
  }

  @override
  String get courseCopyRatingTitle => 'コース評価';

  @override
  String get courseCopyRatingOutOf => '/ 5.0';

  @override
  String get courseCopyNoRatingQuiet => '評価なし';

  @override
  String get courseCopyOfferingsEmpty => '開講記録はまだありません。';

  @override
  String get courseCopyOfferingFocusLabel => 'このクラスの評価のみ表示';

  @override
  String get courseCopyOfferingFocusClear => 'すべての評価を表示';

  @override
  String get courseCopySummaryGenerated => '生成済み';

  @override
  String get courseCopySummaryKeywords => 'キーワード';

  @override
  String get courseCopySummaryPros => '良い点';

  @override
  String get courseCopySummaryCons => '気になる点';

  @override
  String get courseCopySummaryRepresentativeReviews => '代表的な評価';

  @override
  String get courseCopySummarySentimentPositive => '良い';

  @override
  String get courseCopySummarySentimentNeutral => '中立';

  @override
  String get courseCopySummarySentimentNegative => '悪い';

  @override
  String get courseCopySummaryRefresh => '更新';

  @override
  String get courseCopySummaryExpand => '関連コースを表示';

  @override
  String get courseCopySummaryCollapse => '折りたたむ';

  @override
  String get courseCopySummaryDisclaimer =>
      'この内容は学生の評価をもとに AI が自動生成したもので、参考用です。履修の推奨を意味するものではありません。';

  @override
  String get courseCopySummaryInsufficient => '評価が不足しているため、AI まとめを生成できません。';

  @override
  String get courseCopySummaryLoadFailed => 'AI まとめの生成に失敗しました。しばらくしてからお試しください。';

  @override
  String get courseCopyWriteReviewTitle => '評価を書く';

  @override
  String get courseCopyEditReviewTitle => 'レビューを編集';

  @override
  String get courseCopySelectOffering => '開講記録を選択';

  @override
  String get courseCopyRatingLabel => '評価';

  @override
  String get courseCopyContentLabel => '評価内容';

  @override
  String get courseCopyContentPlaceholder => '講義の内容、授業の質、成績評価などを書きましょう…';

  @override
  String get courseCopyRatingRequired => '1〜5 の星評価を選択してください。';

  @override
  String get courseCopyContentRequired => 'レビュー内容を入力してください。';

  @override
  String get courseCopyAnonymousLabel => '匿名で投稿（一般公開では身元を隠す）';

  @override
  String get courseCopySubmitSuccess => '送信しました';

  @override
  String get courseCopyUpdateSuccess => '更新しました';

  @override
  String get courseCopyDelete => '削除';

  @override
  String get courseCopyDeleteReviewTitle => 'レビューを削除';

  @override
  String get courseCopyConfirmDeleteReview => 'この評価を削除しますか？削除すると元に戻せません。';

  @override
  String get courseCopyReviewDeleted => 'レビューを削除しました';

  @override
  String get courseCopyOperationFailed => '操作に失敗しました。あとでもう一度お試しください。';

  @override
  String get courseCopyReviewsLoadFailed => '評価の読み込みに失敗しました。あとでもう一度お試しください。';

  @override
  String get courseCopyAuthorAnonymousLabel => '匿名';

  @override
  String get courseCopyAuthorLegacyLabel => '過去の匿名評価';

  @override
  String get courseCopyRelatedTeacherCoursesTitle => '同じ先生の他の授業';

  @override
  String get courseCopyRelatedOtherTeachersTitle => 'この授業の他の先生';

  @override
  String get courseCopyRelatedEmpty => '関連するコンテンツはありません';

  @override
  String get courseCopyRelationEquivalent => '等価';

  @override
  String get courseCopyRelationRenamed => '改名';

  @override
  String get courseCopyRelationSplit => '分割';

  @override
  String get courseCopyRelationMerged => '統合';

  @override
  String get courseCopyRelationRelated => '関連';

  @override
  String get courseCopySummaryConsensusStrongRecommend => '強くおすすめ';

  @override
  String get courseCopySummaryConsensusRecommend => 'おすすめ';

  @override
  String get courseCopySummaryConsensusNeutral => '賛否両論';

  @override
  String get courseCopySummaryConsensusCautious => '慎重に検討';

  @override
  String get courseCopySummaryConsensusNotRecommend => 'おすすめしない';

  @override
  String get courseCopySummaryConsensusTextStrongRecommend =>
      '多くの学生がこのコースを強くおすすめしています。';

  @override
  String get courseCopySummaryConsensusTextRecommend =>
      '多くの学生がこのコースをおすすめしています。';

  @override
  String get courseCopySummaryConsensusTextNeutral => '学生の評価は賛否両論です。';

  @override
  String get courseCopySummaryConsensusTextCautious =>
      '多くの学生が履修前に慎重な検討をすすめています。';

  @override
  String get courseCopySummaryConsensusTextNotRecommend =>
      '多くの学生がこのコースをおすすめしていません。';

  @override
  String get topicJoinDiscussion => 'ディスカッションに参加';

  @override
  String get updateCheck => 'アップデートを確認';

  @override
  String get updateAvailable => '新しいバージョンがあります';

  @override
  String get updateLatest => '最新バージョンです';

  @override
  String get updateFailed => '更新の確認またはダウンロードに失敗しました。しばらくしてから再試行してください。';

  @override
  String get updateDownload => '更新をダウンロード';

  @override
  String get updateSkip => 'このバージョンをスキップ';

  @override
  String get updateLater => '後で';

  @override
  String get updatePreparing => '最速のダウンロード元を選択中…';

  @override
  String get updateDownloading => 'ダウンロード中…';

  @override
  String get updateReady => '更新の検証が完了しました。インストールできます。';

  @override
  String get updateInstall => '更新をインストール';

  @override
  String get updatePermission =>
      'YourTJ にアプリのインストールを許可し、戻ってもう一度インストールを押してください。';

  @override
  String get updateRetry => '再試行';
}
