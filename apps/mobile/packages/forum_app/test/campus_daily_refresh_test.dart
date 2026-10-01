import 'dart:async';

import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/offline/campus_snapshot_store.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/campus/campus_memory_cache.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';

import 'campus_memory_cache_test.dart' show ControlledCampusRepository, failure;
import 'campus_snapshot_experience_test.dart' show dataAt, scope;
import 'campus_widget_test.dart' show RecordingWidgetBridge;
import 'fixtures/campus_fixtures.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late DateTime now;
  late DateTime previous;
  late AppDatabase db;
  late CampusSnapshotStore store;
  late ControlledCampusRepository repo;
  late RecordingWidgetBridge bridge;

  CampusController controller() {
    final cache = CampusMemoryCache(now: () => now);
    final value = CampusController(
      repo,
      cache: cache,
      persistentStore: store,
      widgetBridge: bridge,
      scope: scope,
    );
    addTearDown(value.dispose);
    addTearDown(cache.dispose);
    return value;
  }

  // Drain the queued snapshot read before asserting on the status/data request.
  Future<void> started() async {
    await store.read(scope);
    await Future<void>.delayed(Duration.zero);
  }

  setUp(() async {
    // Two minutes apart on the same UTC date, across midnight in Shanghai.
    previous = DateTime.utc(2026, 9, 30, 15, 59);
    now = DateTime.utc(2026, 9, 30, 16, 1);
    db = AppDatabase(NativeDatabase.memory());
    store = CampusSnapshotStore(db, now: () => now);
    repo = ControlledCampusRepository()..now = () => now;
    bridge = RecordingWidgetBridge();
    await store.write(
      scope,
      testBinding.revision,
      dataAt(previous),
      committedAt: previous,
    );
  });
  tearDown(() => db.close());

  test(
    'daily entry refreshes once across restart and manual refresh still works',
    () async {
      final first = controller();
      await first.enterTab('today');
      expect(
        repo.requested,
        unorderedEquals({
          ...campusPersistentKeys,
          'messages',
          'calendar-rules',
        }),
      );
      expect(first.state.data['today']!.teachingDay!.date, '2026-10-01');
      expect((await store.read(scope))!.committedAt, now);
      expect(bridge.writes, 1);

      repo.requested.clear();
      await first.enterTab('today');
      await first.loadTab('timetable');
      // A new controller and store instance simulate a fresh process's read.
      store = CampusSnapshotStore(db, now: () => now);
      final restarted = controller();
      await restarted.enterTab('today');
      expect(repo.requested, isEmpty);
      expect(bridge.writes, 1);

      await restarted.refresh();
      expect(repo.requested.where((key) => key == 'timetable'), hasLength(1));
      expect(bridge.writes, 2);
    },
  );

  test(
    'daily refresh waits for verified binding and coalesces other refreshes',
    () async {
      repo.pendingStatus = Completer();
      final value = controller();
      final first = value.enterTab('today');
      await started();
      expect(repo.statusCalls, 1);
      expect(repo.requested, isEmpty);
      final repeated = value.enterTab('today');
      final manual = value.refresh();
      repo.pendingStatus!.complete(testStatus);
      await Future.wait([first, repeated, manual]);
      expect(repo.statusCalls, 1);
      expect(repo.requested.where((key) => key == 'today'), hasLength(1));
      expect(bridge.writes, 1);
    },
  );

  test(
    'daily failure preserves snapshot and widget and allows a later entry to retry',
    () async {
      repo.errors['today'] = failure;
      final value = controller();
      await value.enterTab('today');
      expect(value.state.errors['today'], failure);
      expect(value.state.data['today'], isNull);
      expect((await store.read(scope))!.committedAt, previous);
      expect(bridge.writes, 0);
      expect(bridge.clears, 0);
      repo.requested.clear();
      await value.refreshVisible();
      expect(repo.requested, isEmpty);

      repo.errors.clear();
      await value.enterTab('today');
      expect(value.state.data['today']!.teachingDay!.date, '2026-10-01');
      expect((await store.read(scope))!.committedAt, now);
      expect(bridge.writes, 1);
    },
  );

  test(
    'the visible clock only invalidates; the next entry refreshes the new day',
    () async {
      now = previous;
      final value = controller();
      await value.enterTab('today');
      expect(repo.requested, isEmpty);
      now = now.add(const Duration(minutes: 2));
      await value.refreshVisible();
      expect(value.state.data['today'], isNull);
      expect(repo.requested, isEmpty);
      await value.enterTab('today');
      expect(value.state.data['today']!.teachingDay!.date, '2026-10-01');
      expect(bridge.writes, 1);
    },
  );

  test(
    'daily reads crossing midnight cannot mark yesterday as refreshed today',
    () async {
      repo.pending['timetable'] = Completer();
      final value = controller();
      final refreshing = value.enterTab('today');
      await started();
      expect(repo.requested, contains('timetable'));
      final requestedAt = now;
      now = now.add(const Duration(days: 1));
      repo.pending['timetable']!.complete(
        campusFixture('timetable', now: requestedAt),
      );
      await refreshing;
      expect((await store.read(scope))!.committedAt, previous);
      expect(bridge.writes, 0);
      repo.pending.clear();
      await value.enterTab('today');
      expect(value.state.data['today']!.teachingDay!.date, '2026-10-02');
      expect((await store.read(scope))!.committedAt, now);
    },
  );

  test(
    'clearing during daily refresh cannot republish the old private data',
    () async {
      repo.pending['profile'] = Completer();
      final value = controller();
      final refreshing = value.enterTab('today');
      await started();
      expect(repo.requested, contains('profile'));
      value.discardLocal();
      await store.clearScope(scope);
      repo.pending['profile']!.complete(campusFixture('profile', now: now));
      await refreshing;
      expect(value.state.data, isEmpty);
      expect(await store.read(scope), isNull);
      expect(bridge.writes, 0);
    },
  );

  test('an expired binding cannot trigger daily school reads', () async {
    repo.current = const CampusStatus(
      enabled: true,
      binding: CampusBinding(
        maskedId: '',
        boundAt: '',
        revision: 'demo-revision',
        needsAuthorization: true,
      ),
      candidate: null,
    );
    final value = controller();
    await value.enterTab('today');
    expect(repo.requested, isEmpty);
    expect(value.state.data, isEmpty);
    expect(await store.read(scope), isNull);
    expect(bridge.writes, 0);
  });
}
