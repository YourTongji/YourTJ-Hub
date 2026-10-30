import 'dart:convert';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/schedule/major_picker.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/storage/user_work_database.dart';
import 'package:forum_app/src/theme_mode.dart';
import '../test/pages_behavior_test.dart' show MemTokenStorage;
import '../test/schedule_page_test.dart' show detail;

// Full app, real public dev PK reads and native encrypted local storage. Guest
// mode prevents cloud writes. Only this journey's synthetic local course clears.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'major search preserves plans until explicit confirmation',
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

      Future<void> capture(String name) async {
        await tester.pumpAndSettle();
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot(name);
        }
      }

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
          (p) =>
              p.customEvents.isEmpty &&
              p.stagedCourses.every(
                (c) => c.courseName == 'Duo major acceptance',
              ),
        ),
        isTrue,
        reason: 'Never clear unrelated local work during acceptance',
      );
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
      expect(majors.length, greaterThanOrEqualTo(2));
      final target = majors.firstWhere(
        (m) => m.code == '00304',
        orElse: () => majors[1],
      );
      final previous = majors.firstWhere((m) => m.code != target.code);
      final l = AppLocalizations.of(tester.element(find.byType(SchedulePage)));
      Future<void> open() async {
        if (store.state.isConfigCollapsed) {
          final summary = find
              .textContaining(store.state.majorSelected.majorName!)
              .first;
          await tester.ensureVisible(summary);
          await tester.pumpAndSettle();
          await tester.tap(summary);
          await tester.pumpAndSettle();
        }
        await tester.ensureVisible(find.text(l.scheduleMajor));
        await tester.pumpAndSettle();
        await tester.tap(find.text(l.scheduleMajor));
        await until(
          () => find.byType(ScheduleMajorPicker).evaluate().isNotEmpty,
        );
      }

      Future<Finder> queryTarget() async {
        final search = find.descendant(
          of: find.byKey(const Key('schedule-major-search')),
          matching: find.byType(TextField),
        );
        await tester.enterText(search, target.code);
        final row = find.byKey(
          ValueKey('schedule-major-option-${target.code}'),
        );
        await until(
          () => row.evaluate().isNotEmpty && tester.view.viewInsets.bottom > 0,
        );
        expect(search.hitTestable(), findsOneWidget);
        expect(row.hitTestable(), findsOneWidget);
        final window = MediaQueryData.fromView(tester.view);
        expect(
          tester.getRect(row).bottom,
          lessThanOrEqualTo(window.size.height - window.viewInsets.bottom),
        );
        return row;
      }

      for (final brightness in Brightness.values) {
        container
            .read(themeModeProvider.notifier)
            .setMode(
              brightness == Brightness.dark ? ThemeMode.dark : ThemeMode.light,
            );
        await store.setMajorSelection(
          PkMajorSelection(
            calendarId: term.calendarId,
            grade: grade,
            major: previous.code,
            majorName: previous.name,
          ),
        );
        store.selectClass(detail('DUOE2E.01'), 'Duo major acceptance');
        await store.flush;
        await tester.pumpAndSettle();
        await open();
        await queryTarget();
        expect(store.state.majorSelected.major, previous.code);
        expect(store.activePlan.selectedCourses, ['DUOE2E.01']);
        await capture('1440-search-${brightness.name}');
        await tester.tap(
          find
              .descendant(
                of: find.byType(ScheduleMajorPicker),
                matching: find.byTooltip(l.commonClose),
              )
              .first,
        );
        await until(
          () =>
              find.byType(ScheduleMajorPicker).evaluate().isEmpty &&
              tester.view.viewInsets.bottom == 0,
        );
        expect(store.activePlan.selectedCourses, ['DUOE2E.01']);
        await open();
        await tester.tap(await queryTarget());
        await until(
          () =>
              find.text(l.scheduleConfigChangeTitle).evaluate().isNotEmpty &&
              tester.view.viewInsets.bottom == 0,
        );
        await capture('1440-confirm-${brightness.name}');
        await tester.tap(find.text(l.commonCancel));
        await tester.pumpAndSettle();
        expect(store.state.majorSelected.major, previous.code);
        expect(store.activePlan.selectedCourses, ['DUOE2E.01']);
        await open();
        await tester.tap(await queryTarget());
        await until(
          () =>
              find.text(l.scheduleConfigChangeTitle).evaluate().isNotEmpty &&
              tester.view.viewInsets.bottom == 0,
        );
        await tester.tap(find.text(l.scheduleConfigChangeConfirm));
        await tester.pumpAndSettle();
        await store.flush;
        expect(store.state.majorSelected.major, target.code);
        expect(
          store.state.plans.every(
            (p) => p.stagedCourses.isEmpty && p.selectedCourses.isEmpty,
          ),
          isTrue,
        );
        final values = await container
            .read(userWorkDatabaseProvider)
            .readDomain(writingScope(AppConfig.apiBaseUrl, 0), 'schedule');
        expect(
          (jsonDecode(values[ScheduleStorageKeys.majorSelected]!)
              as Map)['major'],
          target.code,
        );
        await capture('1440-selected-${brightness.name}');
      }
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      container.dispose();
    },
    skip: !const bool.fromEnvironment('YOURTJ_TEST_LIVE_MAJOR_PICKER'),
  );
}
