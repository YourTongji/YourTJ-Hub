import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart' show SemanticsAction, SemanticsNode;
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/messages/chat_message_row.dart';
import 'package:forum_app/src/navigation/route_visibility.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/providers.dart';

import 'pages_behavior_test.dart'
    show
        CountingPageRepository,
        MemTokenStorage,
        NoopCache,
        PollingChatRepository,
        makeChatMessage;

Finder _appBarAvatar() => find.byKey(const Key('chat-peer-avatar-appbar'));

Finder _rowAvatar(int messageId) =>
    find.byKey(Key('chat-peer-avatar-$messageId'));

/// 打开私信页并注册真实的 /u/:userId 路由。[targetUserId] 非空时进入目标会话,
/// 为空时停在会话列表(进入会话走应用真实的 Navigator.push 路径)。
Future<GoRouter> pumpConversation(
  WidgetTester tester, {
  required List<ChatMessagePayload> messages,
  int? targetUserId = 2,
}) async {
  await tester.binding.setSurfaceSize(const Size(390, 700));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final storage = MemTokenStorage()..write('token');
  final client = GfApiClient(
    dio: Dio(),
    tokenStorage: storage,
    baseUrl: 'http://fake.local',
  );
  final container = ProviderContainer(
    overrides: [
      tokenStorageProvider.overrideWithValue(storage),
      currentUserProvider.overrideWith(
        (ref) async => const CurrentUser(id: 1, username: 'alice'),
      ),
      pageRepositoryProvider.overrideWithValue(CountingPageRepository(client)),
      chatRepositoryProvider.overrideWithValue(
        PollingChatRepository(client, messages: messages),
      ),
      offlineTopicCacheProvider.overrideWithValue(NoopCache()),
      offlineChatCacheProvider.overrideWithValue(NoopCache()),
    ],
  );
  addTearDown(container.dispose);
  final router = GoRouter(
    initialLocation: targetUserId == null
        ? '/messages'
        : '/messages?userId=$targetUserId&username=Bob',
    observers: <NavigatorObserver>[VisibilityRouteObserver()],
    routes: <RouteBase>[
      GoRoute(
        path: '/messages',
        builder: (_, GoRouterState state) => MessagesPage(
          targetUserId: int.tryParse(state.uri.queryParameters['userId'] ?? ''),
          targetUsername: state.uri.queryParameters['username'] ?? '',
        ),
      ),
      GoRoute(
        path: '/u/:userId',
        builder: (_, GoRouterState state) =>
            ProfilePage(userId: int.parse(state.pathParameters['userId']!)),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        routerConfig: router,
        theme: gfThemeData(Brightness.light),
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  testWidgets('顶栏对方头像以 44×44 命中区打开对方主页', (tester) async {
    final router = await pumpConversation(
      tester,
      messages: [makeChatMessage(1)],
    );
    expect(_appBarAvatar(), findsOneWidget);
    expect(tester.getSize(_appBarAvatar()), const Size(44, 44));
    final Rect artwork = tester.getRect(
      find.descendant(of: _appBarAvatar(), matching: find.byType(GfAvatar)),
    );
    expect(artwork.size, const Size(36, 36));
    // 标题与旧版保持 10 的视觉间距(44 命中区右侧留白 + 2)。
    final Rect title = tester.getRect(find.text('bob'));
    expect(title.left - artwork.right, 10);

    await tester.tap(_appBarAvatar());
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/u/2');
    expect(find.byType(ProfilePage), findsOneWidget);

    router.pop();
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/messages');
    expect(find.text('消息 1'), findsOneWidget);
  });

  testWidgets('消息行对方头像以 44×44 命中区打开对方主页', (tester) async {
    final router = await pumpConversation(
      tester,
      messages: [makeChatMessage(1)],
    );
    final rowAvatar = _rowAvatar(1);
    expect(rowAvatar, findsOneWidget);
    expect(tester.getSize(rowAvatar), const Size(44, 44));
    final Rect artwork = tester.getRect(
      find.descendant(of: rowAvatar, matching: find.byType(GfAvatar)),
    );
    expect(artwork.size, const Size(32, 32));
    final GfMessageBubble bubble = tester.widget<GfMessageBubble>(
      find.byType(GfMessageBubble),
    );
    final Rect bubbleRect = tester.getRect(find.byKey(bubble.bubbleKey!));
    // 头像贴命中区左上角:与气泡顶部对齐,44 命中区直接接上气泡左边缘。
    expect(artwork.top, bubbleRect.top);
    expect(artwork.left + 44, bubbleRect.left);

    await tester.tap(rowAvatar);
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/u/2');
    expect(find.byType(ProfilePage), findsOneWidget);
  });

  testWidgets('从会话列表进入的会话同样能从头像打开对方主页', (tester) async {
    final router = await pumpConversation(
      tester,
      messages: [makeChatMessage(1)],
      targetUserId: null,
    );
    // 会话列表 → 会话走应用真实入口:rootNavigator.push(MaterialPageRoute),
    // 此处必须验证主页真的可见,而不只是路由地址变化。
    await tester.tap(find.text('bob'));
    await tester.pumpAndSettle();
    expect(find.text('消息 1').hitTestable(), findsOneWidget);

    await tester.tap(_appBarAvatar());
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/u/2');
    expect(find.byType(ProfilePage).hitTestable(), findsOneWidget);
    expect(find.text('消息 1', skipOffstage: false).hitTestable(), findsNothing);

    router.pop();
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/messages');
    expect(find.text('消息 1').hitTestable(), findsOneWidget);
  });

  testWidgets('自己的头像保持展示态,不跳转', (tester) async {
    final router = await pumpConversation(
      tester,
      messages: [makeChatMessage(1), makeChatMessage(2).copyWith(isSelf: true)],
    );
    final selfAvatar = find.descendant(
      of: find.byWidgetPredicate(
        (Widget widget) => widget is ChatMessageRow && widget.mine,
      ),
      matching: find.byType(GfAvatar),
    );
    expect(selfAvatar, findsOneWidget);
    expect(_rowAvatar(2), findsNothing);
    expect(
      find.descendant(of: selfAvatar, matching: find.byType(InkWell)),
      findsNothing,
    );

    await tester.tap(selfAvatar);
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/messages');
    expect(find.text('消息 2'), findsOneWidget);
  });

  testWidgets('头像入口是带标签的按钮,不劫持消息文本朗读', (tester) async {
    final SemanticsHandle handle = tester.ensureSemantics();
    await pumpConversation(tester, messages: [makeChatMessage(1)]);

    final SemanticsNode avatar = tester.getSemantics(_appBarAvatar());
    expect(avatar.label, '查看 bob 的主页');
    expect(avatar.getSemanticsData().hasAction(SemanticsAction.tap), isTrue);

    final SemanticsNode text = tester.getSemantics(find.text('消息 1'));
    expect(text.label, contains('消息 1'));
    expect(text.getSemanticsData().hasAction(SemanticsAction.tap), isFalse);
    handle.dispose();
  });
}
