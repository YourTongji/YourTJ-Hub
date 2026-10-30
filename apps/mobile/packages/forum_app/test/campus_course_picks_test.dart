import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/campus/campus_course_picks.dart';

const picks = CourseListResultPayload(
  list: [
    CourseSummaryPayload(
      id: 101,
      primaryCode: 'M1',
      name: '管理学概论',
      department: '经济与管理学院',
      creditX10: 20,
      teacherName: '郭磊',
      ratingAvg: 3,
    ),
    CourseSummaryPayload(
      id: 102,
      primaryCode: 'M1',
      name: '管理学概论',
      department: '经济与管理学院',
      creditX10: 20,
      teacherName: '解学梅',
      ratingAvg: 5,
    ),
    CourseSummaryPayload(
      id: 103,
      primaryCode: 'M2',
      name: '跨学院的课程和很长的课程名称',
      department: '经济与管理学院',
      creditX10: 20,
      instructors: ['欧阳明远', '司马长风'],
    ),
    CourseSummaryPayload(
      id: 104,
      primaryCode: 'M3',
      name: '教师信息缺失',
      department: '其他学院',
      creditX10: 20,
    ),
  ],
  page: 1,
  size: 4,
  total: 4,
  hasNext: false,
);

Future<GoRouter> mountCoursePicks(
  WidgetTester tester, {
  Brightness brightness = Brightness.light,
  double scale = 1,
}) async {
  final router = GoRouter(
    initialLocation: '/campus',
    routes: [
      GoRoute(
        path: '/campus',
        builder: (_, _) => const Scaffold(
          body: SingleChildScrollView(child: CampusCoursePicks()),
        ),
      ),
      GoRoute(
        path: '/courses/:id',
        builder: (_, s) =>
            Scaffold(body: Text('detail-${s.pathParameters['id']}')),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [campusCoursesProvider.overrideWith((_) async => picks)],
      child: MaterialApp.router(
        routerConfig: router,
        theme: gfThemeData(brightness),
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

void main() {
  for (final brightness in Brightness.values) {
    for (final width in [320.0, 466.0, 951.0]) {
      testWidgets(
        'same-name recommendations show teachers, teams and missing identity ($brightness, $width)',
        (tester) async {
          tester.view.physicalSize = Size(width, 678);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.reset);
          await mountCoursePicks(tester, brightness: brightness, scale: 2);
          expect(find.textContaining('郭磊'), findsOneWidget);
          expect(find.textContaining('解学梅'), findsOneWidget);
          expect(find.textContaining('欧阳明远、司马长风'), findsOneWidget);
          final l = AppLocalizations.of(
            tester.element(find.byType(CampusCoursePicks)),
          );
          expect(find.textContaining(l.courseCopyNoTeacher), findsOneWidget);
          expect(find.text('3.0'), findsOneWidget);
          expect(find.text('5.0'), findsOneWidget);
          expect(tester.takeException(), isNull);
          final second = find.ancestor(
            of: find.textContaining('解学梅'),
            matching: find.byType(ListTile),
          );
          await tester.ensureVisible(second);
          await tester.tap(second);
          await tester.pumpAndSettle();
          expect(find.text('detail-102'), findsOneWidget);
          await tester.pumpWidget(const SizedBox.shrink());
        },
      );
    }
  }
}
