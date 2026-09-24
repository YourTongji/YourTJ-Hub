import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'session';
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

GfApiClient _client() =>
    GfApiClient(dio: Dio(), tokenStorage: _Tokens(), baseUrl: 'http://fake');

class _Pages extends PageRepository {
  _Pages() : super(_client());
  bool liked = true;
  int likeCount = 5;
  bool bookmarked = true;
  bool known = true;
  Completer<PagePayload>? pending;
  PagePayload payload() {
    final data = homePayloadJson();
    final props = data['props'] as Map<String, dynamic>;
    final first = (props['topics'] as List).first as Map<String, dynamic>;
    props['topics'] = [
      for (var i = 0; i < 40; i++)
        {
          ...first,
          'id': 100 + i,
          'title': 'Topic $i',
          'likeCount': likeCount,
          if (known) 'liked': liked,
          if (known) 'bookmarked': bookmarked,
        },
    ];
    return parsePayload(data);
  }

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async =>
      pending == null ? payload() : pending!.future;
}

/// 两级以上分页的 fake:home() 返回第一页,fetch() 按 nextUrl 返回后续页,
/// 用于验证「从详情返回」与「下拉刷新」两条静默路径的不同合并语义。
class _PagedPages extends _Pages {
  _PagedPages() : super();
  int homeCalls = 0;
  int page1LikeCount = 5;
  int page2LikeCount = 5;
  int page3LikeCount = 5;
  bool failHome = false;
  bool emptyHomeProps = false;

  PagePayload _topicPage(
    int id,
    int likeCount, {
    required bool hasNext,
    required String nextUrl,
  }) {
    final data = homePayloadJson();
    final props = data['props'] as Map<String, dynamic>;
    final first = (props['topics'] as List).first as Map<String, dynamic>;
    props['topics'] = [
      {
        ...first,
        'id': id,
        'title': 'Topic $id',
        'likeCount': likeCount,
        'liked': false,
        'bookmarked': false,
      },
    ];
    props['pagination'] = {
      'page': 1,
      'nextPage': hasNext ? 2 : 0,
      'hasNext': hasNext,
      'nextUrl': nextUrl,
    };
    return parsePayload(data);
  }

  @override
  Future<PagePayload> home({String sort = '', Object? cancelToken}) async {
    homeCalls++;
    if (failHome) throw StateError('home refresh failed');
    if (emptyHomeProps) {
      final data = homePayloadJson()..['props'] = <String, dynamic>{};
      return parsePayload(data);
    }
    return _topicPage(
      100,
      page1LikeCount,
      hasNext: true,
      nextUrl: '/?sort=latest&page=2',
    );
  }

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (path.endsWith('page=3')) {
      return _topicPage(300, page3LikeCount, hasNext: false, nextUrl: '');
    }
    if (path.endsWith('page=2')) {
      return _topicPage(
        200,
        page2LikeCount,
        hasNext: true,
        nextUrl: '/?sort=latest&page=3',
      );
    }
    return _topicPage(
      100,
      page1LikeCount,
      hasNext: true,
      nextUrl: '/?sort=latest&page=2',
    );
  }
}

/// page=2 首次请求可挂起的 fake:模拟「加载更多」在途期间发生返回刷新。
class _HangingPages extends _PagedPages {
  Completer<void>? gate;

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (path.endsWith('page=2') && gate != null) {
      final waiter = gate!;
      gate = null;
      await waiter.future;
    }
    return super.fetch(path);
  }
}

class _SortedPages extends _Pages {
  final requestTokens = <String, CancelToken?>{};
  final calls = <String, int>{};
  final nextCalls = <String>[];
  int latestTopicCount = 25;
  bool keepNextPage = false;
  bool sharedTopics = false;
  Completer<PagePayload>? hotRequest;
  Completer<PagePayload>? nextRequest;

  PagePayload sorted(String sort, {bool next = false, bool last = false}) {
    final data = homePayloadJson();
    final props = data['props'] as Map<String, dynamic>;
    final first = (props['topics'] as List).first as Map<String, dynamic>;
    props['sort'] = sort;
    props['topics'] = [
      for (var i = 0; i < (sort == 'latest' ? latestTopicCount : 25); i++)
        {
          ...first,
          'id':
              (sort == 'hot' && !sharedTopics ? 200 : 100) +
              (next ? 25 : 0) +
              i,
          'title': '$sort ${next ? 25 + i : i}',
          'liked': liked,
          'bookmarked': bookmarked,
          'likeCount': likeCount,
        },
    ];
    final hasMore = !next || (keepNextPage && !last);
    props['pagination'] = {
      'page': next ? 2 : 1,
      'nextPage': hasMore ? (next ? 3 : 2) : 0,
      'hasNext': hasMore,
      'nextUrl': hasMore ? '/?sort=$sort&page=${next ? 3 : 2}' : '',
    };
    return parsePayload(data);
  }

  @override
  Future<PagePayload> home({String sort = '', Object? cancelToken}) async {
    final key = sort.isEmpty ? 'latest' : sort;
    requestTokens[key] = cancelToken as CancelToken?;
    calls.update(key, (count) => count + 1, ifAbsent: () => 1);
    if (key == 'hot' && hotRequest != null) return hotRequest!.future;
    return sorted(key);
  }

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    requestTokens['page:$path'] = cancelToken as CancelToken?;
    nextCalls.add(path);
    final query = Uri.parse(path).queryParameters;
    if (query['page'] == '2' && nextRequest != null) return nextRequest!.future;
    return sorted(query['sort']!, next: true, last: query['page'] == '3');
  }
}

/// 详情页 fake:/p/post/ 返回详情 fixture,点赞状态可调。
class _DetailPages extends _Pages {
  bool detailLiked = false;

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (path.startsWith('/p/post/')) {
      final data = topicDetailPayloadJson();
      (data['props']['topic'] as Map<String, dynamic>)['isLiked'] = detailLiked;
      return parsePayload(data);
    }
    return super.fetch(path);
  }
}

/// no-op 离线缓存:详情页失败回退路径不触达真实 sqlite。
class _NoopOfflineCache implements OfflineTopicCache {
  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {}
  @override
  Future<PagePayload?> get(int topicId) async => null;
  @override
  Future<void> clear() async {}
  @override
  Future<void> close() async {}
}

class _Topics extends TopicRepository {
  _Topics(this.pages) : super(_client());
  final _Pages pages;
  final likes = <int>[];
  final bookmarks = <int>[];
  Completer<bool>? pending;
  @override
  Future<bool> likeTopic({required int topicId, required int action}) async {
    likes.add(action);
    final success = await (pending?.future ?? Future.value(true));
    if (success) pages.liked = action == 1;
    return success;
  }

  @override
  Future<bool> bookmarkTopic({
    required int topicId,
    required int action,
  }) async {
    bookmarks.add(action);
    pages.bookmarked = action == 1;
    return true;
  }
}

void main() {
  Future<ProviderContainer> pump(
    WidgetTester tester,
    _Pages pages,
    _Topics topics, {
    bool settle = true,
  }) async {
    SharedPreferences.setMockInitialValues({});
    final container = ProviderContainer(
      overrides: [
        pageRepositoryProvider.overrideWithValue(pages),
        topicRepositoryProvider.overrideWithValue(topics),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          // Isolate return/refresh transitions with explicit pagination;
          // foreground autoload is covered in list_footer/topic_list tests.
          home: TickerMode(
            enabled: pages is! _PagedPages,
            child: const HomePage(),
          ),
        ),
      ),
    );
    if (settle) {
      await tester.pumpAndSettle();
    } else {
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
    }
    return container;
  }

  testWidgets('server state and toggles survive card recycling', (
    tester,
  ) async {
    final pages = _Pages();
    final topics = _Topics(pages);
    await pump(tester, pages, topics);
    expect(find.byIcon(Icons.favorite), findsWidgets);
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('取消收藏').first);
    await tester.pumpAndSettle();
    final position = tester
        .state<ScrollableState>(find.byType(Scrollable).first)
        .position;
    position.jumpTo(position.maxScrollExtent);
    await tester.pumpAndSettle();
    position.jumpTo(0);
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('收藏').first);
    await tester.pumpAndSettle();
    expect(topics.likes, [2, 1]);
    expect(topics.bookmarks, [2, 1]);
  });

  testWidgets('sort tabs retain their own list, cursor and scroll offset', (
    tester,
  ) async {
    final pages = _SortedPages();
    await pump(tester, pages, _Topics(pages));
    // AccountAvatar also reads the home layout during the initial build.
    final initialLatestCalls = pages.calls['latest'];
    var list = tester.widget<GfTopicList>(find.byType(GfTopicList));
    list.onLoadMore();
    await tester.pumpAndSettle();
    list = tester.widget<GfTopicList>(find.byType(GfTopicList));
    expect(list.topics.length, 50);
    list.controller!.jumpTo(800);
    await tester.pump();
    await tester.tap(find.text('热门'));
    await tester.pumpAndSettle();
    expect(
      tester.widget<GfTopicList>(find.byType(GfTopicList)).topics.first.title,
      'hot 0',
    );
    await tester.tap(find.text('最新'));
    await tester.pumpAndSettle();
    list = tester.widget<GfTopicList>(find.byType(GfTopicList));
    expect(list.topics.length, 50);
    expect(list.hasMore, isFalse);
    expect(list.controller!.offset, 800);
    expect(pages.calls['latest'], initialLatestCalls);
  });

  testWidgets(
    'a pending sort keeps tabs usable and never replaces another sort',
    (tester) async {
      final pages = _SortedPages()..hotRequest = Completer<PagePayload>();
      await pump(tester, pages, _Topics(pages));
      await tester.tap(find.text('热门'));
      await tester.pump();
      expect(find.text('最新'), findsOneWidget);
      await tester.tap(find.text('最新'));
      await tester.pump();
      expect(find.text('latest 0'), findsOneWidget);
      expect(pages.requestTokens['hot']!.isCancelled, isFalse);
      pages.hotRequest!.complete(pages.sorted('hot'));
      await tester.pumpAndSettle();
      expect(find.text('latest 0'), findsOneWidget);
      expect(find.text('hot 0'), findsNothing);
      await tester.tap(find.text('热门'));
      await tester.pumpAndSettle();
      expect(find.text('hot 0'), findsOneWidget);
      expect(pages.calls['hot'], 1);
    },
  );

  testWidgets('pagination completes into its original sort after switching', (
    tester,
  ) async {
    final pages = _SortedPages()..nextRequest = Completer<PagePayload>();
    await pump(tester, pages, _Topics(pages));
    tester.widget<GfTopicList>(find.byType(GfTopicList)).onLoadMore();
    await tester.pump();
    await tester.tap(find.text('热门'));
    await tester.pumpAndSettle();
    expect(
      pages.requestTokens['page:/?sort=latest&page=2']!.isCancelled,
      isFalse,
    );
    pages.nextRequest!.complete(pages.sorted('latest', next: true));
    await tester.pumpAndSettle();
    expect(
      tester.widget<GfTopicList>(find.byType(GfTopicList)).topics.length,
      25,
    );
    await tester.tap(find.text('最新'));
    await tester.pumpAndSettle();
    expect(
      tester.widget<GfTopicList>(find.byType(GfTopicList)).topics.length,
      50,
    );
  });
  testWidgets('hidden sorts stop automatic pagination and resume on return', (
    tester,
  ) async {
    final pages = _SortedPages()
      ..latestTopicCount = 1
      ..keepNextPage = true
      ..nextRequest = Completer<PagePayload>();
    await pump(tester, pages, _Topics(pages), settle: false);
    expect(pages.nextCalls, ['/?sort=latest&page=2']);
    await tester.tap(find.text('热门'));
    await tester.pumpAndSettle();
    pages.nextRequest!.complete(pages.sorted('latest', next: true));
    await tester.pumpAndSettle();
    // This short hidden list has another cursor and space to fill, but no
    // request may begin until its TickerMode becomes active again.
    expect(pages.nextCalls, ['/?sort=latest&page=2']);
    await tester.tap(find.text('最新'));
    await tester.pumpAndSettle();
    expect(pages.nextCalls, ['/?sort=latest&page=2', '/?sort=latest&page=3']);
  });

  testWidgets(
    'older sort reads cannot erase a successful shared-topic action',
    (tester) async {
      final pages = _SortedPages()
        ..sharedTopics = true
        ..liked = false
        ..hotRequest = Completer<PagePayload>();
      final stale = pages.sorted('hot');
      final topics = _Topics(pages);
      await pump(tester, pages, topics);
      await tester.tap(find.text('热门'));
      await tester.pump();
      await tester.tap(find.text('最新'));
      await tester.pump();
      await tester.tap(find.byTooltip('点赞').first);
      await tester.pumpAndSettle();
      // A newer read may accept server data, but must not retire the mutation
      // fence needed by the older pending read in the other sort.
      pages.likeCount = 6;
      await tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pumpAndSettle();
      pages.hotRequest!.complete(stale);
      await tester.pumpAndSettle();
      await tester.tap(find.text('热门'));
      await tester.pumpAndSettle();
      final card = tester.widget<GfTopicCard>(find.byType(GfTopicCard).first);
      expect(card.liked, isTrue);
      expect(card.likeCount, 6);
      expect(topics.likes, [1]);
    },
  );

  testWidgets('detail return state survives an older read in another sort', (
    tester,
  ) async {
    final pages = _SortedPages()
      ..sharedTopics = true
      ..liked = false
      ..hotRequest = Completer<PagePayload>();
    final stale = pages.sorted('hot');
    final container = await pump(tester, pages, _Topics(pages));
    await tester.tap(find.text('热门'));
    await tester.pump();
    await tester.tap(find.text('最新'));
    await tester.pump();
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pumpAndSettle();
    // A subsequent detail visit returns a newer unlike and counters. Its
    // state must supersede both the earlier home action and the stale read.
    container.read(topicReturnStatesProvider)[100] = (
      unseen: false,
      liked: false,
      bookmarked: true,
      likeCount: 9,
      replyCount: 8,
      viewCount: 30,
    );
    pages
      ..liked = false
      ..likeCount = 9;
    tester.widget<GfTopicList>(find.byType(GfTopicList)).onReturnFromTopic!();
    await tester.pumpAndSettle();
    pages.hotRequest!.complete(stale);
    await tester.pumpAndSettle();
    await tester.tap(find.text('热门'));
    await tester.pumpAndSettle();
    final topic = tester
        .widget<GfTopicList>(find.byType(GfTopicList))
        .topics
        .first;
    expect(topic.liked, isFalse);
    expect(topic.likeCount, 9);
    expect(topic.replyCount, 8);
    expect(topic.viewCount, 30);
    expect(topic.bookmarked, isTrue);
  });

  testWidgets('late refresh cannot overwrite a successful action', (
    tester,
  ) async {
    final pages = _Pages()..liked = false;
    final topics = _Topics(pages);
    await pump(tester, pages, topics);
    final stale = pages.payload();
    pages.pending = Completer<PagePayload>();
    final refresh = tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pumpAndSettle();
    pages.pending!.complete(stale);
    await refresh;
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.favorite), findsOneWidget);
    pages.pending = null;
    pages.liked = false;
    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.favorite), findsNothing);
  });
  testWidgets('unknown server state never exposes a false toggle', (
    tester,
  ) async {
    final pages = _Pages()..known = false;
    await pump(tester, pages, _Topics(pages));
    expect(find.byTooltip('点赞'), findsNothing);
    expect(find.byTooltip('收藏'), findsNothing);
  });
  testWidgets('recycled cards do not send duplicate pending requests', (
    tester,
  ) async {
    final pages = _Pages()..liked = false;
    final topics = _Topics(pages)..pending = Completer<bool>();
    await pump(tester, pages, topics);
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pump();
    final position = tester
        .state<ScrollableState>(find.byType(Scrollable).first)
        .position;
    position.jumpTo(position.maxScrollExtent);
    await tester.pumpAndSettle();
    position.jumpTo(0);
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pump();
    expect(topics.likes, [1]);
    topics.pending!.complete(true);
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.favorite), findsOneWidget);
  });

  testWidgets('refresh while like is pending preserves its icon and count', (
    tester,
  ) async {
    final pages = _Pages()..liked = false;
    final topics = _Topics(pages)..pending = Completer<bool>();
    await pump(tester, pages, topics);
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pump();
    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.favorite), findsOneWidget);
    expect(
      tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).likeCount,
      6,
    );
    topics.pending!.complete(true);
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.favorite), findsOneWidget);
  });

  testWidgets(
    'failed like cannot discard a concurrent successful bookmark override',
    (tester) async {
      final pages = _Pages()
        ..liked = false
        ..bookmarked = false;
      final topics = _Topics(pages)..pending = Completer<bool>();
      await pump(tester, pages, topics);
      final stale = pages.payload();
      pages.pending = Completer<PagePayload>();
      final refresh = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.tap(find.byTooltip('点赞').first);
      await tester.pump();
      await tester.tap(find.byTooltip('收藏').first);
      await tester.pumpAndSettle();
      topics.pending!.complete(false);
      await tester.pumpAndSettle();
      pages.pending!.complete(stale);
      await refresh;
      await tester.pumpAndSettle();
      expect(find.byTooltip('取消收藏'), findsOneWidget);
    },
  );

  testWidgets('old-session failure cannot roll back the new viewer data', (
    tester,
  ) async {
    final pages = _Pages()..liked = false;
    final topics = _Topics(pages)..pending = Completer<bool>();
    final container = await pump(tester, pages, topics);
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pump();
    pages.liked = true;
    pages.likeCount = 20;
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();
    topics.pending!.complete(false);
    await tester.pumpAndSettle();
    expect(
      tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).liked,
      isTrue,
    );
    expect(
      tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).likeCount,
      20,
    );
  });

  testWidgets('old-session action result is ignored', (tester) async {
    final pages = _Pages()..liked = false;
    final topics = _Topics(pages)..pending = Completer<bool>();
    final container = await pump(tester, pages, topics);
    await tester.tap(find.byTooltip('点赞').first);
    await tester.pump();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    topics.pending!.complete(true);
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.favorite), findsNothing);
  });
  testWidgets(
    'returning from a topic keeps loaded pages and updates in place',
    (tester) async {
      final pages = _PagedPages();
      await pump(tester, pages, _Topics(pages));

      // 用户先滑出第 2、3 页:分页进度在本地累积,游标推进到末页。
      await tester.ensureVisible(find.text('加载更多'));
      await tester.tap(find.text('加载更多'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('加载更多'));
      await tester.tap(find.text('加载更多'));
      await tester.pumpAndSettle();
      expect(find.text('Topic 100'), findsOneWidget);
      expect(find.text('Topic 200'), findsOneWidget);
      expect(find.text('Topic 300'), findsOneWidget);

      // 读者离开期间第一条话题的服务端状态变化(点赞数更新)。
      pages.page1LikeCount = 9;

      // 基线:accountLayoutProvider 等既有消费者在挂载期也会读 home(),
      // 因此只统计返回路径自身的增量。
      final homeCallsBeforeReturn = pages.homeCalls;

      // 从详情返回触发 onReturnFromTopic。
      tester
          .widget<GfTopicList>(find.byType(GfTopicList))
          .onReturnFromTopic
          ?.call();
      await tester.pumpAndSettle();

      // 已加载的多页内容保留:列表不回缩到第一页。
      expect(find.text('Topic 100'), findsOneWidget);
      expect(find.text('Topic 200'), findsOneWidget);
      expect(find.text('Topic 300'), findsOneWidget);
      // 按 id 原位更新:首条话题拿到最新点赞数。
      expect(
        tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).likeCount,
        9,
      );
      // 返回刷新只追加一次第一页读取、不重置分页游标。
      expect(pages.homeCalls, homeCallsBeforeReturn + 1);
      expect(
        tester.widget<GfTopicList>(find.byType(GfTopicList)).hasMore,
        isFalse,
      );
    },
  );

  testWidgets('pull to refresh still resets pagination to the first page', (
    tester,
  ) async {
    final pages = _PagedPages();
    await pump(tester, pages, _Topics(pages));
    await tester.ensureVisible(find.text('加载更多'));
    await tester.tap(find.text('加载更多'));
    await tester.pumpAndSettle();
    expect(find.text('Topic 200'), findsOneWidget);

    pages.page1LikeCount = 9;
    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();

    // 下拉刷新保留既有语义:整页重置回第一页并展示最新状态。
    expect(find.text('Topic 200'), findsNothing);
    expect(find.text('Topic 100'), findsOneWidget);
    expect(
      tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).likeCount,
      9,
    );
    expect(
      tester.widget<GfTopicList>(find.byType(GfTopicList)).hasMore,
      isTrue,
    );
  });

  testWidgets('return while load-more is in flight keeps pagination usable', (
    tester,
  ) async {
    final pages = _HangingPages();
    await pump(tester, pages, _Topics(pages));
    await tester.ensureVisible(find.text('加载更多'));

    // 第二页请求在途时用户点进详情并返回:返回刷新使旧请求失效并接管加载态。
    final gate = Completer<void>();
    pages.gate = gate;
    await tester.tap(find.text('加载更多'));
    await tester.pump();
    tester
        .widget<GfTopicList>(find.byType(GfTopicList))
        .onReturnFromTopic
        ?.call();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    gate.complete();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));

    // 失效的在途请求不得卡死加载态:再次「加载更多」仍能继续翻页。
    await tester.ensureVisible(find.text('加载更多'));
    await tester.tap(find.text('加载更多'));
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.text('Topic 200'), findsOneWidget);
  });

  testWidgets('failed return refresh keeps the list and shows the notice', (
    tester,
  ) async {
    final pages = _PagedPages();
    await pump(tester, pages, _Topics(pages));
    await tester.ensureVisible(find.text('加载更多'));
    await tester.tap(find.text('加载更多'));
    await tester.pumpAndSettle();
    expect(find.text('Topic 200'), findsOneWidget);

    pages.failHome = true;
    tester
        .widget<GfTopicList>(find.byType(GfTopicList))
        .onReturnFromTopic
        ?.call();
    await tester.pumpAndSettle();

    // 列表与分页进度保留,并按 Home 失败刷新的产品约定轻提示。
    expect(find.text('Topic 100'), findsOneWidget);
    expect(find.text('Topic 200'), findsOneWidget);
    expect(find.textContaining('刷新失败'), findsOneWidget);
    expect(
      tester.widget<GfTopicList>(find.byType(GfTopicList)).hasMore,
      isTrue,
    );
  });

  testWidgets('unparseable return payload keeps the list with the notice', (
    tester,
  ) async {
    final pages = _PagedPages();
    await pump(tester, pages, _Topics(pages));
    pages.emptyHomeProps = true;
    tester
        .widget<GfTopicList>(find.byType(GfTopicList))
        .onReturnFromTopic
        ?.call();
    await tester.pumpAndSettle();

    expect(find.text('Topic 100'), findsOneWidget);
    expect(find.textContaining('刷新失败'), findsOneWidget);
    expect(
      tester.widget<GfTopicList>(find.byType(GfTopicList)).hasMore,
      isTrue,
    );
  });

  testWidgets('detail updates hand back to topics beyond page one in place', (
    tester,
  ) async {
    final pages = _PagedPages();
    final container = await pump(tester, pages, _Topics(pages));
    await tester.ensureVisible(find.text('加载更多'));
    await tester.tap(find.text('加载更多'));
    await tester.pumpAndSettle();
    expect(find.text('Topic 200'), findsOneWidget);

    // 详情页带回第 2 页话题的增量:点赞数 5 → 9、已点赞、已读。
    container.read(topicReturnStatesProvider)[200] = (
      unseen: false,
      liked: true,
      bookmarked: false,
      likeCount: 9,
      replyCount: 2,
      viewCount: 10,
    );
    tester
        .widget<GfTopicList>(find.byType(GfTopicList))
        .onReturnFromTopic
        ?.call();
    await tester.pumpAndSettle();

    // 原位合并:第 2 页卡片拿到最新状态,列表形状与游标不动。
    expect(
      tester.widget<GfTopicCard>(find.byType(GfTopicCard).at(1)).likeCount,
      9,
    );
    expect(
      tester.widget<GfTopicCard>(find.byType(GfTopicCard).at(1)).liked,
      isTrue,
    );
    expect(find.text('Topic 100'), findsOneWidget);
    expect(find.text('Topic 200'), findsOneWidget);
    expect(
      tester.widget<GfTopicList>(find.byType(GfTopicList)).hasMore,
      isTrue,
    );
    // 增量消费即清空,不会被后续返回重复应用。
    expect(container.read(topicReturnStatesProvider), isEmpty);
  });

  testWidgets('detail hands fresh topic state back for the return refresh', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final pages = _DetailPages();
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(_Tokens()),
        pageRepositoryProvider.overrideWithValue(pages),
        topicRepositoryProvider.overrideWithValue(_Topics(pages)),
        offlineTopicCacheProvider.overrideWithValue(_NoopOfflineCache()),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const TopicPage(topicId: 100),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // 打开详情即记录服务器读取真相:已读、点赞状态与计数。
    final TopicReturnState? onOpen = container.read(
      topicReturnStatesProvider,
    )[100];
    expect(onOpen, isNotNull);
    expect(onOpen!.unseen, isFalse);
    expect(onOpen.liked, isFalse);
    expect(onOpen.likeCount, 2);

    // 点赞成功后就地更新带回状态(失败回滚不写入)。
    final GfFloatingControls controls = tester.widget(
      find.byType(GfFloatingControls),
    );
    controls.actions
        .firstWhere((GfTopicAction action) => action.symbol == 'heart')
        .onTap();
    await tester.pumpAndSettle();

    final TopicReturnState? afterLike = container.read(
      topicReturnStatesProvider,
    )[100];
    expect(afterLike!.liked, isTrue);
    expect(afterLike.likeCount, 3);

    // markdown_widget 的 VisibilityDetector 会创建 500ms 延迟 Timer,
    // 需推进时钟让其过期,避免 "Timer is still pending"。
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });
}
