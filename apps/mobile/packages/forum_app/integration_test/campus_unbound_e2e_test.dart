import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:webview_flutter/webview_flutter.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/admin/admin_page.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/pages/courses/catalog_page.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/theme_mode.dart';

// Uses the user's existing dev test login. Opens then closes school authorization
// without entering school credentials or confirming a binding. No HTTP mocks.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'unbound timetable survives authorization cancellation and opens public tools',
    (tester) async {
      expect(AppConfig.apiBaseUrl, 'https://dev.yourtj.de');
      final container = ProviderContainer(
        overrides: [pushBootstrapProvider.overrideWith((_) {})],
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

      appRouter.go('/campus');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const GfApp(locale: Locale('zh')),
        ),
      );
      expect(
        await container.read(currentUserProvider.future),
        isNotNull,
        reason: 'existing dev test session is required',
      );
      await until(
        () => container.read(campusControllerProvider).status != null,
      );
      expect(container.read(campusControllerProvider).status!.binding, isNull);
      expect(container.read(campusControllerProvider).status!.enabled, isTrue);
      final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
      await tester.tap(find.text(l.campusTimetable));
      await tester.pumpAndSettle();
      for (final brightness in Brightness.values) {
        container
            .read(themeModeProvider.notifier)
            .setMode(
              brightness == Brightness.dark ? ThemeMode.dark : ThemeMode.light,
            );
        await tester.pumpAndSettle();
        expect(find.text(l.campusTimetableNotConnected), findsOneWidget);
        expect(find.text(l.campusTimetableConnectBody), findsOneWidget);
        await capture('1439-unbound-${brightness.name}');
        if (brightness == Brightness.dark) {
          final connect = find.text(l.campusBind);
          await tester.ensureVisible(connect);
          await tester.pumpAndSettle();
          await tester.tap(connect);
          await until(() => find.byType(AdminPage).evaluate().isNotEmpty);
          await until(() => find.byType(WebViewWidget).evaluate().isNotEmpty);
          final authorization = tester
              .widget<AdminPage>(find.byType(AdminPage))
              .campusAuthorizationUrl;
          expect(authorization, isNotNull);
          expect(authorization!.scheme, 'https');
          await capture('1439-authorization-open');
          await tester.tap(
            find
                .descendant(
                  of: find.byType(AdminPage),
                  matching: find.byTooltip(l.commonClose),
                )
                .first,
          );
          await until(() => find.byType(AdminPage).evaluate().isEmpty);
          expect(
            container.read(campusControllerProvider).status?.binding,
            isNull,
          );
          expect(find.text(l.campusTimetableNotConnected), findsOneWidget);
        }
        for (final tool in ['courses', 'schedule']) {
          final entry = find.byKey(Key('campus-timetable-$tool'));
          await tester.ensureVisible(entry);
          await tester.pumpAndSettle();
          expect(entry.hitTestable(), findsOneWidget);
          await tester.tap(entry);
          await until(
            () => tool == 'courses'
                ? find.byType(CourseCatalogPage).evaluate().isNotEmpty
                : find.byType(SchedulePage).evaluate().isNotEmpty,
          );
          await capture('1439-public-$tool-${brightness.name}');
          appRouter.pop();
          await tester.pumpAndSettle();
          expect(find.text(l.campusTimetableNotConnected), findsOneWidget);
        }
      }
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      container.dispose();
    },
    skip: !const bool.fromEnvironment('YOURTJ_TEST_LIVE_UNBOUND'),
  );
}
