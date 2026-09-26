import 'dart:async';
import 'dart:ui' show Tristate;
import 'package:flutter/services.dart';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

class _Storage implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _Profiles extends PageRepository {
  _Profiles()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Storage(),
          baseUrl: 'http://fake.local',
        ),
      );
  final paths = <String>[];
  final failures = <String>{};
  void Function(String, Map<String, dynamic>)? configure;
  final pending = <String, Completer<PagePayload>>{};
  PagePayload response(String path) {
    final json = userProfilePayloadJson();
    final props = json['props'] as Map<String, dynamic>;
    final activities = props['activities'] as List;
    props['activities'] = List.generate(
      40,
      (i) => {
        ...activities.first as Map<String, dynamic>,
        'id': i + 1,
        'contentPreview': 'activity-$i',
      },
    );
    props['likes'] = List.generate(
      40,
      (i) => {
        'id': i + 1,
        'topicId': i + 1,
        'title': 'like-$i',
        'url': '/p/${i + 1}',
        'likedAt': '2026-01-01T00:00:00Z',
      },
    );
    props['following'] = [
      {
        'id': 2,
        'username': 'bob',
        'nickname': 'Bob',
        'avatarUrl': '',
        'bio': '',
        'isFollowing': false,
        'url': '/u/2',
      },
    ];
    props['followers'] = [
      {
        'id': 3,
        'username': 'carol',
        'nickname': 'Carol',
        'avatarUrl': '',
        'bio': '',
        'isFollowing': false,
        'url': '/u/2',
      },
    ];
    configure?.call(path, props);
    return parsePayload(json);
  }

  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async {
    paths.add(path);
    if (failures.contains(path)) throw StateError('unavailable');
    return pending[path]?.future ?? response(path);
  }
}

class _FollowActions extends TopicRepository {
  _FollowActions()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Storage(),
          baseUrl: 'http://fake.local',
        ),
      );
  final pending = Completer<bool>();
  int calls = 0;
  @override
  Future<bool> followUser({required int userId, required bool isFollowing}) {
    calls++;
    return pending.future;
  }
}

class _ContentActions extends _FollowActions {
  final writes = <(int, bool, int)>[];
  final write = Completer<bool>();
  @override
  Future<bool> likeTopic({required int topicId, required int action}) {
    writes.add((topicId, false, action));
    return write.future;
  }

  @override
  Future<bool> bookmarkTopic({required int topicId, required int action}) {
    writes.add((topicId, true, action));
    return write.future;
  }
}

void _contentFixture(String path, Map<String, dynamic> props) {
  final home = homePayloadJson()['props'] as Map<String, dynamic>;
  props['topics'] = [
    {
      ...(home['topics'] as List).first as Map<String, dynamic>,
      'id': 100,
      'liked': false,
      'bookmarked': false,
      'likeCount': 2,
    },
  ];
  props['activities'] = [
    {
      ...(props['activities'] as List).first as Map<String, dynamic>,
      'action': 2,
      'subjectType': 'Topic',
      'subjectId': 100,
      'liked': false,
      'bookmarked': false,
      'likeCount': 2,
    },
  ];
}

Future<void> _showContent(WidgetTester tester, Type type) async {
  await tester.scrollUntilVisible(
    find.byType(type).first,
    300,
    scrollable: find
        .descendant(
          of: find.byType(CustomScrollView).first,
          matching: find.byType(Scrollable),
        )
        .first,
  );
  await tester.pumpAndSettle();
}

Future<ProviderContainer> _pump(
  WidgetTester tester,
  _Profiles repo, {
  Widget home = const ProfilePage(userId: 1),
  GoRouter? router,
  TopicRepository? topics,
  bool settle = true,
  double scale = 1,
  CurrentUser? currentUser,
}) async {
  final container = ProviderContainer(
    overrides: [
      currentUserProvider.overrideWith((ref) async => currentUser),
      pageRepositoryProvider.overrideWithValue(repo),
      if (topics != null) topicRepositoryProvider.overrideWithValue(topics),
    ],
  );
  addTearDown(container.dispose);
  final app = router == null
      ? MaterialApp(
          home: MediaQuery(
            data: MediaQueryData(textScaler: TextScaler.linear(scale)),
            child: home,
          ),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
        )
      : MaterialApp.router(
          routerConfig: router,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
        );
  await tester.pumpWidget(
    UncontrolledProviderScope(container: container, child: app),
  );
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    for (var i = 0; i < 5; i++) {
      await tester.pump();
    }
  }
  return container;
}

void _select(WidgetTester tester, String label) {
  tester
      .widget<InkWell>(
        find
            .descendant(
              of: find.byTooltip(label),
              matching: find.byType(InkWell),
            )
            .first,
      )
      .onTap!();
}

void main() {
  testWidgets('activity interaction footer fits narrow large-text screens', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await _pump(
      tester,
      _Profiles()..configure = _contentFixture,
      scale: 2,
      currentUser: const CurrentUser(id: 1, username: 'alice'),
    );
    await _showContent(tester, GfContentRow);
    expect(tester.takeException(), isNull);
    expect(
      find.descendant(
        of: find.byType(GfContentRow),
        matching: find.byIcon(Icons.favorite_border),
      ),
      findsOneWidget,
    );
  });

  for (final fail in [false, true]) {
    testWidgets(
      'profile write survives late refresh and reconciles later (fail=$fail)',
      (tester) async {
        final repo = _Profiles()..configure = _contentFixture;
        final actions = _ContentActions();
        await _pump(
          tester,
          repo,
          topics: actions,
          currentUser: const CurrentUser(id: 1, username: 'alice'),
          home: const ProfilePage(userId: 1, initialStream: 'topics'),
        );
        await _showContent(tester, GfTopicCard);
        final card = tester.widget<GfTopicCard>(find.byType(GfTopicCard).first);
        final result = card.onLike!(true);
        await tester.pump();
        expect(
          tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).liked,
          isTrue,
        );
        expect(
          await card.onBookmark!(true),
          isFalse,
        ); // serialize same-content writes
        final pending = Completer<PagePayload>();
        repo.pending['/u/1/activity/topics'] = pending;
        final refresh = tester
            .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
            .onRefresh();
        await tester.pump();
        if (fail) {
          actions.write.completeError(StateError('write failed'));
        } else {
          actions.write.complete(true);
        }
        expect(await result, !fail);
        final stale = repo.response('/u/1/activity/topics');
        pending.complete(stale);
        await refresh;
        await tester.pumpAndSettle();
        final restored = tester.widget<GfTopicCard>(
          find.byType(GfTopicCard).first,
        );
        expect(restored.liked, !fail);
        expect(restored.likeCount, fail ? 2 : 3);
        repo.pending.clear();
        await tester
            .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
            .onRefresh();
        await tester.pumpAndSettle();
        expect(
          tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).liked,
          isFalse,
        );
        expect(actions.writes, [(100, false, 1)]);
      },
    );
  }
  testWidgets(
    'detail return updates retained topics and activity without refetch or scroll reset',
    (tester) async {
      final repo = _Profiles()..configure = _contentFixture;
      final router = GoRouter(
        initialLocation: '/u/1',
        routes: [
          GoRoute(
            path: '/u/1',
            builder: (_, _) =>
                const ProfilePage(userId: 1, initialStream: 'topics'),
          ),
          GoRoute(
            path: '/p/:id',
            builder: (_, _) => const Scaffold(body: Text('detail')),
          ),
        ],
      );
      addTearDown(router.dispose);
      final container = await _pump(
        tester,
        repo,
        router: router,
        currentUser: const CurrentUser(id: 1, username: 'alice'),
      );
      await _showContent(tester, GfTopicCard);
      final controller = tester
          .widget<CustomScrollView>(find.byType(CustomScrollView).first)
          .controller!;
      final offset = controller.offset;
      tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).onTap!();
      await tester.pumpAndSettle();
      container.read(topicReturnStatesProvider)[100] = (
        unseen: false,
        liked: true,
        bookmarked: true,
        likeCount: 3,
        replyCount: 0,
        viewCount: 1,
      );
      router.pop();
      await tester.pumpAndSettle();
      final card = tester.widget<GfTopicCard>(find.byType(GfTopicCard).first);
      expect(card.liked, isTrue);
      expect(card.bookmarked, isTrue);
      expect(card.likeCount, 3);
      expect(controller.offset, offset);
      expect(repo.paths, ['/u/1/activity/topics']);
      _select(tester, '动态');
      await tester.pumpAndSettle();
      await _showContent(tester, GfContentRow);
      // This fresh read is authoritative, then the next detail handoff updates it.
      tester.widget<GfContentRow>(find.byType(GfContentRow).first).onTap!();
      await tester.pumpAndSettle();
      container.read(topicReturnStatesProvider)[100] = (
        unseen: false,
        liked: true,
        bookmarked: false,
        likeCount: 4,
        replyCount: 0,
        viewCount: 1,
      );
      router.pop();
      await tester.pumpAndSettle();
      final row = find.byType(GfContentRow).first;
      expect(
        find.descendant(of: row, matching: find.byIcon(Icons.favorite)),
        findsOneWidget,
      );
      expect(
        find.descendant(of: row, matching: find.text('4')),
        findsOneWidget,
      );
      _select(tester, '内容');
      await tester.pumpAndSettle();
      expect(
        tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).bookmarked,
        isFalse,
      );
    },
  );

  testWidgets(
    'reply return state stays separate from a topic with the same ID',
    (tester) async {
      final repo = _Profiles()
        ..configure = (path, props) {
          _contentFixture(path, props);
          (props['activities'] as List).first['action'] = 5;
          (props['activities'] as List).first['subjectType'] = 'Post';
        };
      final router = GoRouter(
        initialLocation: '/u/1',
        routes: [
          GoRoute(
            path: '/u/1',
            builder: (_, _) => const ProfilePage(userId: 1),
          ),
          GoRoute(
            path: '/p/:id',
            builder: (_, _) => const Scaffold(body: Text('detail')),
          ),
        ],
      );
      addTearDown(router.dispose);
      final container = await _pump(
        tester,
        repo,
        router: router,
        currentUser: const CurrentUser(id: 1, username: 'alice'),
      );
      await _showContent(tester, GfContentRow);
      tester.widget<GfContentRow>(find.byType(GfContentRow).first).onTap!();
      await tester.pumpAndSettle();
      container.read(postReturnStatesProvider)[100] = (
        liked: true,
        bookmarked: true,
        likeCount: 9,
      );
      router.pop();
      await tester.pumpAndSettle();
      expect(
        find.descendant(
          of: find.byType(GfContentRow).first,
          matching: find.byIcon(Icons.favorite),
        ),
        findsOneWidget,
      );
      _select(tester, '内容');
      await tester.pumpAndSettle();
      await _showContent(tester, GfTopicCard);
      expect(
        tester.widget<GfTopicCard>(find.byType(GfTopicCard).first).liked,
        isFalse,
      );
    },
  );

  testWidgets('profile topics expose viewer interaction actions', (
    tester,
  ) async {
    final repo = _Profiles()
      ..configure = (_, props) {
        final home = homePayloadJson()['props'] as Map<String, dynamic>;
        props['topics'] = [
          {
            ...(home['topics'] as List).first as Map<String, dynamic>,
            'liked': true,
            'bookmarked': true,
          },
        ];
      };
    await _pump(
      tester,
      repo,
      home: const ProfilePage(userId: 1, initialStream: 'topics'),
    );
    await tester.scrollUntilVisible(
      find.byType(GfTopicCard).first,
      300,
      scrollable: find
          .descendant(
            of: find.byType(CustomScrollView).first,
            matching: find.byType(Scrollable),
          )
          .first,
    );
    final card = tester.widget<GfTopicCard>(find.byType(GfTopicCard).first);
    expect(card.liked, isTrue);
    expect(card.bookmarked, isTrue);
    expect(card.onLike, isNotNull);
    expect(card.onBookmark, isNotNull);
  });

  testWidgets('older tab response cannot overwrite a newer relationship read', (
    tester,
  ) async {
    final repo = _Profiles()
      ..configure = (path, props) {
        if (path.endsWith('/followers')) {
          props['followers'] = [
            {
              ...(props['following'] as List).first as Map<String, dynamic>,
              'isFollowing': true,
            },
          ];
        }
      };
    await _pump(tester, repo, home: const ProfilePage.connections(userId: 1));
    final pending = Completer<PagePayload>();
    repo.pending['/u/1/following'] = pending;
    final refresh = tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pump();
    _select(tester, '粉丝');
    await tester.pumpAndSettle();
    expect(find.widgetWithText(GfFollowButton, '已关注'), findsOneWidget);
    pending.complete(repo.response('/u/1/following'));
    await refresh;
    await tester.pumpAndSettle();
    _select(tester, '关注');
    await tester.pumpAndSettle();
    expect(find.widgetWithText(GfFollowButton, '已关注'), findsOneWidget);
  });

  testWidgets('post-mutation refresh reconciles external follow changes', (
    tester,
  ) async {
    final repo = _Profiles();
    final actions = _FollowActions();
    await _pump(
      tester,
      repo,
      home: const ProfilePage.connections(userId: 1),
      topics: actions,
      currentUser: const CurrentUser(id: 1, username: 'alice'),
    );
    await tester.tap(find.widgetWithText(GfFollowButton, '关注'));
    await tester.pump();
    actions.pending.complete(true);
    await tester.pumpAndSettle();
    expect(find.widgetWithText(GfFollowButton, '已关注'), findsOneWidget);
    // Another device unfollowed after this mutation settled.
    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();
    expect(find.widgetWithText(GfFollowButton, '关注'), findsOneWidget);
  });

  testWidgets('guest connection action opens login without following', (
    tester,
  ) async {
    final repo = _Profiles();
    final actions = _FollowActions();
    final router = GoRouter(
      initialLocation: '/u/1/following',
      routes: [
        GoRoute(
          path: '/u/1/following',
          builder: (_, _) => const ProfilePage.connections(userId: 1),
        ),
        GoRoute(
          path: '/login',
          builder: (_, _) => const Scaffold(body: Text('login-target')),
        ),
      ],
    );
    addTearDown(router.dispose);
    await _pump(tester, repo, router: router, topics: actions);
    await tester.tap(find.widgetWithText(GfFollowButton, '关注'));
    await tester.pumpAndSettle();
    expect(find.text('login-target'), findsOneWidget);
    expect(router.state.uri.queryParameters['returnTo'], '/u/1/following');
    expect(actions.calls, 0);
  });

  testWidgets('pending connection mutation is discarded at account boundary', (
    tester,
  ) async {
    final repo = _Profiles();
    final actions = _FollowActions();
    final container = await _pump(
      tester,
      repo,
      home: const ProfilePage.connections(userId: 1),
      topics: actions,
      currentUser: const CurrentUser(id: 1, username: 'alice'),
    );
    await tester.tap(find.widgetWithText(GfFollowButton, '关注'));
    await tester.pump();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    actions.pending.complete(true);
    await tester.pumpAndSettle();
    expect(find.widgetWithText(GfFollowButton, '关注'), findsOneWidget);
    expect(
      tester.widget<GfFollowButton>(find.byType(GfFollowButton)).busy,
      isFalse,
    );
  });

  testWidgets(
    'connection follow rolls back on failure without opening profile',
    (tester) async {
      final repo = _Profiles();
      final actions = _FollowActions();
      await _pump(
        tester,
        repo,
        home: const ProfilePage.connections(userId: 1),
        topics: actions,
        currentUser: const CurrentUser(id: 1, username: 'alice'),
      );
      expect(find.text('@bob'), findsOneWidget);
      final toggle = tester
          .widget<GfFollowButton>(find.byType(GfFollowButton))
          .onPressed!;
      toggle();
      toggle();
      expect(actions.calls, 1);
      await tester.pump();
      expect(
        tester.widget<GfFollowButton>(find.byType(GfFollowButton)).busy,
        isTrue,
      );
      actions.pending.completeError(StateError('unavailable'));
      await tester.pumpAndSettle();
      expect(find.widgetWithText(GfFollowButton, '关注'), findsOneWidget);
      expect(
        tester.widget<GfFollowButton>(find.byType(GfFollowButton)).busy,
        isFalse,
      );
      expect(repo.paths, ['/u/1/following']);
    },
  );

  testWidgets(
    'late connection read cannot undo follow and self has no action',
    (tester) async {
      final repo = _Profiles();
      final actions = _FollowActions();
      await _pump(
        tester,
        repo,
        home: const ProfilePage.connections(userId: 1),
        topics: actions,
        currentUser: const CurrentUser(id: 1, username: 'alice'),
      );
      await tester.tap(find.widgetWithText(GfFollowButton, '关注'));
      await tester.pump();
      final pending = Completer<PagePayload>();
      repo.pending['/u/1/following'] = pending;
      final refresh = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pump();
      actions.pending.complete(true);
      await tester.pump();
      pending.complete(repo.response('/u/1/following'));
      await refresh;
      await tester.pumpAndSettle();
      expect(find.widgetWithText(GfFollowButton, '已关注'), findsOneWidget);
      repo.pending.clear();
      repo.configure = (_, props) =>
          (props['following'] as List).first['isSelf'] = true;
      await tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pumpAndSettle();
      expect(find.byType(GfFollowButton), findsNothing);
    },
  );

  testWidgets(
    'liked content uses its author and bookmarks retain reply floor',
    (tester) async {
      final repo = _Profiles()
        ..configure = (_, props) {
          final author = {
            'id': 77,
            'username': 'post-author',
            'nickname': 'Post Author',
            'avatarUrl': '',
          };
          props['likes'] = [
            {
              'id': 1,
              'topicId': 9,
              'title': 'Liked topic',
              'url': '/p/9',
              'likedAt': '2026-01-01',
              'author': author,
              'excerpt': 'Original body',
            },
          ];
          props['bookmarks'] = [
            {
              'id': 1,
              'type': 'post',
              'topicId': 9,
              'postId': 42,
              'postNo': 7,
              'title': 'Saved reply',
              'url': '/p/post/9/7#post-42',
              'bookmarkedAt': '2026-01-01',
              'author': author,
              'excerpt': 'Reply body',
            },
          ];
        };
      final router = GoRouter(
        routes: [
          GoRoute(path: '/', builder: (_, _) => const ProfilePage(userId: 1)),
          GoRoute(
            path: '/p/:id',
            builder: (_, state) =>
                Text('floor=${state.uri.queryParameters['postNo']}'),
          ),
        ],
      );
      addTearDown(router.dispose);
      await _pump(tester, repo, router: router);
      _select(tester, '赞过');
      await tester.pumpAndSettle();
      tester
          .widget<CustomScrollView>(find.byType(CustomScrollView))
          .controller!
          .jumpTo(500);
      await tester.pumpAndSettle();
      var row = tester.widget<GfContentRow>(find.byType(GfContentRow).first);
      expect(row.author, 'Post Author');
      expect(row.text, 'Original body');
      _select(tester, '收藏');
      await tester.pumpAndSettle();
      row = tester.widget<GfContentRow>(find.byType(GfContentRow).first);
      expect(row.author, 'Post Author');
      expect(row.text, 'Reply body');
      row.onTap!();
      await tester.pumpAndSettle();
      expect(find.text('floor=7'), findsOneWidget);
    },
  );

  testWidgets('profile inactive tabs retain visible text', (tester) async {
    await _pump(tester, _Profiles());
    expect(
      find.descendant(of: find.byTooltip('内容'), matching: find.text('内容')),
      findsOneWidget,
    );
    expect(
      find.descendant(of: find.byTooltip('赞过'), matching: find.text('赞过')),
      findsOneWidget,
    );
  });

  testWidgets(
    'profile tab segments stay contiguous as the active segment moves',
    (tester) async {
      await _pump(tester, _Profiles());
      const labels = ['动态', '内容', '赞过', '收藏', '徽章'];
      Finder cell(String label) => find
          .descendant(of: find.byTooltip(label), matching: find.byType(InkWell))
          .first;
      void expectNoGaps() {
        final rects = [for (final label in labels) tester.getRect(cell(label))];
        for (var i = 0; i < rects.length - 1; i++) {
          expect(rects[i].right, closeTo(rects[i + 1].left, .1));
        }
      }

      expectNoGaps();
      final segment = find.byKey(const ValueKey('profile-tab-active-segment'));
      final underline = find.byKey(const ValueKey('profile-tab-indicator'));
      final start = tester.getRect(segment).center.dx;
      expect(tester.getRect(underline).width, greaterThan(40));

      _select(tester, '赞过');
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 110));
      expectNoGaps();
      final halfway = tester.getRect(segment).center.dx;
      expect(halfway, greaterThan(start));
      await tester.pumpAndSettle();
      expect(tester.getRect(segment).center.dx, greaterThan(halfway));
    },
  );

  testWidgets(
    'profile tabs support keyboard activation and selected semantics',
    (tester) async {
      final semantics = tester.ensureSemantics();
      await _pump(tester, _Profiles());
      final tabText = find.descendant(
        of: find.byTooltip('赞过'),
        matching: find.text('赞过'),
      );
      Focus.of(tester.element(tabText)).requestFocus();
      await tester.pump();
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      await tester.pumpAndSettle();
      final node = tester.getSemantics(find.bySemanticsLabel('赞过'));
      expect(
        node.getSemanticsData().flagsCollection.isSelected,
        Tristate.isTrue,
      );
      expect(node.getSemanticsData().flagsCollection.isButton, isTrue);
      semantics.dispose();
    },
  );

  testWidgets(
    'profile streams retain loaded content and independent scroll offsets',
    (tester) async {
      final repo = _Profiles();
      await _pump(tester, repo);
      final scroll = tester
          .widget<CustomScrollView>(find.byType(CustomScrollView))
          .controller!;
      scroll.jumpTo(1100);
      await tester.pump();
      _select(tester, '赞过');
      await tester.pumpAndSettle();
      scroll.jumpTo(850);
      await tester.pump();
      _select(tester, '动态');
      await tester.pumpAndSettle();
      expect(repo.paths.where((path) => path == '/u/1/activity'), hasLength(1));
      expect(scroll.offset, 1100);
      _select(tester, '赞过');
      await tester.pumpAndSettle();
      expect(repo.paths.where((path) => path.endsWith('/likes')), hasLength(1));
      expect(scroll.offset, 850);
    },
  );

  testWidgets(
    'profile slow inactive stream completes without replacing active stream',
    (tester) async {
      final repo = _Profiles();
      final pending = Completer<PagePayload>();
      repo.pending['/u/1/activity/likes'] = pending;
      await _pump(tester, repo);
      _select(tester, '赞过');
      await tester.pump();
      _select(tester, '内容');
      await tester.pumpAndSettle();
      pending.complete(repo.response('/u/1/activity/likes'));
      await tester.pumpAndSettle();
      expect(find.text('like-0'), findsNothing);
      _select(tester, '赞过');
      await tester.pumpAndSettle();
      expect(repo.paths.where((path) => path.endsWith('/likes')), hasLength(2));
      tester
          .widget<CustomScrollView>(find.byType(CustomScrollView))
          .controller!
          .jumpTo(500);
      await tester.pump();
      expect(find.text('like-0'), findsOneWidget);
    },
  );

  testWidgets(
    'profile follower statistic navigates to the real public stream',
    (tester) async {
      final router = GoRouter(
        routes: [
          GoRoute(path: '/', builder: (_, _) => const ProfilePage(userId: 1)),
          GoRoute(
            path: '/u/1/followers',
            builder: (_, _) => const ProfilePage.connections(
              userId: 1,
              initialStream: 'followers',
            ),
          ),
        ],
      );
      addTearDown(router.dispose);
      await _pump(tester, _Profiles(), router: router);
      await tester.tap(
        find.descendant(of: find.byType(GfUserCard), matching: find.text('粉丝')),
      );
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/u/1/followers');
    },
  );
  testWidgets(
    'public profile connections use real routes and return to preserved content',
    (tester) async {
      final repo = _Profiles()
        ..configure = (_, props) {
          props['isOwnProfile'] = false;
          (props['user'] as Map<String, dynamic>)['isSelf'] = false;
        };
      final router = GoRouter(
        initialLocation: '/u/1',
        routes: appRouter.configuration.routes,
      );
      addTearDown(router.dispose);
      await _pump(tester, repo, router: router);
      expect(find.byTooltip('收藏'), findsNothing);
      final stat = find.descendant(
        of: find.byType(GfUserCard),
        matching: find.text('关注'),
      );
      await tester.tap(stat);
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/u/1/following');
      expect(repo.paths, contains('/u/1/following'));
      expect(find.byType(GfUserCard), findsNothing);
      expect(find.text('@alice'), findsOneWidget);
      await tester.tap(find.text('Bob'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/u/2');
      router.pop();
      await tester.pumpAndSettle();
      expect(find.text('Bob'), findsOneWidget);
      router.pop();
      await tester.pumpAndSettle();
      expect(repo.paths.where((p) => p == '/u/1/activity'), hasLength(1));
    },
  );

  testWidgets(
    'own connections require login for guests without fetching a user',
    (tester) async {
      final repo = _Profiles();
      await _pump(tester, repo, home: const ProfilePage.connections());
      expect(repo.paths, isEmpty);
      expect(find.widgetWithText(GfButton, '登录'), findsOneWidget);
    },
  );

  testWidgets(
    'connection pagination completes in its own tab and survives switching back',
    (tester) async {
      final repo = _Profiles()
        ..configure = (path, props) {
          if (path == '/u/1/following') {
            props['pagination'] = {
              'page': 1,
              'nextPage': 2,
              'hasNext': true,
              'nextUrl': '/u/1/following?page=2',
            };
          }
          if (path.endsWith('page=2')) {
            props['following'] = [
              {
                'id': 4,
                'username': 'dan',
                'nickname': 'Dan',
                'avatarUrl': '',
                'bio': '',
                'url': '/u/4',
              },
            ];
          }
        };
      final pending = Completer<PagePayload>();
      repo.pending['/u/1/following?page=2'] = pending;
      await _pump(
        tester,
        repo,
        home: const ProfilePage.connections(userId: 1),
        settle: false,
      );
      expect(repo.paths, contains('/u/1/following?page=2'));
      _select(tester, '粉丝');
      await tester.pumpAndSettle();
      pending.complete(repo.response('/u/1/following?page=2'));
      await tester.pumpAndSettle();
      expect(find.text('Dan'), findsNothing);
      _select(tester, '关注');
      await tester.pumpAndSettle();
      expect(find.text('Bob'), findsOneWidget);
      expect(find.text('Dan'), findsOneWidget);
      expect(
        repo.paths.where((p) => p.startsWith('/u/1/following')),
        hasLength(3),
      );
    },
  );

  testWidgets(
    'connection pagination error preserves rows and waits for explicit retry',
    (tester) async {
      final repo = _Profiles()
        ..configure = (path, props) {
          if (path == '/u/1/following') {
            props['pagination'] = {
              'page': 1,
              'nextPage': 2,
              'hasNext': true,
              'nextUrl': '/u/1/following?page=2',
            };
          }
        };
      repo.failures.add('/u/1/following?page=2');
      await _pump(tester, repo, home: const ProfilePage.connections(userId: 1));
      expect(find.text('Bob'), findsOneWidget);
      expect(
        tester.widget<GfListFooter>(find.byType(GfListFooter)).error,
        isNotNull,
      );
      await tester.pump(const Duration(seconds: 2));
      expect(repo.paths.where((p) => p.endsWith('page=2')), hasLength(1));
      repo.failures.clear();
      tester.widget<GfListFooter>(find.byType(GfListFooter)).onLoadMore();
      await tester.pumpAndSettle();
      expect(
        find.text('Bob'),
        findsOneWidget,
        reason: 'overlapping pages are deduplicated',
      );
      expect(repo.paths.where((p) => p.endsWith('page=2')), hasLength(2));
    },
  );

  testWidgets('refresh invalidates an older pending pagination response', (
    tester,
  ) async {
    final repo = _Profiles()
      ..configure = (path, props) {
        if (path == '/u/1/following') {
          props['pagination'] = {
            'page': 1,
            'nextPage': 2,
            'hasNext': true,
            'nextUrl': '/u/1/following?page=2',
          };
        }
        if (path.endsWith('page=2')) props['following'] = [];
      };
    final pending = Completer<PagePayload>();
    repo.pending['/u/1/following?page=2'] = pending;
    await _pump(
      tester,
      repo,
      home: const ProfilePage.connections(userId: 1),
      settle: false,
    );
    repo.configure = (_, props) {
      props['following'] = [];
    };
    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();
    pending.complete(repo.response('/u/1/following?page=2'));
    await tester.pumpAndSettle();
    expect(find.text('Bob'), findsNothing);
    expect(find.byType(GfListFooter), findsNothing);
  });

  testWidgets(
    'profile account boundary clears cached content and rejects old responses',
    (tester) async {
      final repo = _Profiles();
      final pending = Completer<PagePayload>();
      repo.pending['/u/1/activity/likes'] = pending;
      final container = await _pump(tester, repo);
      _select(tester, '赞过');
      await tester.pump();
      final old = repo.response('/u/1/activity/likes');
      repo.pending.clear();
      repo.configure = (_, props) {
        props['likes'] = [];
      };
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      pending.complete(old);
      await tester.pumpAndSettle();
      tester
          .widget<CustomScrollView>(find.byType(CustomScrollView))
          .controller!
          .jumpTo(500);
      await tester.pump();
      expect(find.text('暂无点赞'), findsOneWidget);
      expect(find.text('like-0'), findsNothing);
    },
  );

  for (final width in [320.0, 1200.0]) {
    testWidgets('profile tabs and statistics support 2x text at width $width', (
      tester,
    ) async {
      tester.view.physicalSize = Size(width, 1000);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await _pump(tester, _Profiles(), scale: 2);
      expect(tester.takeException(), isNull);
      expect(
        tester.getSize(find.byType(CustomScrollView)).width,
        lessThanOrEqualTo(760),
      );
      final scroll = tester
          .widget<CustomScrollView>(find.byType(CustomScrollView))
          .controller!;
      scroll.jumpTo(1000);
      await tester.pump();
      await tester.ensureVisible(find.byTooltip('徽章'));
      await tester.tap(find.byTooltip('徽章'));
      await tester.pumpAndSettle();
      expect(find.text('暂无徽章'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await _pump(
        tester,
        _Profiles(),
        home: const ProfilePage.connections(userId: 1),
        scale: 2,
      );
      expect(find.text('Bob'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
  for (final nextUrl in [
    'https://foreign.example/u/1/following?page=2',
    '/u/2/following?page=2',
    '/u/1/followers?page=2',
  ]) {
    testWidgets(
      'profile pagination rejects a cursor outside its user and stream: $nextUrl',
      (tester) async {
        final repo = _Profiles()
          ..configure = (_, props) {
            props['pagination'] = {
              'page': 1,
              'nextPage': 2,
              'hasNext': true,
              'nextUrl': nextUrl,
            };
          };
        await _pump(
          tester,
          repo,
          home: const ProfilePage.connections(userId: 1),
        );
        expect(repo.paths, ['/u/1/following']);
        expect(find.text('Bob'), findsOneWidget);
      },
    );
  }
  testWidgets(
    'profile refresh begun during follow cannot undo the completed action',
    (tester) async {
      final repo = _Profiles()
        ..configure = (_, props) {
          props['isOwnProfile'] = false;
          props['canFollow'] = true;
          (props['user'] as Map<String, dynamic>)['isSelf'] = false;
        };
      final actions = _FollowActions();
      await _pump(tester, repo, topics: actions);
      await tester.tap(find.widgetWithText(GfFollowButton, '关注'));
      await tester.pump();
      final refresh = Completer<PagePayload>();
      repo.pending['/u/1/activity'] = refresh;
      final refreshDone = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pump();
      actions.pending.complete(true);
      await tester.pump();
      refresh.complete(repo.response('/u/1/activity'));
      await refreshDone;
      await tester.pumpAndSettle();
      expect(find.widgetWithText(GfFollowButton, '已关注'), findsOneWidget);
    },
  );
}
