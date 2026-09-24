import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/drafts/drafts_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Pages extends PageRepository {
  _Pages(super.client);
  bool fail = false;
  @override
  Future<PagePayload> fetch(String path) async {
    if (fail) throw const NetworkException(fallbackMessage: 'offline');
    return parsePayload({
      'component': PageComponent.drafts,
      'url': '/drafts',
      'version': '1',
      'layout': minimalLayoutJson(),
      'meta': {'title': 'Drafts'},
      'props': {
        'total': 1,
        'drafts': [
          {
            'id': 42,
            'title': '云端未完成内容',
            'description': '云端正文',
            'editUrl': '/publish?id=42',
            'replyCount': 0,
            'viewCount': 0,
            'processStatus': 0,
            'createdAt': '2026-09-24',
            'updatedAt': '2026-09-24',
            'categories': [],
          },
        ],
        'pagination': {
          'page': 1,
          'nextPage': 0,
          'hasNext': false,
          'nextUrl': '',
        },
      },
    });
  }
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  testWidgets(
    'drafts share one scroll surface and failed refresh keeps cloud content',
    (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final pages = _Pages(client);
      final store = WritingStore();
      await store.save(
        writingScope('http://fake.local', 1),
        const LocalDraft(
          key: 'reply-100',
          kind: DraftKind.reply,
          topicId: 100,
          title: '回复中的话题',
          content: '本机回复正文',
          contentType: 2,
          categories: [],
          images: [],
          updatedAt: 1,
        ),
      );
      final router = GoRouter(
        initialLocation: '/drafts',
        routes: [
          GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
          GoRoute(
            path: '/p/:id',
            builder: (_, state) => Scaffold(
              body: Text('reply-topic-${state.pathParameters['id']}'),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            apiClientProvider.overrideWithValue(client),
            pageRepositoryProvider.overrideWithValue(pages),
            writingStoreProvider.overrideWithValue(store),
            currentUserProvider.overrideWith(
              (ref) async => const CurrentUser(id: 1, username: 'alice'),
            ),
          ],
          child: MaterialApp.router(
            routerConfig: router,
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(CustomScrollView), findsOneWidget);
      expect(find.byType(ListView), findsNothing);
      expect(find.text('本机回复正文'), findsOneWidget);
      expect(find.text('云端未完成内容'), findsOneWidget);
      pages.fail = true;
      await tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pumpAndSettle();
      expect(find.text('云端未完成内容'), findsOneWidget);
      expect(find.text('本机回复正文'), findsOneWidget);
      expect(find.text('重试'), findsOneWidget);
      pages.fail = false;
      await tester.tap(find.text('重试'));
      await tester.pumpAndSettle();
      expect(find.text('重试'), findsNothing);
      await tester.tap(find.text('回复中的话题'));
      await tester.pumpAndSettle();
      expect(find.text('reply-topic-100'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
