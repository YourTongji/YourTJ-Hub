import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:drift/native.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart'
    show
        MemTokenStorage,
        NoopCache,
        CountingPageRepository,
        PollingChatRepository,
        makeChatMessage;

class _Pages extends PageRepository {
  _Pages(super.client);
  final response = Completer<PagePayload>();
  int calls = 0;
  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) {
    calls++;
    return response.future;
  }
}

class _DelayedChat extends PollingChatRepository {
  _DelayedChat(super.client) : super(messages: []);
  final response = Completer<ChatMessagesResponse>();
  @override
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int aroundId = 0,
    int limit = 30,
    Object? cancelToken,
  }) {
    afterCalls++;
    return response.future;
  }
}

class _Cache extends NoopCache implements OfflineHomeCache {
  @override
  Future<List<ChatMessagePayload>> getMessages(int id) async => [
    makeChatMessage(9).copyWith(content: 'cached older message'),
  ];
  int writes = 0;
  int conversationWrites = 0;
  @override
  Future<List<ChatItemPayload>> getConversations() async =>
      parsePageProps<MessagesPageProps>(
        parsePayload(messagesPayloadJson()),
      )!.conversations;
  @override
  Future<void> putConversations(List<ChatItemPayload> items) async {
    conversationWrites++;
  }

  @override
  Future<PagePayload?> get(int id) async =>
      parsePayload(topicDetailPayloadJson());
  @override
  Future<void> put(int id, Map<String, dynamic> payload) async => writes++;
  @override
  Future<PagePayload?> getHomePage({
    required int accountId,
    required String baseUrl,
    required String sort,
  }) async => parsePayload(homePayloadJson());
  @override
  Future<void> putHomePage({
    required int accountId,
    required String baseUrl,
    required String sort,
    required PagePayload payload,
  }) async => writes++;
}

class _FailedRemovalCache extends DriftOfflineCache {
  _FailedRemovalCache(super.database);
  @override
  DriftOfflineCache capture() => this;
  @override
  Future<PagePayload?> get(int id) async =>
      parsePayload(topicDetailPayloadJson());
  @override
  Future<void> removeTopic(int topicId) async =>
      throw StateError('disk unavailable');
}

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });
  Future<({ProviderContainer container, _Pages pages, _Cache cache})> pump(
    WidgetTester tester,
    Widget page, {
    List<Override> overrides = const [],
  }) async {
    final tokens = MemTokenStorage();
    await tokens.write('token');
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: tokens,
      baseUrl: 'http://fake',
    );
    final pages = _Pages(client);
    final cache = _Cache();
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(tokens),
        currentUserProvider.overrideWith(
          (ref) async => const CurrentUser(id: 1, username: 'alice'),
        ),
        pageRepositoryProvider.overrideWithValue(pages),
        offlineTopicCacheProvider.overrideWithValue(cache),
        offlineChatCacheProvider.overrideWithValue(cache),
        ...overrides,
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: page,
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    return (container: container, pages: pages, cache: cache);
  }

  testWidgets('topic shows a read-only snapshot while network is pending', (
    tester,
  ) async {
    final app = await pump(tester, const TopicPage(topicId: 100));
    expect(find.textContaining('Local copy'), findsOneWidget);
    expect(find.text('Join discussion'), findsNothing);
    app.pages.response.complete(parsePayload(topicDetailPayloadJson()));
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 1));
  });
  testWidgets(
    'clearing forum removes home memory and fences the pending response',
    (tester) async {
      final app = await pump(tester, const HomePage());
      final callsBeforeClear = app.pages.calls;
      app.container.read(cacheClearEpochProvider.notifier).invalidate({
        CacheCategory.forum,
      });
      await tester.pump();
      app.pages.response.complete(parsePayload(homePayloadJson()));
      await tester.pumpAndSettle();
      expect(find.textContaining('Cache cleared'), findsOneWidget);
      expect(app.cache.writes, 0);
      expect(app.pages.calls, callsBeforeClear);
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );
  testWidgets(
    'a malformed messages response does not replace a local snapshot with empty',
    (tester) async {
      final app = await pump(tester, const MessagesPage());
      expect(find.textContaining('Local copy'), findsOneWidget);
      app.pages.response.complete(parsePayload(homePayloadJson()));
      await tester.pumpAndSettle();
      expect(find.textContaining('Local copy'), findsOneWidget);
      expect(app.cache.conversationWrites, 0);
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );
  testWidgets('a different cache category leaves the home request intact', (
    tester,
  ) async {
    final app = await pump(tester, const HomePage());
    app.container.read(cacheClearEpochProvider.notifier).invalidate({
      CacheCategory.chat,
    });
    app.pages.response.complete(parsePayload(homePayloadJson()));
    await tester.pumpAndSettle();
    expect(find.textContaining('Cache cleared'), findsNothing);
    expect(app.cache.writes, 1);
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 1));
  });
  testWidgets('topic floor links skip the unrelated first-page snapshot', (
    tester,
  ) async {
    final app = await pump(
      tester,
      const TopicPage(topicId: 100, initialPostNo: 8),
    );
    expect(find.textContaining('Local copy'), findsNothing);
    app.pages.response.completeError(
      const NetworkException(fallbackMessage: 'offline'),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining('Local copy'), findsNothing);
    expect(app.cache.writes, 0);
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 1));
  });
  testWidgets(
    'valid empty conversation snapshots are shown before the network',
    (tester) async {
      final database = AppDatabase(NativeDatabase.memory());
      addTearDown(database.close);
      final cache = DriftOfflineCache(database);
      await tester.runAsync(() => cache.putConversations([]));
      final app = await pump(
        tester,
        const MessagesPage(),
        overrides: [offlineChatCacheProvider.overrideWithValue(cache)],
      );
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(find.textContaining('Local copy'), findsOneWidget);
      app.pages.response.complete(parsePayload(messagesPayloadJson()));
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );
  testWidgets(
    'cached messages neither acknowledge reads nor survive initial server replacement',
    (tester) async {
      final tokens = MemTokenStorage();
      await tokens.write('token');
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: tokens,
        baseUrl: 'http://fake',
      );
      final chat = _DelayedChat(client);
      final app = await pump(
        tester,
        const MessagesPage(targetUserId: 2),
        overrides: [
          pageRepositoryProvider.overrideWithValue(
            CountingPageRepository(client),
          ),
          chatRepositoryProvider.overrideWithValue(chat),
        ],
      );
      await tester.pump(const Duration(seconds: 1));
      expect(find.text('cached older message'), findsOneWidget);
      expect(
        tester.widget<GfChatInput>(find.byType(GfChatInput)).canSend,
        isFalse,
      );
      expect(chat.visibleReadBatches, isEmpty);
      chat.response.complete(
        const ChatMessagesResponse(
          list: [],
          hasMoreBefore: false,
          hasMoreAfter: false,
          nextBeforeId: 0,
          latestId: 0,
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('cached older message'), findsNothing);
      expect(
        tester.widget<GfChatInput>(find.byType(GfChatInput)).canSend,
        isTrue,
      );
      expect(chat.visibleReadBatches, isEmpty);
      app.container.read(cacheClearEpochProvider.notifier).invalidate({
        CacheCategory.chat,
      });
      await tester.pump(const Duration(seconds: 16));
      expect(find.textContaining('Cache cleared'), findsOneWidget);
      expect(chat.afterCalls, 1);
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );
  for (final status in [403, 404, 410]) {
    testWidgets(
      'topic $status removes the snapshot instead of retaining stale access',
      (tester) async {
        final database = AppDatabase(NativeDatabase.memory());
        addTearDown(database.close);
        final cache = DriftOfflineCache(database);
        final payload = topicDetailPayloadJson();
        (payload['props']['topic'] as Map<String, dynamic>)['topicStatus'] = 1;
        await tester.runAsync(() => cache.put(100, payload));
        final app = await pump(
          tester,
          const TopicPage(topicId: 100),
          overrides: [offlineTopicCacheProvider.overrideWithValue(cache)],
        );
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        });
        await tester.pump();
        expect(find.textContaining('Local copy'), findsOneWidget);
        app.pages.response.completeError(
          ApiException(fallbackMessage: 'Access denied', statusCode: status),
        );
        await tester.pumpAndSettle();
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        });
        await tester.pumpAndSettle();
        expect(find.textContaining('Local copy'), findsNothing);
        expect(find.byType(GfErrorRetry), findsOneWidget);
        expect(await tester.runAsync(() => cache.get(100)), isNull);
        await tester.pumpWidget(const SizedBox());
        await tester.pump(const Duration(seconds: 1));
      },
    );
  }
  for (final status in [403, 404, 410]) {
    testWidgets(
      'home $status removes the snapshot instead of retaining stale access',
      (tester) async {
        final database = AppDatabase(NativeDatabase.memory());
        addTearDown(database.close);
        final cache = DriftOfflineCache(database);
        await tester.runAsync(
          () => cache.putHomePage(
            accountId: 1,
            baseUrl: GfApiClient.defaultBaseUrl,
            sort: '',
            payload: PagePayload.fromJson(homePayloadJson()),
          ),
        );
        final app = await pump(
          tester,
          const HomePage(),
          overrides: [offlineTopicCacheProvider.overrideWithValue(cache)],
        );
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        });
        await tester.pump();
        expect(find.textContaining('Local copy'), findsOneWidget);
        app.pages.response.completeError(
          ApiException(fallbackMessage: 'Access denied', statusCode: status),
        );
        await tester.pumpAndSettle();
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 20));
        });
        await tester.pumpAndSettle();
        expect(find.textContaining('Local copy'), findsNothing);
        expect(find.byType(GfErrorRetry), findsOneWidget);
        expect(
          await tester.runAsync(
            () => cache.getHomePage(
              accountId: 1,
              baseUrl: GfApiClient.defaultBaseUrl,
              sort: '',
            ),
          ),
          isNull,
        );
        await tester.pumpWidget(const SizedBox());
        await tester.pump(const Duration(seconds: 1));
      },
    );
  }
  testWidgets(
    'messages 403 removes the snapshot instead of retaining stale access',
    (tester) async {
      final database = AppDatabase(NativeDatabase.memory());
      addTearDown(database.close);
      final cache = DriftOfflineCache(database);
      await tester.runAsync(() async {
        await cache.putConversations(
          parsePageProps<MessagesPageProps>(
            parsePayload(messagesPayloadJson()),
          )!.conversations,
        );
      });
      final app = await pump(
        tester,
        const MessagesPage(),
        overrides: [offlineChatCacheProvider.overrideWithValue(cache)],
      );
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(find.textContaining('Local copy'), findsOneWidget);
      app.pages.response.completeError(
        const ApiException(fallbackMessage: 'Access denied', statusCode: 403),
      );
      await tester.pumpAndSettle();
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pumpAndSettle();
      expect(find.textContaining('Local copy'), findsNothing);
      expect(find.byType(GfErrorRetry), findsOneWidget);
      expect(
        await tester.runAsync(() => cache.getConversations()),
        isEmpty,
      );
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );
  testWidgets(
    'conversation 403 removes the cached thread instead of retaining stale access',
    (tester) async {
      final database = AppDatabase(NativeDatabase.memory());
      addTearDown(database.close);
      final cache = DriftOfflineCache(database);
      await tester.runAsync(() => cache.putMessages(1, [makeChatMessage(9)]));
      final tokens = MemTokenStorage();
      await tokens.write('token');
      final client = GfApiClient(dio: Dio(), tokenStorage: tokens, baseUrl: 'http://fake');
      final chat = _DelayedChat(client);
      await pump(
        tester,
        const MessagesPage(targetUserId: 2),
        overrides: [
          pageRepositoryProvider.overrideWithValue(CountingPageRepository(client)),
          chatRepositoryProvider.overrideWithValue(chat),
          offlineChatCacheProvider.overrideWithValue(cache),
        ],
      );
      await tester.pump(const Duration(seconds: 1));
      expect(find.text('消息 9'), findsOneWidget);
      expect(find.textContaining('Local copy'), findsOneWidget);
      chat.response.completeError(
        const ApiException(fallbackMessage: 'Access denied', statusCode: 403),
      );
      await tester.pumpAndSettle();
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pumpAndSettle();
      expect(find.text('消息 9'), findsNothing);
      expect(find.textContaining('Local copy'), findsNothing);
      expect(find.byType(GfErrorRetry), findsOneWidget);
      expect(await tester.runAsync(() => cache.getMessages(1)), isEmpty);
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );
  testWidgets(
    'access denial hides a topic even when removing the disk snapshot fails',
    (tester) async {
      final database = AppDatabase(NativeDatabase.memory());
      addTearDown(database.close);
      final app = await pump(
        tester,
        const TopicPage(topicId: 100),
        overrides: [
          offlineTopicCacheProvider.overrideWithValue(
            _FailedRemovalCache(database),
          ),
        ],
      );
      expect(find.textContaining('Local copy'), findsOneWidget);
      app.pages.response.completeError(
        const ApiException(fallbackMessage: 'Access denied', statusCode: 403),
      );
      await tester.pumpAndSettle();
      expect(find.textContaining('Local copy'), findsNothing);
      expect(find.byType(GfErrorRetry), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pump(const Duration(seconds: 1));
    },
  );
}
