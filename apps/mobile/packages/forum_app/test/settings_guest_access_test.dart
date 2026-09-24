import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

class _Tokens implements TokenStorage {
  String? token;
  Future<String?>? pending;
  @override
  Future<String?> read() => pending ?? Future.value(token);
  @override
  Future<void> write(String value) async => token = value;
  @override
  Future<void> clear() async => token = null;
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  for (final section in <String?>[null, 'account']) {
    testWidgets('guest settings retains sign-in intent for $section', (
      tester,
    ) async {
      final router = GoRouter(
        initialLocation: '/settings',
        routes: [
          GoRoute(
            path: '/settings',
            builder: (_, _) => SettingsPage(initialSection: section),
          ),
          GoRoute(path: '/login', builder: (_, _) => const SizedBox()),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [tokenStorageProvider.overrideWithValue(_Tokens())],
          child: MaterialApp.router(
            routerConfig: router,
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final signIn = find.text('Sign in');
      await tester.ensureVisible(signIn);
      await tester.tap(signIn);
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/login');
      expect(
        router.state.uri.queryParameters['returnTo'],
        section == null ? '/settings' : '/settings/$section',
      );
    });
  }

  for (final (locale, delayed) in [
    ('en', false),
    ('en', true),
    ('zh', false),
    ('ja', false),
    ('de', false),
  ]) {
    testWidgets(
      'guest settings in $locale makes no private requests with ${delayed ? 'stale delayed' : 'empty'} token',
      (tester) async {
        tester.view.physicalSize = const Size(320, 844);
        tester.view.devicePixelRatio = 1;
        tester.platformDispatcher.textScaleFactorTestValue = 2;
        addTearDown(tester.view.reset);
        addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
        final calls = <String>[];
        final oldToken = Completer<String?>();
        final storage = _Tokens()..pending = delayed ? oldToken.future : null;
        final dio = Dio()
          ..interceptors.add(
            InterceptorsWrapper(
              onRequest: (options, handler) {
                calls.add(options.path);
                handler.reject(DioException(requestOptions: options));
              },
            ),
          );
        final client = GfApiClient(
          dio: dio,
          tokenStorage: storage,
          baseUrl: 'http://fake.local',
        );
        final container = ProviderContainer(
          overrides: [
            tokenStorageProvider.overrideWithValue(storage),
            pageRepositoryProvider.overrideWithValue(PageRepository(client)),
            userRepositoryProvider.overrideWithValue(UserRepository(client)),
          ],
        );
        addTearDown(container.dispose);
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              theme: gfThemeData(Brightness.light),
              locale: Locale(locale),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const SettingsPage(),
            ),
          ),
        );
        await tester.pump();
        if (delayed) {
          storage.pending = null;
          container.read(offlineCacheEpochProvider.notifier).invalidate();
          await tester.pump();
          oldToken.complete('previous-session-token');
        }
        await tester.pumpAndSettle();
        final l10n = AppLocalizations.of(
          tester.element(find.byType(SettingsPage)),
        );
        expect(calls, isEmpty);
        expect(find.text(l10n.settingsAppearance), findsOneWidget);
        expect(find.text(l10n.settingsAppLanguage), findsOneWidget);
        expect(find.text(l10n.authLoginTitle), findsOneWidget);
        await tester.tap(find.text(l10n.settingsAppLanguage));
        await tester.pumpAndSettle();
        expect(find.text('English'), findsOneWidget);
      },
    );
  }
}
