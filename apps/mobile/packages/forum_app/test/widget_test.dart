import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'package:forum_app/l10n/app_localizations_en.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';

import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

/// 测试用内存 TokenStorage(生产为 SecureTokenStorage)。
class MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

/// 测试用 no-op 离线缓存(避免 widget 测试打开真实 sqlite 库)。
class NoopOfflineCache implements OfflineTopicCache, OfflineChatCache {
  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {}

  @override
  Future<PagePayload?> get(int topicId) async => null;

  @override
  Future<void> putConversations(List<ChatItemPayload> conversations) async {}

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

class RouterPageRepository extends PageRepository {
  RouterPageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
    if (path == '/' || path.startsWith('/?sort=')) {
      return parsePayload(homePayloadJson());
    }
    if (path.startsWith('/u/')) {
      return parsePayload(userProfilePayloadJson());
    }
    if (path == '/drafts') {
      return parsePayload(<String, dynamic>{
        'component': 'drafts.index',
        'props': <String, dynamic>{'drafts': <Object>[]},
        'meta': <String, dynamic>{'title': '草稿箱'},
        'layout': minimalLayoutJson(),
        'url': '/drafts',
        'version': '1.0',
      });
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

void main() {
  testWidgets('app builds with Gf theme and bottom shell', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
          offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
          offlineChatCacheProvider.overrideWithValue(NoopOfflineCache()),
        ],
        child: const GfApp(locale: Locale('zh')),
      ),
    );
    await tester.pumpAndSettle();

    final GfBottomNavigation navigation = tester.widget(
      find.byType(GfBottomNavigation),
    );
    expect(navigation.items.map((item) => item.label), [
      '首页',
      '校园',
      '通知',
      '消息',
    ]);
    expect(navigation.showLabels, isFalse);
    expect(navigation.onAction, isNull);
    expect(find.byType(FloatingActionButton), findsOneWidget);

    navigation.onSelected(1);
    await tester.pumpAndSettle();
    expect(appRouter.state.uri.path, '/campus');
    expect(find.text('从同学的真实评价，发现适合你的课'), findsOneWidget);
    await tester.tap(find.byTooltip('搜索'));
    await tester.pumpAndSettle();
    expect(appRouter.state.uri.path, '/search');
    expect(appRouter.canPop(), isTrue);
    appRouter.pop();
    await tester.pumpAndSettle();
    expect(appRouter.state.uri.path, '/campus');

    appRouter.go('/');
    await tester.pumpAndSettle();
  });

  testWidgets('设置/通知/草稿路由可达:router 注册且可导航', (tester) async {
    appRouter.go('/');
    final MemoryTokenStorage storage = MemoryTokenStorage();
    final GfApiClient client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 1, username: 'alice'),
          ),
          pageRepositoryProvider.overrideWithValue(
            RouterPageRepository(client),
          ),
          offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
          offlineChatCacheProvider.overrideWithValue(NoopOfflineCache()),
        ],
        child: const GfApp(locale: Locale('zh')),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('YourTJ'), findsOneWidget);
    appRouter.push('/profile');
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('更多功能'));
    await tester.pumpAndSettle();
    expect(find.text('设置'), findsOneWidget);
    expect(find.text('内容管理'), findsOneWidget);
    expect(find.text('回收站'), findsOneWidget);
    expect(find.text('草稿箱'), findsOneWidget);

    // 点击“草稿箱”导航到 /drafts。
    await tester.tap(find.text('草稿箱'));
    await tester.pumpAndSettle();
    expect(find.text('草稿箱'), findsWidgets);
  });

  test('router has all browsing routes registered', () {
    final GoRouter router = appRouter;
    final List<RouteBase> routes = router.configuration.routes;

    // 顶层路由包含持久 shell 与全局页面。
    expect(routes.length, greaterThanOrEqualTo(9));

    final RouteBase shell = routes.first;
    expect(shell, isA<StatefulShellRoute>());
    final StatefulShellRoute shellRoute = shell as StatefulShellRoute;
    expect(shellRoute.branches, hasLength(4));
    final List<String> shellPaths = <String>[
      for (final StatefulShellBranch branch in shellRoute.branches)
        for (final RouteBase route in branch.routes)
          if (route is GoRoute) route.path,
    ];
    expect(
      shellPaths,
      containsAll(<String>['/', '/campus', '/notifications', '/messages']),
    );
    expect(shellPaths, isNot(contains('/publish')));

    final List<String> topLevelPaths = <String>[
      for (final RouteBase route in routes)
        if (route is GoRoute) route.path,
    ];
    expect(topLevelPaths, contains('/publish'));
  });
  test('AppLocalizations zh/en 均可用', () {
    expect(AppLocalizationsZh().navHome, '首页');
    expect(AppLocalizationsZh().navPublish, '发布');
    expect(AppLocalizationsEn().navHome, 'Home');
    expect(AppLocalizationsEn().navPublish, 'Publish');
  });
}
