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
  CampusSnapshotStore(this._db);

  static const schemaVersion = 1;
  final AppDatabase _db;

  Future<CampusSnapshot?> read(
    CampusCacheScope scope, [
    String? bindingRevision,
  ]) async {
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
      final payload =
          jsonDecode(row['payload']! as String) as Map<String, dynamic>;
      final data = <String, CampusDataset>{
        for (final entry in payload.entries)
          entry.key: CampusDataset.fromJson(
            entry.value as Map<String, dynamic>,
          ),
      };
      if (!data.keys.toSet().containsAll(campusPersistentKeys) ||
          data.keys.any((key) => !campusPersistentKeys.contains(key))) {
        throw const FormatException();
      }
      return CampusSnapshot(
        bindingRevision: row['binding_revision']! as String,
        committedAt: DateTime.parse(row['committed_at']! as String),
        data: data,
      );
    } catch (_) {
      await clearScope(scope);
      return null;
    }
  }

  Future<CampusSnapshot> write(
    CampusCacheScope scope,
    String bindingRevision,
    Map<String, CampusDataset> data, {
    DateTime? committedAt,
  }) async {
    if (bindingRevision.isEmpty ||
        !data.keys.toSet().containsAll(campusPersistentKeys)) {
      throw ArgumentError('A complete campus snapshot is required.');
    }
    final selected = <String, CampusDataset>{
      for (final key in campusPersistentKeys)
        if (const {'ready', 'empty'}.contains(data[key]!.status))
          key: data[key]!,
    };
    if (selected.length != campusPersistentKeys.length) {
      throw ArgumentError('Campus snapshot datasets must be ready or empty.');
    }
    final now = (committedAt ?? DateTime.now()).toUtc();
    final payload = jsonEncode({
      for (final entry in selected.entries)
        entry.key: _datasetToJson(entry.value),
    });
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
        now.toIso8601String(),
      ],
    );
    return CampusSnapshot(
      bindingRevision: bindingRevision,
      committedAt: now,
      data: selected,
    );
  }

  Future<void> clearScope(CampusCacheScope scope) => _db.customStatement(
    'DELETE FROM campus_snapshots WHERE site = ? AND account_id = ?',
    [scope.site, scope.accountId],
  );

  Future<void> clear() => _db.customStatement('DELETE FROM campus_snapshots');
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
