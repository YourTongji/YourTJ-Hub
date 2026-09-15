import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers.dart';

enum DeliveryState { sending, sent, failed }

class PendingMessage {
  PendingMessage(this.id, this.content, this.afterId);
  final int id;
  final String content;
  final int afterId;
  DeliveryState state = DeliveryState.failed;
  Object? error;
}

/// Session-local outbox survives leaving a conversation. Session invalidation
/// disposes it; failures never remove text or automatically retry a write.
final chatOutboxProvider = ChangeNotifierProvider.family<ChatOutbox, int>((
  ref,
  peerId,
) {
  ref.watch(offlineCacheEpochProvider);
  return ChatOutbox(ref.watch(chatRepositoryProvider), peerId);
});

class ChatOutbox extends ChangeNotifier {
  ChatOutbox(this.repository, this.peerId);
  final ChatRepository repository;
  final int peerId;
  final List<PendingMessage> items = [];
  final Set<int> _matchedIds = {};
  int _serial = 0;
  int conversationId = 0;
  int _latestObservedId = 0;
  bool _disposed = false;

  PendingMessage enqueue(String content, int afterId) {
    final floor = afterId > _latestObservedId ? afterId : _latestObservedId;
    if (items.isEmpty) _matchedIds.clear();
    final message = PendingMessage(++_serial, content, floor);
    items.add(message);
    notifyListeners();
    return message;
  }

  Future<int?> send(PendingMessage message) async {
    if (_disposed ||
        message.state != DeliveryState.failed ||
        !items.contains(message)) {
      return null;
    }
    message.state = DeliveryState.sending;
    message.error = null;
    notifyListeners();
    try {
      final id = await repository.sendMessage(
        peerId: peerId,
        content: message.content,
      );
      if (_disposed) return null;
      conversationId = id;
      message.state = DeliveryState.sent;
      notifyListeners();
      return id;
    } catch (error) {
      message.error = error;
      if (!_disposed) {
        message.state = DeliveryState.failed;
        notifyListeners();
      }
      return null;
    }
  }

  /// The API acknowledges the conversation, not the message ID. Keep an
  /// acknowledged bubble until an unseen matching server message arrives.
  void reconcile(List<ChatMessagePayload> messages) {
    for (final message in messages) {
      if (message.id > _latestObservedId) _latestObservedId = message.id;
    }
    var changed = false;
    items.removeWhere((pending) {
      if (pending.state != DeliveryState.sent) return false;
      for (final message in messages) {
        if (message.isSelf &&
            message.id > pending.afterId &&
            message.content == pending.content &&
            _matchedIds.add(message.id)) {
          changed = true;
          return true;
        }
      }
      return false;
    });
    if (changed) notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
