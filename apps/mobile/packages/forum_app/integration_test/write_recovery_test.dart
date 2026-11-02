import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/post_editor.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'package:integration_test/integration_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/fixtures/campus_fixtures.dart';

/// Actual full-screen reply editor, HTTP failure and settings-route round trip.
/// The settings destination is a fixture: no activation mail or account is touched.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('pending email keeps the imperative editor beneath settings', (
    tester,
  ) async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final writes = <Map<String, dynamic>>[];
    final unexpected = <String>[];
    server.listen((request) async {
      if (request.method != 'POST' ||
          request.uri.path != '/api/forum/posts/update' ||
          request.headers.value(HttpHeaders.authorizationHeader) !=
              'Bearer synthetic-session') {
        unexpected.add('${request.method} ${request.uri}');
      }
      final body = await utf8.decoder.bind(request).join();
      if (body.isNotEmpty) writes.add(jsonDecode(body) as Map<String, dynamic>);
      request.response.statusCode = 403;
      request.response.headers.contentType = ContentType.json;
      request.response.write(
        jsonEncode({'code': 1, 'messageCode': 'permission.emailRequired'}),
      );
      await request.response.close();
    });
    final tokens = CampusMemoryTokenStorage()..value = 'synthetic-session';
    final dio = Dio();
    final client = GfApiClient(
      dio: dio,
      tokenStorage: tokens,
      baseUrl: 'http://127.0.0.1:${server.port}',
    );
    var count = 0;
    for (final locale in ['zh', 'en', 'ja', 'de']) {
      for (final brightness in Brightness.values) {
        final router = GoRouter(
          routes: [
            GoRoute(
              path: '/',
              builder: (context, _) => Scaffold(
                body: Center(
                  child: TextButton(
                    onPressed: () => showPostEditor(
                      context,
                      postId: 42,
                      content: 'original',
                    ),
                    child: const Text('Open editor'),
                  ),
                ),
              ),
            ),
            GoRoute(
              path: '/settings/account',
              builder: (_, _) => const Scaffold(
                body: Center(child: Text('account security fixture')),
              ),
            ),
          ],
        );
        final container = ProviderContainer(
          overrides: [apiClientProvider.overrideWithValue(client)],
        );
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              theme: gfThemeData(brightness),
              locale: Locale(locale),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: const TextScaler.linear(2)),
                child: Center(child: SizedBox(width: 320, child: child)),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.tap(find.text('Open editor'));
        await tester.pumpAndSettle();
        final field = find.descendant(
          of: find.byKey(const Key('post-edit-composer')),
          matching: find.byType(TextField),
        );
        final text = 'Unsent $locale $brightness';
        await tester.enterText(field, text);
        await tester.pumpAndSettle();
        final l = AppLocalizations.of(tester.element(find.byType(PostEditor)));
        await tester.tap(find.text(l.commonSave, skipOffstage: true));
        await tester.pumpAndSettle();
        count++;
        expect(writes.length, count);
        final action = find.text(l.authCheckEmailVerification);
        await tester.ensureVisible(action);
        expect(tester.takeException(), isNull);
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot(
            'write-recovery-$locale-${brightness.name}',
          );
        }
        await tester.tap(action);
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/settings/account');
        expect(find.text('account security fixture'), findsOneWidget);
        expect(writes.length, count);
        router.pop();
        await tester.pumpAndSettle();
        expect(find.byType(PostEditor), findsOneWidget);
        expect(tester.widget<TextField>(field).controller!.text, text);
        expect(writes.last['content'], text);
        expect(writes.last['postId'], 42);
        expect(writes.length, count);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        router.dispose();
        container.dispose();
      }
    }
    expect(unexpected, isEmpty);
    dio.close(force: true);
    await server.close(force: true);
  });
}
