import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import '../test/campus_native_test.dart' show campusTestApp;
import '../test/fixtures/campus_fixtures.dart';

/// Synthetic data only. Exercises the real native widgets without school login.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('native campus overview, notices, timetable and academics', (
    tester,
  ) async {
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    expect(find.textContaining('演示同学'), findsOneWidget);
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    await tester.tap(find.text(l.campusTimetable));
    await tester.pumpAndSettle();
    expect(find.text('课程 1（演示）'), findsOneWidget);
    await tester.tap(find.text(l.campusAcademics).first);
    await tester.pumpAndSettle();
    expect(find.text('综合 GPA'), findsOneWidget);
    await tester.ensureVisible(find.text(l.campusMessages).first);
    await tester.tap(find.text(l.campusMessages).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('图书馆开放时间调整（演示）').first);
    await tester.pumpAndSettle();
    expect(find.byType(SelectableText), findsOneWidget);
    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text(l.campusToday).first);
    await tester.tap(find.text(l.campusToday).first);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
}
