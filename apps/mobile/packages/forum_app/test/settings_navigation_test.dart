import 'dart:async';
import 'dart:ui' show CheckedState;

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/site_theme.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:forum_app/src/widgets/skeletons.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show settingsPayloadJson;
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _CountingPageRepository extends PageRepository {
  _CountingPageRepository(super.client);
  int requests = 0;
  bool fail = false;
  bool malformed = false;
  String email = 'alice@example.com';
  Completer<PagePayload>? pending;
  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async {
    requests++;
    if (fail) throw StateError('Settings temporarily unavailable');
    if (pending != null) return pending!.future;
    final json = malformed ? homePayloadJson() : settingsPayloadJson();
    if (!malformed) ((json['props'] as Map)['user'] as Map)['email'] = email;
    return PagePayload.fromJson(json);
  }
}

class _CountingUserRepository extends UserRepository {
  _CountingUserRepository(super.client);
  int requests = 0;
  bool fail = false;
  int saveAttempts = 0;
  Completer<bool>? savePending;
  @override
  Future<bool> saveUserInfo({
    required String nickname,
    required String bio,
    required String signature,
    required String websiteName,
    required String website,
    String? locale,
    required Map<String, ExternalLinkPayload> externalInformation,
  }) async {
    saveAttempts++;
    if (savePending != null) return savePending!.future;
    throw StateError('Profile save unavailable');
  }

  final credentialFailures = <String>{};
  final emailWrites = <({String email, String password})>[];
  final setupPasswords = <String>[];
  final enabledCodes = <String>[];
  final disabledCodes = <String>[];
  bool totpEnabled = false;
  Completer<bool>? emailPending;

  @override
  Future<bool> setUserEmail(String email, String password) async {
    emailWrites.add((email: email, password: password));
    if (credentialFailures.contains('email')) {
      throw StateError('Email unavailable');
    }
    return emailPending?.future ?? Future.value(true);
  }

  @override
  Future<TotpStatusPayload> getTotpStatus() async =>
      TotpStatusPayload(enabled: totpEnabled);

  @override
  Future<TotpSetupPayload> getTotpSetup({required String password}) async {
    setupPasswords.add(password);
    if (credentialFailures.contains('setup')) {
      throw StateError('Password rejected');
    }
    return const TotpSetupPayload(
      secret: 'TEST-SECRET',
      otpauthUrl: 'otpauth://test',
    );
  }

  @override
  Future<TotpEnablePayload> enableTotp({required String code}) async {
    enabledCodes.add(code);
    if (credentialFailures.contains('enable')) {
      throw StateError('Code rejected');
    }
    return const TotpEnablePayload(recoveryCodes: ['recovery-1']);
  }

  @override
  Future<bool> disableTotp({required String code}) async {
    disabledCodes.add(code);
    if (credentialFailures.contains('disable')) {
      throw StateError('Code rejected');
    }
    return true;
  }

  @override
  Future<List<UserSessionPayload>> listSessions() async {
    requests++;
    if (fail) throw StateError('Sessions temporarily unavailable');
    return const [
      UserSessionPayload(
        id: 1,
        ipMasked: '192.168.*.*',
        userAgent: 'Current phone',
        createdAt: 1786104000000,
        expiresAt: 1787104000000,
        isCurrent: true,
      ),
      UserSessionPayload(
        id: 2,
        ipMasked: '192.168.*.*',
        userAgent: 'A browser session on another device',
        createdAt: 1786104000000,
        expiresAt: 1787104000000,
        isCurrent: false,
      ),
    ];
  }
}

class _DelayedTokenStorage extends MemoryTokenStorage {
  final initialRead = Completer<String?>();
  bool firstRead = true;
  @override
  Future<String?> read() {
    if (firstRead) {
      firstRead = false;
      return initialRead.future;
    }
    return super.read();
  }
}

class _SiteTheme extends SiteThemeController {
  @override
  SiteThemeState build() =>
      const SiteThemeState(available: true, following: true);
}

class _Push extends PushController {
  @override
  PushChannelStatus build() => PushChannelStatus.unsupported;
}

typedef _Harness = ({
  ProviderContainer container,
  _CountingPageRepository page,
  _CountingUserRepository user,
});

Future<_Harness> _mount(
  WidgetTester tester, {
  bool signedIn = false,
  String? section,
  bool autoEditProfile = false,
  bool withOrigin = false,
  String language = 'zh',
  double scale = 1,
  TokenStorage? tokenStorage,
  bool settle = true,
}) async {
  final storage = tokenStorage ?? MemoryTokenStorage();
  if (signedIn) await storage.write('test-session');
  final client = GfApiClient(
    dio: Dio(),
    tokenStorage: storage,
    baseUrl: 'http://fake.local',
  );
  final page = _CountingPageRepository(client);
  final user = _CountingUserRepository(client);
  final container = ProviderContainer(
    overrides: [
      tokenStorageProvider.overrideWithValue(storage),
      pageRepositoryProvider.overrideWithValue(page),
      userRepositoryProvider.overrideWithValue(user),
      siteThemeProvider.overrideWith(_SiteTheme.new),
      pushControllerProvider.overrideWith(_Push.new),
    ],
  );
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: Consumer(
        builder: (_, ref, _) => MaterialApp(
          theme: gfThemeData(Brightness.light),
          darkTheme: gfThemeData(Brightness.dark),
          themeMode: ref.watch(themeModeProvider),
          locale: Locale(language),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: TextScaler.linear(scale)),
            child: child!,
          ),
          home: withOrigin
              ? Builder(
                  builder: (context) => Scaffold(
                    body: TextButton(
                      key: const Key('open-settings'),
                      onPressed: () => Navigator.of(context).push(
                        MaterialPageRoute<void>(
                          builder: (_) => SettingsPage(
                            initialSection: section,
                            autoEditProfile: autoEditProfile,
                          ),
                        ),
                      ),
                      child: const Text('Original profile'),
                    ),
                  ),
                )
              : SettingsPage(
                  initialSection: section,
                  autoEditProfile: autoEditProfile,
                ),
        ),
      ),
    ),
  );
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await tester.pump();
  }
  if (withOrigin) {
    await tester.tap(find.byKey(const Key('open-settings')));
    await tester.pumpAndSettle();
  }
  return (container: container, page: page, user: user);
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  Finder input(String label) => find.byWidgetPredicate(
    (w) => w is TextField && w.decoration?.labelText == label,
  );
  Finder dialogAction(String label) => find.descendant(
    of: find.byType(GfAlertDialog),
    matching: find.widgetWithText(GfButton, label),
  );

  testWidgets('email validation stays beside the entered address', (
    tester,
  ) async {
    final h = await _mount(
      tester,
      signedIn: true,
      section: 'account',
      language: 'en',
    );
    await tester.tap(find.text('Email').first);
    await tester.pumpAndSettle();
    await tester.enterText(input('New email'), 'next@example.com');
    await tester.tap(dialogAction('Save'));
    await tester.pumpAndSettle();

    expect(find.byType(GfAlertDialog), findsOneWidget);
    expect(input('New email'), findsOneWidget);
    expect(
      tester.widget<TextField>(input('New email')).controller!.text,
      'next@example.com',
    );
    expect(find.text('Please fill in all fields'), findsOneWidget);
    expect(h.user.emailWrites, isEmpty);
  });

  testWidgets(
    'email save failure retains input and retry waits for acknowledgement',
    (tester) async {
      final h = await _mount(
        tester,
        signedIn: true,
        section: 'account',
        language: 'en',
      );
      h.user.credentialFailures.add('email');
      await tester.tap(find.text('Email').first);
      await tester.pumpAndSettle();
      await tester.enterText(input('New email'), 'next@example.com');
      await tester.enterText(input('Current password'), ' private password ');
      await tester.tap(dialogAction('Save'));
      await tester.pumpAndSettle();

      expect(find.byType(GfAlertDialog), findsOneWidget);
      expect(
        tester.widget<TextField>(input('New email')).controller!.text,
        'next@example.com',
      );
      expect(
        tester.widget<TextField>(input('Current password')).controller!.text,
        ' private password ',
      );
      expect(find.textContaining('Email update failed:'), findsOneWidget);
      expect(h.user.emailWrites.single.password, ' private password ');

      h.user.credentialFailures.clear();
      h.user.emailPending = Completer<bool>();
      final submit = tester.widget<GfButton>(dialogAction('Save')).onPressed!;
      submit();
      submit();
      await tester.pump();
      expect(h.user.emailWrites.length, 2);
      expect(tester.widget<GfButton>(dialogAction('Save')).onPressed, isNull);
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(find.byType(GfAlertDialog), findsOneWidget);
      h.user.emailPending!.complete(true);
      await tester.pumpAndSettle();
      expect(find.byType(GfAlertDialog), findsNothing);
    },
  );

  testWidgets(
    'TOTP setup and enable failures keep their current inputs for retry',
    (tester) async {
      final h = await _mount(
        tester,
        signedIn: true,
        section: 'security',
        language: 'en',
      );
      h.user.credentialFailures.addAll(['setup', 'enable']);
      await tester.tap(find.text('Enable').first);
      await tester.pumpAndSettle();
      await tester.enterText(input('Password'), ' private password ');
      await tester.tap(dialogAction('Next'));
      await tester.pumpAndSettle();

      expect(
        tester.widget<TextField>(input('Password')).controller!.text,
        ' private password ',
      );
      expect(find.textContaining('TOTP operation failed:'), findsOneWidget);
      expect(h.user.setupPasswords.single, ' private password ');
      h.user.credentialFailures.remove('setup');
      await tester.tap(dialogAction('Next'));
      await tester.pumpAndSettle();
      expect(find.text('TEST-SECRET'), findsOneWidget);
      await tester.enterText(input('Enter the 6-digit code'), '123456');
      await tester.tap(dialogAction('Enable'));
      await tester.pumpAndSettle();

      expect(find.text('TEST-SECRET'), findsOneWidget);
      expect(
        tester
            .widget<TextField>(input('Enter the 6-digit code'))
            .controller!
            .text,
        '123456',
      );
      expect(find.textContaining('TOTP operation failed:'), findsOneWidget);
      h.user.credentialFailures.remove('enable');
      await tester.tap(dialogAction('Enable'));
      await tester.pumpAndSettle();
      expect(find.text('recovery-1'), findsOneWidget);
      expect(h.user.enabledCodes, ['123456', '123456']);
    },
  );

  testWidgets('TOTP disable failure keeps the code and can retry in place', (
    tester,
  ) async {
    final h = await _mount(
      tester,
      signedIn: true,
      section: 'security',
      language: 'en',
    );
    h.user.totpEnabled = true;
    h.user.credentialFailures.add('disable');
    await tester.tap(find.text('Enable').first);
    await tester.pumpAndSettle();
    await tester.enterText(input('Enter the 6-digit code'), '123456');
    await tester.tap(dialogAction('Disable'));
    await tester.pumpAndSettle();

    expect(
      tester
          .widget<TextField>(input('Enter the 6-digit code'))
          .controller!
          .text,
      '123456',
    );
    expect(find.textContaining('TOTP operation failed:'), findsOneWidget);
    h.user.credentialFailures.clear();
    await tester.tap(dialogAction('Disable'));
    await tester.pumpAndSettle();
    expect(find.byType(GfAlertDialog), findsNothing);
    expect(h.user.disabledCodes, ['123456', '123456']);
  });

  testWidgets('email dialog drops private inputs when the account changes', (
    tester,
  ) async {
    final h = await _mount(
      tester,
      signedIn: true,
      section: 'account',
      language: 'en',
    );
    h.user.emailPending = Completer<bool>();
    await tester.tap(find.text('Email').first);
    await tester.pumpAndSettle();
    await tester.enterText(input('New email'), 'next@example.com');
    await tester.enterText(input('Current password'), 'private password');
    await tester.tap(dialogAction('Save'));
    await tester.pump();
    h.container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    expect(input('New email'), findsNothing);
    h.user.emailPending!.complete(true);
    await tester.pumpAndSettle();
    expect(find.text('Email change request submitted.'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('profile shortcut opens the complete editor once after loading', (
    tester,
  ) async {
    await _mount(
      tester,
      signedIn: true,
      section: 'profile',
      autoEditProfile: true,
    );
    expect(find.byKey(const ValueKey('profile-nickname')), findsOneWidget);
    await tester.tap(find.byTooltip('取消'));
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('profile-nickname')), findsNothing);
    expect(find.text('编辑资料'), findsOneWidget);
  });

  for (final save in [false, true]) {
    testWidgets(
      'profile shortcut returns to its origin after ${save ? 'save' : 'cancel'}',
      (tester) async {
        final harness = await _mount(
          tester,
          signedIn: true,
          section: 'profile',
          autoEditProfile: true,
          withOrigin: true,
        );
        if (save) {
          harness.user.savePending = Completer<bool>();
          await tester.enterText(
            find.byKey(const ValueKey('profile-nickname')),
            'Saved profile name',
          );
          await tester.pump();
          await tester.tap(find.byKey(const Key('profile-save')));
          await tester.pump();
          expect(harness.user.saveAttempts, 1);
          harness.user.savePending!.complete(true);
        } else {
          await tester.tap(find.byTooltip('取消'));
        }
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('open-settings')), findsOneWidget);
        expect(find.byType(SettingsPage), findsNothing);
      },
    );
  }

  testWidgets('ordinary profile editor returns to the settings route', (
    tester,
  ) async {
    await _mount(tester, signedIn: true, section: 'profile', withOrigin: true);
    await tester.ensureVisible(find.text('编辑资料'));
    await tester.tap(find.text('编辑资料'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('取消'));
    await tester.pumpAndSettle();
    expect(find.byType(SettingsPage), findsOneWidget);
    expect(find.byKey(const Key('open-settings')), findsNothing);
  });

  testWidgets('profile shortcut does not pop a route opened above its editor', (
    tester,
  ) async {
    await _mount(
      tester,
      signedIn: true,
      section: 'profile',
      autoEditProfile: true,
      withOrigin: true,
    );
    final editorContext = tester.element(
      find.byKey(const Key('profile-nickname')),
    );
    final editorRoute = ModalRoute.of(editorContext)!;
    final navigator = Navigator.of(editorContext);
    unawaited(
      navigator.push<void>(
        MaterialPageRoute<void>(
          builder: (_) => const Scaffold(body: Text('New destination')),
        ),
      ),
    );
    await tester.pumpAndSettle();
    navigator.removeRoute(editorRoute);
    await tester.pumpAndSettle();
    expect(find.text('New destination'), findsOneWidget);
    navigator.pop();
    await tester.pumpAndSettle();
    expect(find.byType(SettingsPage), findsOneWidget);
    expect(find.byKey(const Key('open-settings')), findsNothing);
  });

  testWidgets('invalidated profile shortcut does not pop its settings route', (
    tester,
  ) async {
    final harness = await _mount(
      tester,
      signedIn: true,
      section: 'profile',
      autoEditProfile: true,
      withOrigin: true,
    );
    harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    expect(find.byType(SettingsPage), findsOneWidget);
    expect(find.byKey(const Key('profile-nickname')), findsNothing);
    expect(find.byKey(const Key('open-settings')), findsNothing);
  });

  testWidgets('failed profile save keeps the editor and entered text', (
    tester,
  ) async {
    final harness = await _mount(tester, signedIn: true, section: 'profile');
    final edit = find.text('编辑资料');
    await tester.ensureVisible(edit);
    await tester.tap(edit);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const ValueKey('profile-nickname')),
      'Kept nickname',
    );
    await tester.pump();
    await tester.tap(find.text('保存'));
    await tester.pumpAndSettle();
    expect(harness.user.saveAttempts, 1);
    expect(find.byKey(const ValueKey('profile-nickname')), findsOneWidget);
    expect(find.text('Kept nickname'), findsOneWidget);
  });

  testWidgets('session invalidation removes an unsaved profile editor', (
    tester,
  ) async {
    final harness = await _mount(
      tester,
      signedIn: true,
      section: 'profile',
      autoEditProfile: true,
    );
    await tester.enterText(
      find.byKey(const ValueKey('profile-nickname')),
      'Private previous profile',
    );
    harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('profile-nickname')), findsNothing);
    expect(find.text('Private previous profile'), findsNothing);
    expect(harness.user.saveAttempts, 0);
  });

  testWidgets(
    'late profile acknowledgement cannot restore an invalidated editor',
    (tester) async {
      final harness = await _mount(
        tester,
        signedIn: true,
        section: 'profile',
        autoEditProfile: true,
      );
      harness.user.savePending = Completer<bool>();
      await tester.enterText(
        find.byKey(const ValueKey('profile-nickname')),
        'Previous account pending',
      );
      await tester.pump();
      await tester.tap(find.byKey(const Key('profile-save')));
      await tester.pump();
      expect(harness.user.saveAttempts, 1);
      harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      harness.user.savePending!.complete(true);
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('profile-nickname')), findsNothing);
      expect(find.text('Previous account pending'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('guests reach appearance without fetching account data', (
    tester,
  ) async {
    final harness = await _mount(tester, section: 'appearance');
    expect(harness.page.requests, 0);
    expect(harness.user.requests, 0);
    expect(find.text('跟随系统'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('guest index exposes language, appearance and sign-in', (
    tester,
  ) async {
    final harness = await _mount(tester);
    expect(find.text('登录账号'), findsOneWidget);
    expect(
      find.byKey(const ValueKey('settings-category-profile')),
      findsNothing,
    );
    await tester.tap(find.byKey(const ValueKey('settings-category-language')));
    await tester.pumpAndSettle();
    for (final label in ['简体中文', 'English', '日本語', 'Deutsch']) {
      expect(find.text(label), findsOneWidget);
    }
    expect(harness.page.requests, 0);
    expect(harness.user.requests, 0);
  });

  testWidgets(
    'theme choices expose radio semantics and follow system brightness',
    (tester) async {
      final handle = tester.ensureSemantics();

      tester.platformDispatcher.platformBrightnessTestValue = Brightness.dark;
      addTearDown(tester.platformDispatcher.clearPlatformBrightnessTestValue);
      final harness = await _mount(
        tester,
        section: 'appearance',
        language: 'en',
      );
      final system = find.byKey(const ValueKey('settings-theme-system'));
      expect(find.bySemanticsLabel('Follow system'), findsOneWidget);
      expect(
        tester
            .getSemantics(system)
            .getSemanticsData()
            .flagsCollection
            .isChecked,
        CheckedState.isTrue,
      );
      expect(Theme.of(tester.element(system)).brightness, Brightness.dark);
      await tester.tap(find.byKey(const ValueKey('settings-theme-light')));
      await tester.pumpAndSettle();
      expect(harness.container.read(themeModeProvider), ThemeMode.light);
      expect(Theme.of(tester.element(system)).brightness, Brightness.light);
      await tester.tap(system);
      await tester.pumpAndSettle();
      expect(harness.container.read(themeModeProvider), ThemeMode.system);
      expect(Theme.of(tester.element(system)).brightness, Brightness.dark);
      handle.dispose();
    },
  );

  testWidgets(
    'keyboard opens a category, selects a theme and returns to the index',
    (tester) async {
      final harness = await _mount(tester, language: 'en');
      await tester.sendKeyEvent(LogicalKeyboardKey.tab); // Back
      await tester.sendKeyEvent(LogicalKeyboardKey.tab); // Appearance
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('settings-theme-system')),
        findsOneWidget,
      );
      await tester.sendKeyEvent(LogicalKeyboardKey.tab); // Back
      await tester.sendKeyEvent(LogicalKeyboardKey.tab); // Radio group
      await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
      await tester.pumpAndSettle();
      expect(harness.container.read(themeModeProvider), ThemeMode.light);
      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('settings-category-appearance')),
        findsOneWidget,
      );
      expect(harness.page.requests, 0);
    },
  );

  for (final malformed in [false, true]) {
    testWidgets(
      'profile refresh retains loaded content after ${malformed ? 'malformed payload' : 'network failure'}',
      (tester) async {
        final harness = await _mount(
          tester,
          signedIn: true,
          section: 'profile',
        );
        harness.page.fail = !malformed;
        harness.page.malformed = malformed;
        await tester
            .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
            .onRefresh();
        await tester.pumpAndSettle();
        expect(harness.page.requests, 2);
        expect(find.text('编辑资料'), findsOneWidget);
        expect(find.byType(GfSettingsSkeleton), findsNothing);
        await tester.pump(const Duration(seconds: 4));
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets('session refresh retains previously loaded sessions on failure', (
    tester,
  ) async {
    final harness = await _mount(tester, signedIn: true, section: 'security');
    harness.user.fail = true;
    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();
    expect(harness.user.requests, 2);
    expect(harness.page.requests, 0);
    expect(find.text('当前设备'), findsOneWidget);
    expect(find.byTooltip('吊销此会话'), findsOneWidget);
    await tester.pump(const Duration(seconds: 4));
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'delayed token read cannot restore a previous session after a boundary',
    (tester) async {
      final storage = _DelayedTokenStorage();
      final harness = await _mount(
        tester,
        tokenStorage: storage,
        section: 'account',
        settle: false,
      );
      harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      storage.initialRead.complete('expired-token');
      await tester.pumpAndSettle();
      expect(harness.page.requests, 0);
      expect(harness.user.requests, 0);
      expect(find.text('alice@example.com'), findsNothing);
      expect(find.text('登录账号'), findsOneWidget);
    },
  );

  testWidgets(
    'account boundary clears private content and rejects its pending refresh',
    (tester) async {
      final storage = MemoryTokenStorage();
      final harness = await _mount(
        tester,
        tokenStorage: storage,
        signedIn: true,
        section: 'account',
      );
      expect(find.text('alice@example.com'), findsOneWidget);
      final pending = Completer<PagePayload>();
      harness.page.pending = pending;
      final refreshing = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await storage.clear();
      harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      expect(find.text('alice@example.com'), findsNothing);
      pending.complete(PagePayload.fromJson(settingsPayloadJson()));
      await refreshing;
      await tester.pumpAndSettle();
      expect(find.text('alice@example.com'), findsNothing);
      expect(find.text('登录账号'), findsOneWidget);
      harness.page.pending = null;
      harness.page.email = 'next@example.com';
      await storage.write('next-session');
      harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      expect(find.text('alice@example.com'), findsNothing);
      expect(find.text('next@example.com'), findsOneWidget);
    },
  );

  testWidgets('session boundary clears the previous session list', (
    tester,
  ) async {
    final storage = MemoryTokenStorage();
    final harness = await _mount(
      tester,
      tokenStorage: storage,
      signedIn: true,
      section: 'security',
    );
    expect(find.text('当前设备'), findsOneWidget);
    await storage.clear();
    harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    expect(find.text('当前设备'), findsNothing);
    expect(find.text('登录账号'), findsOneWidget);
  });

  for (final width in [320.0, 390.0, 768.0, 1024.0]) {
    for (final language in ['zh', 'en', 'ja', 'de']) {
      testWidgets(
        'all settings categories fit $width pixels, $language and 200% text',
        (tester) async {
          tester.view.physicalSize = Size(width, 900);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.reset);
          final harness = await _mount(
            tester,
            signedIn: true,
            language: language,
            scale: 2,
          );
          expect(find.byType(GfTabBar), findsNothing);
          expect(harness.page.requests, 0);
          expect(harness.user.requests, 0);
          expect(tester.takeException(), isNull);
          for (final section in [
            'appearance',
            'profile',
            'account',
            'privacy',
            'binding',
            'notifications',
            'security',
          ]) {
            final category = find.byKey(ValueKey('settings-category-$section'));
            await tester.ensureVisible(category);
            await tester.pumpAndSettle();
            await tester.tap(category);
            await tester.pumpAndSettle();
            expect(
              tester.takeException(),
              isNull,
              reason: '$section initial viewport',
            );
            await tester.drag(
              find.byType(ListView).last,
              const Offset(0, -2000),
            );
            await tester.pumpAndSettle();
            expect(
              tester.takeException(),
              isNull,
              reason: '$section scrolled viewport',
            );
            await tester.tap(
              find.byTooltip(
                AppLocalizations.of(
                  tester.element(find.byType(SettingsPage).last),
                ).commonBack,
              ),
            );
            await tester.pumpAndSettle();
            expect(category, findsOneWidget);
          }
        },
      );
    }
  }
}
