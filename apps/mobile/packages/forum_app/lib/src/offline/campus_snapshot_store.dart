import 'dart:convert';

import 'package:core/core.dart';
import 'package:drift/drift.dart';

import 'drift_cache.dart';

const campusPersistentKeys = <String>{
  'profile',
  'calendar',
  'timetable',
  'today',
};

class CampusCacheScope {
  const CampusCacheScope({required this.site, required this.accountId});

  final String site;
  final int accountId;
}

class CampusSnapshot {
  const CampusSnapshot({
    required this.bindingRevision,
    required this.committedAt,
    required this.data,
  });

  final String bindingRevision;
  final DateTime committedAt;
  final Map<String, CampusDataset> data;
}

/// One JSON document is replaced by one SQLite statement, so readers never see
/// a partial campus refresh. Credentials and non-whitelisted datasets never
/// enter this store.
class CampusSnapshotStore {
  CampusSnapshotStore(this._db, {DateTime Function()? now})
    : now = now ?? DateTime.now;

  static const schemaVersion = 1;
  static const maxAge = Duration(days: 30);
  static const maxBytes = 1024 * 1024;
  static const maxScopes = 4;
  final AppDatabase _db;
  final DateTime Function() now;
  int generation = 0;
  Future<void> _tail = Future.value();

  void invalidate() => generation++;

  Future<T> _serialize<T>(Future<T> Function() operation) {
    final result = _tail.then((_) => operation());
    _tail = result.then<void>((_) {}, onError: (Object _, StackTrace _) {});
    return result;
  }

  Future<CampusSnapshot?> read(
    CampusCacheScope scope, [
    String? bindingRevision,
  ]) => _serialize(() => _read(scope, bindingRevision));

  Future<CampusSnapshot?> _read(
    CampusCacheScope scope,
    String? bindingRevision,
  ) async {
    final rows = await _db
        .customSelect(
          'SELECT binding_revision, schema_version, payload, committed_at '
          'FROM campus_snapshots WHERE site = ? AND account_id = ?'
          '${bindingRevision == null ? '' : ' AND binding_revision = ?'}',
          variables: [
            Variable.withString(scope.site),
            Variable.withInt(scope.accountId),
            if (bindingRevision != null) Variable.withString(bindingRevision),
          ],
        )
        .get();
    if (rows.isEmpty) return null;
    try {
      final row = rows.single.data;
      if (row['schema_version'] != schemaVersion) throw const FormatException();
      final encoded = row['payload']! as String;
      final committedAt = DateTime.parse(row['committed_at']! as String);
      if (utf8.encode(encoded).length > maxBytes ||
          now().difference(committedAt) > maxAge ||
          committedAt.isAfter(now().add(const Duration(minutes: 5)))) {
        throw const FormatException();
      }
      final payload = jsonDecode(encoded) as Map<String, dynamic>;
      final data = <String, CampusDataset>{
        for (final entry in payload.entries)
          entry.key: CampusDataset.fromJson(
            entry.value as Map<String, dynamic>,
          ),
      };
      if (data.entries.any(
            (e) =>
                e.key != e.value.key ||
                !const {'ready', 'empty'}.contains(e.value.status),
          ) ||
          !data.keys.toSet().containsAll(campusPersistentKeys) ||
          data.keys.any((key) => !campusPersistentKeys.contains(key))) {
        throw const FormatException();
      }
      return CampusSnapshot(
        bindingRevision: row['binding_revision']! as String,
        committedAt: committedAt,
        data: data,
      );
    } catch (_) {
      // Reads share the write queue: corrupt-data cleanup cannot erase a newer
      // completed refresh while validating an older row.
      await _db.customStatement(
        'DELETE FROM campus_snapshots WHERE site = ? AND account_id = ?',
        [scope.site, scope.accountId],
      );
      return null;
    }
  }

  Future<CampusSnapshot> write(
    CampusCacheScope scope,
    String bindingRevision,
    Map<String, CampusDataset> data, {
    DateTime? committedAt,
    int? expectedGeneration,
  }) {
    final fence = expectedGeneration ?? generation;
    return _serialize(() async {
      if (fence != generation) throw const CampusSnapshotSuperseded();
      if (bindingRevision.isEmpty ||
          !data.keys.toSet().containsAll(campusPersistentKeys)) {
        throw ArgumentError('A complete campus snapshot is required.');
      }
      final selected = <String, CampusDataset>{
        for (final key in campusPersistentKeys)
          if (data[key]!.key == key &&
              const {'ready', 'empty'}.contains(data[key]!.status))
            key: data[key]!,
      };
      if (selected.length != campusPersistentKeys.length) {
        throw ArgumentError('Campus snapshot datasets must be ready or empty.');
      }
      final timestamp = (committedAt ?? now()).toUtc();
      final payload = jsonEncode({
        for (final entry in selected.entries)
          entry.key: _datasetToJson(entry.value),
      });
      if (utf8.encode(payload).length > maxBytes) {
        throw StateError('Campus snapshot exceeds its storage limit');
      }
      await _db.transaction(() async {
        if (fence != generation) throw const CampusSnapshotSuperseded();
        await _db.customStatement(
          'INSERT OR REPLACE INTO campus_snapshots '
          '(site, account_id, binding_revision, schema_version, payload, committed_at) '
          'VALUES (?, ?, ?, ?, ?, ?)',
          [
            scope.site,
            scope.accountId,
            bindingRevision,
            schemaVersion,
            payload,
            timestamp.toIso8601String(),
          ],
        );
        await _db.customStatement(
          'DELETE FROM campus_snapshots WHERE rowid IN '
          '(SELECT rowid FROM campus_snapshots ORDER BY committed_at DESC, rowid DESC LIMIT -1 OFFSET ?)',
          [maxScopes],
        );
      });
      if (fence != generation) throw const CampusSnapshotSuperseded();
      return CampusSnapshot(
        bindingRevision: bindingRevision,
        committedAt: timestamp,
        data: selected,
      );
    });
  }

  Future<void> clearScope(CampusCacheScope scope) {
    invalidate();
    return _serialize(
      () => _db.customStatement(
        'DELETE FROM campus_snapshots WHERE site = ? AND account_id = ?',
        [scope.site, scope.accountId],
      ),
    );
  }

  Future<void> clear() {
    invalidate();
    return _serialize(
      () => _db.customStatement('DELETE FROM campus_snapshots'),
    );
  }
}

class CampusSnapshotSuperseded implements Exception {
  const CampusSnapshotSuperseded();
}

Map<String, Object?> _datasetToJson(CampusDataset data) => {
  'key': data.key,
  'status': data.status,
  'updatedAt': data.updatedAt,
  'metrics': [
    for (final metric in data.metrics)
      {'label': metric.label, 'value': metric.value, 'unit': metric.unit},
  ],
  'columns': data.columns,
  'rows': data.rows,
  'events': [
    for (final event in data.events)
      {
        'name': event.name,
        'teacher': event.teacher,
        'room': event.room,
        'campus': event.campus,
        'day': event.day,
        'start': event.start,
        'end': event.end,
        'weeks': event.weeks,
        'credits': event.credits,
      },
  ],
  'series': [
    for (final point in data.series)
      {'label': point.label, 'value': point.value},
  ],
  // Message summaries/bodies are deliberately never serialized.
  'messages': const <Object>[],
  'teachingDay': data.teachingDay == null
      ? null
      : {
          'date': data.teachingDay!.date,
          'sourceDate': data.teachingDay!.sourceDate,
          'kind': data.teachingDay!.kind,
          'label': data.teachingDay!.label,
          'sectionCount': data.teachingDay!.sectionCount,
        },
};
