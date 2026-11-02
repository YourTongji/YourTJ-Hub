import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/navigation/account_return.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/pages_behavior_test.dart' show settingsPayloadJson;
import '../test/pages_smoke_test.dart' show MemoryTokenStorage;

/// The local runner opens yourtj://account/email-verified through the OS when
/// ACCOUNT_RETURN_READY is printed. Transport stays on synthetic loopback data.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('OS account return opens current account and retains draft', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final failures = <String>[];
    var settingsReads = 0;
    server.listen((request) async {
      if (request.method != 'GET' ||
          request.headers.value('Authorization') !=
              'Bearer synthetic-account-A') {
        failures.add(
          'Unexpected request: ${request.method} ${request.uri.path}',
        );
      }
      request.response.headers.contentType = ContentType.json;
      if (request.uri.path == '/settings') {
        settingsReads++;
        request.response.write(jsonEncode(settingsPayloadJson()));
      } else {
        request.response.write(
          jsonEncode({
            'code': 0,
            'result': request.uri.path == '/api/user/sessions'
                ? []
                : {'enabled': false},
          }),
        );
      }
      await request.response.close();
    });
    final tokens = MemoryTokenStorage();
    await tokens.write('synthetic-account-A');
    final dio = Dio();
    final client = GfApiClient(
      dio: dio,
      tokenStorage: tokens,
      baseUrl: 'http://127.0.0.1:${server.port}',
    );
    ProviderContainer? container;
    GoRouter? router;
    try {
      for (final brightness in Brightness.values) {
        await tester.pumpWidget(const SizedBox());
        container?.dispose();
        router?.dispose();
        container = ProviderContainer(
          overrides: [
            apiClientProvider.overrideWithValue(client),
            tokenStorageProvider.overrideWithValue(tokens),
          ],
        );
        router = GoRouter(
          routes: [
            GoRoute(
              path: '/',
              builder: (_, _) => const Scaffold(
                body: SafeArea(child: TextField(key: Key('draft'))),
              ),
            ),
            GoRoute(
              path: '/settings/account',
              builder: (_, _) => const SettingsPage(initialSection: 'account'),
            ),
          ],
        );
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: AccountReturnHost(
              router: router,
              child: MaterialApp.router(
                routerConfig: router,
                theme: gfThemeData(brightness),
                locale: const Locale('en'),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.enterText(
          find.byKey(const Key('draft')),
          'unfinished account-A draft',
        );
        FocusManager.instance.primaryFocus?.unfocus();
        await tester.pumpAndSettle();
        debugPrint('ACCOUNT_RETURN_READY ${brightness.name}');
        for (
          var i = 0;
          i < 1200 && find.byType(SettingsPage).evaluate().isEmpty;
          i++
        ) {
          await tester.pump(const Duration(milliseconds: 100));
        }
        await tester.pumpAndSettle();
        expect(find.byType(SettingsPage), findsOneWidget);
        expect(find.text('alice@example.com'), findsWidgets);
        expect(await tokens.read(), 'synthetic-account-A');
        expect(tester.takeException(), isNull);
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot('account-return-${brightness.name}');
        }
        router.pop();
        await tester.pumpAndSettle();
        expect(find.text('unfinished account-A draft'), findsOneWidget);
      }
      expect(settingsReads, 2);
      expect(failures, isEmpty);
      debugPrint(
        'Account return: two real OS links; only current-account reads, zero writes; draft and credential retained.',
      );
    } finally {
      await tester.pumpWidget(const SizedBox());
      container?.dispose();
      router?.dispose();
      dio.close(force: true);
      await server.close(force: true);
    }
  });
}
