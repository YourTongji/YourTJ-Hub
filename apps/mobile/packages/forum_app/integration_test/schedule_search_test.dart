import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import '../test/schedule_search_test.dart'
    show PendingSearchRepository, openScheduleSearch, searchResult;

/// Runs the real scheduler and platform keyboard with delayed synthetic results.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  for (final brightness in Brightness.values) {
    testWidgets('scheduler search keeps native keyboard (${brightness.name})', (
      tester,
    ) async {
      final repository = PendingSearchRepository();
      await openScheduleSearch(tester, repository, brightness: brightness);
      await tester.tap(find.byType(TextField));
      await tester.enterText(find.byType(TextField), '高');
      await tester.pump(const Duration(milliseconds: 600));
      await tester.pump();
      expect(repository.queries, ['高']);
      expect(find.byType(TextField), findsOneWidget);
      final editable = tester.widget<EditableText>(find.byType(EditableText));
      expect(editable.controller.text, '高');
      expect(editable.focusNode.hasFocus, isTrue);
      expect(
        tester.view.viewInsets.bottom,
        greaterThan(0),
        reason: 'the native software keyboard remains visible',
      );

      await tester.enterText(find.byType(TextField), '高等数学');
      await tester.pump(const Duration(milliseconds: 600));
      await tester.pump();
      expect(repository.queries, ['高', '高等数学']);
      repository.requests.last.complete(searchResult('高等数学（演示结果）'));
      repository.requests.first.complete(searchResult('高（过期结果）'));
      await tester.pumpAndSettle();
      expect(find.text('高等数学（演示结果）'), findsOneWidget);
      expect(find.text('高（过期结果）'), findsNothing);
      expect(editable.controller.text, '高等数学');
      expect(editable.focusNode.hasFocus, isTrue);
      expect(tester.view.viewInsets.bottom, greaterThan(0));
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot('schedule-search-${brightness.name}');
      }
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }
}
