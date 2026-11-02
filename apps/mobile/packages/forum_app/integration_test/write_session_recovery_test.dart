import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/drafts/drafts_page.dart';
import 'package:forum_app/src/pages/topic/post_editor.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/fixtures/campus_fixtures.dart';
import '../test/fixtures/page_fixtures.dart';

/// Real native encrypted user-work storage and HTTP, synthetic login fixture.
/// Only the owning test simulator and an ephemeral loopback origin are used.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    '401 teardown keeps a reply edit for manual same-account recovery',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      var writes = 0;
      var accepted = false;
      final unexpected = <String>[];
      final bodies = <Map<String, dynamic>>[];
      server.listen((request) async {
        if (request.headers.value(HttpHeaders.authorizationHeader) !=
            'Bearer synthetic-session-A') {
          unexpected.add('incorrect owner credential');
        }
        request.response.headers.contentType = ContentType.json;
        if (request.method == 'GET' && request.uri.path == '/drafts') {
          request.response.write(
            jsonEncode({
              'component': PageComponent.drafts,
              'url': '/drafts',
              'version': '1',
              'layout': {
                ...minimalLayoutJson(),
                'viewer': {
                  'id': 1,
                  'isAuthenticated': true,
                  'username': 'synthetic-A',
                },
              },
              'meta': {'title': 'Drafts'},
              'props': {
                'total': 0,
                'drafts': [],
                'pagination': {
                  'page': 1,
                  'nextPage': 0,
                  'hasNext': false,
                  'nextUrl': '',
                },
              },
            }),
          );
        } else if (request.method == 'POST' &&
            request.uri.path == '/api/forum/posts/update') {
          writes++;
          final body =
              jsonDecode(await utf8.decoder.bind(request).join())
                  as Map<String, dynamic>;
          bodies.add(body);
          request.response.statusCode = accepted ? 200 : 401;
          request.response.write(
            jsonEncode(
              accepted
                  ? {
                      'code': 0,
                      'result': {
                        'id': 42,
                        'content': body['content'],
                        'renderedContent': '',
                        'updatedAt': '',
                      },
                    }
                  : {'code': 1, 'messageCode': 'auth.required'},
            ),
          );
        } else {
          unexpected.add('${request.method} ${request.uri}');
          request.response.statusCode = 404;
        }
        await request.response.close();
      });
      final site = 'http://127.0.0.1:${server.port}';
      final scope = writingScope(site, 1);
      for (final locale in ['zh', 'en', 'ja', 'de']) {
        for (final brightness in Brightness.values) {
          accepted = false;
          var signedIn = true;
          final tokens = CampusMemoryTokenStorage()
            ..value = 'synthetic-session-A';
          late ProviderContainer container;
          late GoRouter router;
          final dio = Dio();
          final client = GfApiClient(
            dio: dio,
            tokenStorage: tokens,
            baseUrl: site,
            onUnauthorized: () {
              // Match the production boundary: epoch invalidates before the HTTP
              // Future throws, then the old route is replaced by sign-in.
              signedIn = false;
              container.read(offlineCacheEpochProvider.notifier).invalidate();
              unawaited(tokens.clear());
              router.go('/login');
            },
          );
          router = GoRouter(
            routes: [
              GoRoute(
                path: '/',
                builder: (context, _) => Scaffold(
                  body: Center(
                    child: TextButton(
                      onPressed: () => showPostEditor(
                        context,
                        postId: 42,
                        content: 'server original',
                      ),
                      child: const Text('Open editor'),
                    ),
                  ),
                ),
              ),
              GoRoute(
                path: '/login',
                builder: (_, _) => Scaffold(
                  body: Center(
                    child: TextButton(
                      onPressed: () async {
                        await tokens.write('synthetic-session-A');
                        signedIn = true;
                        accepted = true;
                        container
                            .read(offlineCacheEpochProvider.notifier)
                            .invalidate();
                        router.go('/drafts');
                      },
                      child: const Text('Sign in as A fixture'),
                    ),
                  ),
                ),
              ),
              GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
            ],
          );
          container = ProviderContainer(
            overrides: [
              apiClientProvider.overrideWithValue(client),
              tokenStorageProvider.overrideWithValue(tokens),
              currentUserProvider.overrideWith((ref) async {
                ref.watch(offlineCacheEpochProvider);
                return signedIn
                    ? const CurrentUser(id: 1, username: 'synthetic-A')
                    : null;
              }),
            ],
          );
          final store = container.read(writingStoreProvider);
          await store.clearAccount(scope);
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
          final content = 'Unsent reply $locale ${brightness.name}';
          await tester.enterText(field, content);
          final l = AppLocalizations.of(
            tester.element(find.byType(PostEditor)),
          );
          final before = writes;
          await tester.tap(find.text(l.commonSave));
          await tester.pumpAndSettle();
          expect(router.state.uri.path, '/login');
          expect(writes, before + 1);
          expect((await store.drafts(scope)).single.content, content);
          expect(await tokens.read(), isNull);
          await tester.tap(find.text('Sign in as A fixture'));
          await tester.pumpAndSettle();
          final row = find.byKey(ValueKey(postEditDraftKey(42)));
          expect(row, findsOneWidget);
          await tester.tap(row);
          await tester.pumpAndSettle();
          expect(tester.widget<TextField>(field).controller!.text, content);
          expect(writes, before + 1);
          expect(tester.takeException(), isNull);
          if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
            await binding.takeScreenshot(
              'session-draft-$locale-${brightness.name}',
            );
          }
          // A second write is explicit; merely logging in/reopening never sends.
          await tester.tap(find.text(l.commonSave));
          await tester.pumpAndSettle();
          expect(find.byType(PostEditor), findsNothing);
          expect(writes, before + 2);
          expect(bodies.last['postId'], 42);
          expect(bodies.last['content'], content);
          expect(await store.drafts(scope), isEmpty);
          await tester.pumpWidget(const SizedBox());
          router.dispose();
          container.dispose();
          dio.close(force: true);
        }
      }
      expect(unexpected, isEmpty);
      await server.close(force: true);
    },
  );
}
