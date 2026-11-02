import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'campus_native_test.dart' show campusTestApp;
import 'fixtures/campus_fixtures.dart';

void main() {
  testWidgets(
    'unbound timetable explains the prerequisite and public next steps',
    (tester) async {
      final repo = FakeCampusRepository()
        ..current = const CampusStatus(
          enabled: true,
          binding: null,
          candidate: null,
        );
      await tester.pumpWidget(campusTestApp(repo));
      await tester.pumpAndSettle();
      final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
      await tester.ensureVisible(find.text(l.campusTimetable));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l.campusTimetable));
      await tester.pumpAndSettle();
      expect(find.text('尚未连接同济身份'), findsOneWidget);
      expect(find.byKey(const Key('campus-timetable-courses')), findsOneWidget);
      expect(
        find.byKey(const Key('campus-timetable-schedule')),
        findsOneWidget,
      );
      expect(find.text(l.campusBind), findsOneWidget);
      expect(repo.requested, isEmpty);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );
  for (final lang in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets(
        'unbound timetable actions fit $lang $brightness at large text',
        (tester) async {
          tester.view.devicePixelRatio = 1;
          tester.view.physicalSize = Size(
            brightness == Brightness.dark ? 320 : 951,
            678,
          );
          addTearDown(tester.view.reset);
          final repo = FakeCampusRepository()
            ..current = const CampusStatus(
              enabled: true,
              binding: null,
              candidate: null,
            );
          await tester.pumpWidget(
            campusTestApp(
              repo,
              locale: Locale(lang),
              brightness: brightness,
              scale: 2,
            ),
          );
          await tester.pumpAndSettle();
          final l = AppLocalizations.of(
            tester.element(find.byType(CampusPage)),
          );
          await tester.ensureVisible(find.text(l.campusTimetable));
          await tester.pumpAndSettle();
          await tester.tap(find.text(l.campusTimetable));
          await tester.pumpAndSettle();
          expect(find.text(l.campusTimetableNotConnected), findsOneWidget);
          expect(find.text(l.campusTimetableConnectBody), findsOneWidget);
          final publicTool = find.byKey(const Key('campus-timetable-schedule'));
          await tester.ensureVisible(publicTool);
          await tester.pumpAndSettle();
          expect(publicTool.hitTestable(), findsOneWidget);
          expect(repo.requested, isEmpty);
          expect(tester.takeException(), isNull);
          await tester.pumpWidget(const SizedBox.shrink());
          await tester.pumpAndSettle();
        },
      );
    }
  }
}
