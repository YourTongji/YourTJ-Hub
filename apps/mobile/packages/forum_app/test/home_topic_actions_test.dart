import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
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
  Future<PagePayload> fetch(String path) async =>
      pending == null ? payload() : pending!.future;
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
    _Topics topics,
  ) async {
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
          home: const HomePage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
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
}
