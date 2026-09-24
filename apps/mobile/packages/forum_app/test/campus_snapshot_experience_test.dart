import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/campus/campus_connection.dart';
import 'package:forum_app/src/pages/settings/campus_cache_clear_tile.dart';
import 'package:forum_app/src/pages/settings/schedule_widget_settings_page.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'campus_native_test.dart' show CampusPkRepository;

import 'package:forum_app/src/offline/campus_snapshot_store.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/campus/campus_memory_cache.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';

import 'campus_memory_cache_test.dart' show ControlledCampusRepository;
import 'campus_widget_test.dart' show RecordingWidgetBridge;
import 'fixtures/campus_fixtures.dart';

const scope = CampusCacheScope(site: 'https://forum.example', accountId: 7);
Map<String, CampusDataset> dataAt(DateTime now) => {
  for (final key in campusPersistentKeys) key: campusFixture(key, now: now),
};

class _PendingRules extends ControlledCampusRepository {
  final rules = Completer<CampusCalendarSettings>();
  final requestedRules = Completer<void>();
  @override
  Future<CampusCalendarSettings> calendarRules({CancelToken? cancelToken}) {
    requestedRules.complete();
    return rules.future;
  }
}

Widget testApp(
  AppDatabase db,
  CampusSnapshotStore store,
  ControlledCampusRepository repo,
  RecordingWidgetBridge bridge, {
  Widget child = const CampusPage(),
  double scale = 1,
}) => ProviderScope(
  overrides: [
    offlineDatabaseProvider.overrideWithValue(db),
    campusSnapshotStoreProvider.overrideWithValue(store),
    campusRepositoryProvider.overrideWithValue(repo),
    scheduleWidgetBridgeProvider.overrideWithValue(bridge),
    pkRepositoryProvider.overrideWithValue(CampusPkRepository()),
    currentUserProvider.overrideWith(
      (ref) async => const CurrentUser(id: 7, username: 'demo'),
    ),
  ],
  child: MaterialApp(
    theme: gfThemeData(Brightness.light),
    locale: const Locale('zh'),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    builder: (context, child) => MediaQuery(
      data: MediaQuery.of(
        context,
      ).copyWith(textScaler: TextScaler.linear(scale)),
      child: child!,
    ),
    home: child,
  ),
);
CampusCacheScope get appScope => CampusCacheScope(
  site: Uri.parse(
    AppConfig.apiBaseUrl.isNotEmpty
        ? AppConfig.apiBaseUrl
        : GfApiClient.defaultBaseUrl,
  ).origin,
  accountId: 7,
);

class _FailingClearStore extends CampusSnapshotStore {
  _FailingClearStore(super.db);
  bool fail = true;
  @override
  Future<void> clear() async {
    if (fail) throw StateError('disk unavailable');
    await super.clear();
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late AppDatabase db;
  late CampusSnapshotStore store;
  setUp(() {
    db = AppDatabase(NativeDatabase.memory());
    store = CampusSnapshotStore(db);
  });
  tearDown(() => db.close());

  for (final locale in ['zh', 'en', 'ja', 'de']) {
    testWidgets(
      'campus privacy explains storage beside clear action ($locale)',
      (tester) async {
        tester.view.physicalSize = const Size(320, 640);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              campusControllerProvider.overrideWith(
                (_) => CampusController(ControlledCampusRepository()),
              ),
            ],
            child: MaterialApp(
              theme: gfThemeData(Brightness.light),
              locale: Locale(locale),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: const TextScaler.linear(2)),
                child: child!,
              ),
              home: const Scaffold(
                body: SingleChildScrollView(
                  child: Padding(
                    padding: EdgeInsets.all(16),
                    child: CampusConnection(),
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final l10n = AppLocalizations.of(
          tester.element(find.byType(CampusConnection)),
        );
        expect(find.text(l10n.campusPrivacy), findsOneWidget);
        await tester.ensureVisible(find.byType(CampusCacheClearTile));
        await tester.tap(find.text(l10n.campusCacheClear));
        await tester.pumpAndSettle();
        expect(find.byType(GfAlertDialog), findsOneWidget);
        expect(find.text(l10n.campusCacheClearDescription), findsWidgets);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
      },
    );
  }

  test(
    'persisted timetable tab reuse does not restart school requests after five minutes',
    () async {
      final now = DateTime.now();
      final fetched = now.subtract(const Duration(minutes: 6));
      await store.write(
        scope,
        testBinding.revision,
        dataAt(now),
        committedAt: fetched,
      );
      final repo = ControlledCampusRepository();
      final cache = CampusMemoryCache(now: () => now);
      final controller = CampusController(
        repo,
        cache: cache,
        persistentStore: store,
        widgetBridge: RecordingWidgetBridge(),
        scope: scope,
      );
      addTearDown(controller.dispose);
      addTearDown(cache.dispose);
      await controller.refresh(reuseCache: true);
      await controller.loadTab('timetable');
      expect(repo.requested, isEmpty);
    },
  );

  test(
    'overlapping manual refreshes issue one status and one dataset batch',
    () async {
      final repo = ControlledCampusRepository()..pendingStatus = Completer();
      final controller = CampusController(repo);
      addTearDown(controller.dispose);
      final first = controller.refresh();
      final second = controller.refresh();
      expect(repo.statusCalls, 1);
      repo.pendingStatus!.complete(testStatus);
      await Future.wait([first, second]);
      expect(
        repo.requested,
        unorderedEquals({...campusPersistentKeys, ...campusTabKeys['today']!}),
      );
    },
  );

  test(
    'disposing during calendar rules cannot resurrect a cleared widget',
    () async {
      final repo = _PendingRules();
      final bridge = RecordingWidgetBridge();
      final controller = CampusController(
        repo,
        persistentStore: store,
        widgetBridge: bridge,
        scope: scope,
      );
      final refreshing = controller.refresh();
      await repo.requestedRules.future;
      controller.dispose();
      await store.clear();
      await bridge.clear();
      repo.rules.complete(
        const CampusCalendarSettings(
          revision: '',
          rules: CampusCalendarRules(holidays: [], moves: []),
        ),
      );
      await refreshing;
      expect(await store.read(scope), isNull);
      expect(bridge.writes, 0);
    },
  );

  test('snapshot past the retention limit is discarded on read', () async {
    final old = DateTime.now().subtract(const Duration(days: 31));
    await store.write(
      scope,
      testBinding.revision,
      dataAt(old),
      committedAt: old,
    );
    expect(await store.read(scope), isNull);
  });
  test(
    'manual refresh clears a replaced binding before failed replacement data',
    () async {
      final repo = ControlledCampusRepository();
      await store.write(scope, testBinding.revision, dataAt(DateTime.now()));
      final controller = CampusController(
        repo,
        persistentStore: store,
        widgetBridge: RecordingWidgetBridge(),
        scope: scope,
      );
      addTearDown(controller.dispose);
      await controller.refresh(reuseCache: true);
      repo.current = const CampusStatus(
        enabled: true,
        candidate: null,
        binding: CampusBinding(
          maskedId: '',
          boundAt: '',
          revision: 'replacement',
          needsAuthorization: false,
        ),
      );
      repo.errors['profile'] = const ApiException(fallbackMessage: 'offline');
      await controller.refresh();
      expect(await store.read(scope), isNull);
      expect(controller.state.snapshot, isNull);
      expect(controller.state.data['profile'], isNull);
    },
  );

  test('old queued writes cannot restore a snapshot after clear', () async {
    final write = store.write(scope, 'binding', dataAt(DateTime.now()));
    final failure = expectLater(
      write,
      throwsA(isA<CampusSnapshotSuperseded>()),
    );
    await store.clear();
    await failure;
    expect(await store.read(scope), isNull);
  });

  test('scope and byte bounds retain the latest complete snapshot', () async {
    final now = DateTime.now();
    for (var i = 1; i <= 5; i++) {
      await store.write(
        CampusCacheScope(site: scope.site, accountId: i),
        'binding',
        dataAt(now),
        committedAt: now.subtract(Duration(minutes: 5 - i)),
      );
    }
    expect(
      await store.read(CampusCacheScope(site: scope.site, accountId: 1)),
      isNull,
    );
    final newest = CampusCacheScope(site: scope.site, accountId: 5);
    expect(await store.read(newest), isNotNull);
    final oversized = dataAt(now);
    oversized['profile'] = CampusDataset.fromJson({
      'key': 'profile',
      'status': 'ready',
      'updatedAt': '',
      'metrics': [
        {
          'label': 'name',
          'value': 'x' * CampusSnapshotStore.maxBytes,
          'unit': '',
        },
      ],
      'columns': [],
      'rows': [],
      'events': [],
      'series': [],
      'messages': [],
    });
    await expectLater(
      store.write(newest, 'replacement', oversized),
      throwsStateError,
    );
    expect((await store.read(newest))!.bindingRevision, 'binding');
    expect(
      (await db.customSelect('SELECT * FROM campus_snapshots').get()).length,
      CampusSnapshotStore.maxScopes,
    );
  });

  test(
    'clearing during a school read cancels the commit and private view',
    () async {
      final repo = ControlledCampusRepository()
        ..pending['profile'] = Completer();
      final bridge = RecordingWidgetBridge();
      final controller = CampusController(
        repo,
        persistentStore: store,
        widgetBridge: bridge,
        scope: scope,
      );
      addTearDown(controller.dispose);
      final refreshing = controller.refresh();
      await Future<void>.delayed(Duration.zero);
      controller.discardLocal();
      await store.clear();
      repo.pending['profile']!.complete(campusFixture('profile'));
      await refreshing;
      expect(controller.state.data, isEmpty);
      expect(await store.read(scope), isNull);
      expect(bridge.writes, 0);
    },
  );

  testWidgets(
    'offline snapshot labels its age and never leaves an idle spinner',
    (tester) async {
      store = CampusSnapshotStore(db);
      final now = DateTime.now();
      await store.write(
        appScope,
        testBinding.revision,
        dataAt(now),
        committedAt: now.subtract(const Duration(minutes: 6)),
      );
      final repo = ControlledCampusRepository()
        ..statusError = const ApiException(fallbackMessage: 'offline');
      await tester.pumpWidget(
        testApp(db, store, repo, RecordingWidgetBridge()),
      );
      await tester.pumpAndSettle();
      expect(find.textContaining('设备快照更新于'), findsOneWidget);
      expect(find.text('连接核验失败，正在显示本设备快照。'), findsOneWidget);
      expect(find.byType(GfLoading), findsNothing);
      expect(repo.requested, isEmpty);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'failed manual refresh keeps usable same-day classes with an error',
    (tester) async {
      store = CampusSnapshotStore(db);
      final now = DateTime.now();
      await store.write(
        appScope,
        testBinding.revision,
        dataAt(now),
        committedAt: now,
      );
      final repo = ControlledCampusRepository()
        ..errors['today'] = const ApiException(fallbackMessage: 'offline');
      await tester.pumpWidget(
        testApp(db, store, repo, RecordingWidgetBridge()),
      );
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('刷新').first);
      await tester.tap(find.text('刷新').first);
      await tester.pumpAndSettle();
      expect(find.text('部分数据更新失败，保留上次可用内容。'), findsOneWidget);
      await tester.ensureVisible(find.text('第四周周二的数学'));
      expect(find.text('第四周周二的数学'), findsOneWidget);
      expect((await store.read(appScope))!.committedAt, now.toUtc());
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'persistent block retry commits the snapshot and republishes its widget',
    (tester) async {
      store = CampusSnapshotStore(db);
      final now = DateTime.now();
      final old = now.subtract(const Duration(minutes: 6));
      final previous = dataAt(now);
      previous['profile'] = const CampusDataset(
        key: 'profile',
        status: 'ready',
        updatedAt: '',
        metrics: [CampusMetric(label: '姓名', value: '旧资料', unit: '')],
        columns: [],
        rows: [],
        events: [],
        series: [],
      );
      await store.write(
        appScope,
        testBinding.revision,
        previous,
        committedAt: old,
      );
      final repo = ControlledCampusRepository()
        ..errors['today'] = const ApiException(fallbackMessage: 'offline');
      final bridge = RecordingWidgetBridge();
      await tester.pumpWidget(testApp(db, store, repo, bridge));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('刷新').first);
      await tester.tap(find.text('刷新').first);
      await tester.pumpAndSettle();
      expect((await store.read(appScope))!.committedAt, old.toUtc());
      expect(bridge.writes, 0);
      repo.errors.clear();
      final retry = find.descendant(
        of: find.byType(GfErrorRetry),
        matching: find.text('重试'),
      );
      await tester.ensureVisible(retry);
      await tester.tap(retry);
      await tester.pumpAndSettle();
      final committed = (await store.read(appScope))!;
      expect(committed.data['profile']!.metrics.single.value, '演示同学');
      expect(committed.committedAt.isAfter(old), isTrue);
      expect(bridge.writes, 1);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      final cache = CampusMemoryCache();
      final restored = CampusController(
        repo,
        cache: cache,
        persistentStore: store,
        scope: appScope,
      );
      await restored.refresh(reuseCache: true);
      expect(restored.state.data['profile']!.metrics.single.value, '演示同学');
      restored.dispose();
      cache.dispose();
    },
  );

  testWidgets(
    'old teaching date shows manual refresh instead of a spinner at large text',
    (tester) async {
      store = CampusSnapshotStore(db);
      tester.view.physicalSize = const Size(320, 640);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final old = DateTime.now().subtract(const Duration(days: 2));
      await store.write(
        appScope,
        testBinding.revision,
        dataAt(old),
        committedAt: old,
      );
      final repo = ControlledCampusRepository();
      await tester.pumpWidget(
        testApp(db, store, repo, RecordingWidgetBridge(), scale: 2),
      );
      await tester.pumpAndSettle();
      expect(find.text('快照可能已过期，请刷新获取最新数据。'), findsOneWidget);
      expect(find.byType(GfLoading), findsNothing);
      expect(repo.requested, isEmpty);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'cache clear is campus-only and retry completes partial failure',
    (tester) async {
      store = CampusSnapshotStore(db);
      final failing = _FailingClearStore(db);
      await failing.write(appScope, 'binding', dataAt(DateTime.now()));
      await db.customStatement(
        "INSERT INTO cached_topics VALUES (7, '{}', 'now')",
      );
      SharedPreferences.setMockInitialValues({
        'schedule_plans': 'plan',
        'composer_draft': 'draft',
      });
      final bridge = RecordingWidgetBridge();
      await tester.pumpWidget(
        testApp(
          db,
          failing,
          ControlledCampusRepository(),
          bridge,
          child: const Scaffold(body: CampusCacheClearTile()),
        ),
      );
      await tester.pumpAndSettle();
      Future<void> clear() async {
        await tester.tap(find.text('清除校园缓存'));
        await tester.pumpAndSettle();
        await tester.tap(find.text('确认'));
        await tester.pumpAndSettle();
      }

      await clear();
      expect(find.text('部分缓存未能清除，请重试。'), findsOneWidget);
      expect(bridge.clears, 1);
      expect(await failing.read(appScope), isNotNull);
      failing.fail = false;
      await clear();
      expect(find.text('部分缓存未能清除，请重试。'), findsNothing);
      expect(await failing.read(appScope), isNull);
      expect(bridge.clears, 2);
      expect(
        (await db.customSelect('SELECT * FROM cached_topics').get()).length,
        1,
      );
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString('schedule_plans'), 'plan');
      expect(prefs.getString('composer_draft'), 'draft');
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'widget settings cannot republish a snapshot deleted during rules lookup',
    (tester) async {
      store = CampusSnapshotStore(db);
      SharedPreferences.setMockInitialValues({
        'schedule_widget_privacy_ack': true,
      });
      await store.write(appScope, 'binding', dataAt(DateTime.now()));
      final repo = _PendingRules();
      final bridge = RecordingWidgetBridge();
      await tester.pumpWidget(
        testApp(
          db,
          store,
          repo,
          bridge,
          child: const ScheduleWidgetSettingsPage(),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('从离线课表刷新'));
      await tester.pumpAndSettle();
      expect(repo.requestedRules.isCompleted, isTrue);
      await tester.runAsync(store.clear);
      repo.rules.complete(
        const CampusCalendarSettings(
          revision: '',
          rules: CampusCalendarRules(holidays: [], moves: []),
        ),
      );
      await tester.pumpAndSettle();
      expect(bridge.writes, 0);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );
}
