import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/messages/chat_drafts.dart';
import 'package:forum_app/src/messages/chat_outbox.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'chat_visible_read_test.dart' show VisibleChatRepository, pumpChat;
import 'pages_behavior_test.dart' show MemTokenStorage, makeChatMessage;

/// Records every send attempt and keeps it pending until the test resolves it,
/// so failure and retry are driven explicitly.
class GatedChatRepository extends VisibleChatRepository {
  GatedChatRepository(super.client, {required super.messages});

  final contents = <String>[];
  final clientMessageIds = <String?>[];
  final pending = <Completer<int>>[];

  @override
  Future<int> sendMessage({
    required int peerId,
    required String content,
    int msgType = 0,
    String? clientMessageId,
    int? replyToMessageId,
  }) {
    contents.add(content);
    clientMessageIds.add(clientMessageId);
    final gate = Completer<int>();
    pending.add(gate);
    return gate.future;
  }
}

TextEditingController composerOf(WidgetTester tester) =>
    tester.widget<GfChatInput>(find.byType(GfChatInput)).controller!;

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  Future<({GatedChatRepository repo, ProviderContainer container})>
  pumpSendChat(WidgetTester tester) async {
    final repo = GatedChatRepository(
      GfApiClient(dio: Dio(), tokenStorage: MemTokenStorage()),
      messages: [makeChatMessage(1)],
    );
    final chat = await pumpChat(tester, repository: (_) => repo);
    return (repo: repo, container: chat.container);
  }

  Future<void> send(WidgetTester tester, String content, {int? caret}) async {
    await tester.enterText(find.byType(TextField), content);
    if (caret != null) {
      composerOf(tester).selection = TextSelection.collapsed(offset: caret);
    }
    await tester.pump();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pump();
  }

  Future<void> failLastSend(
    WidgetTester tester,
    GatedChatRepository repo,
  ) async {
    repo.pending.last.completeError(StateError('offline'));
    await tester.pumpAndSettle();
  }

  testWidgets(
    'server acknowledgement preserves outgoing bubble width and alignment',
    (tester) async {
      final h = await pumpSendChat(tester);
      final content = List.filled(
        6,
        'A longer message keeps its wrapping.',
      ).join(' ');
      await send(tester, content);
      await tester.pumpAndSettle();
      Finder bubble() => find.descendant(
        of: find.byWidgetPredicate(
          (w) => w is GfMessageBubble && w.text == content,
        ),
        matching: find.byWidgetPredicate(
          (w) =>
              w is Container &&
              w.padding ==
                  const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        ),
      );
      final pending = tester.getRect(bubble());
      h.repo.messages.add(
        makeChatMessage(2).copyWith(content: content, isSelf: true),
      );
      h.repo.pending.single.complete(1);
      await tester.pumpAndSettle();
      expect(h.container.read(chatOutboxProvider(2)).items, isEmpty);
      final acknowledged = tester.getRect(bubble());
      expect(acknowledged.right, closeTo(pending.right, .01));
      expect(acknowledged.size, pending.size);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  const drafts = <String, String>{
    'plain text': '先写到一半',
    'sticker': '[:sticker:smile:]',
    'text, stickers and newlines':
        '第一行\n[:sticker:smile:]\n第二行\n[:sticker:cat:]',
  };

  for (final MapEntry(key: label, value: content) in drafts.entries) {
    testWidgets('failed send rehydrates a $label draft', (tester) async {
      final h = await pumpSendChat(tester);
      final controller = composerOf(tester);
      final caret = content.length ~/ 2;
      await send(tester, content, caret: caret);
      expect(h.repo.contents, [content]);

      // The composer loses its value while the request is in flight: the
      // failure path must reinstall the full submission explicitly instead of
      // relying on the controller never being touched.
      controller.clear();
      await tester.pump();
      await failLastSend(tester, h.repo);

      expect(controller.text, content);
      expect(
        controller.value.selection,
        TextSelection.collapsed(offset: caret),
        reason: 'the submitted caret is restored best-effort',
      );
      expect(
        h.container.read(chatDraftsProvider).forPeer(2)!.value.text,
        content,
        reason: 'the restored draft must also outlive a page rebuild',
      );
      await tester.pumpWidget(const SizedBox.shrink());
    });
  }

  testWidgets('a failure never replaces edits made while sending', (
    tester,
  ) async {
    final h = await pumpSendChat(tester);
    final controller = composerOf(tester);
    await send(tester, 'submitted');
    expect(h.repo.contents, ['submitted']);

    await tester.enterText(find.byType(TextField), 'submitted plus newer');
    await tester.pump();
    await failLastSend(tester, h.repo);

    expect(controller.text, 'submitted plus newer');
    expect(
      h.container.read(chatOutboxProvider(2)).items.single.content,
      'submitted',
    );
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('retry reuses the pending message and its clientMessageId', (
    tester,
  ) async {
    final h = await pumpSendChat(tester);
    final controller = composerOf(tester);
    await send(tester, 'hello');
    expect(h.repo.clientMessageIds.single, isNotNull);
    await failLastSend(tester, h.repo);
    expect(h.container.read(chatOutboxProvider(2)).items, hasLength(1));

    await tester.tap(find.text('Retry sending'));
    await tester.pump();
    expect(h.repo.contents, ['hello', 'hello']);
    expect(h.repo.clientMessageIds, hasLength(2));
    expect(h.repo.clientMessageIds.last, h.repo.clientMessageIds.first);
    expect(
      h.container.read(chatOutboxProvider(2)).items,
      hasLength(1),
      reason: 'a retry must not enqueue a duplicate message',
    );

    h.repo.pending.last.complete(5);
    await tester.pumpAndSettle();
    expect(controller.text, isEmpty, reason: 'acknowledged drafts clear');
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('a rehydrated draft stays bound to its pending message', (
    tester,
  ) async {
    final h = await pumpSendChat(tester);
    final controller = composerOf(tester);
    await send(tester, 'hello');
    final submittedId = h.repo.clientMessageIds.single;
    controller.clear();
    await tester.pump();
    await failLastSend(tester, h.repo);
    expect(controller.text, 'hello', reason: 'the failed draft is rehydrated');

    await tester.tap(find.text('Retry sending'));
    await tester.pump();
    expect(h.repo.clientMessageIds, hasLength(2));
    expect(h.repo.clientMessageIds.last, submittedId);
    h.repo.pending.last.complete(5);
    await tester.pumpAndSettle();

    expect(
      controller.text,
      isEmpty,
      reason: 'the acknowledged retry clears the rehydrated draft',
    );
    expect(
      h.container.read(chatOutboxProvider(2)).items,
      hasLength(1),
      reason: 'the retry must not enqueue a duplicate',
    );
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('an older failed bubble never refills an acknowledged composer', (
    tester,
  ) async {
    final h = await pumpSendChat(tester);
    final controller = composerOf(tester);
    await send(tester, 'hello');
    await failLastSend(tester, h.repo);
    expect(controller.text, 'hello');

    // A newer send is acknowledged and empties the composer.
    await send(tester, 'world');
    h.repo.pending.last.complete(6);
    await tester.pumpAndSettle();
    expect(controller.text, isEmpty);

    // Retrying the older bubble must not inject its text back.
    await tester.tap(find.text('Retry sending'));
    await tester.pump();
    await failLastSend(tester, h.repo);
    expect(controller.text, isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('whitespace typed while sending is never replaced', (
    tester,
  ) async {
    final h = await pumpSendChat(tester);
    final controller = composerOf(tester);
    await send(tester, 'hello');
    await tester.enterText(find.byType(TextField), ' ');
    await tester.pump();
    await failLastSend(tester, h.repo);

    expect(controller.text, ' ');
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('a double retry never rehydrates the in-flight attempt', (
    tester,
  ) async {
    final h = await pumpSendChat(tester);
    final controller = composerOf(tester);
    await send(tester, 'hello');
    await failLastSend(tester, h.repo);
    controller.clear();
    await tester.pump();

    // Both callbacks run in one frame, before the pending rebuild disables the
    // button; only the first may claim the failed message.
    final retry = tester.widget<TextButton>(
      find.widgetWithText(TextButton, 'Retry sending'),
    );
    retry.onPressed!();
    retry.onPressed!();
    await tester.pump();
    expect(h.repo.contents, ['hello', 'hello']);
    h.repo.pending.last.complete(5);
    await tester.pumpAndSettle();
    expect(controller.text, isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
