import 'dart:math';
import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers.dart';

enum ForwardDeliveryState { pending, sending, sent, failed }

class ForwardRecipient {
  ForwardRecipient(this.user);
  final UserConnectionPayload user;
  ForwardDeliveryState state = ForwardDeliveryState.pending;
  Object? error;
}

class ForwardBatch {
  ForwardBatch({
    required List<int> messageIds,
    required this.merged,
    required List<UserConnectionPayload> recipients,
  }) : messageIds = List.unmodifiable(messageIds),
       recipients = List.unmodifiable(recipients.map(ForwardRecipient.new));
  final List<int> messageIds;
  final bool merged;
  final List<ForwardRecipient> recipients;
  final String id = List.generate(
    16,
    (_) => Random.secure().nextInt(256).toRadixString(16).padLeft(2, '0'),
  ).join();
  bool get complete => recipients.every(
    (recipient) => recipient.state == ForwardDeliveryState.sent,
  );
}

final chatForwardingProvider =
    ChangeNotifierProvider.family<ChatForwarding, int>((ref, convId) {
      ref.watch(offlineCacheEpochProvider);
      return ChatForwarding(ref.watch(chatRepositoryProvider), convId);
    });

/// Session-local retry ownership survives leaving the recipient page. Every
/// recipient keeps the same immutable operation identity until acknowledged.
class ChatForwarding extends ChangeNotifier {
  ChatForwarding(this.repository, this.convId);
  final ChatRepository repository;
  final int convId;
  ForwardBatch? batch;
  bool busy = false;
  bool _disposed = false;
  bool get unfinished => batch != null && !batch!.complete;
  ForwardBatch create({
    required List<int> messageIds,
    required bool merged,
    required List<UserConnectionPayload> recipients,
  }) {
    if (_disposed ||
        busy ||
        unfinished ||
        messageIds.isEmpty ||
        messageIds.length > 50 ||
        (!merged && messageIds.length > 10) ||
        messageIds.toSet().length != messageIds.length ||
        recipients.isEmpty ||
        recipients.length > 10 ||
        recipients.map((u) => u.id).toSet().length != recipients.length) {
      throw StateError('invalid forward batch');
    }
    batch = ForwardBatch(
      messageIds: messageIds,
      merged: merged,
      recipients: recipients,
    );
    notifyListeners();
    return batch!;
  }

  Future<void> send() async {
    final current = batch;
    if (_disposed || busy || current == null || current.complete) return;
    busy = true;
    notifyListeners();
    try {
      for (final recipient in current.recipients) {
        if (_disposed || !identical(batch, current)) break;
        if (recipient.state == ForwardDeliveryState.sent) continue;
        recipient.state = ForwardDeliveryState.sending;
        recipient.error = null;
        notifyListeners();
        try {
          final result = await repository.forwardMessages(
            convId: convId,
            peerId: recipient.user.id,
            messageIds: current.messageIds,
            merged: current.merged,
            clientForwardId: current.id,
          );
          if (_disposed) return;
          if (result.convId <= 0 || result.messageIds.isEmpty) {
            throw const FormatException('invalid forward acknowledgement');
          }
          recipient.state = ForwardDeliveryState.sent;
        } catch (error) {
          if (_disposed) return;
          recipient.state = ForwardDeliveryState.failed;
          recipient.error = error;
        }
        notifyListeners();
      }
    } finally {
      busy = false;
      if (!_disposed) notifyListeners();
    }
  }

  void discard() {
    if (_disposed || busy) return;
    batch = null;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
