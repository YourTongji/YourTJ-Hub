import 'dart:async';
import 'dart:convert';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'schedule_config_transaction_test.dart'
    show ScheduleConfigTestDatabase, seedScheduleConfig;
import 'schedule_page_test.dart'
    show FakePkRepository, MemoryTokenStorage, detail, makeContainer;

class _Pk extends FakePkRepository {
  Completer<Map<String, List<PkCourseDetailBrief>>>? pendingSync;
  @override
  Future<Map<String, List<PkCourseDetailBrief>>> courseInfoSync({
    required int calendarId,
    List<String> majorCourseCodes = const [],
    List<String> otherCourseCodes = const [],
    (int, String)? majorInfo,
  }) async => pendingSync?.future ?? {};
  _Pk() : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage()));
  @override
  Future<List<PkCalendarItem>> calendars() async => [
    ...FakePkRepository.calendarsFixture,
    const PkCalendarItem(
      calendarId: 120,
      calendarName: '2026-2027-1',
      startDate: '2026-09-07',
      endDate: '2027-01-16',
    ),
  ];
}

Future<({ScheduleStoreNotifier store, ProviderContainer container})>
openScheduleConfig(
  WidgetTester tester, {
  ScheduleStoreNotifier? initialStore,
  Locale locale = const Locale('zh'),
  Brightness brightness = Brightness.light,
  double textScale = 1,
  double? width,
  bool empty = false,
  FakePkRepository? repository,
}) async {
  final store = initialStore ?? ScheduleStoreNotifier();
  await store.ready;
  if (initialStore == null) {
    await store.setMajorSelection(
      PkMajorSelection(
        calendarId: 119,
        grade: 2025,
        major: 'm1',
        majorName: 'Existing major',
      ),
    );
    if (!empty) {
      store.selectClass(detail('X.01'), 'Existing course');
      store.addCustomEvent(
        label: 'Keep this block',
        day: 2,
        sections: [3],
        weeks: [1, 2],
      );
    }
  }
  store.setConfigCollapsed(false);
  await store.flush;
  final container = makeContainer(store, repository: repository ?? _Pk());
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        theme: gfThemeData(brightness),
        locale: locale,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(
            size: width == null
                ? null
                : Size(width, MediaQuery.sizeOf(context).height),
            textScaler: TextScaler.linear(textScale),
          ),
          child: width == null
              ? child!
              : Center(
                  child: SizedBox(width: width, child: child),
                ),
        ),
        home: const SchedulePage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return (store: store, container: container);
}

Future<void> chooseNewTerm(WidgetTester tester) async {
  await tester.tap(find.text('2025-2026-1'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('2026-2027-1'));
  await tester.pumpAndSettle();
}

String plansJson(ScheduleStoreNotifier store) =>
    jsonEncode(store.state.plans.map((p) => p.toJson()).toList());

void main() {
  testWidgets(
    'reselecting the same term keeps grade, major, courses and custom blocks',
    (tester) async {
      final result = await openScheduleConfig(tester);
      final before = jsonEncode(
        result.store.state.plans.map((p) => p.toJson()).toList(),
      );
      await tester.tap(find.text('2025-2026-1'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('2025-2026-1').last);
      await tester.pumpAndSettle();
      await result.store.configurationSettled;
      await result.store.flush;
      expect(result.store.state.majorSelected.grade, 2025);
      expect(result.store.state.majorSelected.major, 'm1');
      expect(
        jsonEncode(result.store.state.plans.map((p) => p.toJson()).toList()),
        before,
      );
      await tester.pumpWidget(const SizedBox());
    },
  );
  testWidgets(
    'a new term waits for explicit confirmation and Cancel preserves all plans',
    (tester) async {
      final result = await openScheduleConfig(tester);
      final before = jsonEncode(
        result.store.state.plans.map((p) => p.toJson()).toList(),
      );
      await tester.tap(find.text('2025-2026-1'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('2026-2027-1'));
      await tester.pumpAndSettle();
      expect(find.text('切换排课配置？'), findsOneWidget);
      expect(result.store.state.majorSelected.calendarId, 119);
      await tester.tap(find.text('取消'));
      await tester.pumpAndSettle();
      await result.store.configurationSettled;
      await result.store.flush;
      expect(result.store.state.majorSelected.calendarId, 119);
      expect(
        jsonEncode(result.store.state.plans.map((p) => p.toJson()).toList()),
        before,
      );
      await tester.pumpWidget(const SizedBox());
    },
  );
  for (final selector in ['grade', 'major']) {
    testWidgets('reselecting the same $selector preserves every plan', (
      tester,
    ) async {
      final result = await openScheduleConfig(tester);
      final before = plansJson(result.store);
      await tester.tap(
        find.text(selector == 'grade' ? '2025 级' : 'Existing major'),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text(selector == 'grade' ? '2025 级' : '软件工程').last);
      await tester.pumpAndSettle();
      await result.store.configurationSettled;
      expect(plansJson(result.store), before);
      expect(result.store.state.majorSelected.major, 'm1');
      expect(find.text('切换排课配置？'), findsNothing);
      await tester.pumpWidget(const SizedBox());
    });
  }
  testWidgets(
    'confirmation lists all affected plans and clears them only after save',
    (tester) async {
      final db = ScheduleConfigTestDatabase();
      addTearDown(db.close);
      final seeded = await seedScheduleConfig(db);
      final identities = seeded.state.plans
          .map((p) => (p.id, p.name, p.createdAt))
          .toList();
      final result = await openScheduleConfig(tester, initialStore: seeded);
      db.batches.clear();
      await chooseNewTerm(tester);
      for (final p in seeded.state.plans) {
        expect(find.textContaining(p.name), findsWidgets);
      }
      expect(find.textContaining('2'), findsWidgets);
      expect(db.batches, isEmpty);
      await tester.tap(find.text('清空方案并切换'));
      await tester.pumpAndSettle();
      await result.store.configurationSettled;
      expect(result.store.state.majorSelected.calendarId, 120);
      expect(
        result.store.state.plans.every(
          (p) => p.stagedCourses.isEmpty && p.customEvents.isEmpty,
        ),
        isTrue,
      );
      expect(
        result.store.state.plans
            .map((p) => (p.id, p.name, p.createdAt))
            .toList(),
        identities,
      );
      expect(db.batches, isNotEmpty);
      for (final batch in db.batches) {
        expect(
          jsonDecode(batch[ScheduleStorageKeys.majorSelected]!)['calendarId'],
          120,
        );
        for (final plan
            in jsonDecode(batch[ScheduleStorageKeys.plans]!) as List) {
          expect(plan['stagedCourses'], isEmpty);
          expect(plan['customEvents'], isEmpty);
        }
      }
      await tester.pumpWidget(const SizedBox());
    },
  );
  testWidgets('empty configuration changes need no destructive confirmation', (
    tester,
  ) async {
    final result = await openScheduleConfig(tester, empty: true);
    await chooseNewTerm(tester);
    await result.store.configurationSettled;
    expect(find.text('切换排课配置？'), findsNothing);
    expect(result.store.state.majorSelected.calendarId, 120);
    await tester.pumpWidget(const SizedBox());
  });
  testWidgets('changed plans invalidate an already open confirmation', (
    tester,
  ) async {
    final result = await openScheduleConfig(tester);
    await chooseNewTerm(tester);
    result.store.renamePlan(result.store.activePlan.id, 'Changed elsewhere');
    await tester.tap(find.text('清空方案并切换'));
    await tester.pumpAndSettle();
    expect(result.store.state.majorSelected.calendarId, 119);
    expect(result.store.activePlan.name, 'Changed elsewhere');
    expect(result.store.activePlan.stagedCourses, isNotEmpty);
    await tester.pumpWidget(const SizedBox());
  });
  testWidgets('storage failure retains data and explicit retry can succeed', (
    tester,
  ) async {
    final db = ScheduleConfigTestDatabase();
    addTearDown(db.close);
    final seeded = await seedScheduleConfig(db);
    await openScheduleConfig(tester, initialStore: seeded);
    final before = plansJson(seeded);
    db.reject = true;
    await chooseNewTerm(tester);
    await tester.tap(find.text('清空方案并切换'));
    await tester.pumpAndSettle();
    expect(seeded.state.majorSelected.calendarId, 119);
    expect(plansJson(seeded), before);
    expect(seeded.persistenceError.value, isNotNull);
    db.reject = false;
    await chooseNewTerm(tester);
    await tester.tap(find.text('清空方案并切换'));
    await tester.pumpAndSettle();
    expect(seeded.state.majorSelected.calendarId, 120);
    expect(
      seeded.state.plans.every(
        (p) => p.stagedCourses.isEmpty && p.customEvents.isEmpty,
      ),
      isTrue,
    );
    await tester.pumpWidget(const SizedBox());
  });
  testWidgets('pending durable commit announces saving and blocks route exit', (
    tester,
  ) async {
    final db = ScheduleConfigTestDatabase();
    addTearDown(db.close);
    final seeded = await seedScheduleConfig(db);
    await openScheduleConfig(tester, initialStore: seeded);
    db.pending = Completer<void>();
    db.entered = Completer<void>();
    await chooseNewTerm(tester);
    await tester.tap(find.text('清空方案并切换'));
    await tester.pump(const Duration(milliseconds: 500));
    await tester.pump();
    expect(seeded.isChangingConfiguration, isTrue);
    expect(tester.widget<PopScope>(find.byType(PopScope)).canPop, isFalse);
    expect(
      find.text(
        AppLocalizations.of(
          tester.element(find.byType(SchedulePage)),
        ).draftLocalSaving,
      ),
      findsOneWidget,
    );
    expect(seeded.state.majorSelected.calendarId, 119);
    db.pending!.complete();
    await tester.pumpAndSettle();
    expect(seeded.state.majorSelected.calendarId, 120);
    expect(tester.widget<PopScope>(find.byType(PopScope)).canPop, isTrue);
    await tester.pumpWidget(const SizedBox());
  });
  testWidgets(
    'an old course refresh cannot update a newly selected configuration',
    (tester) async {
      final repository = _Pk();
      final result = await openScheduleConfig(tester, repository: repository);
      result.store.setUpdateTime('2020-09-01');
      result.store.setLatestUpdateTime('2099-09-01');
      await tester.pumpAndSettle();
      repository.pendingSync = Completer();
      await tester.tap(find.text('同步最新'));
      await tester.pump();
      await result.store.setMajorSelection(PkMajorSelection(calendarId: 120));
      await tester.pump();
      final updateTime = result.store.state.updateTime;
      repository.pendingSync!.complete({});
      await tester.pumpAndSettle();
      expect(result.store.state.updateTime, updateTime);
      await tester.pumpWidget(const SizedBox());
    },
  );
  for (final width in [320.0, 360.0, 390.0]) {
    for (final language in ['zh', 'en', 'ja', 'de']) {
      testWidgets(
        'confirmation actions stay readable at $width with large $language text',
        (tester) async {
          tester.view.physicalSize = Size(width, 800);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.reset);
          final result = await openScheduleConfig(
            tester,
            locale: Locale(language),
            textScale: 2,
          );
          final l = AppLocalizations.of(
            tester.element(find.byType(SchedulePage)),
          );
          await chooseNewTerm(tester);
          final cancel = find.text(l.commonCancel);
          final confirm = find.text(l.scheduleConfigChangeConfirm);
          expect(cancel.hitTestable(), findsOneWidget);
          expect(confirm.hitTestable(), findsOneWidget);
          expect(
            tester.getTopLeft(confirm).dy,
            greaterThan(tester.getBottomLeft(cancel).dy),
          );
          expect(tester.getSize(confirm).width, greaterThan(140));
          expect(tester.takeException(), isNull);
          await tester.ensureVisible(find.text(l.scheduleConfigChangeSync));
          await tester.pumpAndSettle();
          expect(
            find.text(l.scheduleConfigChangeSync).hitTestable(),
            findsOneWidget,
          );
          await tester.tap(cancel);
          await tester.pumpAndSettle();
          expect(result.store.state.majorSelected.calendarId, 119);
          await tester.pumpWidget(const SizedBox());
        },
      );
    }
  }
}
