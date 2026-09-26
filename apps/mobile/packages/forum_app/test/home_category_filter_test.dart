import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String value) async {}
  @override
  Future<void> clear() async {}
}

class _NoopCache implements OfflineTopicCache {
  @override
  Future<void> put(int id, Map<String, dynamic> payload) async {}
  @override
  Future<PagePayload?> get(int id) async => null;
  @override
  Future<void> clear() async {}
  @override
  Future<void> close() async {}
}

class _Pages extends PageRepository {
  _Pages()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Tokens(),
          baseUrl: 'http://fake',
        ),
      );
  final calls = <String>[];
  Completer<PagePayload>? pendingCategory;
  bool failCategory = false;
  bool categoryHasNext = false;

  PagePayload payload({
    bool category = false,
    String sort = 'latest',
    bool next = false,
  }) {
    final data = homePayloadJson();
    final props = data['props'] as Map<String, dynamic>;
    final first = (props['topics'] as List).first as Map<String, dynamic>;
    (data['layout']['sidebar'] as Map)['categories'] = [
      {'id': 1, 'label': '开发', 'url': '/c/dev/1', 'color': '#2563eb'},
      {'id': 2, 'label': '校园生活', 'url': '/c/life/2', 'color': '#888888'},
    ];
    props['sort'] = sort;
    props['topics'] = [
      for (var i = 0; i < (next ? 2 : 25); i++)
        {
          ...first,
          'id': (category ? 1000 : 100) + (next ? 25 : 0) + i,
          'title':
              '${category ? 'Category' : 'All'} $sort ${next ? 25 + i : i}',
        },
    ];
    props['pagination'] = {
      'page': next ? 2 : 1,
      'nextPage': next ? 0 : 2,
      'hasNext': category && categoryHasNext && !next,
      'nextUrl': category && categoryHasNext && !next
          ? '/c/dev/1?page=2&cursor=server-token'
          : '',
    };
    if (category) {
      data['component'] = PageComponent.category;
      data['url'] = '/c/dev/1';
      props.remove('announcement');
      props['category'] = {
        'id': 1,
        'name': '开发',
        'description': '',
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
  Future<PagePayload> home({String sort = '', Object? cancelToken}) async {
    calls.add('home:$sort');
    return payload(sort: sort.isEmpty ? 'latest' : sort);
  }

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    calls.add(path);
    if (path.startsWith('/c/')) {
      if (failCategory) throw StateError('category unavailable');
      if (pendingCategory != null) return pendingCategory!.future;
      return payload(
        category: true,
        sort: path.contains('/l/new') ? 'new' : 'latest',
        next: path.contains('page=2'),
      );
    }
    throw StateError('unexpected fetch $path');
  }
}

Future<GoRouter> _mount(WidgetTester tester, _Pages pages) async {
  SharedPreferences.setMockInitialValues({});
  final router = GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (_, _) => const TickerMode(enabled: false, child: HomePage()),
      ),
      GoRoute(
        path: '/c/:slug/:id',
        builder: (_, _) =>
            const Scaffold(body: Text('separate-category-route')),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        pageRepositoryProvider.overrideWithValue(pages),
        currentUserProvider.overrideWith((ref) async => null),
        accountLayoutProvider.overrideWith(
          (ref) async => pages.payload().layout,
        ),
        offlineTopicCacheProvider.overrideWithValue(_NoopCache()),
        tokenStorageProvider.overrideWithValue(_Tokens()),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        theme: gfThemeData(Brightness.light),
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

Future<void> _chooseCategory(WidgetTester tester, {bool settle = true}) async {
  await tester.tap(find.byTooltip('分类与显示'));
  await tester.pumpAndSettle();
  await tester.tap(
    find.descendant(of: find.byType(BottomSheet), matching: find.text('开发')),
  );
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
  }
}

GfTopicList _feed(WidgetTester tester) =>
    tester.widget<GfTopicList>(find.byType(GfTopicList));
ScrollPosition _position(WidgetTester tester) => tester
    .state<ScrollableState>(
      find
          .descendant(
            of: find.byType(GfTopicList),
            matching: find.byType(Scrollable),
          )
          .first,
    )
    .position;

void main() {
  testWidgets('category option filters home without pushing a category route', (
    tester,
  ) async {
    final pages = _Pages();
    final router = await _mount(tester, pages);
    await _chooseCategory(tester);
    expect(router.state.uri.path, '/');
    expect(pages.calls, contains('/c/dev/1'));
    expect(_feed(tester).topics.first.title, 'Category latest 0');
    expect(find.text('separate-category-route'), findsNothing);
    final tabs = tester.widget<GfTabBar>(find.byType(GfTabBar));
    expect(tabs.tabs.map((tab) => tab.value), ['latest', 'new']);
  });

  testWidgets('topic category chip filters in place', (tester) async {
    final pages = _Pages();
    final router = await _mount(tester, pages);
    await tester.tap(find.text('开发').first);
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/');
    expect(pages.calls, contains('/c/dev/1'));
    expect(_feed(tester).topics.first.title, 'Category latest 0');
    expect(
      find.descendant(of: find.byType(GfTopicCard), matching: find.text('开发')),
      findsNothing,
    );
  });

  testWidgets(
    'clearing category restores the all-feed content and reading position',
    (tester) async {
      final pages = _Pages();
      await _mount(tester, pages);
      _position(tester).jumpTo(420);
      await tester.pumpAndSettle();
      final before = _position(tester).pixels;
      await _chooseCategory(tester);
      _position(tester).jumpTo(210);
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('全部分类'));
      await tester.pumpAndSettle();
      expect(_feed(tester).topics.first.title, 'All latest 0');
      expect(_position(tester).pixels, closeTo(before, 1));
      expect(
        pages.calls.where((path) => path.startsWith('home:')),
        hasLength(1),
      );
    },
  );

  testWidgets('category sort and pagination use server category URLs', (
    tester,
  ) async {
    final pages = _Pages()..categoryHasNext = true;
    await _mount(tester, pages);
    tester.widget<GfTabBar>(find.byType(GfTabBar)).onSelected('hot');
    await tester.pumpAndSettle();
    await _chooseCategory(tester);
    _feed(tester).onLoadMore();
    await tester.pumpAndSettle();
    expect(pages.calls, contains('/c/dev/1?page=2&cursor=server-token'));
    expect(_feed(tester).topics, hasLength(27));
    tester.widget<GfTabBar>(find.byType(GfTabBar)).onSelected('new');
    await tester.pumpAndSettle();
    expect(pages.calls, contains('/c/dev/1/l/new'));
    expect(_feed(tester).topics.first.title, 'Category new 0');
    await tester.tap(find.byTooltip('全部分类'));
    await tester.pumpAndSettle();
    expect(_feed(tester).topics.first.title, 'All hot 0');
  });

  testWidgets(
    'category load failure retries the category rather than the all feed',
    (tester) async {
      final pages = _Pages()..failCategory = true;
      await _mount(tester, pages);
      await _chooseCategory(tester);
      expect(find.byType(GfErrorRetry), findsOneWidget);
      pages.failCategory = false;
      tester.widget<GfErrorRetry>(find.byType(GfErrorRetry)).onRetry();
      await tester.pumpAndSettle();
      expect(_feed(tester).topics.first.title, 'Category latest 0');
      expect(pages.calls.where((path) => path == '/c/dev/1'), hasLength(2));
    },
  );

  testWidgets('late category response does not replace the restored all feed', (
    tester,
  ) async {
    final pending = Completer<PagePayload>();
    final pages = _Pages()..pendingCategory = pending;
    await _mount(tester, pages);
    await _chooseCategory(tester, settle: false);
    await tester.tap(find.byTooltip('全部分类'));
    await tester.pumpAndSettle();
    pending.complete(pages.payload(category: true));
    await tester.pumpAndSettle();
    expect(_feed(tester).topics.first.title, 'All latest 0');
    expect(find.text('Category latest 0'), findsNothing);
    expect(find.byTooltip('全部分类'), findsNothing);
    expect(
      tester
          .widget<GfTabBar>(find.byType(GfTabBar))
          .tabs
          .map((tab) => tab.value),
      ['latest', 'hot'],
    );
  });
}
