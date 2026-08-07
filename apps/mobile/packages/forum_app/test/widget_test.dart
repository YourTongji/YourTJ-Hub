import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'package:forum_app/l10n/app_localizations_en.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';

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

    // 底部导航 5 个 tab 存在(zh)。
    expect(find.text('首页'), findsOneWidget);
    expect(find.text('搜索'), findsOneWidget);
    // 「发布」出现 2 次:底部 tab + GfFloatingAction 浮动按钮。
    expect(find.text('发布'), findsNWidgets(2));
    expect(find.text('消息'), findsOneWidget);
    expect(find.text('我的'), findsOneWidget);
  });

  testWidgets('设置/通知/草稿路由可达:router 注册且可导航', (tester) async {
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

    // 首页齿轮按钮存在(设置入口)。
    expect(find.byIcon(Icons.settings_outlined), findsWidgets);

    // 底部导航切到"我的"。
    await tester.tap(find.text('我的'));
    await tester.pumpAndSettle();

    // 我的页三入口:设置/通知/草稿。
    expect(find.text('设置'), findsOneWidget);
    expect(find.text('通知'), findsOneWidget);
    expect(find.text('草稿箱'), findsOneWidget);

    // 点击"草稿箱"导航到 /drafts。
    await tester.tap(find.text('草稿箱'));
    await tester.pumpAndSettle();
    expect(find.text('草稿箱'), findsWidgets);
  });

  test('router has all browsing routes registered', () {
    final GoRouter router = appRouter;
    final List<RouteBase> routes = router.configuration.routes;

    // 顶层路由:ShellRoute + 4 个独立页。
    expect(routes.length, greaterThanOrEqualTo(5));

    // ShellRoute 内的 5 个 tab 路由。
    final RouteBase shell = routes.first;
    expect(shell, isA<ShellRoute>());
    final ShellRoute shellRoute = shell as ShellRoute;
    final List<String> paths = [
      for (final r in shellRoute.routes) (r as GoRoute).path,
    ];
    expect(paths, contains('/'));
    expect(paths, contains('/search'));
    expect(paths, contains('/publish'));
    expect(paths, contains('/messages'));
    expect(paths, contains('/profile'));
  });
  test('AppLocalizations zh/en 均可用', () {
    expect(AppLocalizationsZh().navHome, '首页');
    expect(AppLocalizationsZh().navPublish, '发布');
    expect(AppLocalizationsEn().navHome, 'Home');
    expect(AppLocalizationsEn().navPublish, 'Publish');
  });
}
