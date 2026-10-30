import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'schedule_page_test.dart'
    show FakePkRepository, MemoryTokenStorage, detail, makeContainer;

Future<
  ({
    ScheduleStoreNotifier notifier,
    ProviderContainer container,
    FakePkRepository repository,
  })
>
openTaskFlow(
  WidgetTester tester, {
  String locale = 'zh',
  Brightness brightness = Brightness.light,
  double width = 390,
  double scale = 1,
  bool configured = true,
}) async {
  SharedPreferences.setMockInitialValues({});
  final notifier = ScheduleStoreNotifier();
  await notifier.ready;
  if (configured) {
    notifier.setMajorSelection(
      PkMajorSelection(
        calendarId: 119,
        grade: 2025,
        major: 'm1',
        majorName: '软件工程',
      ),
    );
  }
  final repository = FakePkRepository(
    GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'http://fake.local',
    ),
  );
  final container = makeContainer(notifier, repository: repository);
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        locale: Locale(locale),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        theme: ThemeData(brightness: brightness),
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: Center(
            child: SizedBox(width: width, child: child!),
          ),
        ),
        home: const SchedulePage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return (notifier: notifier, container: container, repository: repository);
}

Future<void> selectTask(WidgetTester tester, String label) async {
  final tab = find.text(label).first;
  await tester.ensureVisible(tab);
  await tester.tap(tab);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'four tasks share selected classes, conflicts and restored local state',
    (tester) async {
      final scope = await openTaskFlow(tester);
      final l10n = AppLocalizations.of(
        tester.element(find.byType(SchedulePage)),
      );
      expect(find.text('筛选'), findsOneWidget);
      expect(find.text('候选'), findsOneWidget);
      scope.notifier.selectClass(detail('110001.01', room: 'A101'), '高等数学');
      scope.notifier.selectClass(detail('110002.01', room: 'A102'), '大学物理');
      scope.notifier.addCustomEvent(
        label: '实验室',
        day: 3,
        sections: [3],
        weeks: [1, 2],
      );
      scope.notifier.setWeekView(PkWeekView(week: 2, useCurrent: false));
      await scope.notifier.flush;
      await selectTask(tester, l10n.scheduleSelected);
      expect(find.text('高等数学'), findsOneWidget);
      expect(find.text('大学物理'), findsOneWidget);
      expect(find.textContaining('A101'), findsWidgets);
      expect(find.textContaining('张老师'), findsWidgets);
      expect(find.textContaining('大学物理'), findsWidgets);
      await selectTask(tester, l10n.scheduleTabTimetable);
      expect(find.text('高等数学'), findsWidgets);
      await selectTask(tester, '筛选');
      expect(scope.notifier.activePlan.selectedCourses, [
        '110001.01',
        '110002.01',
      ]);
      await selectTask(tester, '候选');
      final restored = ScheduleStoreNotifier();
      await restored.ready;
      expect(restored.activePlan.selectedCourses, ['110001.01', '110002.01']);
      final restoredContainer = makeContainer(restored);
      addTearDown(restoredContainer.dispose);
      final restoredState = restoredContainer.read(scheduleStoreProvider);
      expect(restoredState.weekView.week, 2);
      expect(restoredState.majorSelected.major, 'm1');
      expect(restored.activePlan.customEvents.single.label, '实验室');
      expect(restoredState.stats.conflictCount, 2);
      await selectTask(tester, l10n.scheduleSelected);
      await tester.ensureVisible(find.text(l10n.scheduleRemoveCourse).first);
      await tester.tap(find.text(l10n.scheduleRemoveCourse).first);
      await tester.pumpAndSettle();
      expect(scope.notifier.activePlan.selectedCourses, ['110002.01']);
    },
  );
  testWidgets('candidate query survives each task switch', (tester) async {
    await openTaskFlow(tester);
    await selectTask(tester, '搜索');
    await tester.enterText(find.byType(TextField), '高等数学');
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    for (final label in ['已选', '方案预览', '筛选']) {
      await selectTask(tester, label);
      await selectTask(tester, '候选');
      expect(find.byType(TextField), findsOneWidget);
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        '高等数学',
      );
    }
  });
  testWidgets('retained candidates fence previous course context responses', (
    tester,
  ) async {
    final scope = await openTaskFlow(tester);
    final requests = <Completer<PkSearchResult>>[];
    scope.repository.onSearchCourses = (_) {
      final pending = Completer<PkSearchResult>();
      requests.add(pending);
      return pending.future;
    };
    await selectTask(tester, '搜索');
    await tester.enterText(find.byType(TextField), '数学');
    await tester.pump(const Duration(milliseconds: 400));
    await selectTask(tester, '筛选');
    scope.notifier.setMajorSelection(
      PkMajorSelection(
        calendarId: 119,
        grade: 2024,
        major: 'm2',
        majorName: '土木工程',
      ),
    );
    await tester.pump();
    expect(requests.length, 2);
    PkSearchResult result(String name) => PkSearchResult(
      courses: [
        PkSearchCourseItem(
          courseCode: '110001',
          courseName: name,
          faculty: '',
          facultyI18n: '',
          credit: 4,
          campus: [],
          campusList: [],
          courseNature: [],
        ),
      ],
      sizeLimit: 100,
    );
    requests.last.complete(result('新上下文结果'));
    requests.first.complete(result('旧上下文结果'));
    await tester.pumpAndSettle();
    await selectTask(tester, '候选');
    expect(find.text('新上下文结果'), findsOneWidget);
    expect(find.text('旧上下文结果'), findsNothing);
  });
  testWidgets('first-time planner opens filters and proceeds to candidates', (
    tester,
  ) async {
    final scope = await openTaskFlow(tester, configured: false);
    expect(find.text('学期'), findsWidgets);
    scope.repository.coursesByMajorFixture = const [
      PkCourseByMajorItem(
        courseCode: '110001',
        courseName: '高等数学',
        faculty: '数学系',
        facultyI18n: '',
        credit: 4,
        grade: 2025,
        courseNature: ['必修'],
        courses: [],
      ),
    ];
    scope.repository.onCourseDetails = (_) => const [
      PkCourseDetailBrief(
        code: '110001.01',
        teachers: [PkTeacherRef(teacherName: '张老师', teacherCode: 'T1')],
        campus: '四平',
        teachingLanguage: '中文',
        arrangementInfo: [
          PkArrangementInfo(
            arrangementText: '周一 1-2节',
            occupyDay: 1,
            occupyTime: [1, 2],
            occupyWeek: [1, 2],
          ),
        ],
      ),
    ];
    for (final step in [
      ('学期', '2025-2026-1'),
      ('年级', '2025 级'),
      ('专业', '软件工程'),
    ]) {
      await tester.ensureVisible(find.text(step.$1).first);
      await tester.tap(find.text(step.$1).first);
      await tester.pumpAndSettle();
      await tester.tap(find.text(step.$2).last);
      await tester.pumpAndSettle();
    }
    // The visible primary action and tab lead to the same task.
    await tester.ensureVisible(find.text('候选').last);
    await tester.tap(find.text('候选').last);
    await tester.pumpAndSettle();
    expect(find.text('必修'), findsOneWidget);
    await tester.ensureVisible(find.text('高等数学'));
    await tester.tap(find.text('高等数学'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('110001.01'));
    await tester.pumpAndSettle();
    expect(scope.notifier.activePlan.selectedCourses, ['110001.01']);
    Navigator.of(tester.element(find.text('110001.01'))).pop();
    await tester.pumpAndSettle();
    await selectTask(tester, '方案预览');
    expect(find.text('高等数学'), findsWidgets);
  });
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    for (final width in [320.0, 360.0, 390.0]) {
      for (final scale in [1.0, 2.0]) {
        testWidgets('selected class readable $locale/$width/$scale', (
          tester,
        ) async {
          final scope = await openTaskFlow(
            tester,
            locale: locale,
            width: width,
            scale: scale,
          );
          final l10n = AppLocalizations.of(
            tester.element(find.byType(SchedulePage)),
          );
          scope.notifier.selectClass(
            detail('110001.01', room: 'A101', weeks: [1]),
            'Long course / 高等数学 und angewandte Analysis',
          );
          scope.notifier.setWeekView(PkWeekView(week: 2, useCurrent: false));
          await selectTask(tester, l10n.scheduleSelected);
          final remove = find.text(l10n.scheduleRemoveCourse);
          await tester.ensureVisible(remove);
          await tester.pumpAndSettle();
          expect(find.textContaining('A101'), findsWidgets);
          expect(tester.takeException(), isNull);
          await tester.tap(remove);
          await tester.pumpAndSettle();
          expect(scope.notifier.activePlan.selectedCourses, isEmpty);
          await tester.pumpWidget(const SizedBox());
          await tester.pumpAndSettle();
        });
      }
    }
  }
}
