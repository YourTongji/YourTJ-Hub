import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../local/writing_store.dart';
import '../providers.dart';

/// Peer identity is stable before the first server conversation is created.
class ChatDraft {
  const ChatDraft({
    required this.peerId,
    required this.peerName,
    required this.peerAvatar,
    required this.convId,
    required this.value,
    required this.updatedAt,
    required this.revision,
  });
  final int peerId, convId, updatedAt, revision;
  final String peerName, peerAvatar;
  final TextEditingValue value;
  bool get hasText => value.text.trim().isNotEmpty;

  ChatItemPayload get conversation => ChatItemPayload(
    id: 0,
    peerId: peerId,
    peerUsername: peerName,
    peerAvatar: peerAvatar,
    convId: convId,
    lastMsg: '',
    lastMsgTime: '',
    unreadCount: 0,
    peerUrl: '/u/$peerId',
  );

  Map<String, Object> toJson() => {
    'peerId': peerId,
    'peerName': peerName,
    'peerAvatar': peerAvatar,
    'convId': convId,
    'text': value.text,
    'base': value.selection.baseOffset,
    'extent': value.selection.extentOffset,
    'updatedAt': updatedAt,
  };
  factory ChatDraft.fromJson(Map<String, dynamic> json, int revision) {
    final text = json['text'] as String;
    return ChatDraft(
      peerId: json['peerId'] as int,
      peerName: json['peerName'] as String,
      peerAvatar: json['peerAvatar'] as String,
      convId: json['convId'] as int,
      updatedAt: json['updatedAt'] as int,
      revision: revision,
      value: TextEditingValue(
        text: text,
        selection: TextSelection(
          baseOffset: (json['base'] as int).clamp(0, text.length),
          extentOffset: (json['extent'] as int).clamp(0, text.length),
        ),
      ),
    );
  }
}

final chatDraftStoreProvider = Provider((ref) => ChatDraftStore());
final chatDraftsProvider = ChangeNotifierProvider<ChatDrafts>((ref) {
  final epoch = ref.watch(offlineCacheEpochProvider);
  // A same-site login changes identity even when the API origin is unchanged.
  final scope = ref.watch(writingScopeProvider.future);
  final session = ref.read(offlineCacheEpochProvider.notifier);
  return ChatDrafts(
    store: ref.watch(chatDraftStoreProvider),
    resolveScope: () => scope,
    isCurrent: () => session.isCurrent(epoch),
  );
});

/// Private message drafts use device-bound Keychain items on iOS. Android uses
/// the existing secure-storage file (excluded from backup), with namespaced keys.
/// Do not change Android file/cipher options independently of token storage: the
/// installed plugin shares one native instance with sticky file options.
class ChatDraftStore {
  ChatDraftStore({
    this.secureStorage = const FlutterSecureStorage(
      iOptions: IOSOptions(
        accountName: 'yourtj_chat_drafts',
        accessibility: KeychainAccessibility.first_unlock_this_device,
        synchronizable: false,
      ),
    ),
  });
  final FlutterSecureStorage secureStorage;
  Future<void> _tail = Future.value();
  String _prefix(String scope) => 'yourtj:writing:v1:$scope:chat:';
  bool _isChatKey(String key) =>
      key.startsWith('yourtj:writing:v1:') && key.contains(':chat:');

  Future<T> _serial<T>(Future<T> Function() action) {
    final result = _tail.then((_) => action());
    _tail = result.then<void>((_) {}, onError: (Object _) {});
    return result;
  }

  Future<void> _put(String key, String value) async {
    await secureStorage.write(key: key, value: value);
    if (await secureStorage.read(key: key) != value) {
      throw StateError('Chat draft storage failed');
    }
  }

  Future<void> _removeLegacy(SharedPreferences prefs, String key) async {
    if (prefs.containsKey(key) && !await prefs.remove(key)) {
      throw StateError('Chat draft migration cleanup failed');
    }
  }

  Future<List<ChatDraft>> read(String scope) => _serial(() async {
    if (scope.endsWith(':0')) throw StateError('Draft requires an account');
    final prefs = await SharedPreferences.getInstance();
    await prefs.reload();
    final secured = Map<String, String>.of(await secureStorage.readAll());
    // Migrate every account's legacy chat records without exposing foreign data.
    // A verified secure value always wins over an older plaintext copy. A
    // tombstone also wins, preventing an acknowledged draft from reappearing.
    for (final key in prefs.getKeys().where(_isChatKey)) {
      if (!secured.containsKey(key)) {
        final value = prefs.getString(key);
        if (value == null) continue;
        await _put(key, value);
        secured[key] = value;
      }
      await _removeLegacy(prefs, key);
    }
    final result = <ChatDraft>[];
    for (final entry in secured.entries.where(
      (entry) => entry.key.startsWith(_prefix(scope)),
    )) {
      if (entry.value.isEmpty) {
        await secureStorage.delete(key: entry.key);
        continue;
      }
      try {
        final draft = ChatDraft.fromJson(
          jsonDecode(entry.value) as Map<String, dynamic>,
          0,
        );
        if (draft.peerId > 0 && draft.hasText) result.add(draft);
      } on FormatException {
        /* Keep malformed records for possible recovery. */
      } on TypeError {
        /* A damaged record must not hide other drafts. */
      }
    }
    return result;
  });

  Future<void> write(
    String scope,
    ChatDraft draft, {
    required bool Function() isCurrent,
  }) => _serial(() async {
    if (scope.endsWith(':0')) throw StateError('Draft requires an account');
    final prefs = await SharedPreferences.getInstance();
    await prefs.reload();
    if (!isCurrent()) throw StateError('Draft session changed');
    final key = '${_prefix(scope)}${draft.peerId}';
    // Write deletion intent before touching legacy storage. If cleanup fails,
    // the empty secure value prevents migration from resurrecting old text.
    await _put(key, draft.hasText ? jsonEncode(draft.toJson()) : '');
    await _removeLegacy(prefs, key);
    if (!draft.hasText) await secureStorage.delete(key: key);
  });

  Future<void> clearAccount(String scope) => _serial(() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.reload();
    final keys = {
      ...(await secureStorage.readAll()).keys,
      ...prefs.getKeys(),
    }.where((key) => key.startsWith(_prefix(scope)));
    for (final key in keys) {
      await _put(key, '');
      await _removeLegacy(prefs, key);
      await secureStorage.delete(key: key);
    }
  });
}

/// Session-owned memory survives route disposal; disk writes debounce and flush
/// on exit/inactivity. Failed writes keep the dirty value available for retry.
class ChatDrafts extends ChangeNotifier {
  ChatDrafts({
    required this.store,
    required this.resolveScope,
    required this.isCurrent,
  }) {
    unawaited(_load().then((_) => flush()));
  }
  final ChatDraftStore store;
  final Future<String> Function() resolveScope;
  final bool Function() isCurrent;
  final Map<int, ChatDraft> _items = {};
  final Map<int, int> _dirty = {};
  int _serial = 0;
  String? _scope;
  Future<void>? _loading;
  Future<bool> _flushing = Future.value(true);
  bool _loaded = false, _disposed = false;
  Timer? _debounce;
  Object? error;
  bool get current => !_disposed && isCurrent();
  bool get loading => !_loaded && error == null;
  bool get dirty => _dirty.isNotEmpty;
  bool isDirty(int peerId) => _dirty.containsKey(peerId);
  Iterable<ChatDraft> get items =>
      _items.values.where((draft) => draft.hasText);
  ChatDraft? forPeer(int peerId) => _items[peerId];

  Future<void> _load() {
    if (_loaded || !current) return Future.value();
    return _loading ??= _loadOnce().whenComplete(() => _loading = null);
  }

  Future<void> _loadOnce() async {
    try {
      final scope = await resolveScope();
      final saved = await store.read(scope);
      if (!current) return;
      _scope = scope;
      for (final draft in saved) {
        if (!_items.containsKey(draft.peerId)) {
          _items[draft.peerId] = ChatDraft.fromJson(draft.toJson(), ++_serial);
        }
      }
      _loaded = true;
      error = null;
    } catch (failure) {
      if (!current) return;
      error = failure;
    }
    if (current) notifyListeners();
  }

  void update(ChatItemPayload peer, TextEditingValue input) {
    if (!current || peer.peerId <= 0) return;
    final previous = _items[peer.peerId];
    final selection = input.selection.isValid
        ? TextSelection(
            baseOffset: input.selection.baseOffset.clamp(0, input.text.length),
            extentOffset: input.selection.extentOffset.clamp(
              0,
              input.text.length,
            ),
          )
        : previous?.value.text == input.text
        ? previous!.value.selection
        : TextSelection.collapsed(offset: input.text.length);
    final value = TextEditingValue(text: input.text, selection: selection);
    final convId = peer.convId > 0 ? peer.convId : previous?.convId ?? 0;
    if (previous?.value == value &&
        previous?.convId == convId &&
        previous?.peerName == peer.peerUsername &&
        previous?.peerAvatar == peer.peerAvatar) {
      return;
    }
    final version = ++_serial;
    final sameText = previous?.value.text == value.text;
    _items[peer.peerId] = ChatDraft(
      peerId: peer.peerId,
      peerName: peer.peerUsername,
      peerAvatar: peer.peerAvatar,
      convId: convId,
      value: value,
      updatedAt: sameText
          ? previous!.updatedAt
          : DateTime.now().millisecondsSinceEpoch,
      revision: sameText ? previous!.revision : version,
    );
    _dirty[peer.peerId] = version;
    notifyListeners();
    _debounce?.cancel();
    _debounce = Timer(
      const Duration(milliseconds: 500),
      () => unawaited(flush()),
    );
  }

  void acknowledge(int peerId, int? revision, int conversationId) {
    if (!current || revision == null) return;
    final draft = _items[peerId];
    if (draft == null) return;
    if (draft.convId != conversationId) {
      _items[peerId] = ChatDraft.fromJson({
        ...draft.toJson(),
        'convId': conversationId,
      }, draft.revision);
      _dirty[peerId] = ++_serial;
      notifyListeners();
    }
    if (draft.revision == revision) {
      update(_items[peerId]!.conversation, TextEditingValue.empty);
    }
    unawaited(flush());
  }

  Future<bool> flush() {
    _debounce?.cancel();
    if (!current) return Future.value(false);
    final result = _flushing.then((_) async {
      await _load();
      if (!current || !_loaded || _scope == null) return false;
      for (final entry in Map<int, int>.of(_dirty).entries) {
        final draft = _items[entry.key]!;
        try {
          await store.write(_scope!, draft, isCurrent: () => current);
          if (!current) return false;
          if (_dirty[entry.key] == entry.value) _dirty.remove(entry.key);
        } catch (failure) {
          if (current) {
            error = failure;
            notifyListeners();
          }
          return false;
        }
      }
      if (!current) return false;
      error = null;
      notifyListeners();
      return !dirty;
    });
    _flushing = result;
    return result;
  }

  @override
  void dispose() {
    _disposed = true;
    _debounce?.cancel();
    _items.clear();
    _dirty.clear();
    super.dispose();
  }
}
