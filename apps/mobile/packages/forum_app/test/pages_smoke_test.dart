import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/l10n/app_localizations.dart';
import 'package:core/core.dart';

import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/category/category_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/status_views.dart';

import 'fixtures/page_fixtures.dart';

/// 内存 PageRepository:返回 fixture,不发真实网络。
class FakePageRepository extends PageRepository {
  FakePageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
    if (path == '/' || path.startsWith('/?sort=')) {
      return parsePayload(homePayloadJson());
    }
    if (path.startsWith('/p/post/')) {
      return parsePayload(topicDetailPayloadJson());
    }
    if (path.startsWith('/u/')) {
      return parsePayload(userProfilePayloadJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

class FakeTopicRepository extends TopicRepository {
  FakeTopicRepository(super.client);

  @override
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) async {
    final props = parsePayload(topicDetailPayloadJson());
    return (parsePageProps<TopicDetailProps>(props)!).postStream;
  }
}

/// 测试用内存 TokenStorage(生产已改用 SecureTokenStorage)。
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
  TestWidgetsFlutterBinding.ensureInitialized();

  Future<ProviderContainer> makeContainer() async {
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
        offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  testWidgets('首页在 mock repository 下可构建并渲染话题', (tester) async {
    final container = await makeContainer();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          home: const HomePage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('移动端测试话题'), findsOneWidget);
    expect(find.text('最新'), findsOneWidget);
  });

  testWidgets('话题详情在 mock repository 下可构建并渲染帖子', (tester) async {
    final container = await makeContainer();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          home: const TopicPage(topicId: 100),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('第一楼'), findsOneWidget);
    expect(find.text('移动端测试话题'), findsOneWidget);

    // markdown_widget 的 VisibilityDetector 会创建 500ms 延迟 Timer,
    // 需推进时钟让其过期,避免 "Timer is still pending"。
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('用户主页在 mock repository 下可构建', (tester) async {
    final container = await makeContainer();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          home: const ProfilePage(userId: 1),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Alice'), findsOneWidget);
    expect(find.text('同济学生'), findsOneWidget);
  });

  testWidgets('分类页在 mock repository 下可构建', (tester) async {
    final container = await makeContainer();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          home: const CategoryPage(slug: 'dev', categoryId: 1),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // FakePageRepository 不支持分类页,应显示错误重试而非崩溃。
    expect(find.byType(GfErrorRetry), findsOneWidget);
  });
}
