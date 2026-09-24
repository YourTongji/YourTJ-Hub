import 'dart:convert';
import 'dart:math';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../current_user.dart';
import '../providers.dart';

/// Device-local data is namespaced by API origin and numeric account ID.
/// Tokens are never persisted here. Guest history uses ID 0; drafts require login.
final writingScopeProvider = FutureProvider<String>((ref) async {
  ref.watch(offlineCacheEpochProvider);
  final site = Uri.parse(ref.watch(apiClientProvider).baseUrl).origin;
  final user = await ref.watch(currentUserProvider.future);
  return writingScope(site, user?.id ?? 0);
});
String writingScope(String site, int userId) =>
    '${Uri.encodeComponent(site)}:$userId';

final writingStoreProvider = Provider<WritingStore>((ref) => WritingStore());

enum DraftKind { newTopic, serverDraft, topicEdit, reply }

/// New writing sessions never share a content-type slot. The v1 storage prefix
/// remains readable; metadata is additive so existing recovery copies survive.
String newTopicDraftKey() {
  final random = Random.secure();
  return 'new-${List.generate(16, (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0')).join()}';
}

String topicDraftKey(int topicId, {required bool published}) =>
    '${published ? 'topic-edit' : 'server-draft'}-$topicId';
String replyDraftKey(int topicId) => 'reply-$topicId';

class LocalDraft {
  const LocalDraft({
    required this.key,
    required this.title,
    required this.content,
    required this.contentType,
    required this.topicId,
    required this.categories,
    required this.images,
    required this.updatedAt,
    this.kind = DraftKind.newTopic,
    this.replyToPostId = 0,
    this.replyTargetName,
    this.replyMentionPrefix,
  });
  final DraftKind kind;
  final int replyToPostId;
  final String? replyTargetName, replyMentionPrefix;
  final String key, title, content;
  final int contentType, topicId, updatedAt;
  final List<int> categories;
  final List<String> images;
  bool get isEmpty =>
      (kind == DraftKind.reply || title.trim().isEmpty) &&
      content.trim().isEmpty &&
      images.isEmpty &&
      categories.isEmpty &&
      replyToPostId == 0;
  Map<String, dynamic> toJson() => {
    'version': 2,
    'kind': kind.name,
    'replyToPostId': replyToPostId,
    'replyTargetName': replyTargetName,
    'replyMentionPrefix': replyMentionPrefix,
    'key': key,
    'title': title,
    'content': content,
    'contentType': contentType,
    'topicId': topicId,
    'categories': categories,
    'images': images,
    'updatedAt': updatedAt,
  };
  factory LocalDraft.fromJson(Map<String, dynamic> json) => LocalDraft(
    kind:
        DraftKind.values
            .where((kind) => kind.name == json['kind'])
            .firstOrNull ??
        ((json['topicId'] as int) > 0
            ? DraftKind.topicEdit
            : DraftKind.newTopic),
    replyToPostId: json['replyToPostId'] as int? ?? 0,
    replyTargetName: json['replyTargetName'] as String?,
    replyMentionPrefix: json['replyMentionPrefix'] as String?,
    key: json['key'] as String,
    title: json['title'] as String,
    content: json['content'] as String,
    contentType: json['contentType'] as int,
    topicId: json['topicId'] as int,
    categories: List<int>.from(json['categories'] as List),
    images: List<String>.from(json['images'] as List),
    updatedAt: json['updatedAt'] as int,
  );
}

class WritingStore {
  Future<void> _tail = Future.value();
  String _prefix(String scope) => 'yourtj:writing:v1:$scope:';

  // Serialize mutations so deletion follows any in-flight save. A platform
  // remove(false) is success only when the key was already absent; failure to
  // remove an existing recovery copy must remain visible to the caller.
  Future<void> _remove(SharedPreferences prefs, String key) async {
    final existed = prefs.containsKey(key);
    if (!await prefs.remove(key) && existed) {
      throw StateError('Local draft could not be removed');
    }
  }

  Future<T> _write<T>(Future<T> Function(SharedPreferences) action) {
    final next = _tail.then(
      (_) async => action(await SharedPreferences.getInstance()),
    );
    _tail = next.then<void>((_) {}, onError: (Object _, StackTrace _) {});
    return next;
  }

  Future<void> save(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) => _write((prefs) async {
    if (isCurrent != null && !isCurrent()) {
      throw StateError('Draft session changed');
    }
    if (scope.endsWith(':0')) throw StateError('Draft requires an account');
    final key = '${_prefix(scope)}draft:${draft.key}';
    if (draft.isEmpty) {
      await _remove(prefs, key);
      return;
    }
    final ok = await prefs.setString(key, jsonEncode(draft.toJson()));
    if (!ok) throw StateError('Local draft could not be saved');
  });
  Future<void> delete(String scope, String key, {bool Function()? isCurrent}) =>
      _write((prefs) async {
        if (isCurrent != null && !isCurrent()) {
          throw StateError('Draft session changed');
        }
        await _remove(prefs, '${_prefix(scope)}draft:$key');
      });

  /// Undo only a deletion that is still absent. The check and write share the
  /// mutation queue, so an editor save queued first always wins over recovery.
  Future<bool> restoreIfAbsent(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) => _write((prefs) async {
    // SharedPreferences updates its cache before a platform write succeeds.
    // Reload allows retry after a failed restore without mistaking cache for disk.
    await prefs.reload();
    if (isCurrent != null && !isCurrent()) {
      throw StateError('Draft session changed');
    }
    if (scope.endsWith(':0')) throw StateError('Draft requires an account');
    final key = '${_prefix(scope)}draft:${draft.key}';
    if (prefs.containsKey(key)) return false;
    if (!await prefs.setString(key, jsonEncode(draft.toJson()))) {
      throw StateError('Local draft could not be restored');
    }
    return true;
  });

  Future<List<LocalDraft>> drafts(String scope) async {
    await _tail;
    if (scope.endsWith(':0')) return [];
    final prefs = await SharedPreferences.getInstance();
    final drafts = <LocalDraft>[];
    for (final key in prefs.getKeys().where(
      (key) => key.startsWith('${_prefix(scope)}draft:'),
    )) {
      try {
        drafts.add(
          LocalDraft.fromJson(
            jsonDecode(prefs.getString(key)!) as Map<String, dynamic>,
          ),
        );
      } on FormatException {
        /* A damaged record must not hide other drafts. */
      } on TypeError {
        /* Preserve the raw record for recovery. */
      }
    }
    return drafts..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
  }

  Future<List<String>> history(String scope) async {
    await _tail;
    return (await SharedPreferences.getInstance()).getStringList(
          '${_prefix(scope)}history',
        ) ??
        [];
  }

  Future<void> remember(String scope, String query) => _write((prefs) async {
    final value = query.trim();
    if (value.isEmpty || value.length > 200) return;
    final key = '${_prefix(scope)}history';
    final old = prefs.getStringList(key) ?? [];
    if (!await prefs.setStringList(
      key,
      [value, ...old.where((q) => q != value)].take(10).toList(),
    )) {
      throw StateError('Search history could not be saved');
    }
  });
  Future<void> clearAccount(String scope) => _write((prefs) async {
    for (final key in prefs.getKeys().where(
      (key) => key.startsWith(_prefix(scope)),
    )) {
      await prefs.remove(key);
    }
  });

  Future<void> clearHistory(String scope) => _write((prefs) async {
    await prefs.remove('${_prefix(scope)}history');
  });
}
