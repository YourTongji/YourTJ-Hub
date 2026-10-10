import 'dart:convert';
import 'dart:math';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../storage/user_work_database.dart';
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

final writingStoreProvider = Provider<WritingStore>((ref) {
  ref.watch(offlineCacheEpochProvider);
  return WritingStore(database: ref.watch(userWorkDatabaseProvider));
});

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
    this.agentRepliesDisabled = false,
    this.identity = "member",
    this.replyToPostId = 0,
    this.replyTargetName,
    this.replyMentionPrefix,
  });
  final DraftKind kind;
  final bool agentRepliesDisabled;
  final String identity;
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
      !agentRepliesDisabled &&
      replyToPostId == 0;
  Map<String, dynamic> toJson() => {
    'version': 2,
    'identity': identity,
    'kind': kind.name,
    'agentRepliesDisabled': agentRepliesDisabled,
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
    agentRepliesDisabled: json['agentRepliesDisabled'] as bool? ?? false,
    identity: _draftIdentity(json["identity"]),
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
  WritingStore({UserWorkDatabase? database})
    : _database = database ?? UserWorkDatabase.instance,
      _generation = (database ?? UserWorkDatabase.instance).generation;
  final UserWorkDatabase _database;
  final int _generation;

  Future<void> save(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) async {
    if (scope.endsWith(':0')) throw StateError('Draft requires an account');
    await _database.writeBatch(
      scope,
      'draft',
      {draft.key: draft.isEmpty ? null : jsonEncode(draft.toJson())},
      isCurrent: isCurrent,
      generation: _generation,
    );
  }

  Future<void> delete(String scope, String key, {bool Function()? isCurrent}) =>
      _database.writeBatch(
        scope,
        'draft',
        {key: null},
        isCurrent: isCurrent,
        generation: _generation,
      );

  /// The absence check and restore share a transaction across all store instances.
  Future<bool> restoreIfAbsent(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) async {
    if (scope.endsWith(':0')) throw StateError('Draft requires an account');
    return _database.writeIfAbsent(
      scope,
      'draft',
      draft.key,
      jsonEncode(draft.toJson()),
      isCurrent: isCurrent,
      generation: _generation,
    );
  }

  Future<List<LocalDraft>> drafts(String scope) async {
    if (scope.endsWith(':0')) return [];
    final records = await _database.readDomain(scope, 'draft');
    final drafts = <LocalDraft>[];
    for (final raw in records.values) {
      try {
        drafts.add(
          LocalDraft.fromJson(jsonDecode(raw) as Map<String, dynamic>),
        );
      } on FormatException {
        /* Damaged raw records remain in the database for recovery. */
      } on TypeError {
        /* A damaged record must not hide other drafts. */
      }
    }
    return drafts..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
  }

  Future<List<String>> history(String scope) async {
    final values = await _database.readDomain(scope, 'history');
    return values['search'] == null
        ? []
        : List<String>.from(jsonDecode(values['search']!) as List);
  }

  Future<void> remember(String scope, String query) async {
    final value = query.trim();
    if (value.isEmpty || value.length > 200) return;
    await _database.updateValue(scope, 'history', 'search', (raw) {
      final old = raw == null
          ? <String>[]
          : List<String>.from(jsonDecode(raw) as List);
      return jsonEncode(
        [value, ...old.where((q) => q != value)].take(10).toList(),
      );
    }, generation: _generation);
  }

  Future<void> forget(String scope, String query) async {
    await _database.updateValue(scope, 'history', 'search', (raw) {
      if (raw == null) return null;
      return jsonEncode(
        List<String>.from(
          jsonDecode(raw) as List,
        ).where((q) => q != query).toList(),
      );
    }, generation: _generation);
  }

  Future<void> clearAccount(String scope) => _database.clearScope(scope);
  Future<void> clearHistory(String scope) => _database.writeBatch(
    scope,
    'history',
    {'search': null},
    generation: _generation,
  );
}

String _draftIdentity(Object? value) {
  if (value == null) return "member";
  if (value == "member" || value == "persona") return value as String;
  throw const FormatException("Invalid draft identity");
}
