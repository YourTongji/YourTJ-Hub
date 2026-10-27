import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:forum_app/src/navigation/window_reserved_regions.dart';

import 'chat_avatar_profile_test.dart' show pumpConversation;
import 'chat_visible_read_test.dart' show pumpChat;
import 'fixtures/page_fixtures.dart' show messagesPayloadJson, parsePayload;
import 'pages_behavior_test.dart' show CountingPageRepository, makeChatMessage;

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  for (final size in [const Size(466, 678), const Size(951, 669)]) {
    testWidgets('latest chat message clears the side capsule at $size', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = size;
      tester.view.padding = const FakeViewPadding(right: 84, bottom: 34);
      tester.view.viewPadding = const FakeViewPadding(right: 84, bottom: 34);
      addTearDown(tester.view.reset);
      final reserved = WindowReservedRegions(
        size: size,
        safeArea: const EdgeInsets.only(right: 84, bottom: 34),
        occlusions: [Rect.fromLTWH(size.width - 84, 0, 84, 190)],
      );
      await pumpChat(
        tester,
        surfaceSize: size,
        messages: List.generate(
          40,
          (i) => makeChatMessage(i + 1).copyWith(isSelf: true),
        ),
        pageWrapper: (page) => ReadingWindow(
          reservedRegions: reserved,
          maxContentWidth: 1120,
          rail: const Align(
            alignment: Alignment.bottomCenter,
            child: SizedBox(width: 56, height: 224, key: Key('capsule')),
          ),
          bottomNavigation: const SizedBox(height: 56),
          child: page,
        ),
      );
      final timeline = find.byType(ListView).last;
      final scroll = tester.widget<ListView>(timeline).controller!;
      // Reach the real end of the lazy message list, including its trailing space.
      await tester.drag(timeline, const Offset(0, -5000));
      await tester.pumpAndSettle();
      final bubble = find.text('消息 40');
      expect(bubble, findsOneWidget);
      expect(
        tester.getRect(bubble).bottom,
        lessThan(tester.getRect(find.byKey(const Key('capsule'))).top),
      );
      final input = find.byType(GfChatInput);
      expect(tester.getRect(input).right, lessThanOrEqualTo(size.width - 84));
      final field = find.byType(TextField).last;
      final controller = tester.widget<TextField>(field).controller!;
      await tester.enterText(field, 'draft stays in the safe input');
      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      tester.view.padding = const FakeViewPadding(right: 84);
      await tester.pumpAndSettle();
      expect(
        ReadingWindowScope.edgeNavigationOf(
          tester.element(find.byType(MessagesPage)),
        ),
        isFalse,
      );
      expect(
        tester.widget<ListView>(timeline).padding,
        const EdgeInsets.fromLTRB(12, 12, 12, 18),
      );
      expect(tester.widget<ListView>(timeline).controller, same(scroll));
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(controller),
      );
      expect(controller.text, 'draft stays in the safe input');
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets(
    'wide messages keep the directory and retain chat across resizing',
    (tester) async {
      final app = await pumpChat(
        tester,
        targetUserId: null,
        messages: List.generate(40, (index) => makeChatMessage(index + 1)),
      );
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();

      expect(app.navigator.currentState!.canPop(), isFalse);
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        tester
            .widgetList<GfConversationRow>(find.byType(GfConversationRow))
            .singleWhere((row) => row.name == 'bob')
            .active,
        isTrue,
      );
      final field = find.byType(TextField).last;
      await tester.enterText(field, 'unfinished iPad message');
      final input = tester.widget<TextField>(field).controller!;
      final scroll = tester
          .widget<ListView>(find.byType(ListView).last)
          .controller!;
      await tester.drag(find.byType(ListView).last, const Offset(0, 250));
      await tester.pumpAndSettle();
      final calls = app.repo.afterCalls;
      final reads = app.repo.batches.length;

      await tester.binding.setSurfaceSize(const Size(500, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      expect(input.text, 'unfinished iPad message');
      expect(
        tester.widget<ListView>(find.byType(ListView).last).controller,
        same(scroll),
      );

      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      expect(app.repo.afterCalls, calls);
      expect(app.repo.batches.length, reads);
      expect(scroll.offset, greaterThan(0));
      await tester.binding.setSurfaceSize(const Size(500, 800));
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(find.textContaining('unfinished iPad message'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'compact conversation still pushes and wide rotation shows directory',
    (tester) async {
      final app = await pumpChat(tester, targetUserId: null, messages: []);
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();
      expect(app.navigator.currentState!.canPop(), isTrue);
      expect(find.byType(GfConversationRow), findsNothing);
      final input = tester
          .widget<TextField>(find.byType(TextField).last)
          .controller!;
      await tester.enterText(find.byType(TextField).last, 'compact draft');
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      expect(input.text, 'compact draft');
    },
  );

  testWidgets(
    'switching peers and target routes keeps drafts with their owner',
    (tester) async {
      final router = await pumpConversation(tester, messages: []);
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, 'Bob only');
      await tester.tap(find.text('carol'));
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        isEmpty,
      );
      await tester.enterText(find.byType(TextField).last, 'Carol only');
      router.go('/messages?userId=2&username=bob');
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        'Bob only',
      );
      router.go('/messages?userId=3&username=carol');
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        'Carol only',
      );
    },
  );

  testWidgets('seeded contact data cannot return after an account switch', (
    tester,
  ) async {
    final app = await pumpChat(tester, targetUserId: null, messages: []);
    final contact = ValueNotifier(
      const ChatItemPayload(
        id: 9,
        peerId: 4,
        peerUsername: 'dave',
        peerNickname: 'Private contact A',
        peerAvatar: '',
        lastMsg: 'Private summary A',
        lastMsgTime: '',
        unreadCount: 0,
        convId: 9,
        peerUrl: '/u/4',
      ),
    );
    addTearDown(contact.dispose);
    app.navigator.currentState!.push(
      MaterialPageRoute<void>(
        builder: (_) => ValueListenableBuilder<ChatItemPayload>(
          valueListenable: contact,
          builder: (_, value, _) => MessagesPage(
            targetUserId: value.peerId,
            targetUsername: value.peerUsername,
            initialConversation: value,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Private contact A'), findsOneWidget);
    app.container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pumpAndSettle();
    final historyCalls = app.repo.afterCalls;
    contact.value = contact.value.copyWith(peerNickname: 'Stale contact A');
    await tester.pumpAndSettle();
    await tester.pump(const Duration(seconds: 16));
    await tester.pumpAndSettle();
    expect(find.textContaining('contact A'), findsNothing);
    expect(find.byType(GfChatInput), findsNothing);
    expect(app.repo.afterCalls, historyCalls);
    expect(tester.takeException(), isNull);
  });

  testWidgets('covered and inactive split views stop chat polling', (
    tester,
  ) async {
    final app = await pumpChat(tester, targetUserId: 2, messages: []);
    await tester.binding.setSurfaceSize(const Size(1100, 800));
    await tester.pumpAndSettle();
    app.navigator.currentState!.push(
      MaterialPageRoute<void>(
        builder: (_) => const Scaffold(body: Text('Covered')),
      ),
    );
    await tester.pumpAndSettle();
    final coveredCalls = app.repo.afterCalls;
    await tester.pump(const Duration(seconds: 31));
    expect(app.repo.afterCalls, coveredCalls);
    app.navigator.currentState!.pop();
    await tester.pumpAndSettle();
    app.active.value = false;
    await tester.pumpAndSettle();
    final inactiveCalls = app.repo.afterCalls;
    await tester.pump(const Duration(seconds: 31));
    expect(app.repo.afterCalls, inactiveCalls);
  });

  testWidgets(
    'wide new conversations stay in the detail pane and retain drafts',
    (tester) async {
      final app = await pumpChat(tester, targetUserId: null, messages: []);
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('New message'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Dave'));
      await tester.pumpAndSettle();
      expect(app.navigator.currentState!.canPop(), isFalse);
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      expect(
        app.repo.afterCalls,
        0,
        reason: 'no server conversation exists before the first send',
      );
      await tester.enterText(find.byType(TextField).last, 'Draft for Dave');
      await tester.pumpAndSettle();
      await tester.tap(find.text('bob'));
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        isEmpty,
      );
      final draftRow = find.widgetWithText(GfConversationRow, 'dave');
      await tester.tap(draftRow);
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller!.text,
        'Draft for Dave',
      );
    },
  );

  testWidgets(
    'hidden compact directory stops polling and large text stays single pane',
    (tester) async {
      final app = await pumpChat(tester, targetUserId: 2, messages: []);
      final pages =
          app.container.read(pageRepositoryProvider) as CountingPageRepository;
      final compactCalls = pages.fetchCalls;
      await tester.pump(const Duration(seconds: 16));
      await tester.pumpAndSettle();
      expect(pages.fetchCalls, compactCalls);
      await tester.binding.setSurfaceSize(const Size(1100, 800));
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNWidgets(2));
      final input = tester
          .widget<TextField>(find.byType(TextField).last)
          .controller;
      tester.platformDispatcher.textScaleFactorTestValue = 2;
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      await tester.pumpAndSettle();
      expect(find.byType(GfConversationRow), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField).last).controller,
        same(input),
      );
      final hiddenCalls = pages.fetchCalls;
      await tester.pump(const Duration(seconds: 16));
      await tester.pumpAndSettle();
      expect(pages.fetchCalls, hiddenCalls);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'a conversation the server never creates stops discovery polling',
    (tester) async {
      final app = await pumpChat(
        tester,
        targetUserId: 9,
        targetUsername: 'zed',
        messages: [],
      );
      final pages =
          app.container.read(pageRepositoryProvider) as CountingPageRepository;
      final initial = pages.fetchCalls;
      await tester.pump(const Duration(seconds: 16));
      await tester.pumpAndSettle();
      expect(
        pages.fetchCalls,
        initial + 1,
        reason: 'a silent peer is discovered through the directory',
      );
      await tester.pump(const Duration(minutes: 30));
      await tester.pumpAndSettle();
      final capped = pages.fetchCalls;
      expect(capped - initial, lessThanOrEqualTo(12));
      await tester.pump(const Duration(minutes: 10));
      await tester.pumpAndSettle();
      expect(
        pages.fetchCalls,
        capped,
        reason: 'backoff ends the probe instead of polling forever',
      );
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('a discovered conversation ends directory polling', (
    tester,
  ) async {
    final app = await pumpChat(
      tester,
      targetUserId: 9,
      targetUsername: 'zed',
      messages: [],
      pages: (client) => _RevealingPageRepository(client, revealFrom: 2),
    );
    final pages =
        app.container.read(pageRepositoryProvider) as CountingPageRepository;
    final initial = pages.fetchCalls;
    await tester.pump(const Duration(seconds: 16));
    await tester.pumpAndSettle();
    expect(
      pages.fetchCalls,
      initial + 1,
      reason: 'the probe loads the directory that now holds the chat',
    );
    final resolved = pages.fetchCalls;
    await tester.pump(const Duration(minutes: 5));
    await tester.pumpAndSettle();
    expect(
      pages.fetchCalls,
      resolved,
      reason: 'the created conversation needs no further discovery',
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'visible reads use window coordinates inside an offset reading column',
    (tester) async {
      final app = await pumpChat(
        tester,
        targetUserId: null,
        messages: [makeChatMessage(1)],
      );
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(1600, 800);
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(tester.view.resetPhysicalSize);
      await tester.binding.setSurfaceSize(const Size(1600, 800));
      app.navigator.currentState!.push(
        MaterialPageRoute<void>(
          builder: (context) => Align(
            alignment: Alignment.centerRight,
            child: SizedBox(
              width: 500,
              child: MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(size: const Size(500, 800)),
                child: const MessagesPage(targetUserId: 2),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpAndSettle();
      expect(app.repo.batches, [
        [1],
      ]);
    },
  );
}

/// `/messages` grows the deep-linked peer once the server has created it.
class _RevealingPageRepository extends CountingPageRepository {
  _RevealingPageRepository(super.client, {required this.revealFrom});

  final int revealFrom;

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (path != '/messages') {
      return super.fetch(path, cancelToken: cancelToken);
    }
    fetchCalls++;
    final Map<String, dynamic> json = messagesPayloadJson();
    final Map<String, dynamic> props = Map<String, dynamic>.from(
      json['props']! as Map<dynamic, dynamic>,
    );
    json['props'] = props;
    if (fetchCalls >= revealFrom) {
      props['conversations'] = <dynamic>[
        ...props['conversations']! as List<dynamic>,
        <String, dynamic>{
          'id': 3,
          'peerId': 9,
          'peerUsername': 'zed',
          'peerAvatar': '',
          'lastMsg': '',
          'lastMsgTime': '',
          'unreadCount': 0,
          'convId': 3,
          'peerUrl': '/u/9',
        },
      ];
    }
    return parsePayload(json);
  }
}
