import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import '../test/schedule_task_flow_test.dart' show openTaskFlow, selectTask;
import '../test/schedule_page_test.dart' show detail;

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  const baseline = bool.fromEnvironment('YOURTJ_TASK_FLOW_BASELINE');
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets('scheduler tasks $locale ${brightness.name}', (tester) async {
        final scope = await openTaskFlow(
          tester,
          locale: locale,
          brightness: brightness,
          width: 320,
          scale: 2,
        );
        scope.notifier.selectClass(
          detail('110001.01', room: 'A101'),
          '高等数学 / Analysis',
        );
        scope.notifier.selectClass(
          detail('110002.01', room: 'A102'),
          '大学物理 / Physics',
        );
        await scope.notifier.flush;
        await tester.pumpAndSettle();
        if (!baseline) {
          final l10n = AppLocalizations.of(
            tester.element(find.byType(SchedulePage)),
          );
          await selectTask(tester, l10n.scheduleSelected);
          expect(find.text('高等数学 / Analysis'), findsOneWidget);
          expect(find.textContaining('A101'), findsWidgets);
          await tester.ensureVisible(find.textContaining('A101').first);
          await tester.pumpAndSettle();
        }
        await tester.pump(const Duration(milliseconds: 400));
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot(
            'schedule-tasks-${baseline ? "before" : "after"}-$locale-${brightness.name}',
          );
        }
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        await tester.pumpAndSettle();
      });
    }
  }
}
