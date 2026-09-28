import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/messages/chat_forwarding.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class ForwardRepository extends ChatRepository {
  ForwardRepository()
    : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage()));
  final attempts = <(int, String, List<int>, bool)>[];
  final replies = <Completer<ChatForwardResult>>[];
  @override
  Future<ChatForwardResult> forwardMessages({
    required int convId,
    required int peerId,
    required List<int> messageIds,
    required bool merged,
    required String clientForwardId,
  }) {
    attempts.add((peerId, clientForwardId, List.of(messageIds), merged));
    final reply = Completer<ChatForwardResult>();
    replies.add(reply);
    return reply.future;
  }
}

UserConnectionPayload recipient(int id) => UserConnectionPayload(
  id: id,
  username: 'user$id',
  nickname: '',
  avatarUrl: '',
  bio: '',
  url: '/u/$id',
);
const ack = ChatForwardResult(convId: 4, messageIds: [8]);
void main() {
  test(
    'partial delivery retries only failures with the immutable batch identity',
    () async {
      final repo = ForwardRepository();
      final controller = ChatForwarding(repo, 1);
      final source = [9, 3];
      final users = [recipient(2), recipient(3)];
      final batch = controller.create(
        messageIds: source,
        merged: true,
        recipients: users,
      );
      source.clear();
      users.clear();
      expect(repo.attempts, isEmpty);
      final send = controller.send();
      await controller.send();
      expect(repo.attempts.length, 1);
      repo.replies[0].complete(ack);
      await Future<void>.delayed(Duration.zero);
      repo.replies[1].completeError(StateError('lost acknowledgement'));
      await send;
      expect(controller.unfinished, isTrue);
      expect(batch.recipients[0].state, ForwardDeliveryState.sent);
      expect(
        () => controller.create(
          messageIds: [1],
          merged: false,
          recipients: [recipient(4)],
        ),
        throwsStateError,
      );
      final retry = controller.send();
      expect(repo.attempts.map((a) => a.$1), [2, 3, 3]);
      expect(repo.attempts.every((a) => a.$2 == batch.id && a.$4), isTrue);
      expect(repo.attempts.last.$3, [9, 3]);
      repo.replies[2].complete(ack);
      await retry;
      expect(batch.complete, isTrue);
      await controller.send();
      expect(repo.attempts.length, 3);
      controller.dispose();
    },
  );
  test(
    'session disposal prevents subsequent recipients and late notifications',
    () async {
      final repo = ForwardRepository();
      final controller = ChatForwarding(repo, 1);
      controller.create(
        messageIds: [1],
        merged: false,
        recipients: [recipient(2), recipient(3)],
      );
      final sending = controller.send();
      controller.dispose();
      repo.replies.single.complete(ack);
      await sending;
      expect(repo.attempts.length, 1);
    },
  );
  test(
    'message and recipient bounds are validated before a batch can send',
    () {
      final repo = ForwardRepository();
      final controller = ChatForwarding(repo, 1);
      for (final ids in [
        <int>[],
        List.generate(11, (i) => i + 1),
        [1, 1],
        List.generate(51, (i) => i + 1),
      ]) {
        expect(
          () => controller.create(
            messageIds: ids,
            merged: false,
            recipients: [recipient(2)],
          ),
          throwsStateError,
        );
      }
      expect(
        () => controller.create(
          messageIds: [1],
          merged: true,
          recipients: List.generate(11, (i) => recipient(i + 2)),
        ),
        throwsStateError,
      );
      expect(repo.attempts, isEmpty);
      controller.dispose();
    },
  );
}
