import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
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
      createdAt: '2026-08-07T10:00:00+08:00',
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

  testWidgets('home page golden', (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const HomePage()),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/home_page.png'),
    );
  });

  testWidgets('topic page golden', (tester) async {
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

  testWidgets('notifications page golden', (tester) async {
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

  testWidgets('messages page golden', (tester) async {
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

  testWidgets('login page golden', (tester) async {
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
