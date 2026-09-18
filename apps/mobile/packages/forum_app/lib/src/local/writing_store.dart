import 'dart:convert';
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
  });
  final String key, title, content;
  final int contentType, topicId, updatedAt;
  final List<int> categories;
  final List<String> images;
  bool get isEmpty =>
      title.trim().isEmpty &&
      content.trim().isEmpty &&
      images.isEmpty &&
      categories.isEmpty;
  Map<String, dynamic> toJson() => {
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

  // Serialize mutations, including deletion, so a late autosave cannot undo a
  // successful publish or resurrect cleared history. Failed writes are surfaced.
  Future<void> _write(Future<void> Function(SharedPreferences) action) {
    final next = _tail.then(
      (_) async => action(await SharedPreferences.getInstance()),
    );
    _tail = next.catchError((Object _) {});
    return next;
  }

  Future<void> save(String scope, LocalDraft draft) => _write((prefs) async {
    if (scope.endsWith(':0')) throw StateError('Draft requires an account');
    final key = '${_prefix(scope)}draft:${draft.key}';
    final ok = draft.isEmpty
        ? await prefs.remove(key)
        : await prefs.setString(key, jsonEncode(draft.toJson()));
    if (!ok) throw StateError('Local draft could not be saved');
  });
  Future<void> delete(String scope, String key) => _write((prefs) async {
    await prefs.remove('${_prefix(scope)}draft:$key');
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
