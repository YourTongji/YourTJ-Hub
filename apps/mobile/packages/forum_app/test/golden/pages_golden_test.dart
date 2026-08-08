import 'package:dio/dio.dart';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/pages/search/search_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:core/core.dart';

import '../golden_helper.dart';
import '../pages_smoke_test.dart'
    show
        FakePageRepository,
        FakeTopicRepository,
        MemoryTokenStorage,
        NoopOfflineCache;

/// 返回单条通知的 NotificationRepository(页面 golden 用)。
class OneNotificationRepository extends NotificationRepository {
  OneNotificationRepository(super.client);

  @override
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
  }) async {
    final NotificationPayload n = NotificationPayload(
      id: 1,
      eventType: 'reply',
      isRead: false,
      // 固定过去日期(>7 天):timeAgo 回退绝对日期,不随 CI 运行时刻漂移。
      createdAt: '2025-01-15T10:00:00+08:00',
      title: '有人回复了你的话题',
      content: '回复内容预览',
      actor: const NotificationActorPayload(
        id: 2,
        username: 'bob',
        avatarUrl: '',
      ),
      payload: const NotificationInnerPayload(
        actorId: 2,
        topicId: 100,
        topicTitle: '移动端测试话题',
      ),
    );
    return NotificationListResponse(
      items: [n],
      nextCursor: 0,
      hasNext: false,
      unreadCount: 1,
    );
  }
}

/// Page-level golden baselines (390x844 mobile surface, Roboto, zh locale).
///
/// Regenerate intentionally after a visual change:
/// `flutter test --update-goldens test/golden/pages_golden_test.dart`
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  // Golden baselines are rendered by flutter_tester on Linux CI (see
  // golden_helper.dart). flutter_tester rasterizes text differently per
  // host OS (hinting/AA), so the same PNG cannot match on macOS; run
  // goldens on Linux and skip elsewhere.
  final bool skipGoldens = !Platform.isLinux;

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  Future<ProviderContainer> makeContainer({
    NotificationRepository? notifRepo,
  }) async {
    final storage = MemoryTokenStorage();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
        pageRepositoryProvider.overrideWithValue(FakePageRepository(client)),
        topicRepositoryProvider.overrideWithValue(FakeTopicRepository(client)),
        notificationRepositoryProvider.overrideWithValue(
          notifRepo ?? OneNotificationRepository(client),
        ),
        offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
        offlineChatCacheProvider.overrideWithValue(NoopOfflineCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  Future<void> settleBrandLogo(WidgetTester tester) async {
    final BuildContext context = tester.element(find.byType(HomePage));
    await tester.runAsync(() async {
      await precacheImage(
        const AssetImage('assets/images/brand-default.png'),
        context,
      );
    });
    await tester.pump();
  }

  testWidgets('home page golden', skip: skipGoldens, (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const HomePage()),
    );
    await settleBrandLogo(tester);
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/home_page.png'),
    );
  });

  testWidgets('home page list golden', skip: skipGoldens, (tester) async {
    SharedPreferences.setMockInitialValues(<String, Object>{
      'goose:home-feed-mode': 'list',
    });
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const HomePage()),
    );
    await settleBrandLogo(tester);
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/home_page_list.png'),
    );
  });

  testWidgets('topic page golden', skip: skipGoldens, (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const TopicPage(topicId: 100),
      ),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/topic_page.png'),
    );
    // markdown_widget 的 VisibilityDetector 会创建 500ms 延迟 Timer,
    // 需推进时钟让其过期,避免 "Timer is still pending"。
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('notifications page golden', skip: skipGoldens, (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const NotificationsPage(),
      ),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/notifications_page.png'),
    );
  });

  testWidgets('messages page golden', skip: skipGoldens, (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const MessagesPage(),
      ),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/messages_page.png'),
    );
  });

  testWidgets('messages new chat sheet golden', skip: skipGoldens, (
    tester,
  ) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const MessagesPage(),
      ),
    );
    await tester.tap(find.byTooltip('新私信'));
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(Overlay).first,
      matchesGoldenFile('golden/pages/messages_new_chat.png'),
    );
  });

  testWidgets('search page grouped results golden', skip: skipGoldens, (
    tester,
  ) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const SearchPage(),
      ),
    );
    await tester.enterText(find.byType(TextField), '同济');
    await tester.tap(find.byIcon(Icons.search));
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/search_page.png'),
    );
  });

  testWidgets('login page golden', skip: skipGoldens, (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const LoginPage()),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/login_page.png'),
    );
  });
}
