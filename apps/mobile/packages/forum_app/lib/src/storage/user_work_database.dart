import 'dart:convert';

import 'package:drift/drift.dart';
import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'private_database.dart';

final userWorkDatabaseProvider = Provider<UserWorkDatabase>(
  (ref) => UserWorkDatabase.instance,
);

class UserWorkUsage {
  const UserWorkUsage({
    required this.draftCount,
    required this.planCount,
    required this.recoveryPlanCount,
    required this.legacyPlanCount,
    required this.unsyncedPlanCount,
    required this.payloadBytes,
  });
  final int draftCount,
      planCount,
      recoveryPlanCount,
      legacyPlanCount,
      unsyncedPlanCount,
      payloadBytes;
}

/// Authoritative local work, never subject to cache eviction. Nullable payloads
/// are durable tombstones: an old preferences copy cannot resurrect a deletion.
class UserWorkDatabase extends GeneratedDatabase {
  UserWorkDatabase([QueryExecutor? executor])
    : super(executor ?? openPrivateDatabase(name: 'user_work'));

  static UserWorkDatabase? _instance;
  static UserWorkDatabase get instance => _instance ??= UserWorkDatabase();

  /// Test executables replace only this factory's instance, not SQLite itself.
  static void setInstanceForTesting(UserWorkDatabase? database) =>
      _instance = database;

  @override
  int get schemaVersion => 1;
  @override
  Iterable<TableInfo<Table, dynamic>> get allTables => const [];
  @override
  MigrationStrategy get migration => MigrationStrategy(
    onCreate: (_) async {
      await customStatement(
        'CREATE TABLE work_records (scope TEXT NOT NULL, domain TEXT NOT NULL, entry_key TEXT NOT NULL, payload TEXT, PRIMARY KEY(scope, domain, entry_key))',
      );
      await customStatement(
        'CREATE TABLE work_meta (entry_key TEXT PRIMARY KEY, value TEXT NOT NULL)',
      );
    },
  );

  Future<void>? _tail;
  int _generation = 0;
  bool _migrated = false;
  int get generation => _generation;

  Future<T> _queue<T>(Future<T> Function() action, {int? generation}) {
    final expected = generation ?? _generation;
    Future<T> run() async {
      _check(expected);
      return action();
    }

    final next = _tail?.then((_) => run()) ?? Future<T>.sync(run);
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

  void _check(int generation, [bool Function()? isCurrent]) {
    if (generation != _generation || (isCurrent != null && !isCurrent())) {
      throw StateError('Local work operation invalidated');
    }
  }

  bool _legacyKey(String key) =>
      RegExp(
        r'^yourtj:writing:v1:.+:[0-9]+:(draft:.+|history)$',
      ).hasMatch(key) ||
      key.startsWith('pk.');

  Future<void> _removeLegacy(
    SharedPreferences prefs,
    Iterable<String> keys,
  ) async {
    for (final key in keys) {
      if (prefs.containsKey(key) && !await prefs.remove(key)) {
        // Preferences mutate their memory cache even when the native write fails.
        await prefs.reload();
        if (prefs.containsKey(key)) {
          throw StateError('Legacy local work cleanup failed');
        }
      }
    }
  }

  Future<void> _migrate(int generation) async {
    if (_migrated) return;
    final prefs = await SharedPreferences.getInstance();
    final keys = prefs.getKeys().where(_legacyKey).toList();
    await transaction(() async {
      final disabled = await customSelect(
        "SELECT value FROM work_meta WHERE entry_key='legacy_disabled'",
      ).getSingleOrNull();
      if (disabled == null) {
        for (final legacyKey in keys) {
          final Object? legacy = prefs.get(legacyKey);
          if (legacy == null) continue;
          String scope, domain, key, payload;
          final draft = RegExp(
            r'^yourtj:writing:v1:(.+:[0-9]+):draft:(.+)$',
          ).firstMatch(legacyKey);
          final history = RegExp(
            r'^yourtj:writing:v1:(.+:[0-9]+):history$',
          ).firstMatch(legacyKey);
          if (draft != null) {
            scope = draft[1]!;
            domain = 'draft';
            key = draft[2]!;
          } else if (history != null) {
            scope = history[1]!;
            domain = 'history';
            key = 'search';
          } else if (legacyKey.startsWith('pk.')) {
            // Legacy plan IDs have no origin. Never assign them to today's site.
            scope = 'legacy-unassigned';
            domain = 'schedule-legacy';
            key = legacyKey;
          } else {
            continue;
          }
          payload = legacy is String ? legacy : jsonEncode(legacy);
          final vars = [Variable(scope), Variable(domain), Variable(key)];
          final existing = await customSelect(
            'SELECT payload FROM work_records WHERE scope=? AND domain=? AND entry_key=?',
            variables: vars,
          ).getSingleOrNull();
          if (existing != null) continue; // authority includes deleted rows
          await customStatement(
            'INSERT INTO work_records(scope,domain,entry_key,payload) VALUES(?,?,?,?)',
            [scope, domain, key, payload],
          );
          final verified = await customSelect(
            'SELECT payload FROM work_records WHERE scope=? AND domain=? AND entry_key=?',
            variables: vars,
          ).getSingle();
          if (verified.read<String>('payload') != payload) {
            throw StateError('Local work migration verification failed');
          }
        }
      }
      _check(generation);
    });
    _check(generation);
    await _removeLegacy(prefs, keys);
    _check(generation);
    _migrated = true;
  }

  Future<Map<String, String>> readDomain(String scope, String domain) {
    final lease = _generation;
    return _queue(() async {
      await _migrate(lease);
      final rows = await customSelect(
        'SELECT entry_key,payload FROM work_records WHERE scope=? AND domain=? AND payload IS NOT NULL',
        variables: [Variable(scope), Variable(domain)],
      ).get();
      _check(lease);
      return {
        for (final row in rows)
          row.read<String>('entry_key'): row.read<String>('payload'),
      };
    }, generation: lease);
  }

  Future<void> writeBatch(
    String scope,
    String domain,
    Map<String, String?> entries, {
    bool Function()? isCurrent,
    int? generation,
  }) {
    final lease = generation ?? _generation;
    final copy = Map<String, String?>.of(entries);
    return _queue(() async {
      await _migrate(lease);
      await transaction(() async {
        _check(lease, isCurrent);
        for (final entry in copy.entries) {
          await customStatement(
            'INSERT INTO work_records(scope,domain,entry_key,payload) VALUES(?,?,?,?) ON CONFLICT(scope,domain,entry_key) DO UPDATE SET payload=excluded.payload',
            [scope, domain, entry.key, entry.value],
          );
        }
        _check(lease, isCurrent);
      });
    }, generation: lease);
  }

  Future<bool> writeIfAbsent(
    String scope,
    String domain,
    String key,
    String payload, {
    bool Function()? isCurrent,
    int? generation,
  }) {
    final lease = generation ?? _generation;
    return _queue(() async {
      await _migrate(lease);
      return transaction(() async {
        _check(lease, isCurrent);
        final previous = await customSelect(
          'SELECT payload FROM work_records WHERE scope=? AND domain=? AND entry_key=?',
          variables: [Variable(scope), Variable(domain), Variable(key)],
        ).getSingleOrNull();
        if (previous?.readNullable<String>('payload') != null) return false;
        await customStatement(
          'INSERT INTO work_records(scope,domain,entry_key,payload) VALUES(?,?,?,?) ON CONFLICT(scope,domain,entry_key) DO UPDATE SET payload=excluded.payload',
          [scope, domain, key, payload],
        );
        _check(lease, isCurrent);
        return true;
      });
    }, generation: lease);
  }

  Future<void> updateValue(
    String scope,
    String domain,
    String key,
    String? Function(String?) update, {
    int? generation,
  }) {
    final lease = generation ?? _generation;
    return _queue(() async {
      await _migrate(lease);
      await transaction(() async {
        final row = await customSelect(
          'SELECT payload FROM work_records WHERE scope=? AND domain=? AND entry_key=?',
          variables: [Variable(scope), Variable(domain), Variable(key)],
        ).getSingleOrNull();
        final next = update(row?.readNullable<String>('payload'));
        await customStatement(
          'INSERT INTO work_records(scope,domain,entry_key,payload) VALUES(?,?,?,?) ON CONFLICT(scope,domain,entry_key) DO UPDATE SET payload=excluded.payload',
          [scope, domain, key, next],
        );
        _check(lease);
      });
    }, generation: lease);
  }

  Future<void> transferScope(
    String source,
    String target,
    String domain,
    Map<String, String> entries, {
    required int generation,
    bool Function()? isCurrent,
  }) {
    final copy = Map<String, String>.of(entries);
    return _queue(() async {
      await _migrate(generation);
      await transaction(() async {
        _check(generation, isCurrent);
        for (final e in copy.entries) {
          await customStatement(
            'INSERT INTO work_records(scope,domain,entry_key,payload) VALUES(?,?,?,?) ON CONFLICT(scope,domain,entry_key) DO UPDATE SET payload=excluded.payload',
            [target, domain, e.key, e.value],
          );
        }
        if (source != target) {
          await customStatement(
            'UPDATE work_records SET payload=NULL WHERE scope=? AND domain=?',
            [source, domain],
          );
        }
        _check(generation, isCurrent);
      });
    }, generation: generation);
  }

  /// Explicit recovery is a verified move. Source rows survive every failure.
  Future<void> recoverLegacySchedule(
    String target,
    int owner, {
    required int generation,
    bool Function()? isCurrent,
  }) => _queue(() async {
    await _migrate(generation);
    await transaction(() async {
      _check(generation, isCurrent);
      final rows = await customSelect(
        "SELECT entry_key,payload FROM work_records WHERE scope='legacy-unassigned' AND domain='schedule-legacy' AND payload IS NOT NULL",
      ).get();
      final legacy = {
        for (final row in rows)
          row.read<String>('entry_key'): row.read<String>('payload'),
      };
      if (legacy.isEmpty) return;
      final oldOwner = int.tryParse(legacy['pk.syncOwner'] ?? '');
      if (oldOwner != null && oldOwner != 0 && oldOwner != owner) {
        throw StateError('Legacy plans belong to another account');
      }
      final existingRows = await customSelect(
        "SELECT entry_key,payload FROM work_records WHERE scope=? AND domain='schedule' AND payload IS NOT NULL",
        variables: [Variable(target)],
      ).get();
      final existing = {
        for (final row in existingRows)
          row.read<String>('entry_key'): row.read<String>('payload'),
      };
      final oldPlans = jsonDecode(legacy['pk.plans'] ?? '[]') as List;
      final currentPlans = jsonDecode(existing['pk.plans'] ?? '[]') as List;
      final currentCache =
          jsonDecode(existing['pk.planSync.v3.$owner'] ?? '{}') as Map;
      bool placeholder(dynamic plan) =>
          plan is Map &&
          currentCache['placeholderID'] == plan['id'] &&
          currentCache['placeholderKey'] ==
              schedulePlanKey(PkPlan.fromJson(Map<String, dynamic>.from(plan)));
      final merged = <String, dynamic>{};
      for (final plan in [
        ...(currentPlans.length == 1 && placeholder(currentPlans.single)
            ? []
            : currentPlans),
        ...oldPlans,
      ]) {
        final id = (plan as Map)['id'] as String;
        if (merged.containsKey(id) &&
            jsonEncode(merged[id]) != jsonEncode(plan)) {
          throw StateError('A plan with this identity already exists');
        }
        merged[id] = plan;
      }
      if (merged.length > 10) throw StateError('Too many plans to restore');
      final next = {
        ...Map<String, String>.fromEntries(
          legacy.entries.where((e) => !e.key.startsWith('pk.planSync.v3.')),
        ),
        ...existing,
        'pk.plans': jsonEncode(merged.values.toList()),
        'pk.syncDirty': '1',
      };
      next.remove('pk.syncedAt');
      if (owner == 0) {
        next.remove('pk.syncOwner');
      } else {
        next['pk.syncOwner'] = '$owner';
      }
      final oldCache =
          jsonDecode(legacy['pk.planSync.v3.$owner'] ?? '{}') as Map;
      final newCache =
          jsonDecode(existing['pk.planSync.v3.$owner'] ?? '{}') as Map;
      final oldDrafts = Map<String, dynamic>.from(
        oldCache['drafts'] as Map? ?? {},
      );
      final newDrafts = Map<String, dynamic>.from(
        newCache['drafts'] as Map? ?? {},
      );
      for (final entry in oldDrafts.entries) {
        if (newDrafts.containsKey(entry.key) &&
            jsonEncode(newDrafts[entry.key]) != jsonEncode(entry.value)) {
          throw StateError(
            'A recovery draft with this identity already exists',
          );
        }
        newDrafts[entry.key] = entry.value;
      }
      if (newDrafts.isNotEmpty) {
        next['pk.planSync.v3.$owner'] = jsonEncode({
          'bases': newCache['bases'] ?? {},
          'plans': merged.values.toList(),
          'drafts': newDrafts,
        });
      }
      Future<void> put(String domain, String key, String value) async {
        await customStatement(
          'INSERT INTO work_records(scope,domain,entry_key,payload) VALUES(?,?,?,?) ON CONFLICT(scope,domain,entry_key) DO UPDATE SET payload=excluded.payload',
          [target, domain, key, value],
        );
        final check = await customSelect(
          'SELECT payload FROM work_records WHERE scope=? AND domain=? AND entry_key=?',
          variables: [Variable(target), Variable(domain), Variable(key)],
        ).getSingle();
        if (check.read<String>('payload') != value) {
          throw StateError('Recovery verification failed');
        }
      }

      for (final entry in next.entries) {
        await put('schedule', entry.key, entry.value);
      }
      // Keep a verbatim recovery archive, including other accounts' old caches;
      // their unknown-origin ancestors are never used by synchronization.
      for (final entry in legacy.entries) {
        await put('schedule-recovery', entry.key, entry.value);
      }
      await customStatement(
        "UPDATE work_records SET payload=NULL WHERE scope='legacy-unassigned' AND domain='schedule-legacy'",
      );
      _check(generation, isCurrent);
    });
  }, generation: generation);

  Future<void> clearScope(String scope, {String? domain}) {
    final lease = _generation;
    return _queue(() async {
      await _migrate(lease);
      await transaction(() async {
        await customStatement(
          'UPDATE work_records SET payload=NULL WHERE scope=?${domain == null ? '' : ' AND domain=?'}',
          [scope, ?domain],
        );
        _check(lease);
      });
    }, generation: lease);
  }

  /// The reset fence advances synchronously; earlier queued saves are rejected.
  /// Keep the DB/key and tombstones. A partial preferences cleanup is retryable.
  Future<void> clearAll() {
    final lease = ++_generation;
    _migrated = false;
    return _queue(() async {
      await transaction(() async {
        await customStatement('UPDATE work_records SET payload=NULL');
        await customStatement(
          "INSERT INTO work_meta(entry_key,value) VALUES('legacy_disabled','1') ON CONFLICT(entry_key) DO UPDATE SET value='1'",
        );
        _check(lease);
      });
      final prefs = await SharedPreferences.getInstance();
      await _removeLegacy(prefs, prefs.getKeys().where(_legacyKey).toList());
      // Explicit reset also releases payload pages; keep only authority rows.
      await customStatement('VACUUM');
      await customStatement('PRAGMA wal_checkpoint(TRUNCATE)');
      _check(lease);
      _migrated = true;
    }, generation: lease);
  }

  Future<UserWorkUsage> usage() {
    final lease = _generation;
    return _queue(() async {
      await _migrate(lease);
      final rows = await customSelect(
        'SELECT scope,domain,entry_key,payload FROM work_records WHERE payload IS NOT NULL',
      ).get();
      var drafts = 0,
          plans = 0,
          recovery = 0,
          legacyPlans = 0,
          unsynced = 0,
          bytes = 0;
      final scopes = <String, Map<String, String>>{};
      for (final row in rows) {
        final payload = row.read<String>('payload'),
            domain = row.read<String>('domain');
        bytes += utf8.encode(payload).length;
        if (domain == 'draft') drafts++;
        if (domain == 'schedule' || domain == 'schedule-legacy') {
          scopes.putIfAbsent(
            '${row.read<String>('scope')}:$domain',
            () => {},
          )[row.read<String>('entry_key')] = payload;
        }
      }
      for (final scopeEntry in scopes.entries) {
        final values = scopeEntry.value;
        try {
          final items = jsonDecode(values['pk.plans'] ?? '[]') as List;
          plans += items.length;
          if (scopeEntry.key.endsWith(':schedule-legacy')) {
            legacyPlans += items.length;
          }
          final owner = values['pk.syncOwner'];
          final cache =
              jsonDecode(values['pk.planSync.v3.$owner'] ?? '{}') as Map;
          recovery += (cache['drafts'] as Map? ?? {}).length;
          final bases = cache['bases'] as Map? ?? {};
          unsynced += items
              .where(
                (p) =>
                    jsonEncode(p) !=
                    jsonEncode((bases[(p as Map)['id']] as Map?)?['plan']),
              )
              .length;
          // Quarantined ancestors/recovery copies may belong to several owners.
          for (final entry in values.entries.where(
            (e) =>
                e.key.startsWith('pk.planSync.v3.') &&
                e.key != 'pk.planSync.v3.$owner',
          )) {
            recovery +=
                (jsonDecode(entry.value)['drafts'] as Map? ?? {}).length;
          }
        } catch (_) {
          /* Raw damaged work remains counted in bytes for recovery. */
        }
      }
      _check(lease);
      return UserWorkUsage(
        draftCount: drafts,
        planCount: plans,
        recoveryPlanCount: recovery,
        legacyPlanCount: legacyPlans,
        unsyncedPlanCount: unsynced,
        payloadBytes: bytes,
      );
    }, generation: lease);
  }

  @override
  Future<void> close() async {
    await _tail;
    return super.close();
  }
}
