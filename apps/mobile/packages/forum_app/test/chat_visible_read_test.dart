import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/messages/visible_chat_reads.dart';
import 'package:forum_app/src/navigation/route_visibility.dart';
import 'package:forum_app/src/realtime/realtime_updates.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_behavior_test.dart'
    show
        CountingPageRepository,
        MemTokenStorage,
        NoopCache,
        PollingChatRepository,
        makeChatMessage;

class VisibleChatRepository extends PollingChatRepository {
  VisibleChatRepository(
    super.client, {
    required super.messages,
    super.hasMoreBefore,
  });
  final batches = <List<int>>[];
  Completer<ChatVisibleReadResult>? gate;
  bool failReads = false;
  Object? readError;
  List<ChatMessagePayload> olderMessages = [];
  @override
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int limit = 30,
  }) {
    if (beforeId > 0 && olderMessages.isNotEmpty) {
      beforeCalls++;
      return Future.value(
        ChatMessagesResponse(
          list: olderMessages,
          hasMoreBefore: false,
          hasMoreAfter: false,
          nextBeforeId: 0,
          latestId: olderMessages.last.id,
        ),
      );
    }
    return super
        .getMessages(
          convId: convId,
          beforeId: beforeId,
          afterId: afterId,
          limit: limit,
        )
        .then(
          (response) => afterId > 0
              ? response.copyWith(hasMoreBefore: false, nextBeforeId: 0)
              : response,
        );
  }

  @override
  Future<ChatVisibleReadResult> markVisible({
    required int convId,
    required List<int> messageIds,
  }) async {
    batches.add(List.of(messageIds));
    if (gate != null) return gate!.future;
    if (readError != null) throw readError!;
    if (failReads) throw StateError('offline');
    return ChatVisibleReadResult(
      convId: convId,
      acknowledgedMessageIds: messageIds,
      unreadCount: 99,
    );
  }
}

Future<
  ({
    ProviderContainer container,
    VisibleChatRepository repo,
    GlobalKey<NavigatorState> navigator,
    ValueNotifier<bool> active,
  })
>
pumpChat(
  WidgetTester tester, {
  List<ChatMessagePayload>? messages,
  bool nested = false,
  List<ChatMessagePayload>? older,
}) async {
  await tester.binding.setSurfaceSize(const Size(390, 700));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  final storage = MemTokenStorage()..write('token');
  final client = GfApiClient(
    dio: Dio(),
    tokenStorage: storage,
    baseUrl: 'http://fake.local',
  );
  final repo = VisibleChatRepository(
    client,
    hasMoreBefore: older != null,
    messages:
        messages ?? List.generate(40, (index) => makeChatMessage(index + 1)),
  );
  repo.olderMessages = older ?? [];
  final container = ProviderContainer(
    overrides: [
      tokenStorageProvider.overrideWithValue(storage),
      pageRepositoryProvider.overrideWithValue(CountingPageRepository(client)),
      chatRepositoryProvider.overrideWithValue(repo),
      offlineTopicCacheProvider.overrideWithValue(NoopCache()),
      offlineChatCacheProvider.overrideWithValue(NoopCache()),
    ],
  );
  addTearDown(container.dispose);
  final navigator = GlobalKey<NavigatorState>();
  final active = ValueNotifier(true);
  addTearDown(active.dispose);
  Widget page() => ValueListenableBuilder<bool>(
    valueListenable: active,
    builder: (_, value, _) =>
        TickerMode(enabled: value, child: const MessagesPage(targetUserId: 2)),
  );
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        navigatorKey: navigator,
        navigatorObservers: [VisibilityRouteObserver()],
        theme: gfThemeData(Brightness.light),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: nested
            ? Navigator(
                observers: [VisibilityRouteObserver()],
                onGenerateRoute: (_) =>
                    MaterialPageRoute<void>(builder: (_) => page()),
              )
            : page(),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return (
    container: container,
    repo: repo,
    navigator: navigator,
    active: active,
  );
}

void main() {
  Future<void> dwell(WidgetTester tester) async {
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
  }

  ScrollController scroll(WidgetTester tester) =>
      tester.widget<ListView>(find.byType(ListView).last).controller!;

  testWidgets('other conversation hints cannot erase a covered chat update', (
    tester,
  ) async {
    final h = await pumpChat(tester, messages: [makeChatMessage(1)]);
    h.container.read(realtimeHealthyProvider.notifier).setHealthy(true);
    await tester.pump();
    final before = h.repo.afterCalls;
    h.navigator.currentState!.push(
      DialogRoute<void>(
        context: h.navigator.currentContext!,
        builder: (_) => const AlertDialog(title: Text('Covered')),
      ),
    );
    await tester.pumpAndSettle();
    h.repo.messages.add(makeChatMessage(2));
    h.container.read(realtimeInvalidationsProvider.notifier).chat(1);
    h.container.read(realtimeInvalidationsProvider.notifier).chat(2);
    await tester.pump();
    expect(h.repo.afterCalls, before);

    h.navigator.currentState!.pop();
    await tester.pumpAndSettle();
    expect(h.repo.afterCalls, greaterThan(before));
    expect(find.text('消息 2'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('shell drawer suspends visible-read dwell until closed', (
    tester,
  ) async {
    final h = await pumpChat(tester, messages: [makeChatMessage(1)]);
    addTearDown(() => shellDrawerOpen.value = false);
    shellDrawerOpen.value = true;
    await tester.pump();
    await dwell(tester);
    expect(h.repo.batches, isEmpty);

    shellDrawerOpen.value = false;
    await tester.pump();
    await dwell(tester);
    expect(h.repo.batches.length, 1);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets(
    'prepending older history preserves the anchor without marking unseen rows',
    (tester) async {
      final h = await pumpChat(
        tester,
        messages: List.generate(40, (i) => makeChatMessage(i + 41)),
        older: List.generate(40, (i) => makeChatMessage(i + 1)),
      );
      await dwell(tester);
      await tester.pump(const Duration(seconds: 15));
      await tester.pumpAndSettle();
      scroll(tester).jumpTo(0);
      await tester.pumpAndSettle();
      await dwell(tester);
      expect(tester.takeException(), isNull);
      expect(h.repo.beforeCalls, greaterThan(0));
      expect(h.repo.batches.expand((ids) => ids), isNot(contains(1)));
      expect(
        find.text('消息 41'),
        findsOneWidget,
        reason:
            'built ${tester.widgetList<GfMessageBubble>(find.byType(GfMessageBubble)).map((b) => b.text).toList()} at ${scroll(tester).offset}/${scroll(tester).position.maxScrollExtent}',
      );
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'batches are bounded to 100 exact IDs and acknowledged IDs are not resent',
    (tester) async {
      final batches = <List<int>>[];
      final controller = VisibleChatReads(
        isEnabled: () => true,
        visibleIds: () => List.generate(205, (i) => i + 1).toSet(),
        acknowledge: (ids) async {
          batches.add(ids);
          return ids.toSet();
        },
        onAcknowledged: (_) {},
        onFailure: (_) {},
      );
      await tester.pumpWidget(const SizedBox.shrink());
      controller.changed();
      await tester.pump();
      await dwell(tester);
      expect(batches.map((ids) => ids.length), [100, 100, 5]);
      expect(batches.expand((ids) => ids).toSet().length, 205);
      controller.changed();
      await tester.pump();
      await dwell(tester);
      expect(batches.length, 3);
      controller.dispose();
    },
  );

  testWidgets('keyboard-clipped messages cannot qualify behind the keyboard', (
    tester,
  ) async {
    final h = await pumpChat(
      tester,
      messages: List.generate(8, (i) => makeChatMessage(i + 1)),
    );
    tester.view.viewInsets = FakeViewPadding(
      bottom: 350 * tester.view.devicePixelRatio,
    );
    addTearDown(tester.view.resetViewInsets);
    await tester.pumpAndSettle();
    await dwell(tester);
    final seen = h.repo.batches.expand((ids) => ids).toSet();
    expect(seen, isNotEmpty);
    expect(seen.length, lessThan(8));
    final viewport = tester.getRect(find.byType(ListView).last);
    for (final bubble in tester.widgetList<GfMessageBubble>(
      find.byType(GfMessageBubble),
    )) {
      final id = int.parse(bubble.text.split(' ').last);
      if (!seen.contains(id)) continue;
      final render =
          bubble.bubbleKey!.currentContext!.findRenderObject()! as RenderBox;
      expect(
        bubbleIsVisible(
          render.localToGlobal(Offset.zero) & render.size,
          viewport,
        ),
        isTrue,
      );
    }
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('session changes before dwell produce no read request', (
    tester,
  ) async {
    final h = await pumpChat(tester);
    h.container.read(offlineCacheEpochProvider.notifier).invalidate();
    await dwell(tester);
    expect(h.repo.batches, isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets(
    'returning before an old read finishes still flushes a fresh dwell',
    (tester) async {
      final h = await pumpChat(tester, messages: [makeChatMessage(1)]);
      final gate = h.repo.gate = Completer<ChatVisibleReadResult>();
      await dwell(tester);
      h.navigator.currentState!.push(
        DialogRoute<void>(
          context: h.navigator.currentContext!,
          builder: (_) => const AlertDialog(title: Text('Covered')),
        ),
      );
      await tester.pumpAndSettle();
      h.navigator.currentState!.pop();
      await tester.pumpAndSettle();
      await dwell(tester);
      expect(h.repo.batches.length, 1);
      h.repo.gate = null;
      gate.complete(
        const ChatVisibleReadResult(
          convId: 1,
          acknowledgedMessageIds: [1],
          unreadCount: 0,
        ),
      );
      await tester.pumpAndSettle();
      expect(h.repo.batches.length, 2);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets('unsupported servers preserve unread with no mark-all fallback', (
    tester,
  ) async {
    final h = await pumpChat(tester, messages: [makeChatMessage(1)]);
    h.repo.readError = const ApiException(
      fallbackMessage: 'Not found',
      statusCode: 404,
    );
    await dwell(tester);
    await tester.pump(const Duration(seconds: 4));
    expect(h.repo.batches.length, 1);
    expect(h.repo.markReadCalls, 0);
    expect(find.textContaining('server does not support'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  test(
    'long bubbles use viewport height; clipping and horizontal occlusion count',
    () {
      const viewport = Rect.fromLTWH(0, 0, 300, 500);
      expect(
        bubbleIsVisible(const Rect.fromLTWH(0, -500, 200, 1500), viewport),
        isTrue,
      );
      expect(
        bubbleIsVisible(const Rect.fromLTWH(0, 480, 200, 100), viewport),
        isFalse,
      );
      expect(
        bubbleIsVisible(const Rect.fromLTWH(290, 0, 100, 100), viewport),
        isFalse,
      );
    },
  );
  testWidgets('opening a conversation never marks unseen history read', (
    tester,
  ) async {
    final h = await pumpChat(tester);
    expect(h.repo.markReadCalls, 0);
    expect(h.repo.batches, isEmpty, reason: 'layout alone is not a dwell');
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    final seen = h.repo.batches.expand((batch) => batch).toSet();
    expect(seen, isNotEmpty);
    expect(seen, isNot(contains(1)), reason: 'unseen history stays unread');
    expect(seen.length, lessThan(40));
    final built = tester
        .widgetList<GfMessageBubble>(find.byType(GfMessageBubble))
        .map((bubble) => int.parse(bubble.text.split(' ').last))
        .toSet();
    expect(
      built.difference(seen),
      isNotEmpty,
      reason: 'prefetched offscreen bubbles must not be acknowledged',
    );
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('a bubble taller than the viewport can become read', (
    tester,
  ) async {
    final h = await pumpChat(
      tester,
      messages: [
        makeChatMessage(
          1,
        ).copyWith(content: List.filled(100, 'A long message').join('\n')),
      ],
    );
    await dwell(tester);
    expect(h.repo.batches, [
      [1],
    ]);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets(
    'only incoming unread server IDs qualify, not outgoing or read messages',
    (tester) async {
      final h = await pumpChat(
        tester,
        messages: [
          makeChatMessage(1),
          makeChatMessage(2).copyWith(isSelf: true),
          makeChatMessage(3).copyWith(isRead: 1),
          makeChatMessage(0),
        ],
      );
      await dwell(tester);
      expect(h.repo.batches, [
        [1],
      ]);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  for (final nested in [false, true]) {
    testWidgets(
      'root overlay cancels dwell over nested=$nested and resumes after pop',
      (tester) async {
        final h = await pumpChat(tester, nested: nested);
        h.navigator.currentState!.push(
          DialogRoute<void>(
            context: h.navigator.currentContext!,
            builder: (_) => const AlertDialog(title: Text('Covered')),
          ),
        );
        await tester.pumpAndSettle();
        await dwell(tester);
        expect(h.repo.batches, isEmpty);
        h.repo.messages.add(makeChatMessage(41));
        await tester.pump(const Duration(seconds: 15));
        await tester.pumpAndSettle();
        expect(h.repo.batches, isEmpty);
        h.navigator.currentState!.pop();
        await tester.pumpAndSettle();
        await dwell(tester);
        expect(h.repo.batches, isNotEmpty);
        expect(find.byKey(const Key('chat-new-messages')), findsOneWidget);
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );
  }

  testWidgets('inactive app and inactive tab cancel dwell', (tester) async {
    final h = await pumpChat(tester);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    addTearDown(
      () => tester.binding.handleAppLifecycleStateChanged(
        AppLifecycleState.resumed,
      ),
    );
    await dwell(tester);
    expect(h.repo.batches, isEmpty);
    h.active.value = false;
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();
    await dwell(tester);
    expect(h.repo.batches, isEmpty);
    h.active.value = true;
    await tester.pumpAndSettle();
    await dwell(tester);
    expect(h.repo.batches, isNotEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets(
    'session change cancels old dwell and ignores an old acknowledgement',
    (tester) async {
      final h = await pumpChat(tester);
      final gate = h.repo.gate = Completer<ChatVisibleReadResult>();
      await dwell(tester);
      expect(h.repo.batches.length, 1);
      h.container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      gate.complete(
        ChatVisibleReadResult(
          convId: 1,
          acknowledgedMessageIds: h.repo.batches.single,
          unreadCount: 0,
        ),
      );
      await dwell(tester);
      expect(h.repo.batches.length, 1);
      expect(find.byType(GfMessageBubble), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'reading history preserves new unread messages and jump only reads visible IDs',
    (tester) async {
      final h = await pumpChat(tester);
      await dwell(tester);
      scroll(tester).jumpTo(0);
      await tester.pumpAndSettle();
      await dwell(tester);
      final before = scroll(tester).offset;
      h.repo.messages.addAll(
        List.generate(15, (index) => makeChatMessage(41 + index)),
      );
      await tester.pump(const Duration(seconds: 15));
      await tester.pumpAndSettle();
      expect(scroll(tester).offset, before);
      expect(h.repo.batches.expand((batch) => batch), isNot(contains(55)));
      expect(find.byKey(const Key('chat-new-messages')), findsOneWidget);
      await tester.tap(find.byKey(const Key('chat-new-messages')));
      await tester.pumpAndSettle();
      await dwell(tester);
      final seen = h.repo.batches.expand((batch) => batch).toSet();
      expect(seen, contains(55));
      expect(seen, isNot(contains(25)));
      expect(h.repo.markReadCalls, 0);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets('rapid scrolling restarts dwell and batches are single flight', (
    tester,
  ) async {
    final h = await pumpChat(tester);
    scroll(tester).jumpTo(0);
    await tester.pump(const Duration(milliseconds: 100));
    scroll(tester).jumpTo(scroll(tester).position.maxScrollExtent);
    await tester.pump(const Duration(milliseconds: 100));
    expect(h.repo.batches, isEmpty);
    final gate = h.repo.gate = Completer<ChatVisibleReadResult>();
    await dwell(tester);
    scroll(tester).jumpTo(0);
    await tester.pumpAndSettle();
    await dwell(tester);
    expect(h.repo.batches.length, 1);
    h.repo.gate = null;
    gate.complete(
      ChatVisibleReadResult(
        convId: 1,
        acknowledgedMessageIds: h.repo.batches.first,
        unreadCount: 99,
      ),
    );
    await tester.pumpAndSettle();
    expect(h.repo.batches.length, 2);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets(
    'read failures have one bounded retry and never fall back to mark-all',
    (tester) async {
      final h = await pumpChat(tester, messages: [makeChatMessage(1)]);
      h.repo.failReads = true;
      await dwell(tester);
      await tester.pump(const Duration(seconds: 1));
      await tester.pumpAndSettle();
      await dwell(tester);
      expect(h.repo.batches.length, 2);
      await tester.pump(const Duration(seconds: 3));
      expect(h.repo.batches.length, 2);
      expect(h.repo.markReadCalls, 0);
      expect(find.textContaining('Read status could not sync'), findsOneWidget);
      h.repo.failReads = false;
      await tester.tap(find.text('Retry'));
      await tester.pumpAndSettle();
      await dwell(tester);
      expect(h.repo.batches.length, 3);
      expect(find.textContaining('Read status could not sync'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
