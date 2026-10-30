import 'dart:async';
import 'dart:convert';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'schedule_page_test.dart'
    show FakePkRepository, MemoryTokenStorage, makeContainer, wrapApp, detail;

class ScopeCourses extends FakePkRepository {
  ScopeCourses()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      );
  int loads = 0;
  bool empty = false;
  final pending = <String, Completer<List<PkCourseByMajorItem>>>{};
  @override
  Future<List<PkCourseByMajorItem>> coursesByMajor({
    required int grade,
    required String code,
    required int calendarId,
  }) async {
    loads++;
    if (pending[code] case final request?) return request.future;
    if (empty) return [];
    return [
      PkCourseByMajorItem.fromJson({
        'courseCode': code,
        'courseName': '配置后的课程',
        'credit': 2,
        'courses': [],
      }),
    ];
  }

  @override
  Future<List<PkCourseByNatureItem>> coursesByNature({
    required int calendarId,
    required List<int> ids,
  }) async => [
    PkCourseByNatureItem.fromJson({'courseLabelName': '空课程组', 'courses': []}),
  ];
}

void main() {
  testWidgets(
    'configuration completion fetches courses without changing tabs',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      final notifier = ScheduleStoreNotifier();
      await notifier.ready;
      final repo = ScopeCourses();
      final container = makeContainer(notifier, repository: repo);
      addTearDown(container.dispose);
      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();
      await tester.tap(find.text('候选').first);
      await tester.pumpAndSettle();
      expect(repo.loads, 0);
      await notifier.setMajorSelection(
        PkMajorSelection(
          calendarId: 119,
          grade: 2025,
          major: 'm1',
          majorName: '软件工程',
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('配置后的课程'), findsOneWidget);
      expect(repo.loads, 1);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );
  testWidgets(
    'missing configuration names the prerequisite and opens its controls',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      final notifier = ScheduleStoreNotifier();
      await notifier.ready;
      final container = makeContainer(notifier);
      addTearDown(container.dispose);
      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();
      await tester.tap(find.text('候选').first);
      await tester.pumpAndSettle();
      expect(find.text('先设置排课范围'), findsOneWidget);
      expect(find.byKey(const Key('schedule-configure-scope')), findsOneWidget);
      final before = notifier.state.majorSelected.toJson();
      await tester.ensureVisible(
        find.byKey(const Key('schedule-configure-scope')),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('schedule-configure-scope')));
      await tester.pumpAndSettle();
      expect(find.text('学期').first.hitTestable(), findsOneWidget);
      expect(notifier.state.majorSelected.toJson(), before);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );
  testWidgets('configuration change fences a pending old-scope response', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final notifier = ScheduleStoreNotifier();
    await notifier.ready;
    await notifier.setMajorSelection(
      PkMajorSelection(
        calendarId: 119,
        grade: 2025,
        major: 'old',
        majorName: '旧专业',
      ),
    );
    final repo = ScopeCourses();
    repo.pending['old'] = Completer();
    repo.pending['new'] = Completer();
    final container = makeContainer(notifier, repository: repo);
    addTearDown(container.dispose);
    await tester.pumpWidget(wrapApp(container));
    await tester.pump();
    expect(repo.loads, 1);
    await notifier.setMajorSelection(
      PkMajorSelection(
        calendarId: 119,
        grade: 2025,
        major: 'new',
        majorName: '新专业',
      ),
    );
    await tester.pump();
    await tester.pump();
    expect(repo.loads, 2);
    repo.pending['new']!.complete([
      PkCourseByMajorItem.fromJson({
        'courseCode': 'NEW',
        'courseName': '新专业课程',
        'courses': [],
      }),
    ]);
    await tester.pumpAndSettle();
    repo.pending['old']!.complete([
      PkCourseByMajorItem.fromJson({
        'courseCode': 'OLD',
        'courseName': '过期课程',
        'courses': [],
      }),
    ]);
    await tester.pumpAndSettle();
    expect(find.text('新专业课程'), findsOneWidget);
    expect(find.text('过期课程'), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
  });
  testWidgets(
    'failed retry, zero matches and clearing preserve courses and custom events',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      final notifier = ScheduleStoreNotifier();
      await notifier.ready;
      await notifier.setMajorSelection(
        PkMajorSelection(
          calendarId: 119,
          grade: 2025,
          major: 'm1',
          majorName: '软件工程',
        ),
      );
      notifier.selectClass(detail('X.01'), '已有课程');
      notifier.addCustomEvent(label: '已有安排', day: 1, sections: [3], weeks: [1]);
      final before = jsonEncode(
        notifier.state.plans.map((p) => p.toJson()).toList(),
      );
      var requests = 0;
      final repo = ScopeCourses()..empty = true;
      repo.onSearchCourses = (query) async {
        requests++;
        if (requests == 1) {
          throw const NetworkException(fallbackMessage: 'controlled failure');
        }
        return const PkSearchResult(courses: [], sizeLimit: 100);
      };
      final container = makeContainer(notifier, repository: repo);
      addTearDown(container.dispose);
      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();
      expect(find.text('当前范围未返回必修课程'), findsOneWidget);
      await tester.ensureVisible(
        find.byKey(const Key('schedule-find-by-name')),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('schedule-find-by-name')));
      await tester.pumpAndSettle();
      final input = find.byType(TextField);
      expect(tester.widget<TextField>(input).decoration!.hintText, '搜索课程名称');
      await tester.enterText(input, 'CEM1903');
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpAndSettle();
      expect(find.byType(GfErrorRetry), findsOneWidget);
      expect(find.byKey(const Key('schedule-search-no-match')), findsNothing);
      tester.widget<GfErrorRetry>(find.byType(GfErrorRetry)).onRetry();
      await tester.pumpAndSettle();
      expect(find.text('没有找到名称匹配“CEM1903”的课程'), findsOneWidget);
      expect(find.text('按课程名称搜索：2025-2026-1'), findsOneWidget);
      expect(tester.widget<TextField>(input).controller!.text, 'CEM1903');
      final clear = find.byKey(const Key('schedule-clear-query'));
      await tester.ensureVisible(clear);
      await tester.pumpAndSettle();
      await tester.tap(clear);
      await tester.pumpAndSettle();
      expect(tester.widget<TextField>(input).controller!.text, isEmpty);
      expect(requests, 2);
      expect(
        jsonEncode(notifier.state.plans.map((p) => p.toJson()).toList()),
        before,
      );
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );
  testWidgets('empty optional groups show a scoped empty state', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final notifier = ScheduleStoreNotifier();
    await notifier.ready;
    await notifier.setMajorSelection(
      PkMajorSelection(
        calendarId: 119,
        grade: 2025,
        major: 'm1',
        majorName: '软件工程',
      ),
    );
    final repo = ScopeCourses()..empty = true;
    final container = makeContainer(notifier, repository: repo);
    addTearDown(container.dispose);
    await tester.pumpWidget(wrapApp(container));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('选修'));
    await tester.tap(find.text('选修'));
    await tester.pumpAndSettle();
    expect(find.text('当前学期未返回选修课程'), findsOneWidget);
    expect(find.text('空课程组'), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
  });
  for (final lang in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets(
        'scoped empty recovery actions fit $lang $brightness large text',
        (tester) async {
          tester.view.devicePixelRatio = 1;
          tester.view.physicalSize = Size(
            brightness == Brightness.dark ? 320 : 951,
            678,
          );
          addTearDown(tester.view.reset);
          SharedPreferences.setMockInitialValues({});
          final notifier = ScheduleStoreNotifier();
          await notifier.ready;
          await notifier.setMajorSelection(
            PkMajorSelection(
              calendarId: 119,
              grade: 2025,
              major: 'm1',
              majorName: '软件工程',
            ),
          );
          final repo = ScopeCourses()..empty = true;
          final container = makeContainer(notifier, repository: repo);
          addTearDown(container.dispose);
          await tester.pumpWidget(
            UncontrolledProviderScope(
              container: container,
              child: MaterialApp(
                theme: gfThemeData(brightness),
                locale: Locale(lang),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(
                    context,
                  ).copyWith(textScaler: const TextScaler.linear(2)),
                  child: child!,
                ),
                home: const SchedulePage(),
              ),
            ),
          );
          await tester.pumpAndSettle();
          final l = AppLocalizations.of(
            tester.element(find.byType(SchedulePage)),
          );
          await tester.scrollUntilVisible(
            find.text(l.commonSearch),
            180,
            scrollable: find.byType(Scrollable).first,
          );
          await tester.ensureVisible(find.text(l.commonSearch));
          await tester.pumpAndSettle();
          expect(find.text(l.commonSearch).hitTestable(), findsOneWidget);
          await tester.tap(find.text(l.commonSearch));
          await tester.pumpAndSettle();
          expect(
            tester.widgetList<GfTabBar>(find.byType(GfTabBar)).last.selected,
            2,
          );
          await tester.ensureVisible(find.byType(TextField));
          await tester.enterText(find.byType(TextField), 'Missing course');
          await tester.pump(const Duration(milliseconds: 400));
          await tester.pumpAndSettle();
          expect(
            find.text(l.scheduleSearchNoMatch('Missing course')),
            findsOneWidget,
          );
          final clear = find.byKey(const Key('schedule-clear-query'));
          await tester.ensureVisible(clear);
          await tester.pumpAndSettle();
          expect(clear.hitTestable(), findsOneWidget);
          await tester.tap(clear);
          await tester.pumpAndSettle();
          await tester.enterText(find.byType(TextField), 'Missing course');
          await tester.pump(const Duration(milliseconds: 400));
          await tester.pumpAndSettle();
          final configure = find.byKey(const Key('schedule-configure-scope'));
          await tester.ensureVisible(configure);
          await tester.pumpAndSettle();
          await tester.tap(configure);
          await tester.pumpAndSettle();
          expect(find.text(l.scheduleTerm).first.hitTestable(), findsOneWidget);
          expect(tester.takeException(), isNull);
          await tester.pumpWidget(const SizedBox.shrink());
          await tester.pumpAndSettle();
        },
      );
    }
  }
}
