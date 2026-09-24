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
import 'package:go_router/go_router.dart';
import 'package:forum_app/src/pages/topic/mention_panel.dart';
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
  Completer<PostWindowPayload>? pendingOlder;
  int paginatedCalls = 0;
  @override
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) async {
    if (anchorPostId == null && pendingOlder != null) {
      paginatedCalls++;
      return pendingOlder!.future;
    }
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

class _Pages extends FakePageRepository {
  _Pages(super.client, this.hasMore);
  final bool hasMore;
  @override
  Future<PagePayload> fetch(String path) async {
    if (!hasMore || !path.startsWith('/p/post/')) return super.fetch(path);
    final json = topicDetailPayloadJson();
    final stream = (json['props'] as Map)['postStream'] as Map;
    stream['hasAfter'] = true;
    stream['afterPostNo'] = 3;
    return parsePayload(json);
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
    bool routed = false,
    bool hasMore = false,
    Completer<PostWindowPayload>? pendingPagination,
  }) async {
    if (fresh) {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      store = _Store();
      posts = _Posts(client);
      topics = _Topics(client)..pendingOlder = pendingPagination;
      container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(client),
          currentUserProvider.overrideWith(
            (ref) async => identity == null
                ? const CurrentUser(id: 1, username: 'alice')
                : await identity,
          ),
          pageRepositoryProvider.overrideWithValue(_Pages(client, hasMore)),
          topicRepositoryProvider.overrideWithValue(topics),
          postRepositoryProvider.overrideWithValue(posts),
          writingStoreProvider.overrideWithValue(store),
          offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
        ],
      );
      addTearDown(container.dispose);
    }
    GoRouter? router;
    if (routed) {
      router = GoRouter(
        initialLocation: '/topic',
        routes: [
          GoRoute(
            path: '/',
            builder: (_, _) => const Scaffold(body: Text('返回列表')),
          ),
          GoRoute(
            path: '/topic',
            builder: (_, _) => TopicPage(topicId: 100, initialPostNo: floor),
          ),
        ],
      );
      addTearDown(router.dispose);
    }
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: routed
            ? MaterialApp.router(
                routerConfig: router,
                theme: gfThemeData(Brightness.light),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                locale: const Locale('zh'),
              )
            : MaterialApp(
                theme: gfThemeData(Brightness.light),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                locale: const Locale('zh'),
                home: TopicPage(topicId: 100, initialPostNo: floor),
              ),
      ),
    );
    if (pendingPagination == null) {
      await tester.pumpAndSettle();
    } else {
      await tester.pump(const Duration(milliseconds: 100));
    }
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

  testWidgets('restored reply rebuilds local mention candidates', (
    tester,
  ) async {
    await WritingStore().save(
      scope,
      const LocalDraft(
        key: 'reply-100',
        kind: DraftKind.reply,
        title: '话题',
        content: 'unfinished',
        contentType: 2,
        topicId: 100,
        categories: [],
        images: [],
        updatedAt: 1,
        replyToPostId: 9002,
        replyTargetName: 'bob',
      ),
    );
    await pumpTopic(tester);
    composer(tester).controller.value = const TextEditingValue(
      text: '@',
      selection: TextSelection.collapsed(offset: 1),
    );
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<MentionCandidatesPanel>(find.byType(MentionCandidatesPanel))
          .session
          .candidates
          .map((user) => user.username),
      contains('bob'),
    );
    await disposePage(tester);
  });
  testWidgets('anchoring a sent reply records the updated return-state count', (
    tester,
  ) async {
    await pumpTopic(tester);
    await open(tester);
    composer(tester).controller.text = 'new reply';
    composer(tester).onPublish();
    await tester.pumpAndSettle();
    expect(container.read(topicReturnStatesProvider)[100]!.replyCount, 3);
    await disposePage(tester);
  });
  testWidgets(
    'storage failure offers an explicit unsaved exit without deleting the old copy',
    (tester) async {
      await pumpTopic(tester, routed: true);
      await open(tester);
      composer(tester).controller.text = 'old saved reply';
      await autosave(tester);
      store.fail = true;
      composer(tester).controller.text = 'unsaved changes';
      await tester.tap(find.byTooltip('返回'));
      await tester.pumpAndSettle();
      expect(find.text('放弃修改'), findsOneWidget);
      await tester.tap(find.text('继续编辑'));
      await tester.pumpAndSettle();
      expect(composer(tester).controller.text, 'unsaved changes');
      await tester.tap(find.byTooltip('返回'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('放弃修改'));
      await tester.pumpAndSettle();
      expect(find.text('返回列表'), findsOneWidget);
      expect((await store.drafts(scope)).single.content, 'old saved reply');
      await disposePage(tester);
    },
  );
  testWidgets('anchored send resets an invalidated pagination loading flag', (
    tester,
  ) async {
    final pending = Completer<PostWindowPayload>();
    await pumpTopic(tester, hasMore: true, pendingPagination: pending);
    for (var i = 0; i < 6 && topics.paginatedCalls == 0; i++) {
      await tester.drag(find.byType(CustomScrollView), const Offset(0, -600));
      await tester.pump(const Duration(milliseconds: 100));
    }
    expect(topics.paginatedCalls, 1);
    tester
        .widget<GfFloatingControls>(find.byType(GfFloatingControls))
        .onOpenReply!();
    await tester.pump();
    composer(tester).controller.text = 'reply during pagination';
    composer(tester).onPublish();
    await tester.pump(const Duration(milliseconds: 50));
    await tester.pump(const Duration(milliseconds: 300));
    final original = parsePageProps<TopicDetailProps>(
      parsePayload(topicDetailPayloadJson()),
    )!.postStream;
    topics.pendingOlder!.complete(original);
    await tester.pump(const Duration(milliseconds: 400));
    final l10n = AppLocalizations.of(tester.element(find.byType(TopicPage)));
    expect(
      tester
          .widget<TextButton>(
            find.widgetWithText(TextButton, l10n.topicEarlierReplies),
          )
          .onPressed,
      isNotNull,
    );
    await disposePage(tester);
  });

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
    'unavailable ownership keeps unsent text but not an acknowledged empty reply',
    (tester) async {
      await pumpTopic(tester, identity: Future<CurrentUser?>.value(null));
      await open(tester);
      composer(tester).controller.text = '身份暂不可用时的正文';
      composer(tester).onCollapse!();
      await tester.pumpAndSettle();
      expect(composer(tester).controller.text, '身份暂不可用时的正文');
      expect(find.text('本机保存失败，请重试'), findsOneWidget);
      expect(find.text('已保存到本机'), findsNothing);
      composer(tester).onPublish();
      await tester.pumpAndSettle();
      expect(find.byType(GfPostComposer), findsNothing);
      expect(find.text('刚发出的回复'), findsOneWidget);
      expect(await store.drafts(scope), isEmpty);
      await disposePage(tester);
    },
  );

  testWidgets('late failed ownership cannot reopen a successfully sent reply', (
    tester,
  ) async {
    final identity = Completer<CurrentUser?>();
    await pumpTopic(tester, identity: identity.future);
    await open(tester);
    composer(tester).controller.text = '等待身份时发送';
    await autosave(tester);
    composer(tester).onPublish();
    await tester.pump();
    identity.complete(null);
    await tester.pumpAndSettle();
    expect(find.byType(GfPostComposer), findsNothing);
    expect(find.text('刚发出的回复'), findsOneWidget);
    expect(find.text('本机保存失败，请重试'), findsNothing);
    expect(await store.drafts(scope), isEmpty);
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
