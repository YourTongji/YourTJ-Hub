// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for German (`de`).
class AppLocalizationsDe extends AppLocalizations {
  AppLocalizationsDe([String locale = 'de']) : super(locale);

  @override
  String get campusCourseReviews => 'Kursbewertungen';

  @override
  String get myCourseReviewsTitle => 'Meine Kursbewertungen';

  @override
  String get myCourseReviewsEmpty =>
      'Noch keine Bewertungen. Entdecke den Kurskatalog.';

  @override
  String get myCourseReviewHidden =>
      'Diese Bewertung ist ausgeblendet. Du kannst sie hier löschen.';

  @override
  String get myCourseReviewUnavailable =>
      'Dieser Kurs ist nicht verfügbar. Du kannst deine Bewertung weiterhin verwalten.';

  @override
  String get myCourseReviewOpen => 'Details ansehen';

  @override
  String get courseReviewNetworkError =>
      'Verbindung fehlgeschlagen. Bitte Verbindung prüfen und erneut versuchen; deine Bewertung bleibt erhalten.';

  @override
  String get courseReviewUnknownError =>
      'Die Bewertung konnte nicht gespeichert werden. Der Server hat keinen konkreten Grund angegeben. Bitte später erneut versuchen.';

  @override
  String get commonHideKeyboard => 'Tastatur ausblenden';

  @override
  String get appTitle => 'YourTJ';

  @override
  String get navHome => 'Startseite';

  @override
  String get navSearch => 'Suchen';

  @override
  String get navPublish => 'Veröffentlichen';

  @override
  String get navMessages => 'Nachrichten';

  @override
  String get navProfile => 'Mein Profil';

  @override
  String get commonCancel => 'Abbrechen';

  @override
  String get commonSave => 'Speichern';

  @override
  String imageViewPosition(int index, int count) {
    return 'Bild $index von $count ansehen';
  }

  @override
  String get imageSave => 'Bild speichern';

  @override
  String get imageSaved => 'Bild gespeichert';

  @override
  String get imageSaveFailed =>
      'Bild konnte nicht gespeichert werden. Bitte später erneut versuchen.';

  @override
  String get announcementLabel => 'Ankündigung';

  @override
  String get announcementCollapse => 'Ankündigung einklappen';

  @override
  String get announcementCollapseAction => 'Einklappen';

  @override
  String get announcementExpand => 'Ankündigung ausklappen';

  @override
  String announcementItem(int index) {
    return 'Ankündigung $index';
  }

  @override
  String get commonLoading => 'Wird geladen…';

  @override
  String get commonLoadMore => 'Mehr laden';

  @override
  String get commonRetry => 'Erneut versuchen';

  @override
  String get commonClose => 'Schließen';

  @override
  String get commonSend => 'Senden';

  @override
  String get commonSearch => 'Suchen';

  @override
  String get commonEdit => 'Bearbeiten';

  @override
  String get commonCurrent => 'Aktuell';

  @override
  String get commonEmpty => 'Keine Themen';

  @override
  String get commonBack => 'Zurück';

  @override
  String get commonBackToTop => 'Nach oben';

  @override
  String get commonUseLightTheme => 'Helles Design verwenden';

  @override
  String get commonUseDarkTheme => 'Dunkles Design verwenden';

  @override
  String get timeAgoJustNow => 'gerade eben';

  @override
  String timeAgoMinutes(int count) {
    return 'vor $count Minuten';
  }

  @override
  String timeAgoHours(int count) {
    return 'vor $count Stunden';
  }

  @override
  String timeAgoDays(int count) {
    return 'vor $count Tagen';
  }

  @override
  String timeAgoWeeks(int count) {
    return 'vor $count Wo.';
  }

  @override
  String timeAgoMonths(int count) {
    return 'vor $count Mon.';
  }

  @override
  String timeAgoYears(int count) {
    return 'vor $count J.';
  }

  @override
  String get authLoginTitle => 'Melde dich mit deinem Konto an';

  @override
  String get authRegisterTitle => 'Konto erstellen';

  @override
  String get authForgotTitle => 'Passwort zurücksetzen';

  @override
  String get authContinueAfterLogin =>
      'Melde dich an, um dort weiterzumachen, wo du aufgehört hast.';

  @override
  String get authSignInMethods => 'Weitere Anmeldeoptionen';

  @override
  String get authLoginSubtitle =>
      'Willkommen zurück. Setze deine Diskussionen und Beiträge fort.';

  @override
  String get authRegisterSubtitle =>
      'Erstelle ein Konto und tausche dich auf dem Campus aus.';

  @override
  String get authForgotSubtitle =>
      'Gib deine E-Mail ein und wir senden dir eine Nachricht zum Zurücksetzen des Passworts.';

  @override
  String get authUsernameOrEmail => 'Benutzername oder E-Mail';

  @override
  String get authUsername => 'Benutzername';

  @override
  String get authEmail => 'E-Mail';

  @override
  String get authPassword => 'Passwort';

  @override
  String get authNewPassword => 'Neues Passwort';

  @override
  String get authConfirmPassword => 'Passwort bestätigen';

  @override
  String get authCaptcha => 'Captcha';

  @override
  String get authForgotPassword => 'Passwort vergessen?';

  @override
  String get authCreateAccount => 'Konto erstellen';

  @override
  String get authSendResetEmail => 'Zurücksetzungs-E-Mail senden';

  @override
  String get authBackToLogin => 'Zurück zur Anmeldung';

  @override
  String get authTwoFactorTitle => 'Zwei-Faktor-Authentifizierung';

  @override
  String get authTwoFactorCode => 'TOTP-Code';

  @override
  String get authVerify => 'Überprüfen';

  @override
  String get authGetCode => 'Code anfordern';

  @override
  String get authOidcLogin => 'Mit YourTJ anmelden';

  @override
  String get authRegisterSuccess =>
      'Registrierung erfolgreich. Bitte anmelden.';

  @override
  String get authResetEmailSent =>
      'E-Mail zum Zurücksetzen gesendet. Bitte prüfe deinen Posteingang.';

  @override
  String get authLoading => 'Wird verarbeitet …';

  @override
  String get authCacheClearFailed =>
      'Die Offline-Daten des vorherigen Kontos konnten nicht gelöscht werden. Bitte erneut versuchen.';

  @override
  String get authSessionSaveFailed =>
      'Die neue Sitzung konnte nicht sicher gespeichert werden. Bitte erneut versuchen.';

  @override
  String get loginWelcome => 'Willkommen bei YourTJ';

  @override
  String get loginModeLogin => 'Anmelden';

  @override
  String get loginModeRegister => 'Registrieren';

  @override
  String get loginModeForgot => 'Passwort vergessen';

  @override
  String get publishTitle => 'Thema veröffentlichen';

  @override
  String get publishEditTitle => 'Thema bearbeiten';

  @override
  String get publishPublish => 'Veröffentlichen';

  @override
  String get publishSaveDraft => 'Entwurf speichern';

  @override
  String get publishTitleField => 'Titel';

  @override
  String get publishTitleHint => 'Titel eingeben (5–100 Zeichen)';

  @override
  String get publishBodyPlaceholder => 'Inhalt …';

  @override
  String get publishTitleRequired => 'Bitte einen Titel eingeben';

  @override
  String get publishContentRequired => 'Bitte einen Inhalt eingeben';

  @override
  String get publishSuccess => 'Veröffentlicht';

  @override
  String get publishSavedDraft => 'Als Entwurf gespeichert';

  @override
  String publishFailed(String error) {
    return 'Veröffentlichung fehlgeschlagen: $error';
  }

  @override
  String publishImageFailed(String error) {
    return 'Bild-Upload fehlgeschlagen: $error';
  }

  @override
  String get composePreview => 'Vorschau';

  @override
  String get composeEdit => 'Bearbeiten';

  @override
  String get publishBodyField => 'Inhalt';

  @override
  String get publishCategoryRequired => 'Wähle mindestens eine Kategorie';

  @override
  String get publishPreviewEmpty =>
      'Beginne zu schreiben, um hier eine formatierte Vorschau zu sehen';

  @override
  String publishLoadFailed(String error) {
    return 'Editor-Daten konnten nicht geladen werden: $error';
  }

  @override
  String get publishToolBold => 'Fett';

  @override
  String get publishToolItalic => 'Kursiv';

  @override
  String get publishToolStrike => 'Durchgestrichen';

  @override
  String get publishToolQuote => 'Zitat';

  @override
  String get publishToolCode => 'Inline-Code';

  @override
  String get publishToolBulletList => 'Aufzählungsliste';

  @override
  String get publishToolOrderedList => 'Nummerierte Liste';

  @override
  String get publishToolImage => 'Bild hinzufügen';

  @override
  String get publishRemoveImage => 'Bild entfernen';

  @override
  String topicReplyTarget(String name) {
    return 'Antwort an $name';
  }

  @override
  String get topicTitle => 'Thema';

  @override
  String get topicReply => 'Antworten';

  @override
  String get topicReplySuccess => 'Antwort veröffentlicht';

  @override
  String topicReplyFailed(String error) {
    return 'Antwort fehlgeschlagen: $error';
  }

  @override
  String get topicReplyHint => 'Kommentar schreiben …';

  @override
  String get topicReplyTargetUnavailable =>
      'Die ursprüngliche Antwort ist nicht verfügbar';

  @override
  String get topicReplying => 'Antwort verfassen … (zum Abbrechen tippen)';

  @override
  String get topicReport => 'Beitrag melden';

  @override
  String get topicReportHint => 'Grund beschreiben';

  @override
  String get topicReportSubmit => 'Senden';

  @override
  String get topicReportSubmitted => 'Meldung gesendet';

  @override
  String topicReportFailed(String error) {
    return 'Meldung fehlgeschlagen: $error';
  }

  @override
  String get topicWatch => 'Antworten auf dieses Thema verfolgen';

  @override
  String get topicUnwatch => 'Thema nicht mehr verfolgen';

  @override
  String topicReplies(int count) {
    return '$count Antworten';
  }

  @override
  String get topicNoTitle => 'Ohne Titel';

  @override
  String get profileTitle => 'Profil';

  @override
  String get profileFollow => 'Folgen';

  @override
  String get profileFollowing => 'Folgt';

  @override
  String get profileTopics => 'Themen';

  @override
  String get profilePosts => 'Beiträge';

  @override
  String get profileReplies => 'Antworten';

  @override
  String get profileLikedPosts => 'Gefällt mir';

  @override
  String get profileLikes => 'Gefällt mir';

  @override
  String get profileFollowers => 'Follower';

  @override
  String get profileFollowingCount => 'Folgt';

  @override
  String get profileBadges => 'Badges';

  @override
  String get profileNoBadges => 'Keine Abzeichen';

  @override
  String get profileEmptyActivity => 'Noch keine Aktivität';

  @override
  String get profileEmptyTopics => 'Noch keine Themen';

  @override
  String get profileEmptyLikes => 'Noch keine Likes';

  @override
  String get profileEmptyBookmarks => 'Noch keine Lesezeichen';

  @override
  String get profileEmptyFollowing => 'Folgt noch niemandem';

  @override
  String get profileEmptyFollowers => 'Noch keine Follower';

  @override
  String get profileNotLoggedIn => 'Nicht angemeldet';

  @override
  String get messagesTitle => 'Nachrichten';

  @override
  String get messagesEmpty => 'Noch keine Unterhaltungen';

  @override
  String get messagesEmptyDescription =>
      'Beginne eine Unterhaltung mit jemandem aus der Community.';

  @override
  String get messagesSearchConversations => 'Unterhaltungen suchen';

  @override
  String get messagesConversation => 'Private Unterhaltung';

  @override
  String get messagesStartChat => 'Chat starten';

  @override
  String messagesFirstMessageTo(String user) {
    return 'Sende $user die erste Nachricht.';
  }

  @override
  String get messagesNoMessagesYet => 'Noch keine Nachrichten';

  @override
  String get messagesEmptyDetail => 'Noch keine Nachrichten. Sag Hallo!';

  @override
  String get messagesInputHint => 'Nachricht eingeben …';

  @override
  String get messagesEmoji => 'Emoji';

  @override
  String get messagesKeyboard => 'Tastatur';

  @override
  String get messagesCopyAll => 'Ganze Nachricht kopieren';

  @override
  String messagesSendFailed(String error) {
    return 'Senden fehlgeschlagen: $error';
  }

  @override
  String get notificationsTitle => 'Benachrichtigungen';

  @override
  String get notificationsEmpty => 'Keine Benachrichtigungen';

  @override
  String get notificationsMarkAllRead => 'Alle als gelesen markieren';

  @override
  String get notificationsAll => 'Alle';

  @override
  String get notificationsUnread => 'Ungelesen';

  @override
  String get searchTitle => 'Suchen';

  @override
  String get searchHint => 'Themen, Nutzer, Kategorien suchen …';

  @override
  String get searchEmpty => 'Suchbegriff eingeben';

  @override
  String get searchNoUsers => 'Keine passenden Nutzer';

  @override
  String get searchNoCategories => 'Keine passenden Kategorien';

  @override
  String get searchUnavailable => 'Suche nicht verfügbar';

  @override
  String searchResultCount(int shown, int total) {
    return '$shown angezeigt · $total Treffer';
  }

  @override
  String get searchAll => 'Alle';

  @override
  String get searchTopics => 'Themen';

  @override
  String get searchUsers => 'Benutzer';

  @override
  String get searchCategories => 'Kategorien';

  @override
  String get categoryTitle => 'Kategorie';

  @override
  String get settingsTitle => 'Einstellungen';

  @override
  String get settingsTabProfile => 'Profil';

  @override
  String get settingsTabAccount => 'Konto';

  @override
  String get settingsTabPrivacy => 'Datenschutz';

  @override
  String get settingsTabBinding => 'Verknüpfungen';

  @override
  String get settingsTabSecurity => 'Sicherheit';

  @override
  String get settingsSectionProfile => 'Persönliche Angaben';

  @override
  String get settingsNickname => 'Anzeigename';

  @override
  String get settingsNicknameEdit => 'Anzeigenamen bearbeiten';

  @override
  String get settingsBio => 'Biografie';

  @override
  String get settingsBioEdit => 'Biografie bearbeiten';

  @override
  String get settingsAvatar => 'Profilbild';

  @override
  String get settingsAvatarUpload => 'Profilfoto auswählen und zuschneiden';

  @override
  String get settingsAvatarUploading => 'Wird hochgeladen …';

  @override
  String settingsAvatarUploadFailed(String error) {
    return 'Profilbild-Upload fehlgeschlagen: $error';
  }

  @override
  String get settingsEmail => 'E-Mail';

  @override
  String get settingsEmailEdit => 'Verknüpfte E-Mail-Adresse ändern';

  @override
  String get settingsEmailUpdated =>
      'E-Mail-Adresse aktualisiert. Bitte bestätigen.';

  @override
  String get settingsEmailOAuthReauthRequired =>
      'Dieses reine OAuth-Konto muss über den Anbieter erneut authentifiziert werden, bevor eine E-Mail-Adresse festgelegt werden kann. Kontaktiere bei Bedarf einen Administrator.';

  @override
  String settingsEmailFailed(String error) {
    return 'E-Mail-Änderung fehlgeschlagen: $error';
  }

  @override
  String get settingsNewEmail => 'Neue E-Mail-Adresse';

  @override
  String get settingsChangePassword => 'Passwort ändern';

  @override
  String get settingsChangePasswordSub => 'Anmeldepasswort ändern';

  @override
  String get settingsCurrentPassword => 'Aktuelles Passwort';

  @override
  String get settingsPasswordUpdated => 'Passwort aktualisiert';

  @override
  String settingsPasswordFailed(String error) {
    return 'Passwortänderung fehlgeschlagen: $error';
  }

  @override
  String get settingsBadge => 'Badges';

  @override
  String get settingsBadgeNone => 'Keines ausgewählt';

  @override
  String settingsBadgeCurrent(String name) {
    return 'Aktuell: $name';
  }

  @override
  String get settingsBadgePick => 'Abzeichen auswählen';

  @override
  String get settingsBadgeNoOptions => 'Keine tragbaren Abzeichen';

  @override
  String get settingsBadgeUpdated => 'Abzeichen aktualisiert';

  @override
  String settingsBadgeFailed(String error) {
    return 'Abzeichen konnte nicht aktualisiert werden: $error';
  }

  @override
  String get settingsOAuth => 'Verknüpfte Konten';

  @override
  String get settingsOAuthSub => 'Anmeldung über GitHub und Google';

  @override
  String get settingsOAuthManage => 'Kontoverknüpfungen verwalten';

  @override
  String get settingsOAuthBindings => 'Kontoverknüpfungen';

  @override
  String get settingsBound => 'Verknüpft';

  @override
  String get settingsUnbound => 'Nicht verknüpft';

  @override
  String get settingsUnbind => 'Verknüpfung trennen';

  @override
  String get settingsUnboundDone => 'Verknüpfung getrennt';

  @override
  String settingsUnbindFailed(String error) {
    return 'Trennen fehlgeschlagen: $error';
  }

  @override
  String settingsLoadBindingsFailed(String error) {
    return 'Verknüpfungen konnten nicht geladen werden: $error';
  }

  @override
  String get settingsPrivacyDirect => 'Nachrichten nur für Freunde sichtbar';

  @override
  String get settingsPrivacyLikes => 'Meine Likes öffentlich anzeigen';

  @override
  String get settingsSessions => 'Sitzungen verwalten';

  @override
  String get settingsSessionsEmpty => 'Keine Unterhaltungen';

  @override
  String get settingsRevokeAll => 'Alle Sitzungen beenden';

  @override
  String get settingsRevoked => 'Sitzung widerrufen';

  @override
  String settingsRevokeFailed(String error) {
    return 'Beenden fehlgeschlagen: $error';
  }

  @override
  String get settingsRevokeAllDone => 'Alle Sitzungen beendet';

  @override
  String settingsOpFailed(String error) {
    return 'Aktion fehlgeschlagen: $error';
  }

  @override
  String get settingsDevice => 'Dieses Gerät';

  @override
  String get settingsYourAccount => 'Dein Konto';

  @override
  String get settingsThemeLight => 'Hell';

  @override
  String get settingsThemeDark => 'Dunkel';

  @override
  String get settingsRevokeSession => 'Diese Sitzung widerrufen';

  @override
  String get settingsAppearance => 'Darstellung';

  @override
  String get settingsDarkMode => 'Dunkler Modus';

  @override
  String get settingsDarkCurrent => 'Aktuell: dunkel';

  @override
  String get settingsLightCurrent => 'Aktuell: hell';

  @override
  String get settingsAbout => 'Info';

  @override
  String get settingsAboutVersion =>
      'v0.1.0 · Campusforum der Tongji-Universität';

  @override
  String get settingsEditProfile => 'Profil bearbeiten';

  @override
  String get settingsSignature => 'Signatur';

  @override
  String get settingsSaveInfo => 'Speichern';

  @override
  String get settingsInfoSaved => 'Profil aktualisiert';

  @override
  String settingsInfoFailed(String error) {
    return 'Profiländerung fehlgeschlagen: $error';
  }

  @override
  String get settingsUserDataLoading => 'Kontodaten werden geladen …';

  @override
  String get settingsFillComplete => 'Bitte alle Felder ausfüllen';

  @override
  String get settingsSecondPhase => 'Für Phase 2 geplant';

  @override
  String get settingsTotpTitle => 'Zwei-Faktor-Authentifizierung (TOTP)';

  @override
  String get settingsTotpEnable => 'Aktivieren';

  @override
  String get settingsTotpDisable => 'Deaktivieren';

  @override
  String get settingsTotpPasswordPrompt =>
      'Passwort zur Verwaltung von TOTP eingeben';

  @override
  String get settingsTotpSetupSecret =>
      'In der Authenticator-App scannen oder den Schlüssel eingeben';

  @override
  String get settingsTotpCode => '6-stelligen Code eingeben';

  @override
  String get settingsTotpRecoveryCodes =>
      'Wiederherstellungscodes (sicher aufbewahren):';

  @override
  String get settingsTotpEnabled => 'TOTP aktiviert';

  @override
  String get settingsTotpDisabled => 'TOTP deaktiviert';

  @override
  String settingsTotpFailed(String error) {
    return 'TOTP-Aktion fehlgeschlagen: $error';
  }

  @override
  String get settingsTotpDisableTitle => 'TOTP deaktivieren';

  @override
  String get settingsTotpEnableTitle => 'TOTP aktivieren';

  @override
  String get settingsTotpPassword => 'Passwort';

  @override
  String get settingsTotpNext => 'Nächste';

  @override
  String get settingsTotpScanSecret => 'Scannen oder Schlüssel eingeben';

  @override
  String get settingsTotpDone => 'Fertig';

  @override
  String get settingsTotpUnavailable => 'TOTP nicht verfügbar';

  @override
  String get draftsTitle => 'Entwürfe';

  @override
  String get draftsEmpty => 'Keine Entwürfe';

  @override
  String get draftsNew => 'Neuer Entwurf';

  @override
  String get draftsBlocked => 'Gesperrt';

  @override
  String draftsMetaCreated(Object date) {
    return 'Erstellt am $date';
  }

  @override
  String draftsMetaViews(Object count) {
    return '$count Aufrufe';
  }

  @override
  String draftsMetaReplies(Object count) {
    return '$count Antworten';
  }

  @override
  String get messagesNew => 'Neue Nachricht';

  @override
  String get messagesSearchUsers => 'Benutzer suchen';

  @override
  String get messagesNoContactableUsers => 'Keine erreichbaren Nutzer';

  @override
  String get settingsLogout => 'Abmelden';

  @override
  String get settingsLogoutConfirm => 'Von YourTJ abmelden?';

  @override
  String get commonParseFailed =>
      'Seitendaten konnten nicht verarbeitet werden';

  @override
  String get commonLoadFailed => 'Laden fehlgeschlagen';

  @override
  String get topicEmpty => 'Noch keine Themen';

  @override
  String settingsAvatarUploaded(String url) {
    return 'Profilbild hochgeladen: $url';
  }

  @override
  String get settingsImageDecodeFailed => 'Bild konnte nicht gelesen werden';

  @override
  String dateMonthDayTime(int month, int day, String time) {
    return '$day.$month $time';
  }

  @override
  String dateYearMonthDayTime(int year, int month, int day, String time) {
    return '$day.$month.$year $time';
  }

  @override
  String topicFloorSelected(Object floor) {
    return 'Zu Beitrag $floor gesprungen';
  }

  @override
  String get sortLatest => 'Neueste';

  @override
  String get sortHot => 'Trending';

  @override
  String get sortPopular => 'Beliebt';

  @override
  String get commentSortAsc => 'Älteste zuerst';

  @override
  String get commentSortDesc => 'Neueste zuerst';

  @override
  String get commentSortOnlyOp => 'Nur Autor';

  @override
  String get topicOpRepliesPending =>
      'Im geladenen Bereich gibt es keine Antworten des Autors. Lade weitere Beiträge.';

  @override
  String get topicOpRepliesEmpty => 'Noch keine Antworten des Autors';

  @override
  String get topicLaterReplies => 'Neuere Antworten laden';

  @override
  String get topicFeedModeList => 'Liste';

  @override
  String get topicFeedModeCard => 'Karten';

  @override
  String get topicNewTopic => 'Neues Thema';

  @override
  String get scheduleTitle => 'Stundenplaner';

  @override
  String get scheduleTabTimetable => 'Planvorschau';

  @override
  String get scheduleTabPick => 'Kurse auswählen';

  @override
  String get scheduleTerm => 'Semester';

  @override
  String get scheduleGrade => 'Jahr';

  @override
  String get scheduleMajor => 'Studiengang';

  @override
  String get scheduleSyncLatest => 'Synchronisieren';

  @override
  String get scheduleDataOutdated =>
      'Kursdaten wurden aktualisiert. Zum Synchronisieren tippen.';

  @override
  String scheduleSyncedTo(String date) {
    return 'Stand: $date';
  }

  @override
  String get scheduleSyncConflictTitle =>
      'Konflikt bei der Plan-Synchronisierung';

  @override
  String get scheduleSyncConflictBody =>
      'Cloud- und lokale Pläne unterscheiden sich. Wähle, welche Version behalten wird.';

  @override
  String get scheduleSyncUseCloud => 'Cloud verwenden';

  @override
  String get scheduleSyncKeepLocal => 'Lokal behalten';

  @override
  String get scheduleWeekAll => 'Alle Wochen';

  @override
  String scheduleWeekN(int week) {
    return 'Woche $week';
  }

  @override
  String get scheduleCurrentWeek => 'Aktuelle Woche';

  @override
  String get schedulePlanNew => 'Neuer Plan';

  @override
  String get schedulePlanRename => 'Plan umbenennen';

  @override
  String get schedulePlanDelete => 'Plan löschen';

  @override
  String get schedulePlanClear => 'Kurse entfernen';

  @override
  String schedulePlanN(int n) {
    return 'Plan $n';
  }

  @override
  String scheduleStatsCourses(int count) {
    return '$count Kurse';
  }

  @override
  String get scheduleStatsCredits => 'Credits';

  @override
  String get scheduleStatsHours => 'Stunden';

  @override
  String get scheduleStatsConflicts => 'Konflikte';

  @override
  String get scheduleConflictBadge => 'Konflikt';

  @override
  String scheduleConflictWith(String course) {
    return 'Zeitkonflikt mit \"$course\"';
  }

  @override
  String scheduleConflictsWith(String course, int count) {
    return 'Zeitkonflikt mit \"$course\" und $count weiteren Kursen';
  }

  @override
  String get scheduleConflictCanAdd => 'Kann trotzdem hinzugefügt werden';

  @override
  String get scheduleAddCustomEvent => 'Platzhalter hinzufügen';

  @override
  String get scheduleCustomEventLabel => 'Beschäftigt';

  @override
  String get scheduleExportPng => 'Bild exportieren';

  @override
  String get scheduleExportCsv => 'CSV exportieren';

  @override
  String get scheduleDegraded =>
      'Eingeschränkte Suche. Ergebnisse sind möglicherweise unvollständig.';

  @override
  String get scheduleNoReviewData => 'Noch keine Bewertungen';

  @override
  String get schedulePickClass => 'Wähle eine Klasse';

  @override
  String get scheduleCompulsory => 'Pflichtkurs';

  @override
  String get scheduleOptional => 'Wahlkurs';

  @override
  String get scheduleSearchHint => 'Name, Kennung oder Lehrkraft suchen';

  @override
  String get scheduleStaged => 'Ausstehend';

  @override
  String get scheduleSelected => 'Ausgewählt';

  @override
  String get scheduleRemoveCourse => 'Entfernen';

  @override
  String get scheduleParityOdd => 'Ungerade Wochen';

  @override
  String get scheduleParityEven => 'Gerade Wochen';

  @override
  String scheduleWeeksN(String range) {
    return 'Wochen $range';
  }

  @override
  String get scheduleDayMon => 'Mo';

  @override
  String get scheduleDayTue => 'Di';

  @override
  String get scheduleDayWed => 'Mi';

  @override
  String get scheduleDayThu => 'Do';

  @override
  String get scheduleDayFri => 'Fr';

  @override
  String get scheduleDaySat => 'Sa';

  @override
  String get scheduleDaySun => 'So';

  @override
  String get scheduleMorning => 'Vormittag';

  @override
  String get scheduleAfternoon => 'Nachmittag';

  @override
  String get scheduleEvening => 'Abend';

  @override
  String get coursesTitle => 'Kurskatalog';

  @override
  String get coursesSearchHint => 'Kurse suchen';

  @override
  String get coursesFilterDepartment => 'Fachbereich';

  @override
  String get coursesFilterTerm => 'Semester';

  @override
  String get coursesFilterCampus => 'Campus';

  @override
  String get coursesFilterInstructor => 'Dozent';

  @override
  String get coursesOnlyWithReviews => 'Nur mit Bewertungen';

  @override
  String coursesRatingCount(int count) {
    return '$count Bewertungen';
  }

  @override
  String get coursesNoRating => 'Keine Bewertung';

  @override
  String get courseDetailReviews => 'Bewertungen';

  @override
  String get courseDetailOfferings => 'Lehrveranstaltungen';

  @override
  String get courseDetailLineage => 'Verlauf';

  @override
  String get courseDetailRelated => 'Verwandte Kurse';

  @override
  String get courseDetailAiSummary => 'KI-Zusammenfassung';

  @override
  String get courseDetailAiSummaryEmpty => 'Keine KI-Zusammenfassung';

  @override
  String get courseBookmark => 'Speichern';

  @override
  String get courseBookmarked => 'Gespeichert';

  @override
  String get courseWriteReview => 'Bewertung schreiben';

  @override
  String get reviewAnonymous => 'Anonym';

  @override
  String get reviewSubmit => 'Senden';

  @override
  String get reviewHelpful => 'Hilfreich';

  @override
  String get reviewsEmpty => 'Noch keine Bewertungen';

  @override
  String get wikiTitle => 'Wiki';

  @override
  String get wikiLinkOpenFailed =>
      'Der Link konnte nicht geöffnet werden. Bitte versuche es erneut.';

  @override
  String get wikiRecent => 'Zuletzt aktualisiert';

  @override
  String get wikiEditOnGithub => 'Auf GitHub bearbeiten';

  @override
  String get wikiToc => 'Inhaltsverzeichnis';

  @override
  String get wikiNamespaces => 'Namespaces';

  @override
  String wikiViewCount(int count) {
    return '$count Aufrufe';
  }

  @override
  String get settingsPush => 'Push-Benachrichtigungen';

  @override
  String get settingsPushDenied =>
      'Benachrichtigungen nicht erlaubt. Tippen, um die Einstellungen zu öffnen.';

  @override
  String get settingsFollowSiteTheme => 'Website-Design übernehmen';

  @override
  String get settingsFollowSiteThemeDesc => 'Farben der Website verwenden';

  @override
  String get entryCourses => 'Kurse';

  @override
  String get entrySchedule => 'Stundenplan';

  @override
  String get entryWiki => 'Wiki';

  @override
  String get navCampus => 'Campus';

  @override
  String get campusTitle => 'Mein Campus';

  @override
  String get campusSubtitle =>
      'Finde Kurse, plane deine Woche und teile Campuswissen.';

  @override
  String get campusCoursesHint =>
      'Kurse und studentische Bewertungen entdecken';

  @override
  String get campusScheduleHint =>
      'Woche planen, Überschneidungen prüfen und exportieren';

  @override
  String get campusWikiHint => 'Ein Campusführer von der Community';

  @override
  String get publishMoment => 'Moment';

  @override
  String get publishQuestion => 'Frage';

  @override
  String get publishArticle => 'Artikel';

  @override
  String get publishNext => 'Nächste';

  @override
  String get publishGallery => 'Wähle Fotos und erzähle deine Geschichte';

  @override
  String get publishGalleryHint =>
      'Bis zu 9 Fotos. Zum Sortieren gedrückt halten und ziehen.';

  @override
  String get publishBodyDragHint =>
      'Bild im Text gedrückt halten und in einen beliebigen Absatz ziehen.';

  @override
  String get publishFormatting => 'Formatierung';

  @override
  String get publishClassification => 'Kategorien auswählen';

  @override
  String get publishLeaveTitle => 'Änderungen behalten?';

  @override
  String get publishLeaveBody =>
      'Speichere den Entwurf auf diesem Gerät, bevor du gehst, oder verwirf die Änderungen.';

  @override
  String get publishDiscard => 'Änderungen verwerfen';

  @override
  String get publishContinue => 'Weiter bearbeiten';

  @override
  String get publishImageOnlyTitle => 'Ein Moment zum Teilen';

  @override
  String get publishTypeLocked =>
      'Beim Bearbeiten bleibt der ursprüngliche Typ erhalten';

  @override
  String get profileMore => 'Weitere Optionen';

  @override
  String get profileContent => 'Inhalte verwalten';

  @override
  String get profileTrash => 'Papierkorb';

  @override
  String get profileSecurity => 'Konto, Sicherheit und Datenschutz';

  @override
  String get profileAdmin => 'Administration';

  @override
  String get fabDiscussion => 'Zur Diskussion';

  @override
  String get fabRefresh => 'Aktualisieren und nach oben';

  @override
  String get contentRestore => 'Wiederherstellen';

  @override
  String get contentDelete => 'Löschen';

  @override
  String get contentPurge => 'Endgültig löschen';

  @override
  String get contentDeleteConfirm =>
      'Der Inhalt wird in den Papierkorb verschoben und kann, sofern zulässig, innerhalb von 30 Tagen wiederhergestellt werden.';

  @override
  String get contentPurgeConfirm =>
      'Dies kann nicht rückgängig gemacht werden. Text und Bilder werden endgültig gelöscht.';

  @override
  String get contentPassword => 'Zur Bestätigung aktuelles Passwort eingeben';

  @override
  String get contentSelected => 'Ausgewählt';

  @override
  String get contentSelectAll => 'Geladene Einträge auswählen';

  @override
  String get contentEmpty => 'Noch keine Inhalte';

  @override
  String get contentTrashHint =>
      'Zulässige Inhalte können 30 Tage lang wiederhergestellt werden. Von der Moderation entfernte Inhalte lassen sich hier nicht wiederherstellen.';

  @override
  String get commonConfirm => 'Bestätigen';

  @override
  String get adminUnavailable =>
      'Die Verwaltung ist nicht verfügbar. Bitte Berechtigungen, Sitzung und Verbindung prüfen und erneut versuchen.';

  @override
  String get adminDownloadFailed =>
      'Export konnte nicht heruntergeladen werden. Bitte erneut versuchen.';

  @override
  String get publishGalleryTooMany =>
      'Bis zu 9 Bilder sind möglich. Entferne Bilder oder verwende den Artikeleditor.';

  @override
  String get profileActivity => 'Aktivität';

  @override
  String get profileBookmarks => 'Favoriten';

  @override
  String get profileModeration => 'Moderation';

  @override
  String get settingsCloseAccount => 'Konto schließen';

  @override
  String get settingsCloseAccountWarning =>
      'Die Kontoschließung ist endgültig und meldet alle Geräte ab. Du kannst bisherige Inhalte anonymisiert behalten oder ihre Löschung beantragen. Aufbewahrungsregeln können bestimmte Inhalte erhalten. Gib zur Bestätigung dein aktuelles Passwort ein.';

  @override
  String get settingsCloseKeepContent => 'Anonymisierte Inhalte behalten';

  @override
  String get settingsCloseDeleteContent => 'Löschung der Inhalte beantragen';

  @override
  String get publishUndo => 'Rückgängig';

  @override
  String get publishRedo => 'Wiederholen';

  @override
  String get publishHeading => 'Überschrift · Ebene durch Halten wählen';

  @override
  String get publishHeadingLevel1 => 'Überschrift 1';

  @override
  String get publishHeadingLevel2 => 'Überschrift 2';

  @override
  String get publishHeadingLevel3 => 'Überschrift 3';

  @override
  String get publishToolLink => 'Link einfügen';

  @override
  String get publishLinkInvalid =>
      'Gültigen HTTP-, HTTPS- oder E-Mail-Link eingeben.';

  @override
  String get publishPhotoLibrary => 'Aus Mediathek wählen';

  @override
  String get publishCamera => 'Foto aufnehmen';

  @override
  String get settingsWebsiteName => 'Name der Website';

  @override
  String get settingsWebsite => 'Persönliche Website';

  @override
  String get settingsSocialLinks => 'Social-Media-Links';

  @override
  String get settingsProfileLanguage => 'Profilsprache';

  @override
  String get settingsInvalidLink => 'Gültigen HTTP- oder HTTPS-Link eingeben';

  @override
  String get siteInfoTitle => 'Über die Community';

  @override
  String get siteInfoLinks => 'Community-Links';

  @override
  String get siteInfoSponsors => 'Unterstützer';

  @override
  String get siteInfoTerms => 'Nutzungsbedingungen';

  @override
  String get siteInfoPrivacy => 'Datenschutzerklärung';

  @override
  String get siteInfoEmpty => 'Keine öffentlichen Inhalte';

  @override
  String get settingsUsernameHint =>
      'Dies ist dein Anmeldename. Es gelten die Namensregeln der Website.';

  @override
  String get settingsUsernameUpdated => 'Benutzername aktualisiert';

  @override
  String get settingsPresetAvatar => 'Standard-Avatar auswählen';

  @override
  String get coursesManagement => 'Kurse verwalten';

  @override
  String get coursesReviewModeration => 'Bewertungen moderieren';

  @override
  String get settingsCover => 'Profil-Titelbild';

  @override
  String get settingsCoverDescription => 'Bild wählen und Ausschnitt anpassen';

  @override
  String get settingsCoverRemove => 'Titelbild entfernen';

  @override
  String get settingsCoverRemoveConfirm =>
      'Dein Profil verwendet dann den Standardhintergrund.';

  @override
  String get settingsCoverMinSize =>
      'Bitte ein Bild mit mindestens 1200 × 240 Pixeln wählen';

  @override
  String get settingsCoverSafeArea =>
      'Die helle Mitte ist der mobile Bildausschnitt. Für breitere Bildschirme bleibt die volle Breite erhalten.';

  @override
  String get settingsCropHint =>
      'Zum Verschieben ziehen, zum Zoomen zwei Finger verwenden.';

  @override
  String get settingsCropPreview => 'Zuschnitt-Vorschau';

  @override
  String get settingsCropZoom => 'Zoom';

  @override
  String get settingsCropReset => 'Position zurücksetzen';

  @override
  String get settingsImageSaved => 'Bild aktualisiert';

  @override
  String settingsImageTooLarge(int maxMb) {
    return 'Das Bild darf höchstens $maxMb MB groß sein';
  }

  @override
  String get settingsOAuthUnavailable => 'Auf dieser Website nicht aktiviert';

  @override
  String get settingsOAuthOpenBrowser => 'Verknüpfungen im Browser verwalten';

  @override
  String settingsOAuthBrowserHint(String username) {
    return 'Melde dich im Browser als @$username an, um ein Konto zu verknüpfen. Bei der Rückkehr zur App werden die Verknüpfungen aktualisiert.';
  }

  @override
  String get commonRefresh => 'Aktualisieren';

  @override
  String get topicEarliest => 'Erster Beitrag';

  @override
  String get topicLatest => 'Neueste';

  @override
  String get topicEarlierReplies => 'Frühere Antworten laden';

  @override
  String get topicHistory => 'Versionsverlauf';

  @override
  String get topicHistoryUnavailable => 'Diese Version ist nicht verfügbar';

  @override
  String get topicHistoryEmpty => 'Noch keine Versionen';

  @override
  String get topicDeleteConfirm =>
      'Diesen Inhalt löschen? Wiederherstellbare Einträge findest du im Papierkorb.';

  @override
  String get topicModerateBan => 'Inhalt ausblenden';

  @override
  String get topicModerateUnban => 'Anzeigen';

  @override
  String get topicModerateConfirm => 'Sichtbarkeit dieses Inhalts ändern?';

  @override
  String get topicShare => 'Teilen';

  @override
  String get topicEditReply => 'Antwort bearbeiten';

  @override
  String get topicRemoved => 'Dieser Inhalt wurde gelöscht oder entfernt';

  @override
  String get topicBookmark => 'Speichern';

  @override
  String get topicBookmarked => 'Lesezeichen entfernen';

  @override
  String get topicLike => 'Gefällt mir';

  @override
  String get authEmailPrefix => 'E-Mail-Präfix';

  @override
  String get authEmailDomain => 'E-Mail-Domain';

  @override
  String get authAgreePolicies =>
      'Ich habe die veröffentlichten Richtlinien gelesen und stimme ihnen zu';

  @override
  String get authPasswordMismatch =>
      'Die beiden Passwörter stimmen nicht überein';

  @override
  String get schedulerWebTitle => 'Vollständigen Stundenplaner im Web nutzen';

  @override
  String get schedulerWebAction => 'f.yourtj.de öffnen';

  @override
  String get schedulerPlanDisclaimer =>
      'Dies ist ein Kursplan. Die endgültige Einschreibung erfolgt über die Universität.';

  @override
  String get campusExploreCourses =>
      'Finde deinen nächsten Kurs anhand studentischer Bewertungen';

  @override
  String get campusPlanTitle => 'Aus deiner Kursauswahl wird ein Plan';

  @override
  String get campusPlanDescription =>
      'Vergleiche Lehrveranstaltungen und prüfe Überschneidungen vor der Auswahl.';

  @override
  String get loginGoogle => 'Mit Google fortfahren';

  @override
  String get loginGithub => 'Mit GitHub fortfahren';

  @override
  String get wikiSearchUnavailable =>
      'Die Suche ist nicht verfügbar. Du kannst weiterhin das Verzeichnis durchsuchen.';

  @override
  String get wikiSearchHint => 'Campuswissen suchen';

  @override
  String get wikiExploreTitle => 'Dein Campusbegleiter';

  @override
  String notificationComment(String actor) {
    return '$actor hat dein Thema kommentiert';
  }

  @override
  String notificationMention(String actor) {
    return '$actor hat dich erwähnt';
  }

  @override
  String get mentionListboxLabel => 'Benutzer erwähnen';

  @override
  String get mentionLoading => 'Benutzer werden gesucht…';

  @override
  String get mentionNoResults => 'Keine passenden Benutzer';

  @override
  String get mentionSearchFailed =>
      'Benutzersuche fehlgeschlagen, weiter tippen';

  @override
  String get mentionKeepTyping => 'Weiter tippen, um Benutzer zu suchen';

  @override
  String get mentionTagReplyTarget => 'Antworte auf';

  @override
  String get mentionTagTopicAuthor => 'Themenersteller';

  @override
  String get mentionTagParticipant => 'Teilnehmer';

  @override
  String notificationPostReply(String actor) {
    return '$actor hat dir geantwortet';
  }

  @override
  String notificationTopicPost(String actor) {
    return '$actor hat in einem von dir verfolgten Thema geschrieben';
  }

  @override
  String notificationFollow(String actor) {
    return '$actor folgt dir jetzt';
  }

  @override
  String notificationLike(String actor) {
    return '$actor gefällt deine Antwort';
  }

  @override
  String notificationWikiUpdated(String actor) {
    return '$actor hat eine von dir verfolgte Wiki-Seite aktualisiert';
  }

  @override
  String notificationBadge(String badge) {
    return 'Du hast das Abzeichen „$badge“ erhalten';
  }

  @override
  String get notificationNew => 'Neue Benachrichtigung';

  @override
  String get notificationSomeone => 'Jemand';

  @override
  String get profileRoleAdmin => 'Administration';

  @override
  String get profileActionSignup => 'Der Community beigetreten';

  @override
  String get profileActionPost => 'Hat ein Thema veröffentlicht';

  @override
  String get profileActionLike => 'Vergebene \"Gefällt mir\"';

  @override
  String get profileActionFollow => 'Folgt einem Benutzer';

  @override
  String get profileActionComment => 'Geantwortet';

  @override
  String get replyQuoteExpand => 'Zitat vollständig anzeigen';

  @override
  String get replyQuoteCollapse => 'Zitat einklappen';

  @override
  String get notificationBadgeUnnamed => 'Du hast ein neues Abzeichen erhalten';

  @override
  String get settingsAppLanguage => 'App-Sprache';

  @override
  String get settingsLanguageSystem => 'Systemsprache verwenden';

  @override
  String scheduleGradeYear(String year) {
    return 'Jahrgang $year';
  }

  @override
  String get schedulePeriods => 'Unterrichtsstunden';

  @override
  String get scheduleWeeksLabel => 'Wochen';

  @override
  String schedulePeriodRange(String range) {
    return 'Stunden $range';
  }

  @override
  String get courseCopyCreditUnit => 'Credits';

  @override
  String get courseCopyNoTeacher => 'Kein Dozent';

  @override
  String get courseCopyCatalogEmptyTitle => 'Noch keine Kurse';

  @override
  String get courseCopyCatalogEmptyDescription =>
      'Der Kurskatalog wurde noch nicht importiert.';

  @override
  String get courseCopyNoFilterResults => 'Kein Kurs entspricht den Filtern.';

  @override
  String get courseCopyNoFilterResultsDescription =>
      'Ändere oder entferne die Filter, um mehr Kurse zu sehen.';

  @override
  String get courseCopyClearSearch => 'Suche löschen';

  @override
  String get courseCopyDone => 'Fertig';

  @override
  String get courseCopyNoOptions => 'Keine Optionen verfügbar';

  @override
  String courseCopySelectedCount(int count) {
    return '$count ausgewählt';
  }

  @override
  String get courseCopyInstructorInputHint =>
      'Lehrperson eingeben und mit Enter hinzufügen';

  @override
  String get courseCopyInstructorAdd => 'Hinzufügen';

  @override
  String get courseCopyInstructorEmptyHint =>
      'Lehrpersonen über das Feld oben hinzufügen';

  @override
  String get courseCopyAliasesLabel => 'Alternative Namen: ';

  @override
  String get courseCopyLegacyNamesLabel => 'Früherer Name: ';

  @override
  String get courseCopyReviewScopeTeam => 'Lehrteam';

  @override
  String get courseCopyReviewScopeCourse => 'Bewertung auf Kursebene';

  @override
  String get courseCopyTeamInstructorsPrefix => 'Lehrteam · ';

  @override
  String courseCopyTeamInstructorsSuffix(int count) {
    return ' und $count weitere Dozenten';
  }

  @override
  String get courseCopyRatingTitle => 'Kursbewertung';

  @override
  String get courseCopyRatingOutOf => '/ 5.0';

  @override
  String get courseCopyNoRatingQuiet => 'Keine Bewertung';

  @override
  String get courseCopyOfferingsEmpty => 'Noch keine Angebote.';

  @override
  String get courseCopyOfferingFocusLabel =>
      'Nur Bewertungen dieser Klasse anzeigen';

  @override
  String get courseCopyOfferingFocusClear => 'Alle Bewertungen anzeigen';

  @override
  String get courseCopySummaryGenerated => 'Generiert';

  @override
  String get courseCopySummaryKeywords => 'Schlüsselwörter';

  @override
  String get courseCopySummaryPros => 'Stärken';

  @override
  String get courseCopySummaryCons => 'Schwächen';

  @override
  String get courseCopySummaryRepresentativeReviews =>
      'Repräsentative Bewertungen';

  @override
  String get courseCopySummarySentimentPositive => 'Positiv';

  @override
  String get courseCopySummarySentimentNeutral => 'Neutral';

  @override
  String get courseCopySummarySentimentNegative => 'Negativ';

  @override
  String get courseCopySummaryRefresh => 'Aktualisieren';

  @override
  String get courseCopySummaryExpand => 'Verwandte Kurse anzeigen';

  @override
  String get courseCopySummaryCollapse => 'Einklappen';

  @override
  String get courseCopySummaryDisclaimer =>
      'Automatisch generierter KI-Inhalt auf der Grundlage der Studierendenbewertungen. Nur zu Informationszwecken; keine Auswahlberatung.';

  @override
  String get courseCopySummaryInsufficient =>
      'Zu wenige Bewertungen, um eine KI-Zusammenfassung zu generieren.';

  @override
  String get courseCopySummaryLoadFailed =>
      'KI-Zusammenfassung konnte nicht generiert werden. Versuche es später erneut.';

  @override
  String get courseCopyWriteReviewTitle => 'Eine Bewertung schreiben';

  @override
  String get courseCopyEditReviewTitle => 'Bewertung bearbeiten';

  @override
  String get courseCopySelectOffering => 'Angebot auswählen';

  @override
  String get courseCopyRatingLabel => 'Bewertung';

  @override
  String get courseCopyContentLabel => 'Bewertung';

  @override
  String get courseCopyContentPlaceholder =>
      'Teile deine Erfahrungen mit dem Kurs, der Lehrqualität oder den Noten…';

  @override
  String get courseCopyRatingRequired =>
      'Wähle eine Bewertung von 1 bis 5 Sternen.';

  @override
  String get courseCopyContentRequired =>
      'Der Inhalt der Bewertung darf nicht leer sein.';

  @override
  String get courseCopyAnonymousLabel =>
      'Anonym veröffentlichen (Identität der Öffentlichkeit verborgen)';

  @override
  String get courseCopySubmitSuccess => 'Gesendet';

  @override
  String get courseCopyUpdateSuccess => 'Aktualisiert';

  @override
  String get courseCopyDelete => 'Löschen';

  @override
  String get courseCopyDeleteReviewTitle => 'Bewertung löschen';

  @override
  String get courseCopyConfirmDeleteReview =>
      'Diese Bewertung löschen? Dieser Vorgang kann nicht rückgängig gemacht werden.';

  @override
  String get courseCopyReviewDeleted => 'Bewertung gelöscht';

  @override
  String get courseCopyOperationFailed =>
      'Vorgang fehlgeschlagen. Versuche es später erneut.';

  @override
  String get courseCopyReviewsLoadFailed =>
      'Bewertungen konnten nicht geladen werden. Versuche es später erneut.';

  @override
  String get courseCopyAuthorAnonymousLabel => 'Anonym';

  @override
  String get courseCopyAuthorLegacyLabel => 'Historische anonyme Bewertung';

  @override
  String get courseCopyRelatedTeacherCoursesTitle =>
      'Weitere Kurse derselben Dozenten';

  @override
  String get courseCopyRelatedOtherTeachersTitle =>
      'Weitere Dozenten dieses Kurses';

  @override
  String get courseCopyRelatedEmpty => 'Keine verwandten Inhalte';

  @override
  String get courseCopyRelationEquivalent => 'Äquivalent';

  @override
  String get courseCopyRelationRenamed => 'Umbenannt';

  @override
  String get courseCopyRelationSplit => 'Aufgeteilt';

  @override
  String get courseCopyRelationMerged => 'Zusammengeführt';

  @override
  String get courseCopyRelationRelated => 'Verwandt';

  @override
  String get courseCopySummaryConsensusStrongRecommend => 'Stark empfohlen';

  @override
  String get courseCopySummaryConsensusRecommend => 'Empfohlen';

  @override
  String get courseCopySummaryConsensusNeutral => 'Gemischte Meinungen';

  @override
  String get courseCopySummaryConsensusCautious => 'Mit Vorsicht abwägen';

  @override
  String get courseCopySummaryConsensusNotRecommend => 'Nicht empfohlen';

  @override
  String get courseCopySummaryConsensusTextStrongRecommend =>
      'Die meisten Studierenden empfehlen diesen Kurs stark.';

  @override
  String get courseCopySummaryConsensusTextRecommend =>
      'Die meisten Studierenden empfehlen diesen Kurs.';

  @override
  String get courseCopySummaryConsensusTextNeutral =>
      'Die Meinungen der Studierenden zu diesem Kurs sind gemischt.';

  @override
  String get courseCopySummaryConsensusTextCautious =>
      'Die meisten Studierenden empfehlen, diesen Kurs vor der Wahl mit Vorsicht abzuwägen.';

  @override
  String get courseCopySummaryConsensusTextNotRecommend =>
      'Die meisten Studierenden empfehlen diesen Kurs nicht.';

  @override
  String get topicJoinDiscussion => 'An Diskussion teilnehmen';

  @override
  String get updateCheck => 'Nach Updates suchen';

  @override
  String get updateAvailable => 'Update verfügbar';

  @override
  String get updateLatest => 'Du bist auf dem neuesten Stand';

  @override
  String get updateFailed =>
      'Updates konnten nicht geprüft oder heruntergeladen werden. Bitte später erneut versuchen.';

  @override
  String get updateDownload => 'Update herunterladen';

  @override
  String get updateSkip => 'Diese Version überspringen';

  @override
  String get updateLater => 'Später';

  @override
  String get updatePreparing => 'Schnellste Downloadquelle wird ausgewählt…';

  @override
  String get updateDownloading => 'Wird heruntergeladen…';

  @override
  String get updateReady => 'Update geprüft und bereit zur Installation.';

  @override
  String get updateInstall => 'Update installieren';

  @override
  String get updatePermission =>
      'Erlaube YourTJ die Installation von Apps. Kehre dann zurück und tippe erneut auf Installieren.';

  @override
  String get updateRetry => 'Erneut versuchen';

  @override
  String get settingsPushConsent =>
      'Systemmitteilungen über Apple (iOS) oder JPush und Gerätehersteller (Android) aktivieren. Diese Dienste verarbeiten Push-Kennungen und Mitteilungsinhalte.';

  @override
  String get settingsPushUnsupported =>
      'Push ist in diesem Build nicht konfiguriert';

  @override
  String get settingsPushServerDisabled =>
      'Der Server hat diesen Kanal nicht aktiviert. Zum Wiederholen tippen.';

  @override
  String get settingsPushFailed =>
      'Push-Registrierung fehlgeschlagen. Verbindung prüfen und erneut tippen.';

  @override
  String get settingsPushPrivacy => 'JPush-Datenschutz (Android)';

  @override
  String get accountFollowing => 'Folge ich';

  @override
  String get accountFollowers => 'Follower';

  @override
  String get publishMomentHint => 'Teile einen Moment vom Campus';

  @override
  String get publishArticleHint =>
      'Teile Erfahrungen, Geschichten und Gedanken';

  @override
  String get publishQuestionHint => 'Stelle deine Frage an die Community';

  @override
  String get accountContent => 'Meine Inhalte';

  @override
  String get commonEndOfList => 'Alles gelesen';

  @override
  String get campusTools => 'Campus-Werkzeuge';

  @override
  String get campusCoursesTitle => 'Kurse entdecken';

  @override
  String get campusCoursesEmpty =>
      'Noch keine Kursbewertungen. Entdecke den Kurskatalog.';

  @override
  String get searchDiscoveryDescription =>
      'Finde Beiträge, Personen und Kategorien oder öffne deine Campus-Werkzeuge.';

  @override
  String get notificationsEmptyDescription =>
      'Antworten, Erwähnungen und neue Follower erscheinen hier. Entdecke Neues auf dem Campus.';

  @override
  String get draftsEmptyDescription =>
      'Speichere unfertige Ideen als Entwürfe und schreibe später weiter.';

  @override
  String get messagesSending => 'Wird gesendet…';

  @override
  String get messagesSent => 'Gesendet';

  @override
  String get messagesFailed => 'Nicht gesendet';

  @override
  String get messagesRetry => 'Erneut senden';

  @override
  String get draftLocalSaved => 'Auf diesem Gerät gespeichert';

  @override
  String get draftLocalSaving => 'Wird gespeichert…';

  @override
  String get draftLocalSaveFailed =>
      'Lokales Speichern fehlgeschlagen. Bitte erneut versuchen.';

  @override
  String get draftLocalRestored =>
      'Dein letzter Entwurf wurde wiederhergestellt';

  @override
  String get draftLocalSection => 'Auf diesem Gerät';

  @override
  String get draftCloudSection => 'Cloud-Entwürfe';

  @override
  String get draftKeepAndLeave => 'Speichern und schließen';

  @override
  String get draftDeleteLocal => 'Lokalen Entwurf löschen';

  @override
  String get draftLocalOnly => 'Nur auf diesem Gerät gespeichert';

  @override
  String get searchRecent => 'Letzte Suchanfragen';

  @override
  String get searchClearRecent => 'Verlauf löschen';

  @override
  String get searchCourses => 'Kurse suchen';

  @override
  String get searchWiki => 'Wiki durchsuchen';

  @override
  String get refreshFailedRetained =>
      'Aktualisierung fehlgeschlagen. Die bisherigen Inhalte bleiben sichtbar.';

  @override
  String get scheduleOpenWebShort => 'Webversion';

  @override
  String get settingsProfileLinks => 'Website und soziale Links';

  @override
  String get settingsAvatarSources => 'Vorlage wählen oder Foto hochladen';

  @override
  String get settingsEmailChangeStaged =>
      'Die Anfrage zur Änderung der E-Mail-Adresse wurde gesendet.';

  @override
  String settingsEmailPending(String email) {
    return 'Bestätigung ausstehend: $email';
  }

  @override
  String get campusOfficialTitle => 'Mein Campus · offizielle Identität';

  @override
  String get campusOfficialSubtitle =>
      'Tongji verbinden: Stundenplan, Noten und Mitteilungen';

  @override
  String get campusToday => 'Heute';

  @override
  String get campusTimetable => 'Mein Stundenplan';

  @override
  String get campusAcademics => 'Studienleistungen';

  @override
  String get campusMessages => 'Campus-Mitteilungen';

  @override
  String get campusCalendars => 'Semesterkalender';

  @override
  String get campusConnection => 'Verbindung';

  @override
  String get campusExplore => 'Campus entdecken';

  @override
  String get campusTodayCourses => 'Heutige Kurse';

  @override
  String get campusNoClasses => 'Heute keine Kurse. Genieße den Tag.';

  @override
  String get campusNoNotices => 'Noch keine Mitteilungen';

  @override
  String get campusAllNotices => 'Alle Mitteilungen';

  @override
  String get campusBind => 'Tongji-Identität verbinden';

  @override
  String get campusReauthorize => 'Hochschulzugriff erneuern';

  @override
  String get campusReplace => 'Verknüpfte Identität ändern';

  @override
  String get campusUnbind => 'Verknüpfung aufheben';

  @override
  String get campusUnbindBody =>
      'Dies löscht die Campus-Zugangsdaten und deaktiviert die Anmeldung mit dieser Tongji-Identität. E-Mail und Beiträge bleiben erhalten. Sie können ein Passwort per E-Mail-Wiederherstellung festlegen oder erneut eine Identität verknüpfen.';

  @override
  String get campusConfirmBinding => 'Verbindung bestätigen';

  @override
  String get campusConfirmUpdate => 'Erneuerten Zugriff bestätigen';

  @override
  String get campusConfirmBody =>
      'Prüfen Sie die Hochschulidentität. Nach der Bestätigung dient sie zur Anmeldung und ersetzt eine zuvor verknüpfte Identität. Die E-Mail-Adresse bleibt unverändert.';

  @override
  String get campusPrivacy =>
      'Ein Konto, eine Hochschulidentität. Name, Kalender und Stundenplan werden für die Offline-Nutzung auf diesem Gerät gespeichert. Unten können Sie diese Daten löschen.';

  @override
  String get campusDisabled => 'Die Campus-Verbindung ist hier nicht aktiviert';

  @override
  String get campusUnavailable =>
      'Hochschuldaten sind derzeit nicht verfügbar. Bitte später erneut versuchen.';

  @override
  String get campusNoData => 'Noch keine Einträge';

  @override
  String get campusAuthRequired =>
      'Bitte den Hochschulzugriff erneuern. Die Identität bleibt verknüpft.';

  @override
  String get campusIdentityConflict =>
      'Diese Identität ist bereits verknüpft oder die Verbindung wurde geändert. Bitte aktualisieren.';

  @override
  String get campusAuthExpired =>
      'Dieser Anmeldeversuch ist abgelaufen. Bitte erneut starten.';

  @override
  String get campusMessageUnavailable =>
      'Diese Mitteilung ist nicht verfügbar. Bitte die Liste aktualisieren.';

  @override
  String get campusMorning => 'Guten Morgen';

  @override
  String get campusNoon => 'Guten Tag';

  @override
  String get campusAfternoon => 'Guten Nachmittag';

  @override
  String get campusEvening => 'Guten Abend';

  @override
  String get campusNight => 'Noch wach?';

  @override
  String get campusWish1 =>
      'Möge deine Inspiration stabiler sein als das Campus-WLAN.';

  @override
  String get campusWish2 =>
      'Das Glück ist unterwegs. Vielleicht steht es gerade an einer roten Ampel.';

  @override
  String get campusWish3 =>
      'Warum war das Mathebuch traurig? Es hatte zu viele Probleme.';

  @override
  String get campusWish4 =>
      'Lass dir Zeit. Auch Bäume wachsen nicht an einem Tag.';

  @override
  String get campusAnotherWish => 'Ein anderer Gedanke';

  @override
  String get campusWeekUnknown => 'Semesterwoche nicht verfügbar';

  @override
  String get campusCreditProgress => 'Studienfortschritt';

  @override
  String get campusGradeTrend => 'Notendurchschnitt je Semester';

  @override
  String get campusCet => 'CET-Ergebnisse';

  @override
  String get campusCourses => 'Kursleistungen';

  @override
  String get campusSchoolLogin => 'Hochschulanmeldung';

  @override
  String get campusUpstreamGaps =>
      'Sport-, Gesundheits- und Kursänderungsdaten sind nicht verfügbar. Diese App hat keinen Zugriff auf Prüfungsdaten.';

  @override
  String get campusExportCalendar => 'Kurskalender exportieren';

  @override
  String get campusExportingCalendar => 'Wird exportiert…';

  @override
  String get campusExportCalendarHint =>
      'Das ganze Semester als .ics-Datei speichern oder mit einer Kalender-App teilen. Änderungen werden nicht automatisch synchronisiert. Die Datei enthält Kurse und Orte.';

  @override
  String get campusCalendarIncomplete =>
      'Semestertermine, Unterrichtswochen oder Kurszeiten sind unvollständig. Bitte aktualisieren und erneut versuchen.';

  @override
  String get campusCalendarEmpty =>
      'Für dieses Semester sind keine Kurstermine zum Exportieren vorhanden.';

  @override
  String get campusApplyAdjustments => 'Terminänderungen anwenden';

  @override
  String get campusApplyAdjustmentsHint =>
      'Bestätigte Feiertage und Ersatztermine verwenden; sonst gilt der ursprüngliche Stundenplan.';

  @override
  String get campusCalendarRules => 'Veröffentlichte Änderungen ansehen';

  @override
  String get campusCalendarRulesHint =>
      'Die heutigen Kurse und Exporte mit aktivierten Anpassungen nutzen veröffentlichte Regeln. Die Wochenansicht bleibt unverändert. Beim Aktualisieren oder Exportieren werden die neuesten Regeln geladen.';

  @override
  String get campusNoCalendarRules =>
      'Keine Änderungen hinterlegt. Der ursprüngliche Stundenplan wird exportiert.';

  @override
  String campusMakeupDate(String original, String actual) {
    return 'Unterricht vom $original findet am $actual statt';
  }

  @override
  String get loginTongji => 'Mit Tongji-SSO anmelden';

  @override
  String get loginTongjiHint =>
      'Nach der ersten Hochschulprüfung Benutzername und Passwort wählen. Matrikelnummer@tongji.edu.cn und Campus-Verbindung werden ohne erneute E-Mail-Prüfung bestätigt.';

  @override
  String get loginTongjiPolicies =>
      'Mit dem Fortfahren stimmen Sie den veröffentlichten Richtlinien zu:';

  @override
  String get linkPreviewExternalTitle => 'YourTJ verlassen';

  @override
  String linkPreviewExternalBody(String domain) {
    return 'Du wechselst zu $domain. Prüfe die Adresse, bevor du Konto-, Bestätigungs- oder Zahlungsdaten eingibst.';
  }

  @override
  String get linkPreviewRememberDomain =>
      'Für diese Domain in dieser Sitzung nicht erneut fragen';

  @override
  String get linkPreviewContinue => 'Weiter';

  @override
  String get linkPreviewCampusFallbackTitle => 'Campusnetz';

  @override
  String get linkPreviewCampusFallbackDescription =>
      'Zugriff nur aus dem Campusnetz';

  @override
  String campusTodayMakeup(String name, String date) {
    return '$name: Heute gilt der Stundenplan vom $date.';
  }

  @override
  String campusTodayHoliday(String name) {
    return '$name: Heute finden keine Lehrveranstaltungen statt.';
  }

  @override
  String campusTodayMoved(String name) {
    return '$name: Die heutigen Lehrveranstaltungen wurden auf einen anderen Tag verlegt.';
  }

  @override
  String get campusRulesUnavailable =>
      'Feiertagsregeln konnten nicht geladen werden. Später erneut versuchen oder Anpassungen deaktivieren, um den Originalstundenplan zu exportieren.';

  @override
  String get planSyncTitle => 'Plankonflikte lösen';

  @override
  String get planSyncBody =>
      'Ein anderes Gerät hat diese Einträge ebenfalls geändert. Wähle die Werte, die bleiben sollen. Andere Änderungen werden automatisch zusammengeführt.';

  @override
  String get planSyncLocal => 'Lokal behalten';

  @override
  String get planSyncRemote => 'Cloud verwenden';

  @override
  String get planSyncDeleted => 'Gelöscht';

  @override
  String get planSyncPlan => 'Plan';

  @override
  String get planSyncName => 'Name';

  @override
  String get planSyncCreatedAt => 'Erstellt';

  @override
  String get planSyncCourse => 'Kurs';

  @override
  String get planSyncEvent => 'Eigener Termin';

  @override
  String get planSyncLabel => 'Terminname';

  @override
  String get planSyncDay => 'Tag';

  @override
  String get planSyncSections => 'Stunden';

  @override
  String get planSyncWeeks => 'Wochen';

  @override
  String get planSyncApply => 'Zusammenführen und speichern';

  @override
  String get planSyncDrafts => 'Wiederherstellungsentwürfe';

  @override
  String get planSyncDraftHint =>
      'Diese Entwürfe bleiben auf diesem Gerät und belegen keinen Cloud-Platz. Beim Wiederherstellen entsteht ein neuer Plan.';

  @override
  String get planSyncRestore => 'Als neuen Plan wiederherstellen';

  @override
  String get planSyncAdopt =>
      'Lokale Pläne dieses Geräts mit diesem Konto synchronisieren';

  @override
  String get planSyncAdoptHint =>
      'Diese lokalen Pläne gehören noch nicht zu diesem Konto. Bitte vor dem Hochladen bestätigen.';

  @override
  String get planSyncCapacity =>
      'Die Cloud speichert höchstens zehn Pläne. Schaffe Platz und versuche es erneut.';

  @override
  String get planSyncRejected =>
      'Speichern wurde abgelehnt. Das Konto kann nicht schreiben oder die Plandaten waren ungültig.';

  @override
  String get planSyncArchived =>
      'Lokale Änderungen wurden als Entwurf gesichert.';

  @override
  String get privateNoteEdit => 'Notiz bearbeiten';

  @override
  String get privateNoteLabel => 'Private Notiz';

  @override
  String get privateNoteHint =>
      'Nur für dich sichtbar. Bis zu 64 Zeichen; leer speichern zum Entfernen.';

  @override
  String get badgeDisplayTitle => 'Profilabzeichen';

  @override
  String get badgeDisplayHint =>
      'Bis zu 5 Abzeichen auswählen und sortieren oder alle abwählen. Unabhängig vom Avatar-Abzeichen.';

  @override
  String get badgeDisplayUp => 'Nach oben';

  @override
  String get badgeDisplayDown => 'Nach unten';

  @override
  String get notificationsMarkRead => 'Als gelesen markieren';

  @override
  String get scheduleWidgetSettingsTitle => 'Startbildschirm-Stundenplan';

  @override
  String get scheduleWidgetPrivacyDescription =>
      'Widgets zeigen Kursnamen, Zeiten, Lehrkräfte und Orte aus dem auf diesem Gerät gespeicherten Stundenplan. Nach einer Aktualisierung erscheint der neue Stand auf dem Startbildschirm.';

  @override
  String get scheduleWidgetRefresh => 'Aus Offline-Stundenplan aktualisieren';

  @override
  String get scheduleWidgetClear => 'Startbildschirmdaten löschen';

  @override
  String get scheduleWidgetCleared => 'Startbildschirm-Stundenplan gelöscht';

  @override
  String get scheduleWidgetDiagnostics => 'Aktualisierungsdiagnose';

  @override
  String get scheduleWidgetDiagnosticsDescription =>
      'Wenn sich Datum oder Kursstatus verzögert aktualisieren, prüfe die Hintergrundaktivität und Akkueinstellungen für YourTJ. Die Menünamen unterscheiden sich je nach Gerät.';

  @override
  String get scheduleWidgetTransparencyTitle => 'Widget-Hintergrundtransparenz';

  @override
  String get scheduleWidgetTransparencyDescription =>
      'Bei höheren Werten scheint mehr vom Hintergrund durch. Der Bereich von 0 % bis 15 % verbindet den Hintergrundeffekt mit gut lesbarem Kurstext.';

  @override
  String campusSnapshotUpdated(String time) {
    return 'Geräte-Snapshot aktualisiert: $time';
  }

  @override
  String get campusSnapshotStale =>
      'Dieser Snapshot ist möglicherweise veraltet. Bitte aktualisieren.';

  @override
  String get campusSnapshotOffline =>
      'Verbindung konnte nicht geprüft werden. Der Geräte-Snapshot wird angezeigt.';

  @override
  String get campusSnapshotRefreshFailed =>
      'Einige Daten konnten nicht aktualisiert werden. Bisherige Inhalte bleiben erhalten.';

  @override
  String get campusDataNeedsRefresh =>
      'Dieser Inhalt wurde noch nicht aktualisiert. Bitte manuell laden.';

  @override
  String get campusCacheClear => 'Campus-Cache leeren';

  @override
  String get campusCacheClearDescription =>
      'Campus-Snapshots und Stundenplan-Widgets auf diesem Gerät entfernen. Entwürfe, Stundenplanentwürfe und die Hochschulverknüpfung bleiben erhalten.';

  @override
  String get campusCacheCleared => 'Campus-Cache geleert';

  @override
  String get campusCacheClearFailed =>
      'Einige Cache-Daten konnten nicht gelöscht werden. Bitte erneut versuchen.';

  @override
  String get scheduleTimeAxis => 'Stunden';

  @override
  String scheduleEmptyCell(String day, int section) {
    return '$day, Stunde $section, Kurs auswählen';
  }

  @override
  String scheduleSectionsN(String range) {
    return 'Stunden $range';
  }

  @override
  String get scheduleGridScrollHint =>
      'Seitlich wischen, um die ganze Woche zu sehen';

  @override
  String get coursesFilterSearchHint => 'Filteroptionen suchen';

  @override
  String get coursesFilterNoMatches => 'Keine passenden Filteroptionen';

  @override
  String get coursesClearSelection => 'Auswahl löschen';

  @override
  String get coursesResetSearch => 'Suche und Filter zurücksetzen';

  @override
  String get coursesFilterLoadFailed =>
      'Filteroptionen konnten nicht geladen werden.';

  @override
  String get coursesPaginationStalled =>
      'Keine weiteren Kurse erhalten. Laden Sie diese Seite erneut.';

  @override
  String get messagesNewMessages => 'Neue Nachrichten · Zum Ende';

  @override
  String get messagesReadSyncFailed =>
      'Lesestatus konnte nicht synchronisiert werden. Ungelesene Nachrichten bleiben erhalten.';

  @override
  String get messagesReadUnavailable =>
      'Dieser Server unterstützt noch keine einzelnen Lesebestätigungen. Ungelesene Nachrichten bleiben erhalten.';

  @override
  String get messagesDraftLabel => 'Entwurf';

  @override
  String get messagesDraftStorageFailed =>
      'Nachrichtenentwürfe konnten auf diesem Gerät nicht gelesen oder gespeichert werden. Erneut versuchen.';

  @override
  String get draftCollapse => 'Einklappen';

  @override
  String get draftKindNew => 'Neues Thema';

  @override
  String get draftKindServer => 'Lokale Kopie eines Cloud-Entwurfs';

  @override
  String get draftKindEdit => 'Themenbearbeitung';

  @override
  String get draftKindReply => 'Antwort';

  @override
  String get draftLocalEmpty =>
      'Unfertige Beiträge werden auf diesem Gerät gespeichert und hier angezeigt.';

  @override
  String get draftReplyLeaveUnsaved =>
      'Die letzten Änderungen an der Antwort konnten nicht gespeichert werden. Weiterbearbeiten und erneut versuchen oder diese Änderungen verwerfen und die Seite verlassen. Eine zuvor gespeicherte lokale Kopie bleibt erhalten.';

  @override
  String get publishMediaQueueTitle => 'Fotos hochladen';

  @override
  String get publishMediaPendingWarning =>
      'Einige Fotos sind noch nicht hochgeladen. Laden Sie sie hoch oder entfernen Sie sie, bevor Sie speichern, veröffentlichen, den Typ wechseln oder die Seite verlassen.';

  @override
  String get publishMediaTemporary =>
      'Ausstehende Fotos bleiben nur während dieser Bearbeitung erhalten. Nach dem Schließen der App müssen Sie sie erneut auswählen.';

  @override
  String get publishMediaUploading => 'Wird hochgeladen';

  @override
  String get publishMediaWaiting => 'Warten auf das vorherige Foto';

  @override
  String get publishMediaSavedPartial =>
      'Text und hochgeladene Fotos auf diesem Gerät gespeichert';

  @override
  String get draftSearchHint => 'Titel und Text durchsuchen';

  @override
  String get draftSearchScope =>
      'Durchsucht lokale Entwürfe und die aktuelle Cloud-Liste.';

  @override
  String draftMatchCount(int count) {
    return 'Angezeigte Entwürfe: $count';
  }

  @override
  String get draftNoMatches => 'Keine Entwürfe entsprechen diesen Bedingungen.';

  @override
  String get draftClearFilters => 'Suche und Filter zurücksetzen';

  @override
  String get draftDeleteDone =>
      'Lokaler Entwurf gelöscht. Die letzte Löschung kann rückgängig gemacht werden.';

  @override
  String get draftDeleteRestored =>
      'Entwurf auf diesem Gerät wiederhergestellt';

  @override
  String get draftRestoreConflict =>
      'Eine vorhandene Kopie wurde unverändert beibehalten.';

  @override
  String get draftRestoreFailed =>
      'Wiederherstellung fehlgeschlagen. Erneut rückgängig machen.';
}
