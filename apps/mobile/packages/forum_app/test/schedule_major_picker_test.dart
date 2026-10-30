import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/schedule/major_picker.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'schedule_page_test.dart' show makeContainer, wrapApp, detail;

void main() {
  testWidgets('major picker exposes name and code search before selection', (
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
    notifier.setConfigCollapsed(false);
    final container = makeContainer(notifier);
    addTearDown(container.dispose);
    await tester.pumpWidget(wrapApp(container));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('专业'));
    await tester.tap(find.text('专业'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(TextField, '搜索专业名称或代码'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
  });
  testWidgets('changing major asks before clearing existing plan content', (
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
    notifier.selectClass(detail('X.01'), '原方案课程');
    final firstPlan = notifier.activePlan.id;
    notifier.createPlan();
    notifier.addCustomEvent(
      label: '另一个方案的安排',
      day: 2,
      sections: [2],
      weeks: [1],
    );
    notifier.switchPlan(firstPlan);
    notifier.setConfigCollapsed(false);
    final container = makeContainer(notifier);
    addTearDown(container.dispose);
    await tester.pumpWidget(wrapApp(container));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('专业'));
    await tester.tap(find.text('专业'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('土木工程'));
    await tester.pumpAndSettle();
    expect(find.text('切换排课配置？'), findsOneWidget);
    expect(notifier.state.majorSelected.major, 'm1');
    expect(notifier.activePlan.selectedCourses, ['X.01']);
    expect(find.textContaining('2 个方案有内容（1 门课程、1 个安排）'), findsOneWidget);
    await tester.tap(find.text('取消'));
    await tester.pumpAndSettle();
    expect(notifier.state.majorSelected.major, 'm1');
    expect(notifier.activePlan.selectedCourses, ['X.01']);
    await tester.tap(find.text('专业'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('schedule-major-option-m1')));
    await tester.pumpAndSettle();
    expect(notifier.activePlan.selectedCourses, ['X.01']);
    expect(find.text('切换排课配置？'), findsNothing);
    await tester.tap(find.text('专业'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('schedule-major-option-m2')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('清空方案并切换'));
    await tester.pumpAndSettle();
    await notifier.flush;
    expect(notifier.state.majorSelected.major, 'm2');
    expect(notifier.activePlan.selectedCourses, isEmpty);
    expect(
      notifier.state.plans.every((plan) => plan.customEvents.isEmpty),
      isTrue,
    );
    final restored = ScheduleStoreNotifier();
    await restored.ready;
    addTearDown(restored.dispose);
    expect(restored.state.majorSelected.major, 'm2');
    expect(restored.activePlan.selectedCourses, isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
  });
  testWidgets(
    'a plan update while confirming rejects stale destructive approval',
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
      notifier.selectClass(detail('X.01'), '原方案课程');
      notifier.setConfigCollapsed(false);
      final container = makeContainer(notifier);
      addTearDown(container.dispose);
      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('专业'));
      await tester.tap(find.text('专业'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('土木工程'));
      await tester.pumpAndSettle();
      notifier.addCustomEvent(label: '期间新增', day: 1, sections: [3], weeks: [1]);
      await tester.tap(find.text('清空方案并切换'));
      await tester.pumpAndSettle();
      expect(notifier.state.majorSelected.major, 'm1');
      expect(notifier.activePlan.selectedCourses, ['X.01']);
      expect(notifier.activePlan.customEvents.single.label, '期间新增');
      expect(find.text('方案已发生变化，请重新选择并确认。'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );

  for (final lang in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets(
        'major names/codes, empty/clear and keyboard fit $lang $brightness',
        (tester) async {
          tester.view.devicePixelRatio = 1;
          tester.view.physicalSize = Size(
            brightness == Brightness.dark ? 320 : 951,
            678,
          );
          addTearDown(tester.view.reset);
          PkMajor? chosen;
          await tester.pumpWidget(
            MaterialApp(
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
              home: Scaffold(
                body: Builder(
                  builder: (context) => TextButton(
                    onPressed: () async {
                      chosen = await showGfBottomSheet<PkMajor>(
                        context,
                        keyboardAware: true,
                        builder: (_) => const ScheduleMajorPicker(
                          contextLabel: '2026-2027 · 2026',
                          selectedCode: '00304',
                          items: [
                            PkMajor(
                              code: '00304',
                              name: '2026(00304 基础学科拔尖基地(数学))',
                            ),
                            PkMajor(code: '00305', name: '基础学科拔尖基地(数学)'),
                            PkMajor(code: 'CSE01', name: '软件工程（试验班与长名称）'),
                          ],
                        ),
                      );
                    },
                    child: const Text('open'),
                  ),
                ),
              ),
            ),
          );
          await tester.tap(find.text('open'));
          await tester.pumpAndSettle();
          final l = AppLocalizations.of(
            tester.element(find.byType(ScheduleMajorPicker)),
          );
          final search = find.descendant(
            of: find.byKey(const Key('schedule-major-search')),
            matching: find.byType(TextField),
          );
          await tester.enterText(search, '00304');
          tester.view.viewInsets = const FakeViewPadding(bottom: 300);
          await tester.pumpAndSettle();
          expect(
            find.byKey(const ValueKey('schedule-major-option-00304')),
            findsOneWidget,
          );
          expect(
            find.byKey(const ValueKey('schedule-major-option-00305')),
            findsNothing,
          );
          expect(find.text('基础学科拔尖基地(数学)'), findsOneWidget);
          expect(search.hitTestable(), findsOneWidget);
          expect(find.byTooltip(l.commonClose).hitTestable(), findsOneWidget);
          await tester.enterText(search, '基础学科拔尖基地');
          await tester.pumpAndSettle();
          final sameName = find.byKey(
            const ValueKey('schedule-major-option-00305'),
          );
          await tester.scrollUntilVisible(
            sameName,
            100,
            scrollable: find
                .descendant(
                  of: find.byType(ScheduleMajorPicker),
                  matching: find.byType(Scrollable),
                )
                .last,
          );
          expect(sameName, findsOneWidget);
          await tester.enterText(search, 'no such major');
          await tester.pumpAndSettle();
          expect(find.text(l.scheduleMajorNoMatch), findsOneWidget);
          await tester.tap(find.byTooltip(l.courseCopyClearSearch));
          await tester.pumpAndSettle();
          expect(tester.widget<TextField>(search).controller!.text, isEmpty);
          expect(
            tester
                .widget<ListTile>(
                  find.byKey(const ValueKey('schedule-major-option-00304')),
                )
                .selected,
            isTrue,
          );
          await tester.enterText(search, ' cse01 ');
          await tester.pumpAndSettle();
          final result = find.byKey(
            const ValueKey('schedule-major-option-CSE01'),
          );
          await Scrollable.ensureVisible(
            tester.element(result),
            alignment: 0.5,
          );
          await tester.pumpAndSettle();
          expect(result.hitTestable(), findsOneWidget);
          await tester.tap(result);
          await tester.pumpAndSettle();
          expect(chosen?.code, 'CSE01');
          expect(tester.takeException(), isNull);
          await tester.pumpWidget(const SizedBox.shrink());
        },
      );
    }
  }
}
