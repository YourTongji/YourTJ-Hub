import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/schedule/schedule_grid.dart';
import 'package:forum_app/src/widgets/schedule_time_grid.dart';
import 'package:ui_kit/ui_kit.dart';

PkCourseOnTable course({
  String code = 'X.01',
  String name = '高等数学',
  int day = 1,
  List<int> sections = const [1, 2],
  List<int> weeks = const [1, 3, 5],
}) => PkCourseOnTable(
  showText: '',
  courseName: name,
  code: code,
  occupyTime: sections,
  occupyDay: day,
  occupyWeek: weeks,
  occupyRoom: '教学北楼 A101',
  teacherAndCode: '张老师、李老师(T001)',
);

Widget gridApp({
  double width = 320,
  TextScaler scaler = TextScaler.noScaling,
  Brightness brightness = Brightness.light,
  List<PkCourseOnTable>? courses,
  ValueChanged<PkCourseOnTable>? onCourse,
  void Function(int, int)? onEmpty,
  bool conflicted = false,
}) => MaterialApp(
  theme: gfThemeData(brightness),
  locale: const Locale('zh'),
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(
    body: SingleChildScrollView(
      child: Center(
        child: SizedBox(
          width: width,
          child: MediaQuery(
            data: MediaQueryData(textScaler: scaler),
            child: ScheduleTimeGrid(
              grid: testGrid(courses ?? [course()], conflicted: conflicted),
              times: sectionTimesFor(11, null),
              onTapCourse: onCourse ?? (_) {},
              onTapEmptyCell: onEmpty,
            ),
          ),
        ),
      ),
    ),
  ),
);

ScheduleGridData testGrid(
  List<PkCourseOnTable> courses, {
  bool conflicted = false,
}) {
  final grid = buildScheduleGridData(courses, [], maxRows: 4);
  return ScheduleGridData(
    cellCourses: grid.cellCourses,
    cellSpans: grid.cellSpans,
    occupiedGrid: grid.occupiedGrid,
    rowHeights: grid.rowHeights,
    conflicts: conflicted
        ? {
            'X': [const PkConflictItem(code: 'Y.01', courseName: '物理')],
          }
        : grid.conflicts,
  );
}

void main() {
  testWidgets('spanning cards show both room and teachers', (tester) async {
    await tester.pumpWidget(gridApp());
    expect(find.text('教学北楼 A101'), findsOneWidget);
    expect(find.text('张老师、李老师'), findsOneWidget);
  });

  testWidgets('wide timetable uses the available width', (tester) async {
    tester.view.physicalSize = const Size(1100, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(gridApp(width: 1000));
    final grid = tester.getRect(find.byType(ScheduleTimeGrid));
    expect(tester.getCenter(find.text('周日')).dx, greaterThan(grid.right - 100));
  });

  testWidgets('time labels remain visible when scrolling to the weekend', (
    tester,
  ) async {
    await tester.pumpWidget(gridApp());
    final start = tester.getTopLeft(find.text('08:00\n08:45'));
    final horizontal = find.byWidgetPredicate(
      (w) => w is SingleChildScrollView && w.scrollDirection == Axis.horizontal,
    );
    await tester.drag(horizontal, const Offset(-400, 0));
    await tester.pumpAndSettle();
    expect(tester.getTopLeft(find.text('08:00\n08:45')), start);
  });

  testWidgets('empty cells have a localized button and keyboard action', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    (int, int)? picked;
    await tester.pumpWidget(
      gridApp(courses: [], onEmpty: (day, section) => picked = (day, section)),
    );
    final cell = find.bySemanticsLabel('周一，第 1 节，选课');
    expect(cell, findsOneWidget);
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    expect(picked, (1, 1));
    semantics.dispose();
    await tester.pumpWidget(const SizedBox());
  });
  for (final brightness in Brightness.values) {
    for (final scaler in [
      const TextScaler.linear(2),
      const UnevenTextScaler(),
    ]) {
      testWidgets('stacked cards fit narrow ${brightness.name} $scaler text', (
        tester,
      ) async {
        final courses = [
          course(name: '高等数学与线性代数', sections: [1]),
          course(
            code: 'Y.01',
            name: '物理实验（双周）',
            sections: [1],
            weeks: [2, 4, 6],
          ),
          course(
            code: 'Z.01',
            name: '第三门重叠课程',
            sections: [1],
            weeks: [1, 2, 3],
          ),
        ];
        await tester.pumpWidget(
          gridApp(
            width: 320,
            scaler: scaler,
            brightness: brightness,
            courses: courses,
          ),
        );
        for (final item in courses) {
          final semantics = find
              .ancestor(
                of: find.text(item.courseName),
                matching: find.byWidgetPredicate(
                  (w) => w is Semantics && w.properties.button == true,
                ),
              )
              .first;
          final size = tester.getSize(semantics);
          expect(size.height, greaterThanOrEqualTo(44));
          expect(size.width, greaterThanOrEqualTo(44));
        }
        expect(find.text('单周'), findsOneWidget);
        expect(find.text('双周'), findsOneWidget);
        expect(tester.takeException(), isNull);
      });
    }
  }

  testWidgets('course action exposes full details and keyboard activation', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    PkCourseOnTable? selected;
    await tester.pumpWidget(gridApp(onCourse: (value) => selected = value));
    final card = find.bySemanticsLabel(
      RegExp('高等数学.*周一.*第 1-2 节.*教学北楼 A101.*张老师、李老师.*单周'),
    );
    expect(card, findsOneWidget);
    expect(tester.getSemantics(card).flagsCollection.isButton, isTrue);
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    expect(selected?.code, 'X.01');
    semantics.dispose();
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('conflict is announced and its icon does not cover the title', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(gridApp(conflicted: true));
    expect(find.bySemanticsLabel(RegExp('高等数学.*冲突')), findsOneWidget);
    final icon = tester.getRect(find.byIcon(Icons.warning_amber_rounded));
    expect(icon.overlaps(tester.getRect(find.text('高等数学'))), isFalse);
    semantics.dispose();
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets(
    'read-only empty cells and custom events have no misleading button',
    (tester) async {
      final semantics = tester.ensureSemantics();
      var calls = 0;
      await tester.pumpWidget(
        gridApp(
          courses: [course(code: 'custom:1', name: '自习')],
          onCourse: (_) => calls++,
        ),
      );
      expect(find.bySemanticsLabel(RegExp('选课')), findsNothing);
      final card = find.bySemanticsLabel(RegExp('自习.*周一'));
      expect(tester.getSemantics(card).flagsCollection.isButton, isFalse);
      await tester.tap(find.text('自习'));
      expect(calls, 0);
      semantics.dispose();
      await tester.pumpWidget(const SizedBox());
    },
  );
}

class UnevenTextScaler extends TextScaler {
  const UnevenTextScaler();
  @override
  double scale(double fontSize) => fontSize * (fontSize <= 10 ? 2.8 : 1.8);
  @override
  double get textScaleFactor => 2.8;
}
