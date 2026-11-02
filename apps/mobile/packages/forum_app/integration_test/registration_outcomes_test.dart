import 'dart:convert';
import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/server_messages.dart';
import 'package:core/core.dart';
import 'package:go_router/go_router.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/fixtures/page_fixtures.dart';
import '../test/pages_behavior_test.dart' show NoopCache;
import '../test/pages_smoke_test.dart' show MemoryTokenStorage;

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('native created-account mail and session recovery copy', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final origin = 'http://127.0.0.1:${server.port}';
    var outcome = 'auth.register.emailRetry';
    var submissions = 0;
    final requestFailures = <String>[];
    server.listen((request) async {
      request.response.headers.contentType = ContentType.json;
      if (request.uri.path == '/login') {
        request.response.write(
          jsonEncode({
            'component': 'auth.login',
            'props': {
              'initialMode': 'login',
              'redirectUrl': '/',
              'githubUrl': '',
              'googleReady': false,
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
      } else if (request.uri.path == '/api/register') {
        submissions++;
        final body = jsonDecode(await utf8.decoder.bind(request).join()) as Map;
        if (body['userName'] != 'native_outcome' ||
            body['email'] != 'native@example.test') {
          requestFailures.add('Unexpected registration payload');
        }
        if (outcome == 'auth.register.emailRetry') {
          request.response.headers.set(
            'New-Token',
            'synthetic-created-session',
          );
        }
        request.response.write(
          jsonEncode({
            'code': outcome == 'auth.register.emailRetry' ? 0 : 1,
            'result': outcome == 'auth.register.emailRetry' ? 'created' : null,
            'messageCode': outcome,
          }),
        );
      } else {
        if (request.method != 'GET') {
          requestFailures.add('Unexpected write: ${request.uri.path}');
        }
        request.response.write(
          jsonEncode({
            'code': 0,
            'result': {
              'enabled': false,
              'captchaId': 'test-captcha',
              'captchaImg':
                  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/1ioAAAAASUVORK5CYII=',
            },
          }),
        );
      }
      await request.response.close();
    });
    ProviderContainer? container;
    Dio? dio;
    GoRouter? router;
    Finder input(String label) => find.byWidgetPredicate(
      (w) => w is TextField && w.decoration?.labelText == label,
    );
    Future<void> show(String language, Brightness theme) async {
      await tester.pumpWidget(const SizedBox());
      container?.dispose();
      dio?.close(force: true);
      router?.dispose();
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
      router = GoRouter(
        initialLocation: '/login',
        routes: [
          GoRoute(path: '/login', builder: (_, _) => const LoginPage()),
          GoRoute(
            path: '/',
            builder: (_, _) => const Scaffold(body: Text('Signed in')),
          ),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container!,
          child: MaterialApp.router(
            routerConfig: router,
            theme: gfThemeData(theme),
            locale: Locale(language),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final l = AppLocalizations.of(tester.element(find.byType(LoginPage)));
      await tester.tap(find.text(l.loginModeRegister).first);
      await tester.pumpAndSettle();
      await tester.enterText(input(l.authUsername), 'native_outcome');
      await tester.enterText(input(l.authEmail), 'native@example.test');
      await tester.enterText(input(l.authPassword), 'password123');
      await tester.enterText(input(l.authConfirmPassword), 'password123');
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      final submit = find.byWidgetPredicate(
        (w) => w is GfButton && w.size == GfButtonSize.extraLarge,
      );
      await tester.ensureVisible(submit);
      await tester.tap(submit);
      await tester.pumpAndSettle();
      final message = resolveErrorMessage(
        l,
        ApiException(fallbackMessage: '', messageCode: outcome),
      );
      expect(find.text(message), findsOneWidget);
      expect(find.text(l.authRegisterEmailVerify), findsNothing);
      if (outcome == 'auth.register.retryLogin') {
        await tester.ensureVisible(find.text(message));
      }
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot(
          'registration-${outcome.split('.').last}-$language-${theme.name}',
        );
      }
    }

    try {
      for (final language in ['zh', 'en', 'ja', 'de']) {
        for (final theme in Brightness.values) {
          for (final code in [
            'auth.register.emailRetry',
            'auth.register.retryLogin',
          ]) {
            outcome = code;
            await show(language, theme);
          }
        }
      }
      expect(submissions, 16);
      expect(requestFailures, isEmpty);
      debugPrint(
        'Registration outcomes: 16 synthetic loopback submissions, both themes and four locales; no live registration or SMTP.',
      );
    } finally {
      await tester.pumpWidget(const SizedBox());
      container?.dispose();
      dio?.close(force: true);
      router?.dispose();
      await server.close(force: true);
    }
  });
}
