import 'dart:convert';

import 'package:drift/drift.dart';
import 'package:drift_flutter/drift_flutter.dart';

import 'package:core/core.dart';

/// 离线话题缓存抽象(生产用 drift 实现,测试可注入 no-op)。
abstract class OfflineTopicCache {
  Future<void> put(int topicId, Map<String, dynamic> payload);
  Future<PagePayload?> get(int topicId);
  Future<void> clear();
  Future<void> close();
}

/// 离线 IM 会话缓存抽象(会话列表 + 单会话消息)。
abstract class OfflineChatCache {
  Future<void> putConversations(List<ChatItemPayload> conversations);
  Future<List<ChatItemPayload>> getConversations();
  Future<void> putMessages(int convId, List<ChatMessagePayload> messages);
  Future<List<ChatMessagePayload>> getMessages(int convId);
  Future<void> clear();
}

/// 已浏览话题/会话的 drift 离线缓存。
///
/// 不依赖 build_runner 生成代码:数据库承载三张 raw SQL 表
/// (`cached_topics` / `cached_conversations` / `cached_messages`),
/// 建表在 `beforeOpen` 中执行,读写走 customSelect/customStatement。
class AppDatabase extends GeneratedDatabase {
  AppDatabase(super.e);

  @override
  int get schemaVersion => 1;

  @override
  Iterable<TableInfo<Table, dynamic>> get allTables => const [];

  @override
  MigrationStrategy get migration => MigrationStrategy(
    beforeOpen: (details) async {
      await customStatement(
        'CREATE TABLE IF NOT EXISTS cached_topics ('
        'id INTEGER PRIMARY KEY, '
        'payload TEXT NOT NULL, '
        'cached_at TEXT NOT NULL)',
      );
      await customStatement(
        'CREATE TABLE IF NOT EXISTS cached_conversations ('
        'conv_id INTEGER PRIMARY KEY, '
        'payload TEXT NOT NULL, '
        'cached_at TEXT NOT NULL)',
      );
      await customStatement(
        'CREATE TABLE IF NOT EXISTS cached_messages ('
        'conv_id INTEGER NOT NULL, '
        'msg_id INTEGER NOT NULL, '
        'payload TEXT NOT NULL, '
        'cached_at TEXT NOT NULL, '
        'PRIMARY KEY (conv_id, msg_id))',
      );
    },
  );
}

/// 创建 drift 数据库实例(drift_flutter 管理原生 sqlite 文件)。
AppDatabase openDatabase() {
  return AppDatabase(driftDatabase(name: 'yourtj_cache'));
}

/// 基于 drift 的离线缓存(话题 + IM 会话),共享同一数据库实例。
class DriftOfflineCache implements OfflineTopicCache, OfflineChatCache {
  DriftOfflineCache(this._db);

  final AppDatabase _db;

  static const int _maxTopics = 50;
  static const int _maxConversations = 50;
  static const int _maxMessagesPerConv = 200;

  // ---- OfflineTopicCache ----

  /// 保存话题详情页负载(insertOrReplace)。
  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {
    final String json = jsonEncode(payload);
    await _db.customStatement(
      'INSERT OR REPLACE INTO cached_topics (id, payload, cached_at) '
      'VALUES (?, ?, ?)',
      [topicId, json, DateTime.now().toUtc().toIso8601String()],
    );
    await _trimTopics();
  }

  /// 读取缓存的负载;无缓存时返回 null。
  @override
  Future<PagePayload?> get(int topicId) async {
    final rows = await _db
        .customSelect(
          'SELECT payload FROM cached_topics WHERE id = ?',
          variables: [Variable.withInt(topicId)],
        )
        .get();
    if (rows.isEmpty) return null;
    final raw = rows.first.data['payload'] as String?;
    if (raw == null || raw.isEmpty) return null;
    try {
      return PagePayload.fromJson(jsonDecode(raw) as Map<String, dynamic>);
    } catch (_) {
      return null;
    }
  }

  Future<void> _trimTopics() async {
    final rows = await _db
        .customSelect(
          'SELECT id FROM cached_topics ORDER BY cached_at DESC LIMIT -1 OFFSET ?',
          variables: [Variable.withInt(_maxTopics)],
        )
        .get();
    for (final row in rows) {
      await _db.customStatement('DELETE FROM cached_topics WHERE id = ?', [
        row.data['id'],
      ]);
    }
  }

  // ---- OfflineChatCache ----

  /// 批量保存会话列表,用一个事务完成写入并只裁剪一次。
  @override
  Future<void> putConversations(List<ChatItemPayload> conversations) async {
    if (conversations.isEmpty) return;
    final String cachedAt = DateTime.now().toUtc().toIso8601String();
    await _db.transaction(() async {
      for (final ChatItemPayload conversation in conversations) {
        await _db.customStatement(
          'INSERT OR REPLACE INTO cached_conversations '
          '(conv_id, payload, cached_at) VALUES (?, ?, ?)',
          [conversation.convId, jsonEncode(conversation.toJson()), cachedAt],
        );
      }
      await _trimConversations();
    });
  }

  /// 读取全部已缓存会话(按最近活跃排序)。
  @override
  Future<List<ChatItemPayload>> getConversations() async {
    final rows = await _db
        .customSelect(
          'SELECT payload FROM cached_conversations ORDER BY cached_at DESC',
        )
        .get();
    final result = <ChatItemPayload>[];
    for (final row in rows) {
      final raw = row.data['payload'] as String?;
      if (raw == null || raw.isEmpty) continue;
      try {
        result.add(
          ChatItemPayload.fromJson(jsonDecode(raw) as Map<String, dynamic>),
        );
      } catch (_) {
        // 跳过损坏条目。
      }
    }
    return result;
  }

  Future<void> _trimConversations() async {
    final rows = await _db
        .customSelect(
          'SELECT conv_id FROM cached_conversations ORDER BY cached_at DESC LIMIT -1 OFFSET ?',
          variables: [Variable.withInt(_maxConversations)],
        )
        .get();
    for (final row in rows) {
      await _db.customStatement(
        'DELETE FROM cached_conversations WHERE conv_id = ?',
        [row.data['conv_id']],
      );
    }
  }

  /// 保存单会话消息列表。
  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {
    for (final m in messages) {
      await _db.customStatement(
        'INSERT OR REPLACE INTO cached_messages (conv_id, msg_id, payload, cached_at) '
        'VALUES (?, ?, ?, ?)',
        [
          convId,
          m.id,
          jsonEncode(m.toJson()),
          DateTime.now().toUtc().toIso8601String(),
        ],
      );
    }
    await _trimMessages(convId);
  }

  /// 读取单会话已缓存消息(按消息 id 升序)。
  @override
  Future<List<ChatMessagePayload>> getMessages(int convId) async {
    final rows = await _db
        .customSelect(
          'SELECT payload FROM cached_messages WHERE conv_id = ? ORDER BY msg_id ASC',
          variables: [Variable.withInt(convId)],
        )
        .get();
    final result = <ChatMessagePayload>[];
    for (final row in rows) {
      final raw = row.data['payload'] as String?;
      if (raw == null || raw.isEmpty) continue;
      try {
        result.add(
          ChatMessagePayload.fromJson(jsonDecode(raw) as Map<String, dynamic>),
        );
      } catch (_) {
        // 跳过损坏条目。
      }
    }
    return result;
  }

  Future<void> _trimMessages(int convId) async {
    final rows = await _db
        .customSelect(
          'SELECT msg_id FROM cached_messages WHERE conv_id = ? '
          'ORDER BY cached_at DESC LIMIT -1 OFFSET ?',
          variables: [
            Variable.withInt(convId),
            Variable.withInt(_maxMessagesPerConv),
          ],
        )
        .get();
    for (final row in rows) {
      await _db.customStatement(
        'DELETE FROM cached_messages WHERE conv_id = ? AND msg_id = ?',
        [convId, row.data['msg_id']],
      );
    }
  }

  /// 清除全部缓存(登出/清理)。
  @override
  Future<void> clear() async {
    await _db.customStatement('DELETE FROM cached_topics');
    await _db.customStatement('DELETE FROM cached_conversations');
    await _db.customStatement('DELETE FROM cached_messages');
  }

  @override
  Future<void> close() => _db.close();
}
