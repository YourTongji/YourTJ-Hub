import 'dart:async';

import 'package:drift/native.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/offline/campus_snapshot_store.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/campus/campus_memory_cache.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/pages/campus/timetable/campus_timetable.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import 'campus_memory_cache_test.dart' show ControlledCampusRepository;
import 'campus_snapshot_experience_test.dart'
    show appScope, dataAt, scope, testApp;
import 'campus_widget_test.dart' show RecordingWidgetBridge;
import 'fixtures/campus_fixtures.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('controller', () {
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

    Future<void> started() async {
      await store.read(scope);
      await Future<void>.delayed(Duration.zero);
    }

    setUp(() async {
      // Saved late yesterday in Shanghai; the tap comes just after midnight.
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
      'widget entry after background shows the snapshot until fresh reads replace it',
      () async {
        final value = controller()..suspend();
        repo.pendingStatus = Completer();
        final timetable = repo.pending['timetable'] = Completer();
        final entry = value.enterFromWidget('timetable');
        await started();
        final cached = value.state.data['timetable'];
        expect(cached, isNotNull);
        expect(value.state.loading, isFalse);
        expect(value.state.refreshing, isTrue);
        expect(value.withholdDataUntilFresh, isFalse);
        // Yesterday's teaching data stays visible while it is replaced.
        await value.refreshVisible();
        expect(value.state.data['timetable'], same(cached));

        repo.pendingStatus!.complete(testStatus);
        await Future<void>.delayed(Duration.zero);
        await Future<void>.delayed(Duration.zero);
        expect(value.state.data['timetable'], same(cached));
        // Background resume still forces every allowlisted read.
        expect(repo.requested, containsAll(campusPersistentKeys));

        final fresh = campusFixture('timetable', now: now);
        timetable.complete(fresh);
        await entry;
        expect(value.state.data['timetable'], same(fresh));
        expect((await store.read(scope))!.committedAt, now);
        expect(bridge.writes, 1);
      },
    );

    test('plain resume still waits for verified fresh data', () async {
      final value = controller()..suspend();
      repo.pendingStatus = Completer();
      final entry = value.enterTab('timetable');
      await started();
      expect(value.state.data, isEmpty);
      repo.pendingStatus!.complete(testStatus);
      await entry;
      expect(value.state.data['timetable'], isNotNull);
    });

    test(
      'a snapshot from another binding leaves once status disagrees',
      () async {
        await store.write(
          scope,
          'replaced-binding',
          dataAt(previous),
          committedAt: previous,
        );
        final value = controller()..suspend();
        repo.pendingStatus = Completer();
        final timetable = repo.pending['timetable'] = Completer();
        final entry = value.enterFromWidget('timetable');
        await started();
        expect(value.state.data['timetable'], isNotNull);
        repo.pendingStatus!.complete(testStatus);
        for (var i = 0; i < 4; i++) {
          await Future<void>.delayed(Duration.zero);
        }
        expect(value.state.data['timetable'], isNull);
        timetable.complete(campusFixture('timetable', now: now));
        await entry;
        expect(
          (await store.read(scope))!.bindingRevision,
          testBinding.revision,
        );
      },
    );
  });

  group('page', () {
    late AppDatabase db;
    setUp(() => db = AppDatabase(NativeDatabase.memory()));
    tearDown(() => db.close());

    for (final tapFirst in [true, false]) {
      testWidgets(
        'widget tap opens the cached timetable (tap first: $tapFirst)',
        (tester) async {
          final store = CampusSnapshotStore(db);
          final repo = ControlledCampusRepository();
          await store.write(
            appScope,
            testBinding.revision,
            dataAt(DateTime.now()),
          );
          await tester.pumpWidget(
            testApp(db, store, repo, RecordingWidgetBridge()),
          );
          await tester.pumpAndSettle();
          final container = ProviderScope.containerOf(
            tester.element(find.byType(CampusPage)),
          );
          expect(
            tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
            'today',
          );

          tester.binding.handleAppLifecycleStateChanged(
            AppLifecycleState.paused,
          );
          await tester.pumpAndSettle();
          repo.pendingStatus = Completer();
          void tap() =>
              container.read(scheduleWidgetEntryProvider.notifier).state++;
          if (tapFirst) tap();
          tester.binding.handleAppLifecycleStateChanged(
            AppLifecycleState.resumed,
          );
          await tester.pump();
          if (!tapFirst) {
            await tester.pump();
            tap();
          }
          await tester.pump(const Duration(milliseconds: 50));
          await tester.pump(const Duration(milliseconds: 50));
          // Status is still pending: the snapshot, not a skeleton, is shown.
          expect(
            tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
            'timetable',
          );
          expect(find.byType(CampusWeekTimetable), findsOneWidget);

          repo.pendingStatus!.complete(testStatus);
          await tester.pumpAndSettle();
          expect(find.byType(CampusWeekTimetable), findsOneWidget);
          await tester.pumpWidget(const SizedBox());
          await tester.pumpAndSettle();
        },
      );
    }
  });
}
