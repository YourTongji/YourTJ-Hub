import 'dart:async';
import 'dart:math' as math;
import 'dart:convert';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:image/image.dart' as img;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/topic/post_actions.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage, NoopOfflineCache;

/// 通知锚定与回复窗口(issue #878):
/// 1. 通知深链进入的锚点窗口仍可向前/向后续载;
/// 2. 回复后已加载评论不消失,新回复按 id 去重并入并滚入视野;
/// 3. 合并后前后游标仍指向已加载窗口边界,续载不重复也不跳楼。
PostPayload floorPost(
  int floor, {
  String? content,
  int? replyToPostId,
  String? replyToUsername,
}) {
  return PostPayload(
    id: 9000 + floor,
    topicId: 100,
    postNo: floor,
    content: content ?? '$floor楼内容',
    renderedContent: '',
    processStatus: 0,
    isHidden: false,
    canModerate: false,
    author: UserBriefPayload(id: floor, username: 'user$floor', avatarUrl: ''),
    createdAt: '2025-01-15T09:00:00+08:00',
    isOwnPost: false,
    revisionCount: 1,
    likeCount: 0,
    isLiked: false,
    isBookmarked: false,
    replyToPostId: replyToPostId,
    replyToUsername: replyToUsername,
  );
}

/// 通知锚点页:与服务端 topicInitialPosts 一致,返回锚点起的窗口。
Map<String, dynamic> anchoredPageJson({
  required List<PostPayload> posts,
  required bool hasBefore,
  required bool hasAfter,
  List<ReplyTargetPayload> replyTargets = const <ReplyTargetPayload>[],
}) {
  final Map<String, dynamic> json = topicDetailPayloadJson();
  final Map<String, dynamic> props = json['props'] as Map<String, dynamic>;
  final Map<String, dynamic> topic = props['topic'] as Map<String, dynamic>;
  final Map<String, dynamic> stream =
      props['postStream'] as Map<String, dynamic>;
  final int maxPostNo = posts.isEmpty ? 1 : posts.last.postNo;
  topic
    ..['replyCount'] = maxPostNo - 1
    ..['maxPostNo'] = maxPostNo;
  stream
    ..['posts'] = <Map<String, dynamic>>[
      for (final PostPayload post in posts)
        <String, dynamic>{...post.toJson(), 'author': post.author.toJson()},
    ]
    ..['replyTargets'] = <Map<String, dynamic>>[
      for (final ReplyTargetPayload target in replyTargets)
        <String, dynamic>{...target.toJson(), 'author': target.author.toJson()},
    ]
    ..['hasBefore'] = hasBefore
    ..['hasAfter'] = hasAfter
    ..['beforePostNo'] = posts.isEmpty ? null : posts.first.postNo
    ..['afterPostNo'] = posts.isEmpty ? null : posts.last.postNo
    ..['total'] = maxPostNo
    ..['maxPostNo'] = maxPostNo;
  return json;
}

class _NotificationPageRepository extends PageRepository {
  _NotificationPageRepository(super.client, this.page);

  final Map<String, dynamic> page;

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (!path.startsWith('/p/post/')) {
      throw UnimplementedError('unexpected page path: $path');
    }
    return parsePayload(page);
  }
}

/// 内存版帖子窗口服务:复刻后端 PostWindow 的 anchor/before/after 分支语义,
/// 以便断言客户端请求的 limit 与游标。
class TopicWindowServer extends TopicRepository {
  TopicWindowServer(super.client);

  /// 服务端楼层(按 postNo 升序)。
  final List<PostPayload> posts = <PostPayload>[];

  /// 非版主视角下被服务端过滤的楼层(如待审回复,payload.go 的 pending 过滤)。
  final Set<int> hiddenPostIds = <int>{};

  /// before 窗口是否包含游标楼层本身(真实服务端严格早于游标,这里用于
  /// 构造重叠窗口,验证客户端仍按 id 去重)。
  bool overlapBeforeCursor = false;
  bool delayNextTail = false;
  Completer<PostWindowPayload>? delayedTail;
  int failTailRequests = 0;
  final List<String> calls = <String>[];

  PostWindowPayload _payload(
    List<PostPayload> window, {
    required bool hasBefore,
    required bool hasAfter,
    int? anchorPostId,
  }) {
    // 与服务端一致:先过滤不可见楼层,再由剩余楼层推导前后游标。
    final List<PostPayload> visible = window
        .where((PostPayload post) => !hiddenPostIds.contains(post.id))
        .toList(growable: false);
    return PostWindowPayload(
      posts: visible,
      replyTargets: const <ReplyTargetPayload>[],
      anchorPostId: anchorPostId,
      beforePostNo: visible.isEmpty ? null : visible.first.postNo,
      afterPostNo: visible.isEmpty ? null : visible.last.postNo,
      hasBefore: hasBefore,
      hasAfter: hasAfter,
      total: posts.length,
      maxPostNo: posts.isEmpty ? 1 : posts.last.postNo,
    );
  }

  @override
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) async {
    calls.add(
      'anchor=${anchorPostId ?? 0}${anchorPostNo == null ? '' : ' anchorNo=$anchorPostNo'} '
      'before=${beforePostNo ?? 0} '
      'after=${afterPostNo ?? 0} limit=${limit ?? 0}',
    );
    final int size = limit ?? 20;
    if (beforePostNo != null && beforePostNo >= 0x7fffffffffffffff) {
      if (failTailRequests > 0) {
        failTailRequests--;
        throw StateError('temporary window failure');
      }
      if (delayNextTail) {
        delayNextTail = false;
        delayedTail = Completer<PostWindowPayload>();
        return delayedTail!.future;
      }
    }
    if (anchorPostId != null) {
      final int index = posts.indexWhere((post) => post.id == anchorPostId);
      if (index < 0) {
        return _payload(
          const <PostPayload>[],
          hasBefore: false,
          hasAfter: false,
          anchorPostId: anchorPostId,
        );
      }
      final int beforeLimit = math.min(5, size ~/ 2);
      final int afterLimit = size - beforeLimit - 1;
      final List<PostPayload> before = posts.sublist(
        math.max(0, index - beforeLimit - 1),
        index,
      );
      final List<PostPayload> after = posts.sublist(
        index + 1,
        math.min(posts.length, index + afterLimit + 2),
      );
      final bool hasBefore = before.length > beforeLimit;
      final bool hasAfter = after.length > afterLimit;
      return _payload(
        <PostPayload>[
          ...(hasBefore ? before.sublist(1) : before),
          posts[index],
          ...(hasAfter ? after.sublist(0, afterLimit) : after),
        ],
        hasBefore: hasBefore,
        hasAfter: hasAfter,
        anchorPostId: anchorPostId,
      );
    }
    if (beforePostNo != null) {
      final int found = posts.indexWhere((post) => post.postNo >= beforePostNo);
      final int cursor = found < 0 ? posts.length : found;
      final int stop = overlapBeforeCursor
          ? math.min(posts.length, cursor + 1)
          : cursor;
      final int begin = math.max(0, stop - size - 1);
      final bool hasBefore = stop - begin > size;
      return _payload(
        posts.sublist(hasBefore ? begin + 1 : begin, stop),
        hasBefore: hasBefore,
        hasAfter: true,
      );
    }
    final int found = afterPostNo == null
        ? 0
        : posts.indexWhere((post) => post.postNo > afterPostNo);
    final int from = found < 0 ? posts.length : found;
    final int end = math.min(posts.length, from + size + 1);
    final bool hasAfter = end - from > size;
    return _payload(
      posts.sublist(from, hasAfter ? from + size : end),
      hasBefore: afterPostNo != null,
      hasAfter: hasAfter,
    );
  }
}

class _CreatedReplyPostRepository extends PostRepository {
  _CreatedReplyPostRepository(
    super.client, {
    required this.onCreated,
    this.requireCaptcha = false,
    this.captchaAction = 'post.create',
    this.postNo = 16,
  });

  final void Function(String content) onCreated;
  final bool requireCaptcha;
  final String captchaAction;
  final int postNo;

  int get postId => 9000 + postNo;

  @override
  Future<CreatePostResult> createPost({
    required int topicId,
    required String content,
    int replyToPostId = 0,
    String? captchaId,
    String? captchaCode,
  }) async {
    if (requireCaptcha &&
        (captchaId != 'challenge' ||
            captchaCode == null ||
            captchaCode.isEmpty)) {
      throw ApiException(
        fallbackMessage: 'Captcha required',
        messageCode: 'common.captchaRequired',
        params: <String, dynamic>{'action': captchaAction},
      );
    }
    if (requireCaptcha && captchaCode != 'ABCD') {
      throw const ApiException(
        fallbackMessage: 'Captcha invalid',
        messageCode: 'auth.captcha.invalid',
      );
    }
    onCreated(content);
    return CreatePostResult(id: postId, postNo: postNo, renderedContent: '');
  }
}

class _CaptchaAuthRepository extends AuthRepository {
  _CaptchaAuthRepository(super.client);

  @override
  Future<CaptchaPayload> getCaptcha() async => CaptchaPayload(
    captchaId: 'challenge',
    captchaImg: base64Encode(img.encodePng(img.Image(width: 2, height: 2))),
  );
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  late TopicWindowServer server;
  late ProviderContainer container;

  Future<void> pumpTopic(
    WidgetTester tester, {
    required Map<String, dynamic> page,
    int floors = 15,
    int? postNo,
    bool pendingReply = false,
    bool requireReplyCaptcha = false,
    String replyCaptchaAction = 'post.create',
    bool trailingReplies = false,
    int createdPostNo = 16,
  }) async {
    final GfApiClient client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'http://fake.local',
    );
    server = TopicWindowServer(client)
      ..posts.addAll(<PostPayload>[
        for (int floor = 1; floor <= floors; floor++) floorPost(floor),
      ]);
    container = ProviderContainer(
      overrides: <Override>[
        apiClientProvider.overrideWithValue(client),
        pageRepositoryProvider.overrideWithValue(
          _NotificationPageRepository(client, page),
        ),
        topicRepositoryProvider.overrideWithValue(server),
        postRepositoryProvider.overrideWithValue(
          _CreatedReplyPostRepository(
            client,
            requireCaptcha: requireReplyCaptcha,
            captchaAction: replyCaptchaAction,
            postNo: createdPostNo,
            onCreated: (String content) {
              server.posts.add(floorPost(createdPostNo, content: content));
              // 待审回复对非版主不可见:服务端窗口会过滤掉它。
              if (pendingReply) {
                server.hiddenPostIds.add(9000 + createdPostNo);
              }
              // 提交成功到取回锚点窗口之间,其他人又发了更晚的楼层。
              if (trailingReplies) {
                for (
                  int floor = createdPostNo + 1;
                  floor <= createdPostNo + 9;
                  floor++
                ) {
                  server.posts.add(floorPost(floor));
                }
              }
            },
          ),
        ),
        currentUserProvider.overrideWith(
          (ref) async => const CurrentUser(id: 1, username: 'alice'),
        ),
        authRepositoryProvider.overrideWithValue(
          _CaptchaAuthRepository(client),
        ),
        writingStoreProvider.overrideWithValue(WritingStore()),
        offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('zh'),
          home: TopicPage(topicId: 100, initialPostNo: postNo),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  GfPostComposer composer(WidgetTester tester) =>
      tester.widget<GfPostComposer>(find.byType(GfPostComposer));

  AppLocalizations l10nOf(WidgetTester tester) =>
      AppLocalizations.of(tester.element(find.byType(TopicPage)));

  // 展开视口以便一次断言整个已加载窗口,避免 sliver 懒构建漏检。
  Future<void> expandViewport(WidgetTester tester) async {
    tester.view.physicalSize = const Size(1080, 3000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpAndSettle();
  }

  // markdown_widget 的 VisibilityDetector 会创建 500ms 延迟 Timer,
  // 需推进时钟让其过期,避免 "Timer is still pending"。
  Future<void> disposePage(WidgetTester tester) async {
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  }

  Future<void> replyToFloor(WidgetTester tester, int floor, String text) async {
    final List<PostActions> actions = tester
        .widgetList<PostActions>(find.byType(PostActions))
        .toList();
    actions
        .firstWhere((PostActions action) => action.post.postNo == floor)
        .onReply!();
    await tester.pumpAndSettle();
    composer(tester).controller.text = text;
    composer(tester).onPublish();
    await tester.pumpAndSettle();
  }

  testWidgets('通知锚定进入后前后楼层仍可通过分页入口续载', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: true,
    );
    await pumpTopic(tester, page: page, floors: 40, postNo: 8);
    final AppLocalizations l10n = AppLocalizations.of(
      tester.element(find.byType(TopicPage)),
    );

    // 锚点窗口内的多条评论都在,而不是只剩目标评论。
    expect(find.text('8楼内容'), findsOneWidget);
    expect(find.text('9楼内容'), findsOneWidget);
    expect(find.text(l10n.topicEarlierReplies), findsOneWidget);

    // 向后:页脚自动续载锚点之后的楼层。
    await tester.drag(find.byType(CustomScrollView), const Offset(0, -2000));
    await tester.pumpAndSettle();
    expect(
      server.calls.any((String call) => call.contains('after=15')),
      isTrue,
    );
    expect(find.text('20楼内容'), findsOneWidget);

    // 向前:更早楼层通过游标续载,且不重复已加载楼层。
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 8000));
    await tester.pumpAndSettle();
    await tester.tap(find.text(l10n.topicEarlierReplies));
    await tester.pumpAndSettle();
    expect(server.calls.last, 'anchor=0 before=8 after=0 limit=0');
    expect(find.text('7楼内容'), findsOneWidget);
    expect(find.text('8楼内容'), findsOneWidget);
    await disposePage(tester);
  });

  testWidgets('深链中间楼层切换倒序时直接读取最新尾窗', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: true,
    );
    await pumpTopic(tester, page: page, floors: 40, postNo: 8);

    await tester.tap(find.text('倒序'));
    await tester.pumpAndSettle();

    expect(server.calls.last, contains('before=9223372036854775807'));
    expect(find.text('40楼内容'), findsOneWidget);
    expect(find.text('15楼内容'), findsNothing);
    await disposePage(tester);
  });

  testWidgets('普通首屏切换倒序时直接读取最新尾窗', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 1; floor <= 20; floor++) floorPost(floor),
      ],
      hasBefore: false,
      hasAfter: true,
    );
    await pumpTopic(tester, page: page, floors: 40);

    await tester.tap(find.text('倒序'));
    await tester.pumpAndSettle();

    expect(server.calls.last, contains('before=9223372036854775807'));
    expect(find.text('40楼内容'), findsOneWidget);
    expect(find.text('20楼内容'), findsNothing);
    await disposePage(tester);
  });

  testWidgets('倒序下静默刷新重新读取最新尾窗', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 1; floor <= 20; floor++) floorPost(floor),
      ],
      hasBefore: false,
      hasAfter: true,
    );
    await pumpTopic(tester, page: page, floors: 40);

    await tester.tap(find.text('倒序'));
    await tester.pumpAndSettle();
    expect(find.text('40楼内容'), findsOneWidget);
    final int tailRequests = server.calls
        .where((String call) => call.contains('before=9223372036854775807'))
        .length;
    expect(tailRequests, 1);

    await tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pumpAndSettle();

    expect(find.text('40楼内容'), findsOneWidget);
    expect(find.text('20楼内容'), findsNothing);
    expect(
      server.calls
          .where((String call) => call.contains('before=9223372036854775807'))
          .length,
      tailRequests + 1,
    );
    await disposePage(tester);
  });

  testWidgets('倒序加载失败保留原窗口,切换后可重试', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: false,
    );
    await pumpTopic(tester, page: page, floors: 40, postNo: 8);
    server.failTailRequests = 1;

    await tester.tap(find.text('倒序'));
    await tester.pumpAndSettle();
    expect(find.text('8楼内容'), findsOneWidget);
    expect(find.text('40楼内容'), findsNothing);

    await tester.tap(find.text('加载更新回复'));
    await tester.pumpAndSettle();
    expect(find.text('40楼内容'), findsOneWidget);
    await disposePage(tester);
  });

  testWidgets('快速切换排序会丢弃过期的倒序窗口响应', (tester) async {
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: true,
    );
    await pumpTopic(tester, page: page, floors: 40, postNo: 8);
    server.delayNextTail = true;

    await tester.tap(find.text('倒序'));
    await tester.pump();
    expect(server.delayedTail, isNotNull);
    await tester.tap(find.text('正序'));
    await tester.pump();
    await tester.tap(find.text('倒序'));
    await tester.pumpAndSettle();
    expect(find.text('40楼内容'), findsOneWidget);

    server.delayedTail!.complete(
      PostWindowPayload(
        posts: <PostPayload>[floorPost(20)],
        replyTargets: const <ReplyTargetPayload>[],
        beforePostNo: 20,
        afterPostNo: 20,
        hasBefore: true,
        hasAfter: false,
        total: 40,
        maxPostNo: 40,
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('40楼内容'), findsOneWidget);
    expect(find.text('20楼内容'), findsNothing);
    await disposePage(tester);
  });

  testWidgets('回复通知锚定的楼层后保留已加载评论并露出新回复', (tester) async {
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: false,
    );
    await pumpTopic(tester, page: page, postNo: 8);

    expect(find.text('8楼内容'), findsOneWidget);
    await replyToFloor(tester, 8, '刚发出的回复');

    // 新回复自动滚入视野,且只渲染一次。
    expect(find.text('刚发出的回复').hitTestable(), findsOneWidget);

    // 已加载评论不因回复消失;两个窗口重叠的楼层不重复。
    await expandViewport(tester);
    expect(find.text('8楼内容'), findsOneWidget);
    expect(find.text('11楼内容'), findsOneWidget);
    await disposePage(tester);
  });

  testWidgets('回复合并后已加载楼层的引用目标不丢失', (tester) async {
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++)
          floorPost(
            floor,
            replyToPostId: floor == 9 ? 9008 : null,
            replyToUsername: floor == 9 ? 'user8' : null,
          ),
      ],
      replyTargets: <ReplyTargetPayload>[
        ReplyTargetPayload(
          id: 9008,
          postNo: 8,
          author: const UserBriefPayload(
            id: 8,
            username: 'user8',
            avatarUrl: '',
          ),
          renderedContent: '<p>被引用楼层预览</p>',
        ),
      ],
      hasBefore: true,
      hasAfter: false,
    );
    await pumpTopic(tester, page: page, postNo: 8);
    expect(find.text('8楼内容'), findsOneWidget);
    expect(find.text(l10nOf(tester).topicReplyTargetUnavailable), findsNothing);

    await replyToFloor(tester, 8, '刚发出的回复');
    await expandViewport(tester);
    expect(find.text('被引用楼层预览'), findsOneWidget);
    expect(find.text(l10nOf(tester).topicReplyTargetUnavailable), findsNothing);
    await disposePage(tester);
  });

  testWidgets('回复较远楼层时替换窗口并把新回复滚入视野', (tester) async {
    // 通知锚定窗口 [2..5];回复第 2 楼后新回复落在第 16 楼,与已加载窗口不相接。
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 2; floor <= 5; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: true,
    );
    await pumpTopic(tester, page: page, postNo: 2);
    expect(find.text('2楼内容'), findsOneWidget);
    await replyToFloor(tester, 2, '刚发出的回复');

    // 整窗替换为 [11..16]:窗口顶部不是新回复,必须把新回复本身滚入视野。
    expect(server.calls.last, 'anchor=9016 before=0 after=0 limit=20');
    expect(find.text('刚发出的回复').hitTestable(), findsOneWidget);
    await disposePage(tester);
  });

  testWidgets('回复待审时保留已加载窗口与游标', (tester) async {
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: false,
    );
    await pumpTopic(tester, page: page, postNo: 8, pendingReply: true);
    expect(find.text('8楼内容'), findsOneWidget);
    await replyToFloor(tester, 8, '待审核的回复');

    // 服务端不把待审楼层放进任何窗口:不能因此清空已加载评论与游标。
    expect(server.calls.last, 'anchor=9016 before=0 after=0 limit=20');
    expect(find.text('待审核的回复'), findsNothing);
    await expandViewport(tester);
    expect(find.text('8楼内容'), findsOneWidget);
    expect(find.text('15楼内容'), findsOneWidget);
    await tester.tap(find.text(l10nOf(tester).topicEarlierReplies));
    await tester.pumpAndSettle();
    expect(server.calls.last, 'anchor=0 before=8 after=0 limit=0');
    await disposePage(tester);
  });

  testWidgets('新回复之后仍有他人楼层时也把它滚入视野', (tester) async {
    // 已加载窗口 24..35;回复第 24 楼后新回复落在 41 楼,取回的锚点窗口
    // 是 36..50(其后还有他人 42..50 楼),新回复不在列表末尾。
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 24; floor <= 35; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: true,
    );
    await pumpTopic(
      tester,
      page: page,
      floors: 40,
      postNo: 24,
      createdPostNo: 41,
      trailingReplies: true,
    );
    expect(find.text('24楼内容'), findsOneWidget);
    await replyToFloor(tester, 24, '刚发出的回复');

    expect(server.calls.last, 'anchor=9041 before=0 after=0 limit=20');
    expect(find.text('刚发出的回复').hitTestable(), findsOneWidget);
    await disposePage(tester);
  });

  testWidgets('回复合并后前后游标仍指向已加载窗口边界', (tester) async {
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: false,
    );
    await pumpTopic(tester, page: page, postNo: 8);
    await replyToFloor(tester, 8, '刚发出的回复');
    final AppLocalizations l10n = AppLocalizations.of(
      tester.element(find.byType(TopicPage)),
    );

    // 展开视口以便完整断言已加载窗口;合并已到话题末尾,不再提供向后续载入口。
    await expandViewport(tester);
    expect(find.text(l10n.commonLoadMore), findsNothing);

    // 向前续载使用已加载窗口上界(第 8 楼),而不是新回复楼层;
    // 服务端即使返回与已加载窗口重叠的楼层(游标漂移),也不得重复渲染。
    server.overlapBeforeCursor = true;
    await tester.tap(find.text(l10n.topicEarlierReplies));
    await tester.pumpAndSettle();
    expect(server.calls.last, 'anchor=0 before=8 after=0 limit=0');
    expect(find.text('7楼内容'), findsOneWidget);
    expect(find.text('8楼内容'), findsOneWidget);
    await disposePage(tester);
  });

  testWidgets('新用户高频回复验证码显示原因说明', (tester) async {
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: false,
    );
    await pumpTopic(tester, page: page, postNo: 8, requireReplyCaptcha: true);

    await replyToFloor(tester, 8, '需要验证码的回复');

    expect(find.text(l10nOf(tester).publishCaptchaExplanation), findsOneWidget);
    expect(find.byKey(const Key('reply-captcha')), findsOneWidget);
    await disposePage(tester);
  });

  testWidgets('非回复 action 的验证码不显示回复说明', (tester) async {
    final Map<String, dynamic> page = anchoredPageJson(
      posts: <PostPayload>[
        for (int floor = 8; floor <= 15; floor++) floorPost(floor),
      ],
      hasBefore: true,
      hasAfter: false,
    );
    await pumpTopic(
      tester,
      page: page,
      postNo: 8,
      requireReplyCaptcha: true,
      replyCaptchaAction: 'login',
    );

    await replyToFloor(tester, 8, '其他验证码 action');

    expect(find.byKey(const Key('reply-captcha')), findsOneWidget);
    expect(find.text(l10nOf(tester).publishCaptchaExplanation), findsNothing);
    await disposePage(tester);
  });
}
