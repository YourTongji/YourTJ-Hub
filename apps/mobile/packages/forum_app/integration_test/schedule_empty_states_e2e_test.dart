import 'dart:convert';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import '../test/pages_behavior_test.dart' show MemTokenStorage;
import '../test/schedule_page_test.dart' show detail;

// Full app, live public PK reads, native keyboard and encrypted local storage.
// Only the first search per theme gets a controlled transport failure; retry
// reaches the real dev API. Guest mode prevents cloud writes.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'scheduler setup and search recovery preserve the local plan',
    (tester) async {
      expect(AppConfig.apiBaseUrl, 'https://dev.yourtj.de');
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(MemTokenStorage()),
          pushBootstrapProvider.overrideWith((_) {}),
        ],
      );
      Future<void> until(bool Function() ready) async {
        final end = DateTime.now().add(const Duration(seconds: 45));
        while (!ready() && DateTime.now().isBefore(end)) {
          await tester.pump(const Duration(milliseconds: 100));
        }
        expect(ready(), isTrue);
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
      }

      Future<void> tap(Finder target) async {
        await tester.ensureVisible(target);
        await tester.pumpAndSettle();
        expect(target.hitTestable(), findsOneWidget);
        await tester.tap(target);
        await tester.pumpAndSettle();
      }

      Future<void> capture(String name) async {
        await tester.pumpAndSettle();
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot(name);
        }
      }

      var majorReads = 0;
      var searchReads = 0;
      var failNext = false;
      container
          .read(apiClientProvider)
          .dio
          .interceptors
          .add(
            InterceptorsWrapper(
              onRequest: (options, handler) {
                if (options.path.endsWith('/courses-by-major')) majorReads++;
                if (options.path.endsWith('/course-search')) {
                  searchReads++;
                  if (failNext) {
                    failNext = false;
                    handler.reject(
                      DioException(
                        requestOptions: options,
                        type: DioExceptionType.connectionError,
                        error: StateError('controlled acceptance failure'),
                      ),
                    );
                    return;
                  }
                }
                handler.next(options);
              },
            ),
          );
      appRouter.go('/schedule');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const GfApp(locale: Locale('zh')),
        ),
      );
      final store = container.read(scheduleStoreProvider.notifier);
      await store.ready;
      await until(() => find.byType(SchedulePage).evaluate().isNotEmpty);
      expect(
        store.state.plans.every(
          (p) => p.stagedCourses.isEmpty && p.customEvents.isEmpty,
        ),
        isTrue,
        reason: 'Refuse to replace unrelated local work',
      );
      await store.setMajorSelection(PkMajorSelection());
      await tester.pumpAndSettle();
      final l = AppLocalizations.of(tester.element(find.byType(SchedulePage)));
      expect(find.text(l.scheduleSetupTitle), findsOneWidget);
      await tap(find.byKey(const Key('schedule-configure-scope')));
      expect(store.state.isConfigCollapsed, isFalse);
      expect(find.text(l.scheduleTerm).first.hitTestable(), findsOneWidget);
      await capture('1441-setup');
      final pk = container.read(pkRepositoryProvider);
      final terms = await pk.calendars();
      final term = terms.firstWhere(
        (t) => t.calendarName.contains('2026-2027'),
        orElse: () => terms.first,
      );
      final grades = await pk.grades(calendarId: term.calendarId);
      final grade = grades.gradeList.contains(2026)
          ? 2026
          : grades.gradeList.first;
      final majors = await pk.majors(grade: grade, calendarId: term.calendarId);
      await store.setMajorSelection(
        PkMajorSelection(
          calendarId: term.calendarId,
          grade: grade,
          major: majors.first.code,
          majorName: majors.first.name,
        ),
      );
      await until(() => majorReads > 0);
      store.selectClass(detail('DUOSTATE.01'), 'Duo empty-state acceptance');
      store.addCustomEvent(
        label: 'Duo empty-state acceptance',
        day: 1,
        sections: [3],
        weeks: [1],
      );
      await store.flush;
      final before = jsonEncode(
        store.state.plans.map((p) => p.toJson()).toList(),
      );
      for (final brightness in Brightness.values) {
        container
            .read(themeModeProvider.notifier)
            .setMode(
              brightness == Brightness.dark ? ThemeMode.dark : ThemeMode.light,
            );
        await tester.pumpAndSettle();
        if (find.byType(TextField).evaluate().isEmpty) {
          await tap(find.text(l.commonSearch));
        }
        final input = find.byType(TextField);
        await tap(input);
        await until(() => tester.view.viewInsets.bottom > 0);
        failNext = true;
        final calls = searchReads;
        final query = 'Duo nonexistent course ${brightness.name} 1441';
        await tester.enterText(input, query);
        await until(() => find.byType(GfErrorRetry).evaluate().isNotEmpty);
        expect(searchReads, calls + 1);
        expect(find.byKey(const Key('schedule-search-no-match')), findsNothing);
        expect(tester.widget<TextField>(input).controller!.text, query);
        FocusManager.instance.primaryFocus?.unfocus();
        await until(() => tester.view.viewInsets.bottom == 0);
        await capture('1441-error-${brightness.name}');
        await tap(find.text(l.commonRetry));
        await until(
          () => find
              .byKey(const Key('schedule-search-no-match'))
              .evaluate()
              .isNotEmpty,
        );
        expect(searchReads, calls + 2);
        expect(find.text(l.scheduleSearchNoMatch(query)), findsOneWidget);
        expect(
          find.text(l.scheduleSearchScope(term.calendarName)),
          findsOneWidget,
        );
        expect(tester.widget<TextField>(input).controller!.text, query);
        await tester.ensureVisible(
          find.byKey(const Key('schedule-clear-query')),
        );
        await tester.pumpAndSettle();
        await capture('1441-zero-${brightness.name}');
        await tap(find.byKey(const Key('schedule-clear-query')));
        expect(tester.widget<TextField>(input).controller!.text, isEmpty);
        expect(find.text(l.scheduleSearchReady), findsOneWidget);
        expect(
          jsonEncode(store.state.plans.map((p) => p.toJson()).toList()),
          before,
        );
      }
      store.removeCourse('DUOSTATE');
      for (final event in store.activePlan.customEvents.toList()) {
        if (event.label == 'Duo empty-state acceptance') {
          store.removeCustomEvent(event.id);
        }
      }
      await store.flush;
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      container.dispose();
    },
    skip: !const bool.fromEnvironment('YOURTJ_TEST_LIVE_SCHEDULE_STATES'),
  );
}
