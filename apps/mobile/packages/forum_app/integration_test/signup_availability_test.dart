import 'dart:convert';
import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/fixtures/page_fixtures.dart';
import '../test/pages_behavior_test.dart' show NoopCache;
import '../test/pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('native signup availability and read-only recovery', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final origin = 'http://127.0.0.1:${server.port}';
    var state = 'disabled';
    var writes = 0;
    var reads = 0;
    server.listen((request) async {
      if (request.method != 'GET') writes++;
      request.response.headers.contentType = ContentType.json;
      if (request.uri.path == '/login') {
        reads++;
        request.response.write(
          jsonEncode({
            'component': 'auth.login',
            'props': {
              'initialMode': 'login',
              'redirectUrl': '/',
              'githubUrl': '',
              'googleReady': false,
              'passwordSignupState': state,
              'allowedDomains': [],
              'termsOfServiceEnabled': false,
              'privacyPolicyEnabled': false,
            },
            'layout': minimalLayoutJson(),
            'url': '/login',
            'version': '1',
            'meta': {'title': 'Login'},
          }),
        );
      } else {
        request.response.write(
          jsonEncode({
            'code': 0,
            'result': {'enabled': false},
          }),
        );
      }
      await request.response.close();
    });
    ProviderContainer? container;
    Dio? dio;
    Future<void> show(
      String locale,
      Brightness brightness, {
      double? width,
      double scale = 1,
    }) async {
      await tester.pumpWidget(const SizedBox());
      container?.dispose();
      dio?.close(force: true);
      dio = Dio()
        ..interceptors.add(
          InterceptorsWrapper(
            onRequest: (options, handler) {
              options.baseUrl = origin;
              handler.next(options);
            },
          ),
        );
      final tokens = MemoryTokenStorage();
      container = ProviderContainer(
        overrides: [
          authDioProvider.overrideWithValue(dio!),
          tokenStorageProvider.overrideWithValue(tokens),
          offlineTopicCacheProvider.overrideWithValue(NoopCache()),
          offlineChatCacheProvider.overrideWithValue(NoopCache()),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container!,
          child: MaterialApp(
            theme: gfThemeData(brightness),
            locale: Locale(locale),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(textScaler: TextScaler.linear(scale)),
              child: Center(
                child: SizedBox(width: width, child: child),
              ),
            ),
            home: const LoginPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final l = AppLocalizations.of(tester.element(find.byType(LoginPage)));
      await tester.tap(find.text(l.loginModeRegister).first);
      await tester.pumpAndSettle();
    }

    Future<void> capture(String name) async {
      expect(tester.takeException(), isNull);
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot(name);
      }
    }

    try {
      for (final theme in Brightness.values) {
        for (final availability in [
          'disabled',
          'daily_limit',
          'mail_unavailable',
          'unavailable',
        ]) {
          state = availability;
          await show('zh', theme);
          expect(find.byKey(const Key('signup-availability')), findsOneWidget);
          expect(find.byType(TextField), findsNothing);
          await capture('signup-$availability-${theme.name}');
          final l = AppLocalizations.of(tester.element(find.byType(LoginPage)));
          state = 'open';
          await tester.tap(find.text(l.authCheckSignupAvailability));
          await tester.pumpAndSettle();
          expect(find.byType(TextField), findsWidgets);
          expect(writes, 0);
        }
      }
      for (final locale in ['zh', 'en', 'ja', 'de']) {
        state = 'unavailable';
        await show(locale, Brightness.dark, width: 320, scale: 2);
        expect(find.byKey(const Key('signup-availability')), findsOneWidget);
        final l = AppLocalizations.of(tester.element(find.byType(LoginPage)));
        await tester.scrollUntilVisible(
          find.text(l.authCheckSignupAvailability),
          200,
          scrollable: find.byType(Scrollable).first,
        );
        await Scrollable.ensureVisible(
          tester.element(find.text(l.authCheckSignupAvailability)),
          alignment: .7,
        );
        await tester.pumpAndSettle();
        await capture('signup-narrow-$locale');
        state = 'open';
        await tester.tap(find.text(l.authCheckSignupAvailability));
        await tester.pumpAndSettle();
        expect(find.byType(TextField), findsWidgets);
        expect(writes, 0);
      }
      expect(writes, 0);
      expect(reads, greaterThanOrEqualTo(20));
      debugPrint(
        'Signup acceptance: $reads anonymous loopback page reads, zero writes, four states/both themes/four locales/narrow 200%.',
      );
    } finally {
      await tester.pumpWidget(const SizedBox());
      container?.dispose();
      dio?.close(force: true);
      await server.close(force: true);
    }
  });
}
