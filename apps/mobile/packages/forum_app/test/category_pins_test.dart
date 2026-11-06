import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/category/category_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage, NoopOfflineCache;

/// Synthetic category content, shared with native visual acceptance.
class CategoryPinsRepository extends PageRepository {
  CategoryPinsRepository({this.pins = 2, this.hasNext = false})
    : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage()));

  final int pins;
  final bool hasNext;
  final calls = <String>[];

  PagePayload payload({
    bool category = false,
    bool next = false,
    String sort = 'latest',
  }) {
    final data = homePayloadJson();
    final props = data['props'] as Map<String, dynamic>;
    final first = (props['topics'] as List).first as Map<String, dynamic>;
    (data['layout']['sidebar'] as Map)['categories'] = [
      {'id': 1, 'label': '开发', 'url': '/c/dev/1', 'color': '#2563eb'},
    ];
    props['sort'] = sort;
    props['topics'] = [
      for (final id in next ? [102, 104] : [101, 102, 103])
        {
          ...first,
          'id': id,
          'title': id <= 100 + pins ? '置顶公告 ${id - 100}（演示）' : '校园话题 $id（演示）',
          'description': '分类置顶验收使用的合成内容',
          'url': '/p/post/$id',
          'pinWeight': id <= 100 + pins ? 104 - id : 0,
        },
    ];
    props['pagination'] = {
      'page': next ? 2 : 1,
      'nextPage': 2,
      'hasNext': hasNext && !next,
      'nextUrl': '/c/dev/1?page=2',
    };
    if (category) {
      data['component'] = PageComponent.category;
      props.remove('announcement');
      props['category'] = {
        'id': 1,
        'name': '开发',
        'description': '分类置顶 · 合成验收数据',
        'icon': '',
        'color': '#2563eb',
        'url': '/c/dev/1',
      };
      props['tabs'] = [
        {
          'key': 'latest',
          'label': '最新回复',
          'url': '/c/dev/1',
          'active': sort == 'latest',
        },
        {
          'key': 'new',
          'label': '新主题',
          'url': '/c/dev/1/l/new',
          'active': sort == 'new',
        },
      ];
    }
    return parsePayload(data);
  }

  @override
  Future<PagePayload> home({String sort = '', Object? cancelToken}) async =>
      payload();

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    calls.add(path);
    return payload(
      category: true,
      next: path.contains('page=2'),
      sort: path.contains('/l/new') ? 'new' : 'latest',
    );
  }
}

GoRouter categoryPinsRouter({bool home = false, bool tickersEnabled = false}) =>
    GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => home
              ? TickerMode(enabled: tickersEnabled, child: const HomePage())
              : const CategoryPage(slug: 'dev', categoryId: 1),
        ),
      ],
    );

Widget categoryPinsApp(
  CategoryPinsRepository pages,
  GoRouter router,
  Brightness brightness,
) => ProviderScope(
  overrides: [
    pageRepositoryProvider.overrideWithValue(pages),
    currentUserProvider.overrideWith((_) async => null),
    tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
    offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
    accountLayoutProvider.overrideWith((_) async => pages.payload().layout),
  ],
  child: MaterialApp.router(
    theme: gfThemeData(brightness),
    routerConfig: router,
    locale: const Locale('zh'),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
  ),
);

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  Future<void> mount(
    WidgetTester tester,
    CategoryPinsRepository pages, {
    bool home = false,
  }) async {
    final router = categoryPinsRouter(home: home);
    addTearDown(router.dispose);
    await tester.pumpWidget(categoryPinsApp(pages, router, Brightness.light));
    await tester.pumpAndSettle();
  }

  testWidgets(
    'category latest groups pins and expansion leaves regular cards intact',
    (tester) async {
      await mount(tester, CategoryPinsRepository());
      expect(find.byKey(const ValueKey('pinned-topic-strip')), findsOneWidget);
      expect(find.text('置顶公告 1（演示）'), findsOneWidget);
      expect(find.text('置顶公告 2（演示）'), findsNothing);
      expect(find.byType(GfTopicCard), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey('pinned-strip-line')));
      await tester.pumpAndSettle();
      expect(find.text('置顶公告 2（演示）'), findsOneWidget);
      expect(find.byType(GfTopicCard), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey('pinned-strip-line')));
      await tester.pumpAndSettle();
      expect(find.text('置顶公告 2（演示）'), findsNothing);
    },
  );

  testWidgets('category pagination does not duplicate pins or regular topics', (
    tester,
  ) async {
    final pages = CategoryPinsRepository(hasNext: true);
    await mount(tester, pages);
    tester.widget<GfTopicList>(find.byType(GfTopicList)).onLoadMore();
    await tester.pumpAndSettle();
    final topics = tester.widget<GfTopicList>(find.byType(GfTopicList)).topics;
    expect(topics.map((topic) => topic.id), [101, 102, 103, 104]);
    expect(pages.calls, contains('/c/dev/1?page=2'));
  });

  for (final pins in [0, 1, 3]) {
    testWidgets('category handles $pins pins without hiding ordinary content', (
      tester,
    ) async {
      await mount(tester, CategoryPinsRepository(pins: pins));
      expect(
        find.byKey(const ValueKey('pinned-topic-strip')),
        pins == 0 ? findsNothing : findsOneWidget,
      );
      expect(find.byType(GfTopicCard), findsNWidgets(3 - pins));
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets(
    'Home category latest groups pins while new keeps its ordered rows',
    (tester) async {
      await mount(tester, CategoryPinsRepository(), home: true);
      await tester.tap(find.byKey(const ValueKey('home-category-1')));
      await tester.pumpAndSettle();
      expect(
        tester.widget<GfTopicList>(find.byType(GfTopicList)).collapsePinned,
        isTrue,
      );
      expect(find.byKey(const ValueKey('pinned-topic-strip')), findsOneWidget);
      tester.widget<GfTabBar>(find.byType(GfTabBar)).onSelected('new');
      await tester.pumpAndSettle();
      expect(
        tester.widget<GfTopicList>(find.byType(GfTopicList)).collapsePinned,
        isFalse,
      );
      expect(find.byKey(const ValueKey('pinned-topic-strip')), findsNothing);
      expect(find.byType(GfTopicCard), findsNWidgets(3));
    },
  );
}
