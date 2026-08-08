import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/pages/search/search_page.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/topic_list.dart';

import 'fixtures/page_fixtures.dart';

/// 内存 TokenStorage(与 pages_smoke_test 同款)。
class MemTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

/// no-op 离线缓存。
class NoopCache implements OfflineTopicCache, OfflineChatCache {
  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {}

  @override
  Future<PagePayload?> get(int topicId) async => null;

  @override
  Future<void> putConversation(ChatItemPayload conv) async {}

  @override
  Future<List<ChatItemPayload>> getConversations() async => const [];

  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {}

  @override
  Future<List<ChatMessagePayload>> getMessages(int convId) async => const [];

  @override
  Future<void> clear() async {}

  @override
  Future<void> close() async {}
}

class RecordingChatRepository extends ChatRepository {
  RecordingChatRepository(super.client);

  final List<(int, String)> sent = <(int, String)>[];

  @override
  Future<int> sendMessage({
    required int peerId,
    required String content,
    int msgType = 1,
  }) async {
    sent.add((peerId, content));
    return 9;
  }

  @override
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int limit = 30,
  }) async {
    return const ChatMessagesResponse(
      list: <ChatMessagePayload>[],
      hasMoreBefore: false,
      hasMoreAfter: false,
      nextBeforeId: 0,
      latestId: 0,
    );
  }

  @override
  Future<bool> markRead({required int convId}) async => true;
}

/// 记录 fetch 调用次数的 PageRepository。
class CountingPageRepository extends PageRepository {
  CountingPageRepository(super.client);

  int fetchCalls = 0;

  @override
  Future<PagePayload> fetch(String path) async {
    fetchCalls++;
    if (path == '/' || path.startsWith('/?sort=')) {
      return parsePayload(homePayloadJson());
    }
    if (path == '/settings') {
      return parsePayload(settingsPayloadJson());
    }
    if (path == '/messages') {
      return parsePayload(messagesPayloadJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

/// 记录 search 调用 page 的 TopicRepository。
class PagingTopicRepository extends TopicRepository {
  PagingTopicRepository(super.client);

  final List<int> searchPages = [];

  @override
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
  }) async {
    searchPages.add(page);
    if (page <= 1) {
      return SearchPageProps(
        query: query,
        scope: scope,
        topics: [makeTopic(1, '结果-第一页')],
        users: const [],
        categories: const [],
        total: 2,
        usersTotal: 0,
        categoriesTotal: 0,
        totalPages: 2,
        pagination: const PaginationPayload(
          page: 1,
          nextPage: 2,
          hasNext: true,
          nextUrl: '/search?page=2',
        ),
      );
    }
    return SearchPageProps(
      query: query,
      scope: scope,
      topics: [makeTopic(2, '结果-第二页')],
      users: const [],
      categories: const [],
      total: 2,
      usersTotal: 0,
      categoriesTotal: 0,
      totalPages: 2,
      pagination: const PaginationPayload(
        page: 2,
        nextPage: 0,
        hasNext: false,
        nextUrl: '',
      ),
    );
  }
}

class AggregateSearchRepository extends PagingTopicRepository {
  AggregateSearchRepository(super.client);

  final List<String> scopes = <String>[];

  @override
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
  }) async {
    scopes.add(scope);
    return SearchPageProps(
      query: query,
      scope: scope,
      topics: <TopicPayload>[makeTopic(1, '聚合帖子结果')],
      users: const <UserSearchPayload>[
        UserSearchPayload(
          id: 2,
          username: 'bob',
          nickname: 'Bob',
          avatarUrl: '',
          bio: '校园开发者',
        ),
      ],
      categories: const <CategorySearchPayload>[
        CategorySearchPayload(
          id: 3,
          name: '开发',
          slug: 'dev',
          icon: '#',
          color: '#2563eb',
          desc: '技术交流',
        ),
      ],
      total: 1,
      usersTotal: 1,
      categoriesTotal: 1,
      totalPages: 1,
      pagination: const PaginationPayload(
        page: 1,
        nextPage: 0,
        hasNext: false,
        nextUrl: '',
      ),
    );
  }
}

/// 记录 filter 的 NotificationRepository。
class FilteringNotificationRepository extends NotificationRepository {
  FilteringNotificationRepository(super.client);

  final List<String> filters = [];

  @override
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
  }) async {
    filters.add(filter);
    final NotificationPayload n = NotificationPayload(
      id: 1,
      eventType: 'reply',
      isRead: filter == 'all',
      createdAt: '2026-08-07T10:00:00+08:00',
      title: filter == 'all' ? '全部通知' : '未读通知',
      content: '内容',
      actor: const NotificationActorPayload(id: 1, username: 'alice'),
      payload: const NotificationInnerPayload(actorId: 1),
    );
    return NotificationListResponse(
      items: [n],
      nextCursor: 0,
      hasNext: false,
      unreadCount: 1,
    );
  }
}

/// 记录 logout 的 AuthRepository。
class LogoutAuthRepository extends AuthRepository {
  LogoutAuthRepository(super.client);

  int logoutCalls = 0;

  @override
  Future<bool> logout() async {
    logoutCalls++;
    return true;
  }
}

/// 构造最小 TopicPayload。
TopicPayload makeTopic(int id, String title) {
  return TopicPayload(
    id: id,
    title: title,
    description: '摘要',
    url: '/p/post/$id',
    author: const UserBriefPayload(id: 1, username: 'alice', avatarUrl: ''),
    participants: const [],
    categories: const [],
    replyCount: 0,
    viewCount: 10,
    pinWeight: 0,
    processStatus: 0,
    activityText: '2026-08-07T10:00:00+08:00',
    lastUpdateTime: '2026-08-07T10:00:00+08:00',
  );
}

/// 构造最小 settings payload(登录态用户 + 空会话)。
Map<String, dynamic> settingsPayloadJson() {
  return {
    'component': 'user.settings',
    'props': {
      'user': {
        'id': 1,
        'username': 'alice',
        'email': 'alice@example.com',
        'nickname': 'Alice',
        'locale': 'zh',
        'avatarUrl': '',
        'profileCoverUrl': '',
        'bio': '',
        'signature': '',
        'websiteName': '',
        'website': '',
        'prestige': 0,
        'createdAt': '2026-01-01T00:00:00+08:00',
        'externalInformation': <String, Object>{},
        'wornBadgeCode': '',
        'badges': <Object>[],
        'wearableBadges': <Object>[],
      },
      'stats': {
        'topicCount': 0,
        'replyCount': 0,
        'followerCount': 0,
        'followingCount': 0,
        'likeReceivedCount': 0,
        'likeGivenCount': 0,
        'collectionCount': 0,
        'createdAt': '2026-01-01T00:00:00+08:00',
      },
      'tabs': <Object>[],
    },
    'meta': {'title': '设置'},
    'layout': minimalLayoutJson(),
    'url': '/settings',
    'version': '1.0',
  };
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  Future<ProviderContainer> makeContainer({
    required CountingPageRepository pageRepo,
    PagingTopicRepository? topicRepo,
    FilteringNotificationRepository? notifRepo,
    LogoutAuthRepository? authRepo,
    ChatRepository? chatRepo,
  }) async {
    final storage = MemTokenStorage()..write('token');
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(storage),
        pageRepositoryProvider.overrideWithValue(pageRepo),
        topicRepositoryProvider.overrideWithValue(
          topicRepo ?? PagingTopicRepository(client),
        ),
        notificationRepositoryProvider.overrideWithValue(
          notifRepo ?? FilteringNotificationRepository(client),
        ),
        authRepositoryProvider.overrideWithValue(
          authRepo ?? LogoutAuthRepository(client),
        ),
        if (chatRepo != null)
          chatRepositoryProvider.overrideWithValue(chatRepo),
        offlineTopicCacheProvider.overrideWithValue(NoopCache()),
        offlineChatCacheProvider.overrideWithValue(NoopCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  Widget app(ProviderContainer container, Widget home) {
    return UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('zh'),
        home: home,
      ),
    );
  }

  group('首页下拉刷新', () {
    testWidgets('下拉触发重载第一页(fetch 再次调用)', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const HomePage()));
      await tester.pumpAndSettle();

      expect(find.text('移动端测试话题'), findsOneWidget);
      final int callsBefore = pageRepo.fetchCalls;
      expect(callsBefore, greaterThanOrEqualTo(1));

      // 下拉手势触发 RefreshIndicator。
      await tester.fling(find.byType(GfTopicList), const Offset(0, 400), 1200);
      await tester.pumpAndSettle();

      expect(
        pageRepo.fetchCalls,
        greaterThan(callsBefore),
        reason: '下拉刷新应再次调用 fetch 重载第一页',
      );
    });
  });

  group('首页话题布局', () {
    testWidgets('卡片和列表可通过胶囊切换且记住选择', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const HomePage()));
      await tester.pumpAndSettle();

      expect(find.byType(GfTopicCard), findsOneWidget);
      expect(find.byType(GfTopicRow), findsNothing);
      expect(find.text('新建话题'), findsNothing);
      final Finder feedSwitch = find.byType(GfPillSwitch<GfTopicFeedMode>);
      expect(tester.getSize(feedSwitch).height, 32);
      expect(
        tester.getCenter(feedSwitch).dy,
        closeTo(tester.getCenter(find.byType(GfTabBar)).dy, 1),
      );
      final Finder brandLogo = find.byType(Image);
      expect(brandLogo, findsOneWidget);
      expect(tester.getTopLeft(brandLogo).dx, inInclusiveRange(0, 24));
      expect(tester.getSize(brandLogo), const Size(128, 34));

      await tester.tap(find.text('列表'));
      await tester.pumpAndSettle();

      expect(find.byType(GfTopicCard), findsNothing);
      expect(find.byType(GfTopicRow), findsOneWidget);
      final SharedPreferences preferences =
          await SharedPreferences.getInstance();
      expect(preferences.getString('goose:home-feed-mode'), 'list');
    });
  });

  group('登录表单', () {
    testWidgets('忘记密码入口不被遮挡并可进入重置模式', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const LoginPage()));
      await tester.pumpAndSettle();

      final Finder forgotPassword = find.text('忘记密码？');
      expect(forgotPassword, findsOneWidget);
      expect(forgotPassword.hitTestable(), findsOneWidget);

      await tester.tap(forgotPassword);
      await tester.pumpAndSettle();

      expect(find.text('重置密码'), findsOneWidget);
      expect(find.text('邮箱'), findsOneWidget);
      expect(find.text('返回登录'), findsOneWidget);
    });
  });

  group('搜索分页', () {
    testWidgets('加载更多调用 page+1 并追加结果', (tester) async {
      final topicRepo = PagingTopicRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(
        pageRepo: pageRepo,
        topicRepo: topicRepo,
      );
      await tester.pumpWidget(app(container, const SearchPage()));
      await tester.pumpAndSettle();

      // 输入关键词并搜索。
      await tester.enterText(find.byType(TextField), 'flutter');
      await tester.tap(find.byIcon(Icons.search));
      await tester.pumpAndSettle();

      expect(find.text('结果-第一页'), findsOneWidget);
      expect(topicRepo.searchPages, [1]);

      // 点击"加载更多"触发 page+1。
      await tester.tap(find.text('加载更多'));
      await tester.pumpAndSettle();

      expect(topicRepo.searchPages, [1, 2], reason: '加载更多应调用 search(page+1)');
      expect(find.text('结果-第一页'), findsOneWidget);
      expect(find.text('结果-第二页'), findsOneWidget, reason: '第二页结果应追加到列表');
    });

    testWidgets('全部范围同时展示帖子、用户与分类，并使用复数 scope', (tester) async {
      final topicRepo = AggregateSearchRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(
        pageRepo: pageRepo,
        topicRepo: topicRepo,
      );
      await tester.pumpWidget(app(container, const SearchPage()));
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextField), '同济');
      await tester.tap(find.byIcon(Icons.search));
      await tester.pumpAndSettle();

      expect(topicRepo.scopes, <String>['']);
      expect(find.text('聚合帖子结果'), findsOneWidget);
      expect(find.text('Bob'), findsOneWidget);
      expect(find.text('@bob'), findsOneWidget);
      expect(find.text('开发'), findsOneWidget);

      await tester.tap(find.text('用户').first);
      await tester.pumpAndSettle();
      expect(topicRepo.scopes.last, 'users');
    });
  });

  group('私信列表', () {
    testWidgets('新私信弹层可渲染并搜索用户', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('新私信'));
      await tester.pumpAndSettle();

      expect(find.text('新私信'), findsOneWidget);
      expect(find.text('Bob'), findsOneWidget);
      expect(find.text('@bob'), findsOneWidget);
      expect(tester.takeException(), isNull);

      await tester.enterText(find.byType(TextField).last, 'bob');
      await tester.pumpAndSettle();
      expect(find.text('Bob'), findsOneWidget);

      await tester.tap(find.byIcon(Icons.close));
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets('选择新用户不发空消息，首条真实消息才建立会话', (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final chatRepo = RecordingChatRepository(client);
      final pageRepo = CountingPageRepository(client);
      final container = await makeContainer(
        pageRepo: pageRepo,
        chatRepo: chatRepo,
      );
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();

      await tester.tap(find.byTooltip('新私信'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Dave'));
      await tester.pumpAndSettle();

      expect(chatRepo.sent, isEmpty, reason: '选联系人时不应向后端发送空消息');
      expect(find.text('开始聊天'), findsOneWidget);
      expect(find.text('给 Dave 发出第一条消息。'), findsOneWidget);

      await tester.enterText(find.byType(TextField), '你好');
      await tester.pump();
      await tester.tap(find.text('发送'));
      await tester.pumpAndSettle();
      expect(chatRepo.sent, <(int, String)>[(4, '你好')]);

      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets('会话搜索同时过滤用户名和消息预览', (tester) async {
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(pageRepo: pageRepo);
      await tester.pumpWidget(app(container, const MessagesPage()));
      await tester.pumpAndSettle();

      expect(find.text('bob'), findsOneWidget);
      expect(find.text('carol'), findsOneWidget);
      await tester.enterText(find.byType(TextField), '樱花');
      await tester.pumpAndSettle();

      expect(find.text('bob'), findsNothing);
      expect(find.text('carol'), findsOneWidget);

      await tester.pumpWidget(const SizedBox.shrink());
    });
  });

  group('通知筛选', () {
    testWidgets('all/unread 切换触发重新请求(filter 参数变化)', (tester) async {
      final notifRepo = FilteringNotificationRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final container = await makeContainer(
        pageRepo: pageRepo,
        notifRepo: notifRepo,
      );
      await tester.pumpWidget(app(container, const NotificationsPage()));
      await tester.pumpAndSettle();

      expect(notifRepo.filters, ['all']);
      expect(find.text('全部通知'), findsOneWidget);

      // 切到"未读"。
      await tester.tap(find.text('未读'));
      await tester.pumpAndSettle();

      expect(notifRepo.filters, [
        'all',
        'unread',
      ], reason: '切换 tab 应以 filter=unread 重新请求');
      expect(find.text('未读通知'), findsOneWidget);
    });
  });

  group('设置登出', () {
    testWidgets('登出调用 authRepository.logout、清 token 并跳转登录页', (tester) async {
      final authRepo = LogoutAuthRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final pageRepo = CountingPageRepository(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
      final storage = MemTokenStorage()..write('token');
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: storage,
        baseUrl: 'http://fake.local',
      );
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          pageRepositoryProvider.overrideWithValue(pageRepo),
          topicRepositoryProvider.overrideWithValue(
            PagingTopicRepository(client),
          ),
          notificationRepositoryProvider.overrideWithValue(
            FilteringNotificationRepository(client),
          ),
          authRepositoryProvider.overrideWithValue(authRepo),
          offlineTopicCacheProvider.overrideWithValue(NoopCache()),
          offlineChatCacheProvider.overrideWithValue(NoopCache()),
        ],
      );
      addTearDown(container.dispose);

      final router = GoRouter(
        initialLocation: '/settings',
        routes: [
          GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
          GoRoute(
            path: '/login',
            builder: (_, _) =>
                const Scaffold(body: Center(child: Text('login-page'))),
          ),
        ],
      );

      // 设置更大视口,保证安全 tab 内"退出登录"按钮无需滚动即可见。
      tester.view.physicalSize = const Size(1080, 2400);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.reset);

      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // 切到 Security tab 找登出按钮。
      await tester.tap(find.text('安全'));
      await tester.pumpAndSettle();
      expect(find.text('退出登录'), findsOneWidget);

      // 点击登出 → 确认对话框 → 确认。
      await tester.tap(find.text('退出登录'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('退出登录').last);
      await tester.pumpAndSettle();

      expect(authRepo.logoutCalls, 1, reason: '登出应调用 authRepository.logout');
      expect(await storage.read(), isNull, reason: '登出应清空本地 token');
      expect(router.state.uri.path, '/login', reason: '登出后应跳转登录页');
      expect(find.text('login-page'), findsOneWidget);
    });
  });
}
