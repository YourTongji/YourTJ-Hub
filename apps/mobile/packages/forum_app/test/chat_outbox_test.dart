import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/messages/chat_outbox.dart';
import 'package:forum_app/src/providers.dart';

class _Storage implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _Chat extends ChatRepository {
  _Chat() : super(GfApiClient(dio: Dio(), tokenStorage: _Storage()));
  final attempts = <Completer<int>>[];
  @override
  Future<int> sendMessage({
    required int peerId,
    required String content,
    int msgType = 0,
  }) {
    final result = Completer<int>();
    attempts.add(result);
    return result.future;
  }
}

ChatMessagePayload message(int id) => ChatMessagePayload(
  id: id,
  senderId: 1,
  content: 'same',
  msgType: 0,
  isRead: 0,
  createdAt: '',
  isSelf: true,
);
void main() {
  test(
    'failed text survives, retry is single-flight and success stays until server echo',
    () async {
      final repo = _Chat();
      final outbox = ChatOutbox(repo, 2);
      final pending = outbox.enqueue('same', 10);
      final first = outbox.send(pending);
      expect(pending.state, DeliveryState.sending);
      expect(await outbox.send(pending), isNull);
      expect(repo.attempts.length, 1);
      repo.attempts.single.completeError(Exception('offline'));
      expect(await first, isNull);
      expect(outbox.items.single.content, 'same');
      expect(pending.state, DeliveryState.failed);
      final retry = outbox.send(pending);
      repo.attempts.last.complete(9);
      await retry;
      expect(outbox.conversationId, 9);
      expect(pending.state, DeliveryState.sent);
      outbox.reconcile([message(10)]);
      expect(outbox.items, [pending]);
      outbox.reconcile([message(11)]);
      expect(outbox.items, isEmpty);
      outbox.dispose();
    },
  );
  test(
    'one server message cannot acknowledge two identical pending bubbles',
    () async {
      final repo = _Chat();
      final outbox = ChatOutbox(repo, 2);
      final a = outbox.enqueue('same', 10), b = outbox.enqueue('same', 10);
      final sends = [outbox.send(a), outbox.send(b)];
      for (final attempt in repo.attempts) {
        attempt.complete(9);
      }
      await Future.wait(sends);
      outbox.reconcile([message(11)]);
      expect(outbox.items, [b]);
      outbox.reconcile([message(11), message(12)]);
      expect(outbox.items, isEmpty);
      outbox.dispose();
    },
  );
  test(
    'session switch drops late acknowledgements and starts an empty outbox',
    () async {
      final repo = _Chat();
      final container = ProviderContainer(
        overrides: [chatRepositoryProvider.overrideWithValue(repo)],
      );
      final old = container.read(chatOutboxProvider(2));
      final sending = old.send(old.enqueue('same', 0));
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      final next = container.read(chatOutboxProvider(2));
      expect(next.items, isEmpty);
      repo.attempts.single.complete(9);
      expect(await sending, isNull);
      expect(next.conversationId, 0);
      container.dispose();
    },
  );
}
