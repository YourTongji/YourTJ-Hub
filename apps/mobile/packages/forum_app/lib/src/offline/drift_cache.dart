import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:drift/drift.dart';
import '../storage/private_database.dart';

import 'cache_schema.dart';

/// A server answer that denies, deletes or revokes access to a resource must
/// revoke its visible snapshot and disk copy instead of falling back to it.
/// 401 additionally triggers the global session invalidation.
bool revokesSnapshot(Object error) {
  final status = switch (error) {
    ApiException(statusCode: final status) => status,
    DioException(:final response?) => response.statusCode,
    _ => null,
  };
  return status == 401 || status == 403 || status == 404 || status == 410;
}

abstract class OfflineTopicCache {
  Future<void> put(int topicId, Map<String, dynamic> payload);
  Future<PagePayload?> get(int topicId);
  Future<void> clear();
  Future<void> close();
}

abstract class OfflineChatCache {
  /// Replaces a complete server snapshot; [] is an authoritative empty result.
  Future<void> putConversations(List<ChatItemPayload> conversations);
  Future<List<ChatItemPayload>> getConversations();

  /// Merges a server delta. An empty delta does not remove history.
  Future<void> putMessages(int convId, List<ChatMessagePayload> messages);
  Future<List<ChatMessagePayload>> getMessages(int convId);
  Future<void> clear();
}

abstract interface class OfflineHomeCache {
  Future<void> putHomePage({
    required int accountId,
    required String baseUrl,
    required String sort,
    required PagePayload payload,
  });
  Future<PagePayload?> getHomePage({
    required int accountId,
    required String baseUrl,
    required String sort,
  });
}

enum CacheCategory { forum, chat, campus, media }

/// Stable identity. Session epochs and cleanup generations are separate fences.
class CacheScope {
  CacheScope(String site, this.accountId, {this.language = 'zh'})
    : origin = Uri.parse(site).origin;
  final String origin;
  final int accountId;
  final String language;
  String get key => jsonEncode([origin, accountId, language]);
}

class AppDatabase extends GeneratedDatabase {
  AppDatabase(super.e, {this.physicalBytes});

  /// Includes native database/WAL/SHM files. Logical payload budgets stay
  /// independent: 32 MiB forum, 16 MiB chat and at most 4 MiB campus documents.
  final Future<int> Function()? physicalBytes;
  static const physicalBudgetBytes = 64 * 1024 * 1024;
  static const _reclaimThreshold = 4 * 1024 * 1024;
  int? _lastVacuumBytes;

  @override
  int get schemaVersion => 2;

  @override
  Iterable<TableInfo<Table, dynamic>> get allTables => const [];

  Future<void> _createSchema() async {
    for (final statement in cacheSchema) {
      await customStatement(statement);
    }
  }

  final Map<CacheCategory, int> _generations = {};
  final Set<CacheCategory> _suspended = {};
  bool available(CacheCategory category) => !_suspended.contains(category);
  void suspend(Set<CacheCategory> categories) {
    invalidate(categories);
    _suspended.addAll(categories);
  }

  void release(Set<CacheCategory> categories) =>
      _suspended.removeAll(categories);
  Future<void>? _tail;

  Map<CacheCategory, int> capture() => {
    for (final category in CacheCategory.values)
      category: _generations[category] ?? 0,
  };

  bool accepts(Map<CacheCategory, int>? lease, CacheCategory category) =>
      available(category) &&
      (lease == null || lease[category] == (_generations[category] ?? 0));

  void invalidate(Iterable<CacheCategory> categories) {
    for (final category in categories) {
      _generations[category] = (_generations[category] ?? 0) + 1;
    }
  }

  Future<T> serial<T>(Future<T> Function() action) {
    final next = _tail?.then((_) => action()) ?? Future<T>.sync(action);
    late final Future<void> completion;
    void released() {
      if (identical(_tail, completion)) _tail = null;
    }

    completion = next.then<void>(
      (_) => released(),
      onError: (Object _, StackTrace _) => released(),
    );
    _tail = completion;
    return next;
  }

  @override
  MigrationStrategy get migration => MigrationStrategy(
    onCreate: (_) => _createSchema(),
    onUpgrade: (m, from, to) async {
      if (from < 2) {
        // Legacy topic/chat rows have no trustworthy account/site identity.
        // Rebuild disposable snapshots, preserving the allowlisted campus table.
        for (final table in [
          'cached_topics',
          'cached_conversations',
          'cached_messages',
          'cached_home_pages',
        ]) {
          await customStatement('DROP TABLE IF EXISTS $table');
        }
        await _createSchema();
      }
    },
    beforeOpen: (_) async {
      await customStatement('PRAGMA foreign_keys = ON');
      if (await operation('reset') != null) {
        _suspended.addAll(CacheCategory.values);
      }
      final pending = await operation('clear');
      if (pending != null) {
        try {
          final names = (jsonDecode(pending) as List).cast<String>();
          _suspended.addAll(
            CacheCategory.values.where((c) => names.contains(c.name)),
          );
        } catch (_) {
          _suspended.addAll(CacheCategory.values);
        }
      }
      // Schema upgrades can free a large legacy cache without shrinking its
      // file. beforeOpen is outside Drift's migration transaction.
      await _maintainBudget();
    },
  );

  Future<void> clearCategories(Set<CacheCategory> categories) {
    // Advance synchronously, before any queued old writer can enter a transaction.
    invalidate(categories);
    return serial(
      () => transaction(() async {
        for (final category in categories) {
          if (category == CacheCategory.campus) {
            await customStatement('DELETE FROM campus_snapshots');
          } else {
            await customStatement(
              'DELETE FROM cache_entries WHERE domain = ?',
              [category.name],
            );
          }
        }
      }),
    );
  }

  Future<void> purgePrivateScopes() => serial(
    () => transaction(() async {
      // Null-safe and fail-closed: a scope that is not a JSON array with a
      // numeric account cannot prove it belongs to a guest, so it is removed.
      await customStatement(
        r"DELETE FROM cache_entries WHERE json_valid(scope) IS NOT 1 OR json_extract(scope, '$[1]') IS NOT 0",
      );
      await customStatement('DELETE FROM campus_snapshots');
    }),
  );

  Future<Map<CacheCategory, int>> cacheUsage() => serial(() async {
    final rows = await customSelect(
      'SELECT domain, SUM(byte_size) AS total FROM cache_entries GROUP BY domain',
    ).get();
    final result = <CacheCategory, int>{
      for (final category in CacheCategory.values) category: 0,
    };
    for (final row in rows) {
      final category = CacheCategory.values
          .where((c) => c.name == row.read<String>('domain'))
          .firstOrNull;
      if (category != null) result[category] = row.read<int>('total');
    }
    final campus = await customSelect(
      'SELECT COALESCE(SUM(LENGTH(CAST(payload AS BLOB))), 0) AS total FROM campus_snapshots',
    ).getSingle();
    result[CacheCategory.campus] = campus.read<int>('total');
    return result;
  });

  Future<void> setOperation(String name, String value) => customStatement(
    'INSERT OR REPLACE INTO storage_operations (operation, value) VALUES (?, ?)',
    [name, value],
  );
  Future<String?> operation(String name) async => (await customSelect(
    'SELECT value FROM storage_operations WHERE operation = ?',
    variables: [Variable.withString(name)],
  ).getSingleOrNull())?.read<String>('value');
  Future<void> finishOperation(String name) => customStatement(
    'DELETE FROM storage_operations WHERE operation = ?',
    [name],
  );

  /// Reclaims SQLite free pages after explicit cleanup, outside UI rendering.
  Future<void> compact() => serial(() async {
    await customStatement('PRAGMA wal_checkpoint(TRUNCATE)');
    await customStatement('VACUUM');
    await customStatement('PRAGMA wal_checkpoint(TRUNCATE)');
  });

  /// Run after a write transaction. Reclaim meaningful free space or a physical
  /// high watermark, rather than rewriting the file for every small update.
  /// This does not evict campus snapshots or user work to satisfy a byte target.
  Future<void> maintainBudget() => serial(_maintainBudget);

  Future<int> _pragma(String name) async =>
      (await customSelect('PRAGMA $name').getSingle()).read<int>(name);

  Future<void> _maintainBudget() async {
    final pageSize = await _pragma('page_size');
    var allocated = (await _pragma('page_count')) * pageSize;
    var free = (await _pragma('freelist_count')) * pageSize;
    var physical = await physicalBytes?.call() ?? allocated;
    if (free < _reclaimThreshold && physical <= physicalBudgetBytes) return;
    if (free > 0) {
      // Consume all pragma rows: SQLite may yield once per reclaimed page.
      await customSelect('PRAGMA incremental_vacuum').get();
    }
    await customSelect('PRAGMA wal_checkpoint(TRUNCATE)').get();
    allocated = (await _pragma('page_count')) * pageSize;
    free = (await _pragma('freelist_count')) * pageSize;
    physical = await physicalBytes?.call() ?? allocated;
    // An older non-incremental import or heavily fragmented live pages
    // need a full rewrite. If live data itself exceeds the target, do not repeat
    // that rewrite until allocation grows meaningfully again.
    if (free >= _reclaimThreshold ||
        (physical > physicalBudgetBytes &&
            (_lastVacuumBytes == null ||
                physical >= _lastVacuumBytes! + _reclaimThreshold))) {
      await customStatement('VACUUM');
      await customSelect('PRAGMA wal_checkpoint(TRUNCATE)').get();
      _lastVacuumBytes =
          await physicalBytes?.call() ??
          (await _pragma('page_count')) * pageSize;
    }
  }
}

AppDatabase openDatabase() => AppDatabase(
  openPrivateDatabase(
    name: 'cache',
    disposable: true,
    legacyName: 'yourtj_cache',
  ),
  physicalBytes: () => privateDatabaseBytes('cache'),
);

/// All production instances resolve a stable account/site scope and capture a
/// session fence. A per-request lease must be captured BEFORE starting network IO.
class DriftOfflineCache
    implements OfflineTopicCache, OfflineChatCache, OfflineHomeCache {
  DriftOfflineCache(
    this._db, {
    this.resolveScope,
    this.sessionCurrent,
    this.clearAllCaches,
    DateTime Function()? now,
    this.forumBudget = 32 * 1024 * 1024,
    this.chatBudget = 16 * 1024 * 1024,
    Map<CacheCategory, int>? lease,
  }) : now = now ?? DateTime.now,
       // ignore: prefer_initializing_formals
       _lease = lease;

  final AppDatabase _db;
  final Future<CacheScope> Function()? resolveScope;
  final bool Function()? sessionCurrent;
  final Future<void> Function()? clearAllCaches;
  final DateTime Function() now;
  final int forumBudget, chatBudget;
  final Map<CacheCategory, int>? _lease;
  DateTime? snapshotTime;
  bool snapshotFound = false;
  bool snapshotFresh = false;

  bool isCurrent(CacheCategory category) =>
      (sessionCurrent?.call() ?? true) && _db.accepts(_lease, category);

  DriftOfflineCache capture() => DriftOfflineCache(
    _db,
    resolveScope: resolveScope,
    sessionCurrent: sessionCurrent,
    clearAllCaches: clearAllCaches,
    now: now,
    forumBudget: forumBudget,
    chatBudget: chatBudget,
    lease: _db.capture(),
  );

  Future<CacheScope> _scope() async => resolveScope == null
      ? CacheScope('https://local.invalid', 0)
      : resolveScope!();

  Future<String?> _homeScope(int accountId, String baseUrl) async {
    final owner = await _scope();
    final requested = CacheScope(baseUrl, accountId, language: owner.language);
    if (resolveScope != null && requested.key != owner.key) return null;
    return requested.key;
  }

  static Duration retention(CacheCategory category) =>
      category == CacheCategory.chat
      ? const Duration(days: 30)
      : const Duration(days: 7);

  Future<Map<String, dynamic>?> _read(
    CacheCategory category,
    String key, {
    String? scope,
    Duration freshFor = const Duration(minutes: 5),
  }) async {
    final owner = scope ?? (await _scope()).key;
    snapshotFound = false;
    snapshotFresh = false;
    snapshotTime = null;
    return _db.serial(() async {
      if (!isCurrent(category)) return null;
      final row = await _db
          .customSelect(
            'SELECT * FROM cache_entries WHERE scope = ? AND domain = ? AND entry_key = ?',
            variables: [
              Variable.withString(owner),
              Variable.withString(category.name),
              Variable.withString(key),
            ],
          )
          .getSingleOrNull();
      if (row == null || !isCurrent(category)) return null;
      final stamp = DateTime.fromMillisecondsSinceEpoch(
        row.read<int>('saved_at'),
        isUtc: true,
      );
      try {
        if (row.read<int>('document_version') != 1 ||
            now().difference(stamp) > retention(category) ||
            stamp.isAfter(now().add(const Duration(minutes: 5)))) {
          throw const FormatException('Expired snapshot');
        }
        final json =
            jsonDecode(row.read<String>('payload')) as Map<String, dynamic>;
        snapshotFound = true;
        snapshotTime = stamp;
        snapshotFresh = now().difference(stamp) < freshFor;
        // Access is not validation. It never extends retention or freshness.
        if (now().millisecondsSinceEpoch - row.read<int>('accessed_at') >
            const Duration(minutes: 1).inMilliseconds) {
          await _db.customStatement(
            'UPDATE cache_entries SET accessed_at = ? WHERE scope = ? AND domain = ? AND entry_key = ?',
            [now().millisecondsSinceEpoch, owner, category.name, key],
          );
        }
        return json;
      } on FormatException {
        await _remove(owner, category, key);
        return null;
      } on TypeError {
        await _remove(owner, category, key);
        return null;
      }
    });
  }

  Future<void> _remove(String scope, CacheCategory category, String key) async {
    await _db.customStatement(
      'DELETE FROM cache_entries WHERE scope = ? AND domain = ? AND entry_key = ?',
      [scope, category.name, key],
    );
    if (category == CacheCategory.chat && key == 'conversations') {
      await _db.customStatement(
        "DELETE FROM cache_entries WHERE scope = ? AND domain = 'chat' AND entry_key LIKE 'messages:%'",
        [scope],
      );
    }
  }

  Future<void> removeTopic(int topicId) async {
    final scope = (await _scope()).key;
    await _db.serial(() async {
      if (isCurrent(CacheCategory.forum)) {
        await _remove(scope, CacheCategory.forum, 'topic:$topicId');
      }
    });
  }

  /// Access denial on the home feed revokes every sort of the owner scope the
  /// page addressed. A resolver mismatch removes nothing, like reads and writes.
  Future<void> removeHome({
    required int accountId,
    required String baseUrl,
  }) async {
    final scope = await _homeScope(accountId, baseUrl);
    if (scope == null) return;
    await _db.serial(() async {
      if (!isCurrent(CacheCategory.forum)) return;
      await _db.customStatement(
        "DELETE FROM cache_entries WHERE scope = ? AND domain = 'forum' AND entry_key LIKE 'home:%'",
        [scope],
      );
    });
  }

  /// Access denial on the conversation list revokes the snapshot and every
  /// cached message thread of the resolved scope.
  Future<void> removeConversations() async {
    final scope = (await _scope()).key;
    await _db.serial(() async {
      if (!isCurrent(CacheCategory.chat)) return;
      await _remove(scope, CacheCategory.chat, 'conversations');
      await _db.customStatement(
        "DELETE FROM cache_entries WHERE scope = ? AND domain = 'chat' AND entry_key LIKE 'messages:%'",
        [scope],
      );
    });
  }

  /// Access denial inside one conversation revokes only that thread.
  Future<void> removeMessages(int convId) async {
    final scope = (await _scope()).key;
    await _db.serial(() async {
      if (isCurrent(CacheCategory.chat)) {
        await _remove(scope, CacheCategory.chat, 'messages:$convId');
      }
    });
  }

  Future<void> _store(
    String scope,
    CacheCategory category,
    String key,
    Map<String, dynamic> value,
  ) async {
    if (!isCurrent(category)) return;
    final encoded = jsonEncode(value);
    final size = utf8.encode(encoded).length;
    final budget = category == CacheCategory.chat ? chatBudget : forumBudget;
    // Oversized documents are not cacheable; never retain an older contradictory copy.
    if (size > budget || size > 2 * 1024 * 1024) {
      await _remove(scope, category, key);
      return;
    }
    await _db.customStatement(
      'INSERT OR REPLACE INTO cache_entries '
      '(scope, domain, entry_key, payload, saved_at, accessed_at, byte_size, document_version) '
      'VALUES (?, ?, ?, ?, ?, ?, ?, 1)',
      [
        scope,
        category.name,
        key,
        encoded,
        now().millisecondsSinceEpoch,
        now().millisecondsSinceEpoch,
        size,
      ],
    );
  }

  Future<void> _write(
    CacheCategory category,
    String key,
    Map<String, dynamic> value, {
    String? scope,
  }) async {
    final owner = scope ?? (await _scope()).key;
    // Calls on a request lease retain its generation across scope resolution.
    await _db.serial(
      () => _db.transaction(() async {
        if (!isCurrent(category)) return;
        await _store(owner, category, key, value);
        await _trim(category);
      }),
    );
    await _db.maintainBudget();
  }

  Future<void> _trim(CacheCategory category) async {
    final domain = category.name;
    final cutoff = now().subtract(retention(category)).millisecondsSinceEpoch;
    final expired = await _db
        .customSelect(
          'SELECT scope, entry_key FROM cache_entries WHERE domain = ? AND (saved_at < ? OR saved_at > ?)',
          variables: [
            Variable(domain),
            Variable(cutoff),
            Variable(
              now().add(const Duration(minutes: 5)).millisecondsSinceEpoch,
            ),
          ],
        )
        .get();
    for (final row in expired) {
      await _remove(
        row.read<String>('scope'),
        category,
        row.read<String>('entry_key'),
      );
    }
    final rows = await _db
        .customSelect(
          'SELECT scope, entry_key, byte_size FROM cache_entries WHERE domain = ? ORDER BY accessed_at DESC, saved_at DESC, entry_key DESC',
          variables: [Variable.withString(domain)],
        )
        .get();
    final budget = category == CacheCategory.chat ? chatBudget : forumBudget;
    final allocated = rows.fold<int>(
      0,
      (sum, row) => sum + row.read<int>('byte_size'),
    );
    final target = allocated > budget ? (budget * .8).floor() : budget;
    var total = 0;
    var count = 0;
    for (final row in rows) {
      total += row.read<int>('byte_size');
      count++;
      if ((total > target && count > 1) || total > budget || count > 100) {
        await _remove(
          row.read<String>('scope'),
          category,
          row.read<String>('entry_key'),
        );
      }
    }
  }

  @override
  Future<void> putHomePage({
    required int accountId,
    required String baseUrl,
    required String sort,
    required PagePayload payload,
  }) async {
    final scope = await _homeScope(accountId, baseUrl);
    if (scope == null) return;
    final projection = _pageProjection(payload);
    if (projection == null) return;
    await _write(CacheCategory.forum, 'home:$sort', projection, scope: scope);
  }

  @override
  Future<PagePayload?> getHomePage({
    required int accountId,
    required String baseUrl,
    required String sort,
  }) async {
    final scope = await _homeScope(accountId, baseUrl);
    if (scope == null) return null;
    return _decodePage(
      await _read(
        CacheCategory.forum,
        'home:$sort',
        scope: scope,
        freshFor: const Duration(minutes: 1),
      ),
    );
  }

  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {
    final projection = _pageProjection(PagePayload.fromJson(payload));
    if (projection == null) {
      await removeTopic(topicId);
      return;
    }
    await _write(CacheCategory.forum, 'topic:$topicId', projection);
  }

  @override
  Future<PagePayload?> get(int topicId) async =>
      _decodePage(await _read(CacheCategory.forum, 'topic:$topicId'));

  @override
  Future<void> putConversations(List<ChatItemPayload> conversations) async {
    final scope = (await _scope()).key;
    await _db.serial(
      () => _db.transaction(() async {
        if (!isCurrent(CacheCategory.chat)) return;
        final selected = conversations.take(50).toList();
        final keys = {for (final item in selected) 'messages:${item.convId}'};
        final rows = await _db
            .customSelect(
              "SELECT entry_key FROM cache_entries WHERE scope = ? AND domain = 'chat'",
              variables: [Variable.withString(scope)],
            )
            .get();
        for (final row in rows) {
          if (row.read<String>('entry_key').startsWith('messages:') &&
              !keys.contains(row.read<String>('entry_key'))) {
            await _remove(
              scope,
              CacheCategory.chat,
              row.read<String>('entry_key'),
            );
          }
        }
        await _store(scope, CacheCategory.chat, 'conversations', {
          'items': selected.map((c) => c.toJson()).toList(),
        });
        await _trim(CacheCategory.chat);
      }),
    );
    await _db.maintainBudget();
  }

  @override
  Future<List<ChatItemPayload>> getConversations() async {
    final json = await _read(
      CacheCategory.chat,
      'conversations',
      freshFor: Duration.zero,
    );
    if (json == null) return [];
    try {
      return (json['items'] as List)
          .map(
            (item) => ChatItemPayload.fromJson(
              Map<String, dynamic>.from(item as Map),
            ),
          )
          .toList();
    } catch (_) {
      snapshotFound = false;
      return [];
    }
  }

  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {
    if (messages.isEmpty) return; // A delta, unlike the conversation snapshot.
    final scope = (await _scope()).key;
    await _db.serial(
      () => _db.transaction(() async {
        if (!isCurrent(CacheCategory.chat)) return;
        final key = 'messages:$convId';
        final row = await _db
            .customSelect(
              "SELECT * FROM cache_entries WHERE scope = ? AND domain = 'chat' AND entry_key = ?",
              variables: [Variable.withString(scope), Variable.withString(key)],
            )
            .getSingleOrNull();
        final byId = <int, ChatMessagePayload>{};
        final validatedAt = <int, int>{};
        final cutoff = now()
            .subtract(retention(CacheCategory.chat))
            .millisecondsSinceEpoch;
        if (row != null &&
            now().millisecondsSinceEpoch - row.read<int>('saved_at') <=
                retention(CacheCategory.chat).inMilliseconds) {
          try {
            for (final item
                in (jsonDecode(row.read<String>('payload')) as Map)['items']
                    as List) {
              final message = ChatMessagePayload.fromJson(
                Map<String, dynamic>.from(item as Map),
              );
              final stamp =
                  item['_validatedAt'] as int? ?? row.read<int>('saved_at');
              if (stamp >= cutoff &&
                  stamp <=
                      now()
                          .add(const Duration(minutes: 5))
                          .millisecondsSinceEpoch) {
                byId[message.id] = message;
                validatedAt[message.id] = stamp;
              }
            }
          } catch (_) {
            /* Rebuild only this disposable snapshot. */
          }
        }
        for (final message in messages) {
          byId[message.id] = message;
          validatedAt[message.id] = now().millisecondsSinceEpoch;
        }
        final ordered = byId.values.toList()
          ..sort((a, b) => b.id.compareTo(a.id));
        await _store(scope, CacheCategory.chat, key, {
          'items': ordered
              .take(200)
              .toList()
              .reversed
              .map((m) => {...m.toJson(), '_validatedAt': validatedAt[m.id]})
              .toList(),
        });
        await _trim(CacheCategory.chat);
      }),
    );
    await _db.maintainBudget();
  }

  @override
  Future<List<ChatMessagePayload>> getMessages(int convId) async {
    final json = await _read(
      CacheCategory.chat,
      'messages:$convId',
      freshFor: Duration.zero,
    );
    if (json == null) return [];
    try {
      return (json['items'] as List)
          .where((item) {
            final stamp = (item as Map)['_validatedAt'] as int?;
            return stamp != null &&
                now().millisecondsSinceEpoch - stamp <=
                    retention(CacheCategory.chat).inMilliseconds &&
                stamp <=
                    now()
                        .add(const Duration(minutes: 5))
                        .millisecondsSinceEpoch;
          })
          .map(
            (item) => ChatMessagePayload.fromJson(
              Map<String, dynamic>.from(item as Map),
            ),
          )
          .toList();
    } catch (_) {
      snapshotFound = false;
      return [];
    }
  }

  @override
  Future<void> clear() =>
      clearAllCaches?.call() ??
      _db.clearCategories({
        CacheCategory.forum,
        CacheCategory.chat,
        CacheCategory.campus,
      });

  /// Clearing either view already clears every cache owned by this database.
  bool sharesStorageWith(Object other) =>
      other is DriftOfflineCache && identical(_db, other._db);

  @override
  Future<void> close() => _db.close();
}

OfflineTopicCache captureTopicCache(OfflineTopicCache cache) =>
    cache is DriftOfflineCache ? cache.capture() : cache;
OfflineChatCache captureChatCache(OfflineChatCache cache) =>
    cache is DriftOfflineCache ? cache.capture() : cache;
bool cacheRequestCurrent(Object cache, CacheCategory category) =>
    cache is! DriftOfflineCache || cache.isCurrent(category);

/// Explicit device-reading projection, not an HTTP response cache. Never save
/// layout.viewer, email, admin permissions, notification counters or page chrome.
Map<String, dynamic>? _pageProjection(PagePayload page) {
  final props = jsonDecode(jsonEncode(page.props)) as Map<String, dynamic>;
  void stripTrace(dynamic value) {
    if (value is Map) {
      value.remove('feedTrace');
      value.remove('feedPosition');
      for (final child in value.values) {
        stripTrace(child);
      }
    } else if (value is List) {
      for (final child in value) {
        stripTrace(child);
      }
    }
  }

  stripTrace(props);
  if (page.component == PageComponent.topicDetail) {
    final topic = props['topic'] as Map<String, dynamic>?;
    // Non-normal/moderated/deleted content must remain online-only.
    if (topic == null ||
        topic['topicStatus'] != 1 ||
        topic['processStatus'] != 0 ||
        topic['authorDeleted'] == true ||
        topic['moderatorRemoved'] == true) {
      return null;
    }
    props['permissions'] = {
      'isOwnTopic': false,
      'canPost': false,
      'canModerateTopic': false,
    };
    props['hotTopics'] = <Object>[];
    (props['postStream'] as Map)['replyTargets'] = <Object>[];
    for (final post in (props['postStream'] as Map)['posts'] as List) {
      post['canModerate'] = false;
      post['isOwnPost'] = false;
      if (post['isHidden'] == true ||
          post['isAuthorDeleted'] == true ||
          post['isModeratorRemoved'] == true ||
          (post['processStatus'] != null && post['processStatus'] != 0)) {
        post['content'] = '';
        post['renderedContent'] = '';
      }
    }
  } else if (page.component != PageComponent.home) {
    return null;
  }
  return {
    'component': page.component,
    'props': props,
    'categories': page.component == PageComponent.home
        ? page.layout.sidebar.categories.map((c) => c.toJson()).toList()
        : <Object>[],
  };
}

PagePayload? _decodePage(Map<String, dynamic>? json) {
  if (json == null) return null;
  try {
    return PagePayload.fromJson({
      'component': json['component'],
      'props': json['props'],
      'meta': {'title': ''},
      'url': '',
      'version': 'device-snapshot-v1',
      'layout': {
        'site': {
          'name': '',
          'description': '',
          'logo': '',
          'favicon': '',
          'brandType': 'text',
          'brandText': '',
          'brandImage': '',
        },
        'viewer': {
          'id': 0,
          'username': '',
          'email': '',
          'avatarUrl': '',
          'isAuthenticated': false,
          'canAccessAdmin': false,
          'isModerator': false,
          'requiresEmailVerification': false,
        },
        'sidebar': {
          'categories': json['categories'] ?? <Object>[],
          'activeKey': '',
        },
        'footer': {'links': <Object>[], 'primary': <Object>[]},
        'unread': {'notifications': false, 'messages': false},
        'theme': {'enabled': false, 'current': 'light', 'themeColor': ''},
      },
    });
  } catch (_) {
    return null;
  }
}
