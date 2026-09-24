import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/pages/topic/post_actions.dart';
import 'package:forum_app/src/providers.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart'
    show
        FakePageRepository,
        FakeTopicRepository,
        MemoryTokenStorage,
        NoopOfflineCache;
import 'fixtures/page_fixtures.dart';

class _Store extends WritingStore {
  bool fail = false;
  Completer<void>? pendingSave;
  @override
  Future<void> save(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) async {
    if (fail) throw StateError('disk unavailable');
    if (pendingSave != null) await pendingSave!.future;
    await super.save(scope, draft, isCurrent: isCurrent);
  }
}

class _Posts extends PostRepository {
  _Posts(super.client);
  Completer<CreatePostResult>? pending;
  bool fail = false;
  @override
  Future<CreatePostResult> createPost({
    required int topicId,
    required String content,
    int replyToPostId = 0,
    String? captchaId,
    String? captchaCode,
  }) async {
    if (fail) throw const NetworkException(fallbackMessage: 'offline');
    return pending == null
        ? const CreatePostResult(id: 9100, postNo: 4, renderedContent: '')
        : pending!.future;
  }
}

class _Topics extends FakeTopicRepository {
  _Topics(super.client);
  int? requestedAnchor;
  @override
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) async {
    requestedAnchor = anchorPostId;
    final original = parsePageProps<TopicDetailProps>(
      parsePayload(topicDetailPayloadJson()),
    )!.postStream;
    return original.copyWith(
      posts: [
        original.posts.last.copyWith(
          id: 9100,
          postNo: 4,
          content: '刚发出的回复',
          replyToPostId: null,
          replyToUsername: null,
        ),
      ],
      beforePostNo: 4,
      afterPostNo: 4,
      maxPostNo: 4,
      total: 4,
      hasBefore: true,
      hasAfter: false,
    );
  }
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  const scope = 'http%3A%2F%2Ffake.local:1';
  late ProviderContainer container;
  late _Store store;
  late _Posts posts;
  late _Topics topics;

  Future<void> pumpTopic(
    WidgetTester tester, {
    bool fresh = true,
    int? floor,
    Future<CurrentUser?>? identity,
  }) async {
    if (fresh) {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      store = _Store();
      posts = _Posts(client);
      topics = _Topics(client);
      container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(client),
          currentUserProvider.overrideWith(
            (ref) async => identity == null
                ? const CurrentUser(id: 1, username: 'alice')
                : await identity,
          ),
          pageRepositoryProvider.overrideWithValue(FakePageRepository(client)),
          topicRepositoryProvider.overrideWithValue(topics),
          postRepositoryProvider.overrideWithValue(posts),
          writingStoreProvider.overrideWithValue(store),
          offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
        ],
      );
      addTearDown(container.dispose);
    }
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          home: TopicPage(topicId: 100, initialPostNo: floor),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  GfPostComposer composer(WidgetTester tester) =>
      tester.widget<GfPostComposer>(find.byType(GfPostComposer));
  Future<void> open(WidgetTester tester) async {
    tester
        .widget<GfFloatingControls>(find.byType(GfFloatingControls))
        .onOpenReply!();
    await tester.pumpAndSettle();
  }

  Future<void> disposePage(WidgetTester tester) async {
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  }

  Future<void> autosave(WidgetTester tester) async {
    await tester.pump(const Duration(milliseconds: 800));
    await tester.pumpAndSettle();
  }

  testWidgets('viewing a topic creates no empty reply draft', (tester) async {
    await pumpTopic(tester);
    await disposePage(tester);
    expect(await store.drafts(scope), isEmpty);
  });

  testWidgets('collapse awaits unresolved ownership before reporting a save', (
    tester,
  ) async {
    final identity = Completer<CurrentUser?>();
    await pumpTopic(tester, identity: identity.future);
    await open(tester);
    composer(tester).controller.text = '身份加载中输入';
    await tester.pump();
    expect(
      tester
          .widget<PopScope>(
            find.byWidgetPredicate((widget) => widget is PopScope),
          )
          .canPop,
      isFalse,
    );
    composer(tester).onCollapse!();
    await tester.pump();
    expect(find.byType(GfPostComposer), findsOneWidget);
    expect(await store.drafts(scope), isEmpty);
    expect(find.text('已保存到本机'), findsNothing);
    identity.complete(const CurrentUser(id: 1, username: 'alice'));
    await tester.pumpAndSettle();
    expect(find.byType(GfPostComposer), findsNothing);
    expect((await store.drafts(scope)).single.content, '身份加载中输入');
    await disposePage(tester);
  });

  testWidgets('background transition flushes before the autosave delay', (
    tester,
  ) async {
    await pumpTopic(tester);
    await open(tester);
    composer(tester).controller.text = '后台恢复内容';
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    await tester.pump();
    expect((await store.drafts(scope)).single.content, '后台恢复内容');
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await disposePage(tester);
  });

  testWidgets('reply survives collapse and page recreation', (tester) async {
    await pumpTopic(tester);
    await open(tester);
    composer(tester).controller.text = '不要丢失的回复';
    composer(tester).onCollapse!();
    await tester.pumpAndSettle();
    await disposePage(tester);
    await pumpTopic(tester, fresh: false);
    expect((await store.drafts(scope)).single.content, '不要丢失的回复');
    expect(composer(tester).controller.text, '不要丢失的回复');
    await disposePage(tester);
  });

  testWidgets(
    'choosing another reply target preserves body and collapse preserves target',
    (tester) async {
      await pumpTopic(tester);
      await open(tester);
      composer(tester).controller.text = '已经写好的正文';
      final actions = tester
          .widgetList<PostActions>(find.byType(PostActions))
          .toList();
      actions.firstWhere((action) => action.post.id == 9002).onReply!();
      await tester.pumpAndSettle();
      expect(composer(tester).controller.text, '@bob 已经写好的正文');
      expect(composer(tester).targetName, 'bob');
      actions.firstWhere((action) => action.post.id == 9003).onReply!();
      await tester.pumpAndSettle();
      expect(composer(tester).controller.text, '@carol 已经写好的正文');
      composer(tester).onCollapse!();
      await tester.pumpAndSettle();
      await disposePage(tester);
      await pumpTopic(tester, fresh: false);
      expect(composer(tester).targetName, 'carol');
      expect(composer(tester).controller.text, '@carol 已经写好的正文');
      expect((await store.drafts(scope)).single.replyToPostId, 9003);
      await disposePage(tester);
    },
  );

  testWidgets('floor navigation retains unsent reply', (tester) async {
    await pumpTopic(tester);
    await open(tester);
    composer(tester).controller.text = '跳楼也保留';
    await pumpTopic(tester, fresh: false, floor: 2);
    expect(composer(tester).controller.text, '跳楼也保留');
    await disposePage(tester);
    expect((await store.drafts(scope)).single.content, '跳楼也保留');
  });

  testWidgets('failed storage keeps composer open and exposes retry', (
    tester,
  ) async {
    await pumpTopic(tester);
    await open(tester);
    store.fail = true;
    composer(tester).controller.text = '保存失败也不丢';
    composer(tester).onCollapse!();
    await tester.pumpAndSettle();
    expect(composer(tester).controller.text, '保存失败也不丢');
    expect(find.text('本机保存失败，请重试'), findsOneWidget);
    expect(find.text('已保存到本机'), findsNothing);
    store.fail = false;
    await tester.tap(find.text('重试'));
    await tester.pumpAndSettle();
    expect((await store.drafts(scope)).single.content, '保存失败也不丢');
    await disposePage(tester);
  });

  testWidgets(
    'failed send keeps draft and successful send clears only acknowledged text',
    (tester) async {
      await pumpTopic(tester);
      await open(tester);
      composer(tester).controller.text = '第一条';
      await autosave(tester);
      posts.fail = true;
      composer(tester).onPublish();
      await tester.pumpAndSettle();
      expect(composer(tester).controller.text, '第一条');
      expect((await store.drafts(scope)).single.content, '第一条');
      posts.fail = false;
      posts.pending = Completer<CreatePostResult>();
      composer(tester).onPublish();
      await tester.pump();
      composer(tester).controller.text = '发送期间写的第二条';
      posts.pending!.complete(
        const CreatePostResult(id: 9100, postNo: 4, renderedContent: ''),
      );
      await autosave(tester);
      expect(composer(tester).controller.text, '发送期间写的第二条');
      expect((await store.drafts(scope)).single.content, '发送期间写的第二条');
      expect(topics.requestedAnchor, 9100);
      expect(find.text('刚发出的回复'), findsOneWidget);
      posts.pending = null;
      composer(tester).onPublish();
      await tester.pumpAndSettle();
      expect(await store.drafts(scope), isEmpty);
      await disposePage(tester);
    },
  );

  testWidgets(
    'typing during a pending collapse save keeps the new revision open',
    (tester) async {
      await pumpTopic(tester);
      await open(tester);
      store.pendingSave = Completer<void>();
      composer(tester).controller.text = '保存开始时的内容';
      composer(tester).onCollapse!();
      await tester.pump();
      composer(tester).controller.text = '保存等待期间的新内容';
      store.pendingSave!.complete();
      await tester.pumpAndSettle();
      expect(find.byType(GfPostComposer), findsOneWidget);
      expect(composer(tester).controller.text, '保存等待期间的新内容');
      await autosave(tester);
      expect((await store.drafts(scope)).single.content, '保存等待期间的新内容');
      await disposePage(tester);
    },
  );

  testWidgets('session invalidation suppresses a pending draft write', (
    tester,
  ) async {
    await pumpTopic(tester);
    await open(tester);
    composer(tester).controller.text = '旧会话未保存正文';
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await autosave(tester);
    expect(find.byType(GfPostComposer), findsNothing);
    expect(await store.drafts(scope), isEmpty);
    await disposePage(tester);
    expect(await store.drafts(scope), isEmpty);
  });
}
