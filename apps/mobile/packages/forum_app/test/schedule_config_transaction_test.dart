import 'dart:async';
import 'dart:convert';
import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/storage/user_work_database.dart';
import 'schedule_page_test.dart' show detail;

class ScheduleConfigTestDatabase extends UserWorkDatabase {
  ScheduleConfigTestDatabase() : super(NativeDatabase.memory());
  bool reject = false;
  Completer<void>? pending, entered;
  final batches = <Map<String, String?>>[];
  @override
  Future<void> writeBatch(
    String scope,
    String domain,
    Map<String, String?> entries, {
    bool Function()? isCurrent,
    int? generation,
  }) async {
    if (reject) throw StateError('simulated durable write failure');
    if (pending != null) {
      if (entered != null && !entered!.isCompleted) entered!.complete();
      await pending!.future;
    }
    await super.writeBatch(
      scope,
      domain,
      entries,
      isCurrent: isCurrent,
      generation: generation,
    );
    if (domain == 'schedule') batches.add(Map.of(entries));
  }
}

const _site = 'https://schedule.test';
final _oldSelection = PkMajorSelection(
  calendarId: 119,
  grade: 2025,
  major: 'm1',
  majorName: 'Original major',
);
final _newSelection = PkMajorSelection(
  calendarId: 120,
  grade: 2024,
  major: 'm2',
  majorName: 'New major',
);
Future<void> _select(
  ScheduleStoreNotifier store,
  PkMajorSelection selection,
) async {
  await store.setMajorSelection(selection);
  await store.flush;
}

Future<ScheduleStoreNotifier> seedScheduleConfig(
  ScheduleConfigTestDatabase database,
) async {
  final store = ScheduleStoreNotifier(database: database, site: _site);
  await store.ready;
  await _select(store, _oldSelection);
  store.selectClass(detail('X.01'), 'Existing course X');
  store.addCustomEvent(
    label: 'First plan block',
    day: 1,
    sections: [1],
    weeks: [1, 2],
  );
  store.createPlan();
  store.selectClass(detail('Y.01'), 'Existing course Y');
  store.addCustomEvent(
    label: 'Second plan block',
    day: 2,
    sections: [2],
    weeks: [1, 2],
  );
  await store.flush;
  return store;
}

String _plans(ScheduleStoreNotifier store) =>
    jsonEncode(store.state.plans.map((p) => p.toJson()).toList());
void main() {
  test(
    'choosing identical configuration does not clear or persist any plan',
    () async {
      final database = ScheduleConfigTestDatabase();
      addTearDown(database.close);
      final store = await seedScheduleConfig(database);
      addTearDown(store.dispose);
      final before = _plans(store);
      final writes = database.batches.length;
      await _select(store, _oldSelection);
      expect(_plans(store), before);
      expect(
        database.batches.length,
        writes,
        reason: 'same value is a true no-op',
      );
    },
  );
  test(
    'new configuration and emptied plans have only one durable snapshot',
    () async {
      final database = ScheduleConfigTestDatabase();
      addTearDown(database.close);
      final store = await seedScheduleConfig(database);
      addTearDown(store.dispose);
      final identities = store.state.plans
          .map((p) => '${p.id}:${p.name}:${p.createdAt}')
          .toList();
      database.batches.clear();
      await _select(store, _newSelection);
      expect(database.batches, hasLength(1));
      final batch = database.batches.single;
      expect(
        jsonDecode(batch[ScheduleStorageKeys.majorSelected]!)['calendarId'],
        120,
      );
      final plans = jsonDecode(batch[ScheduleStorageKeys.plans]!) as List;
      expect(plans, hasLength(2));
      for (final plan in plans) {
        expect(plan['stagedCourses'], isEmpty);
        expect(plan['selectedCourses'], isEmpty);
        expect(plan['customEvents'], isEmpty);
      }
      final reopened = ScheduleStoreNotifier(database: database, site: _site);
      addTearDown(reopened.dispose);
      await reopened.ready;
      expect(reopened.state.majorSelected.calendarId, 120);
      expect(
        reopened.state.plans
            .map((p) => '${p.id}:${p.name}:${p.createdAt}')
            .toList(),
        identities,
      );
      expect(
        reopened.state.plans.every(
          (p) => p.stagedCourses.isEmpty && p.customEvents.isEmpty,
        ),
        isTrue,
      );
    },
  );
  test(
    'failed configuration write retains the complete previous memory and disk state',
    () async {
      final database = ScheduleConfigTestDatabase();
      addTearDown(database.close);
      final store = await seedScheduleConfig(database);
      addTearDown(store.dispose);
      final before = _plans(store);
      final scope = writingScope(_site, 0);
      final disk = await database.readDomain(scope, 'schedule');
      database.reject = true;
      await _select(store, _newSelection);
      expect(store.state.majorSelected.toJson(), _oldSelection.toJson());
      expect(_plans(store), before);
      expect(await database.readDomain(scope, 'schedule'), disk);
      expect(store.persistenceError.value, isNotNull);
    },
  );
  test(
    'configuration is published only after commit and excludes old-context commands',
    () async {
      final database = ScheduleConfigTestDatabase();
      addTearDown(database.close);
      final store = await seedScheduleConfig(database);
      addTearDown(store.dispose);
      final before = _plans(store);
      final id = store.activePlan.id;
      database.batches.clear();
      database.pending = Completer<void>();
      database.entered = Completer<void>();
      final result = store.setMajorSelection(_newSelection);
      await database.entered!.future;
      expect(store.isChangingConfiguration, isTrue);
      expect(store.state.majorSelected.toJson(), _oldSelection.toJson());
      expect(_plans(store), before);
      expect(
        store.selectClass(detail('Z.01'), 'Old-context click').added,
        isFalse,
      );
      store.renamePlan(id, 'Old-context rename');
      expect(_plans(store), before);
      database.pending!.complete();
      expect(await result, isTrue);
      expect(store.isChangingConfiguration, isFalse);
      expect(database.batches, hasLength(1));
      expect(store.activePlan.name, isNot('Old-context rename'));
      expect(
        store.state.plans.every(
          (p) => p.stagedCourses.isEmpty && p.customEvents.isEmpty,
        ),
        isTrue,
      );
    },
  );

  test(
    'a cancelled session fence cannot publish or persist the configuration',
    () async {
      final database = ScheduleConfigTestDatabase();
      addTearDown(database.close);
      final store = await seedScheduleConfig(database);
      addTearDown(store.dispose);
      final before = _plans(store);
      final scope = writingScope(_site, 0);
      final disk = await database.readDomain(scope, 'schedule');
      database.pending = Completer<void>();
      database.entered = Completer<void>();
      var current = true;
      final result = store.setMajorSelection(
        _newSelection,
        canWrite: () => current,
      );
      await database.entered!.future;
      current = false;
      database.pending!.complete();
      expect(await result, isFalse);
      expect(store.isChangingConfiguration, isFalse);
      expect(_plans(store), before);
      expect(store.state.majorSelected.toJson(), _oldSelection.toJson());
      expect(await database.readDomain(scope, 'schedule'), disk);
    },
  );
}
