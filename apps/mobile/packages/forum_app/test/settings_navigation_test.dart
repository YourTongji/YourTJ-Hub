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
          home: SettingsPage(initialSection: section),
        ),
      ),
    ),
  );
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await tester.pump();
  }
  return (container: container, page: page, user: user);
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

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
      await tester.tap(find.byIcon(Icons.arrow_back));
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
        expect(find.text('昵称'), findsOneWidget);
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
    expect(find.text('Current phone'), findsOneWidget);
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
    expect(find.text('Current phone'), findsOneWidget);
    await storage.clear();
    harness.container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    expect(find.text('Current phone'), findsNothing);
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
            await tester.tap(find.byIcon(Icons.arrow_back));
            await tester.pumpAndSettle();
            expect(category, findsOneWidget);
          }
        },
      );
    }
  }
}
