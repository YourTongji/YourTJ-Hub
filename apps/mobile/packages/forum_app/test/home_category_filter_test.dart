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
import 'package:forum_app/src/widgets/cache_snapshot_hint.dart';
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
  bool longCategoryRail = false;
  bool hotMoment = false;

  PagePayload payload({
    bool category = false,
    String sort = 'latest',
    bool next = false,
  }) {
    final data = homePayloadJson();
    final props = data['props'] as Map<String, dynamic>;
    final first = (props['topics'] as List).first as Map<String, dynamic>;
    (data['layout']['sidebar'] as Map)['categories'] = [
      if (longCategoryRail)
        for (var id = 3; id <= 15; id++)
          {
            'id': id,
            'label': '分类 $id',
            'url': '/c/category/$id',
            'color': '#888888',
          },
      {'id': 1, 'label': '开发', 'url': '/c/dev/1', 'color': '#2563eb'},
      {'id': 2, 'label': '校园生活', 'url': '/c/life/2', 'color': '#888888'},
    ];
    props['sort'] = sort;
    props['topics'] = [
      for (var i = 0; i < (next ? 2 : 25); i++)
        {
          ...first,
          'id': (category ? 1000 : 100) + (next ? 25 : 0) + i,
          'title': hotMoment && sort == 'hot'
              ? ''
              : '${category ? 'Category' : 'All'} $sort ${next ? 25 + i : i}',
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

Future<GoRouter> _mount(
  WidgetTester tester,
  _Pages pages, {
  Size size = const Size(600, 800),
  double textScale = 1,
  VoidCallback? onScroll,
  bool tickersEnabled = false,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  SharedPreferences.setMockInitialValues({});
  final router = GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (_, _) => NotificationListener<ScrollNotification>(
          onNotification: (_) {
            onScroll?.call();
            return false;
          },
          child: TickerMode(enabled: tickersEnabled, child: const HomePage()),
        ),
      ),
      GoRoute(
        path: '/search',
        builder: (_, _) => const Scaffold(body: Text('search-destination')),
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
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(textScale)),
          child: child!,
        ),
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
  await tester.tap(find.byKey(const ValueKey('home-category-1')));
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
  testWidgets(
    'wide discovery drops deleted topics without waiting for a fresh hot feed',
    (tester) async {
      final pages = _Pages();
      await _mount(tester, pages, size: const Size(1120, 900));
      final panel = find.byKey(const ValueKey('home-discovery-panel'));
      final hot = find.descendant(of: panel, matching: find.text('All hot 0'));
      expect(hot, findsOneWidget);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(HomePage)),
      );
      final calls = List<String>.of(pages.calls);
      container.read(removedTopicsProvider.notifier).add(100);
      await tester.pumpAndSettle();
      expect(hot, findsNothing);
      expect(
        find.descendant(of: panel, matching: find.text('All hot 1')),
        findsOneWidget,
      );
      expect(
        pages.calls,
        calls,
        reason: 'the deletion fence applies to the cached hot response',
      );
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('feed toggle shows the destination layout glyph', (tester) async {
    await _mount(tester, _Pages());
    for (final (label, symbol) in [
      ('切换为列表', 'list-square'),
      ('切换为卡片', 'layout-grid'),
    ]) {
      final button = find.byTooltip(label);
      expect(
        find.descendant(
          of: button,
          matching: find.byWidgetPredicate(
            (widget) => widget is GfSymbol && widget.name == symbol,
          ),
        ),
        findsOneWidget,
      );
      await tester.tap(button);
      await tester.pumpAndSettle();
      expect(find.byType(BottomSheet), findsNothing);
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'discovery retains untitled moments and respects cache clearing',
    (tester) async {
      final pages = _Pages()..hotMoment = true;
      await _mount(tester, pages, size: const Size(1120, 900));
      final panel = find.byKey(const ValueKey('home-discovery-panel'));
      expect(
        find.descendant(of: panel, matching: find.text('这是一条用于 smoke 测试的话题摘要')),
        findsNWidgets(5),
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(HomePage)),
      );
      final calls = List<String>.of(pages.calls);
      container.read(cacheClearEpochProvider.notifier).invalidate({
        CacheCategory.forum,
      });
      await tester.pumpAndSettle();
      expect(pages.calls, calls);
      expect(
        find.descendant(of: panel, matching: find.byType(CacheSnapshotHint)),
        findsOneWidget,
      );
      expect(
        find.descendant(of: panel, matching: find.text('这是一条用于 smoke 测试的话题摘要')),
        findsNothing,
      );
    },
  );

  testWidgets(
    'tablet discovery uses real hot feed and preserves reading on resize',
    (tester) async {
      final pages = _Pages();
      final router = await _mount(tester, pages, size: const Size(1120, 900));
      final panel = find.byKey(const ValueKey('home-discovery-panel'));
      expect(panel, findsOneWidget);
      expect(pages.calls.where((call) => call == 'home:hot'), hasLength(1));
      expect(find.text('All hot 0'), findsOneWidget);
      _position(tester).jumpTo(210);
      await tester.pumpAndSettle();
      final position = _position(tester);
      final offset = position.pixels;
      for (final width in [600.0, 1120.0]) {
        tester.view.physicalSize = Size(width, 900);
        await tester.pumpAndSettle();
        expect(_position(tester), same(position));
        expect(position.pixels, offset);
        expect(pages.calls.where((call) => call == 'home:hot'), hasLength(1));
        expect(tester.takeException(), isNull);
      }
      await tester.tap(find.byKey(const ValueKey('home-discovery-search')));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/search');
    },
  );

  testWidgets(
    'tablet categories stay above the feed and resize retains reading state',
    (tester) async {
      final pages = _Pages();
      final router = await _mount(tester, pages, size: const Size(1000, 900));
      final rail = find.byKey(const ValueKey('home-category-rail'));
      expect(find.byKey(const ValueKey('home-category-sidebar')), findsNothing);
      expect(rail, findsOneWidget);
      expect(tester.getRect(rail).width, lessThanOrEqualTo(720));
      expect(tester.getRect(rail).height, lessThan(100));
      await _chooseCategory(tester);
      _position(tester).jumpTo(210);
      await tester.pumpAndSettle();
      final position = _position(tester);
      final offset = position.pixels;
      final calls = List<String>.of(pages.calls);
      for (final width in [600.0, 1000.0]) {
        tester.view.physicalSize = Size(width, 900);
        await tester.pumpAndSettle();
        expect(
          find.byKey(const ValueKey('home-category-sidebar')),
          findsNothing,
        );
        expect(rail, findsOneWidget);
        expect(
          tester.getRect(find.byType(GfTopicList)).width,
          lessThanOrEqualTo(720),
        );
        expect(_position(tester), same(position));
        expect(_position(tester).pixels, closeTo(offset, 1));
        expect(_feed(tester).topics.first.title, 'Category latest 0');
        expect(pages.calls, calls);
      }
      expect(router.state.uri.path, '/');
      await tester.tap(find.byKey(const ValueKey('home-category-all')));
      await tester.pumpAndSettle();
      expect(_feed(tester).topics.first.title, 'All latest 0');
    },
  );

  testWidgets(
    'large text keeps a readable single feed rather than squeezing it',
    (tester) async {
      await _mount(tester, _Pages(), size: const Size(1000, 900), textScale: 2);
      expect(find.byKey(const ValueKey('home-category-sidebar')), findsNothing);
      expect(find.byKey(const ValueKey('home-category-rail')), findsOneWidget);
      expect(tester.getRect(find.byType(GfTopicList)).width, 720);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('scrolling tablet categories leaves reading chrome unchanged', (
    tester,
  ) async {
    final pages = _Pages()..longCategoryRail = true;
    var scrollNotifications = 0;
    await _mount(
      tester,
      pages,
      size: const Size(1000, 500),
      onScroll: () => scrollNotifications++,
      tickersEnabled: true,
    );
    final rail = find.byKey(const ValueKey('home-category-rail'));
    scrollNotifications = 0;
    final feedPosition = _position(tester);
    final offset = feedPosition.pixels;
    await tester.drag(rail, const Offset(-350, 0));
    await tester.pumpAndSettle();
    final categoryPosition = tester
        .state<ScrollableState>(
          find.descendant(of: rail, matching: find.byType(Scrollable)),
        )
        .position;
    expect(categoryPosition.pixels, greaterThan(0));
    expect(scrollNotifications, 0);
    expect(feedPosition.pixels, offset);
    await tester.drag(find.byType(GfTopicList), const Offset(0, -150));
    await tester.pumpAndSettle();
    expect(scrollNotifications, greaterThan(0));
    expect(feedPosition.pixels, greaterThan(offset));
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'visible category rail selects and filters home without pushing a route',
    (tester) async {
      final pages = _Pages();
      final router = await _mount(tester, pages);
      expect(find.byKey(const ValueKey('home-category-rail')), findsOneWidget);
      expect(find.byType(BottomSheet), findsNothing);
      await _chooseCategory(tester);
      final selected = tester.widget<Semantics>(
        find
            .descendant(
              of: find.byKey(const ValueKey('home-category-1')),
              matching: find.byWidgetPredicate(
                (widget) =>
                    widget is Semantics && widget.properties.selected == true,
              ),
            )
            .first,
      );
      expect(selected.properties.selected, isTrue);
      expect(router.state.uri.path, '/');
      expect(pages.calls, contains('/c/dev/1'));
      expect(_feed(tester).topics.first.title, 'Category latest 0');
      expect(find.text('separate-category-route'), findsNothing);
      final tabs = tester.widget<GfTabBar>(find.byType(GfTabBar));
      expect(tabs.tabs.map((tab) => tab.value), ['latest', 'new']);
    },
  );

  testWidgets('topic category chip filters in place', (tester) async {
    final pages = _Pages();
    final router = await _mount(tester, pages);
    await tester.tap(
      find.descendant(
        of: find.byType(GfTopicCard).first,
        matching: find.text('开发'),
      ),
    );
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
    'a topic chip reveals its offscreen category without later recentering',
    (tester) async {
      tester.view.physicalSize = const Size(320, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final pages = _Pages()..longCategoryRail = true;
      await _mount(tester, pages, size: const Size(320, 800));
      final rail = find.byKey(const ValueKey('home-category-rail'));
      final selected = find.byKey(const ValueKey('home-category-1'));
      expect(selected.hitTestable(), findsNothing);
      await tester.tap(
        find.descendant(
          of: find.byType(GfTopicCard).first,
          matching: find.text('开发'),
        ),
      );
      await tester.pumpAndSettle();
      expect(selected.hitTestable(), findsOneWidget);
      final viewport = tester.getRect(rail);
      final selectedRect = tester.getRect(selected);
      expect(selectedRect.left, greaterThanOrEqualTo(viewport.left));
      expect(selectedRect.right, lessThanOrEqualTo(viewport.right));
      expect(_feed(tester).topics.first.title, 'Category latest 0');
      final horizontal = tester
          .state<ScrollableState>(
            find.descendant(of: rail, matching: find.byType(Scrollable)),
          )
          .position;
      horizontal.jumpTo(80);
      await tester.pumpAndSettle();
      tester.widget<GfTabBar>(find.byType(GfTabBar)).onSelected('new');
      await tester.pumpAndSettle();
      expect(horizontal.pixels, closeTo(80, .1));
      expect(_feed(tester).topics.first.title, 'Category new 0');
    },
  );

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
      await tester.tap(find.byKey(const ValueKey('home-category-all')));
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
    await tester.tap(find.byKey(const ValueKey('home-category-all')));
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
    await tester.tap(find.byKey(const ValueKey('home-category-all')));
    await tester.pumpAndSettle();
    pending.complete(pages.payload(category: true));
    await tester.pumpAndSettle();
    expect(_feed(tester).topics.first.title, 'All latest 0');
    expect(find.text('Category latest 0'), findsNothing);
    expect(
      find.descendant(
        of: find.byKey(const ValueKey('home-category-all')),
        matching: find.byWidgetPredicate(
          (widget) => widget is Semantics && widget.properties.selected == true,
        ),
      ),
      findsOneWidget,
    );
    expect(
      tester
          .widget<GfTabBar>(find.byType(GfTabBar))
          .tabs
          .map((tab) => tab.value),
      ['latest', 'hot'],
    );
  });
}
