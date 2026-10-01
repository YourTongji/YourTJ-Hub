import 'dart:async';
import 'dart:math';
import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers.dart';
import '../local/writing_store.dart';
import 'chat_drafts.dart';

final _messageRandom = Random.secure();

String newChatMessageId() => List.generate(
  16,
  (_) => _messageRandom.nextInt(256).toRadixString(16).padLeft(2, '0'),
).join();

enum DeliveryState { sending, sent, failed }

class PendingMessage {
  PendingMessage(
    this.id,
    this.content,
    this.afterId, {
    this.draftRevision,
    this.draftValue,
    this.replyToMessageId,
    this.msgType = 1,
    String? clientMessageId,
  }) : clientMessageId = clientMessageId ?? newChatMessageId();
  final int id;
  final String clientMessageId;
  final String content;
  final int afterId;
  final int? draftRevision;
  final int? replyToMessageId;
  final int msgType;

  /// Composer snapshot captured on submit, so a failed send can rehydrate text,
  /// sticker tokens and selection instead of relying on untouched state.
  final TextEditingValue? draftValue;
  DeliveryState state = DeliveryState.failed;
  Object? error;
}

/// Session-local presentation survives leaving a conversation. Uploaded image
/// intents additionally use account-scoped secure recovery; restoration never
/// automatically retries a write.
final chatOutboxProvider = ChangeNotifierProvider.family<ChatOutbox, int>((
  ref,
  peerId,
) {
  ref.watch(offlineCacheEpochProvider);
  return ChatOutbox(
    ref.watch(chatRepositoryProvider),
    peerId,
    imageStore: ref.watch(chatDraftStoreProvider),
    // Epoch changes replace the outbox. A profile refresh within the same
    // account must not discard in-flight text messages by recreating it.
    imageScope: ref.read(writingScopeProvider.future),
  );
});

class ChatOutbox extends ChangeNotifier {
  ChatOutbox(this.repository, this.peerId, {this.imageStore, this.imageScope})
    : _imageGeneration = imageStore?.imageGeneration {
    if (imageStore != null) unawaited(restoreImages());
  }
  final ChatRepository repository;
  final int peerId;
  final ChatDraftStore? imageStore;
  final Future<String>? imageScope;
  final int? _imageGeneration;
  Object? imageRecoveryError;
  Future<void>? _restoringImages;
  final List<PendingMessage> items = [];
  final Set<int> _matchedIds = {};
  int _serial = 0;
  int conversationId = 0;
  int _latestObservedId = 0;
  bool _disposed = false;

  bool get sending => items.any((item) => item.state == DeliveryState.sending);

  /// Restore as explicit retries, never as automatic network writes. The same
  /// clientMessageId also reconciles a server success whose response was lost.
  Future<void> restoreImages() => _restoringImages ??= _restoreImages();

  Future<void> _restoreImages() async {
    try {
      final store = imageStore;
      if (store == null) return;
      final scope = await imageScope!;
      final records = await store.readImages(scope, peerId);
      if (_disposed) return;
      for (final record in records) {
        final key = record['clientMessageId'] as String;
        final content = record['content'] as String;
        final afterId = record['afterId'] as int;
        if (key.isEmpty || content.isEmpty || afterId < 0) {
          throw const FormatException('Invalid image recovery record');
        }
        if (items.any((item) => item.clientMessageId == key)) continue;
        items.add(
          PendingMessage(
            ++_serial,
            content,
            afterId,
            msgType: 2,
            clientMessageId: key,
          ),
        );
      }
      imageRecoveryError = null;
    } catch (error) {
      imageRecoveryError = error;
      _restoringImages = null;
    }
    if (!_disposed) notifyListeners();
  }

  Future<void> prepareImageUpload() async {
    await restoreImages();
    if (imageRecoveryError != null) throw imageRecoveryError!;
    if (_disposed || sending) throw StateError('Chat cannot send now');
    final scope = await imageScope;
    if (scope?.endsWith(':0') ?? false) {
      throw StateError('Image requires an account');
    }
  }

  /// Commit the upload URL before handing it to the page or attempting chat/send.
  /// A late upload may still save to its captured account after session disposal;
  /// the explicit erase generation prevents resurrection after user-data removal.
  Future<PendingMessage> enqueueImage(
    String content,
    int afterId, {
    required String clientMessageId,
  }) async {
    final store = imageStore;
    if (store != null) {
      await store.writeImage(await imageScope!, peerId, clientMessageId, {
        'content': content,
        'afterId': afterId,
        'clientMessageId': clientMessageId,
      }, generation: _imageGeneration!);
    }
    final existing = items
        .where((item) => item.clientMessageId == clientMessageId)
        .firstOrNull;
    if (existing != null) return existing;
    final message = PendingMessage(
      ++_serial,
      content,
      afterId,
      msgType: 2,
      clientMessageId: clientMessageId,
    );
    if (!_disposed) {
      items.add(message);
      notifyListeners();
    }
    return message;
  }

  PendingMessage enqueue(
    String content,
    int afterId, {
    int? draftRevision,
    TextEditingValue? draftValue,
    int? replyToMessageId,
    int msgType = 1,
  }) {
    final floor = afterId > _latestObservedId ? afterId : _latestObservedId;
    if (items.isEmpty) _matchedIds.clear();
    final message = PendingMessage(
      ++_serial,
      content,
      floor,
      draftRevision: draftRevision,
      draftValue: draftValue,
      replyToMessageId: replyToMessageId,
      msgType: msgType,
    );
    items.add(message);
    notifyListeners();
    return message;
  }

  Future<int?> send(PendingMessage message) async {
    if (_disposed ||
        sending ||
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
        msgType: message.msgType,
        clientMessageId: message.clientMessageId,
        replyToMessageId: message.replyToMessageId,
      );
      if (message.msgType == 2 && imageStore != null) {
        // Cleanup failure retains an idempotent retry, never deletes the upload.
        await imageStore!.removeImage(
          await imageScope!,
          peerId,
          message.clientMessageId,
        );
      }
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
            (message.msgType == 0 ? 1 : message.msgType) == pending.msgType &&
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
