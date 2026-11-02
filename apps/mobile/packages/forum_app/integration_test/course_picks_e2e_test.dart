import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/campus/campus_course_picks.dart';
import 'package:forum_app/src/pages/courses/course_common.dart';
import 'package:forum_app/src/pages/courses/detail_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/theme_mode.dart';
import '../test/pages_behavior_test.dart' show MemTokenStorage;

// Read-only live dev API, full app routes and native rendering. No HTTP mocks.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'public recommendations retain teacher identity through detail navigation',
    (tester) async {
      expect(AppConfig.apiBaseUrl, 'https://dev.yourtj.de');
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(MemTokenStorage()),
          pushBootstrapProvider.overrideWith((_) {}),
        ],
      );
      Future<void> until(bool Function() ready) async {
        final end = DateTime.now().add(const Duration(seconds: 40));
        while (!ready() && DateTime.now().isBefore(end)) {
          await tester.pump(const Duration(milliseconds: 100));
        }
        expect(ready(), isTrue);
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
      }

      Future<void> capture(String name) async {
        await tester.pumpAndSettle();
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot(name);
        }
      }

      appRouter.go('/campus');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const GfApp(locale: Locale('zh')),
        ),
      );
      final courses = await container
          .read(courseRepositoryProvider)
          .list(size: 3, onlyWithReviews: true);
      expect(courses.list.length, greaterThanOrEqualTo(2));
      await until(() => find.byType(CampusCoursePicks).evaluate().isNotEmpty);
      for (final brightness in Brightness.values) {
        container
            .read(themeModeProvider.notifier)
            .setMode(
              brightness == Brightness.dark ? ThemeMode.dark : ThemeMode.light,
            );
        await tester.pumpAndSettle();
        for (final course in courses.list.take(2)) {
          final row = find.byKey(ValueKey('campus-course-${course.id}'));
          await until(() => row.evaluate().isNotEmpty);
          await tester.ensureVisible(row);
          await tester.pumpAndSettle();
          final l = AppLocalizations.of(
            tester.element(find.byType(CampusCoursePicks)),
          );
          final teacher = courseTeacherLabel(course, CourseCopy(l));
          expect(
            find.descendant(of: row, matching: find.textContaining(teacher)),
            findsOneWidget,
          );
          expect(
            find.descendant(
              of: row,
              matching: find.text(course.ratingAvg?.toStringAsFixed(1) ?? '—'),
            ),
            findsOneWidget,
          );
          await capture('1438-picks-${brightness.name}-${course.id}');
          expect(row.hitTestable(), findsOneWidget);
          await tester.tap(row);
          await until(
            () => find.byType(CourseDetailPage).evaluate().isNotEmpty,
          );
          expect(
            tester
                .widget<CourseDetailPage>(find.byType(CourseDetailPage))
                .courseId,
            course.id,
          );
          await until(() => find.textContaining(teacher).evaluate().isNotEmpty);
          await capture('1438-detail-${brightness.name}-${course.id}');
          await tester.tap(
            find
                .descendant(
                  of: find.byType(CourseDetailPage),
                  matching: find.byTooltip(l.commonBack),
                )
                .first,
          );
          await tester.pumpAndSettle();
        }
      }
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      container.dispose();
    },
    skip: !const bool.fromEnvironment('YOURTJ_TEST_LIVE_COURSE_PICKS'),
  );
}
