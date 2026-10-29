import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('navigation uses usable width and consumes each side once', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    late MediaQueryData contentMedia;
    late bool hasRail;
    final app = MaterialApp(
      theme: gfThemeData(Brightness.light),
      home: ReadingWindow(
        navigationRailWidth: 88,
        rail: const SizedBox.expand(key: Key('rail')),
        bottomNavigation: const SizedBox(height: 56, key: Key('bottom')),
        child: Builder(
          builder: (context) {
            contentMedia = MediaQuery.of(context);
            hasRail = ReadingWindowScope.hasRailOf(context);
            return const SafeArea(child: SizedBox.expand(key: Key('content')));
          },
        ),
      ),
    );

    // Synthetic asymmetric windows, not hard-coded device specifications.
    for (final scenario in [
      (width: 640.0, left: 0.0, right: 96.0, rail: false),
      (width: 640.0, left: 96.0, right: 0.0, rail: false),
      (width: 840.0, left: 32.0, right: 76.0, rail: true),
      (width: 840.0, left: 76.0, right: 32.0, rail: true),
      (width: 1440.0, left: 40.0, right: 80.0, rail: true),
    ]) {
      tester.view.physicalSize = Size(scenario.width, 700);
      final insets = FakeViewPadding(
        top: 24,
        bottom: 20,
        left: scenario.left,
        right: scenario.right,
      );
      tester.view.padding = insets;
      tester.view.viewPadding = insets;
      await tester.pumpWidget(app);
      await tester.pumpAndSettle();
      expect(hasRail, scenario.rail);
      expect(
        find.byKey(const Key('rail')),
        scenario.rail ? findsOneWidget : findsNothing,
      );
      expect(
        find.byKey(const Key('bottom')),
        scenario.rail ? findsNothing : findsOneWidget,
      );
      expect(contentMedia.padding.left, 0);
      expect(contentMedia.padding.right, 0);
      expect(contentMedia.viewPadding.left, 0);
      expect(contentMedia.viewPadding.right, 0);
      expect(contentMedia.padding.top, 24);
      expect(contentMedia.padding.bottom, 20);
      final rect = tester.getRect(find.byKey(const Key('content')));
      final available = scenario.width - scenario.left - scenario.right;
      final workspaceMargin = scenario.rail
          ? ((available - 88 - 1120) / 2).clamp(0.0, double.infinity)
          : 0.0;
      expect(
        rect.left,
        scenario.left + workspaceMargin + (scenario.rail ? 89 : 0),
      );
      expect(
        rect.width,
        (available - workspaceMargin * 2 - (scenario.rail ? 88 : 0)).clamp(
              0.0,
              720.0,
            ) -
            (scenario.rail ? 2 : 0),
      );
      expect(rect.right, lessThanOrEqualTo(scenario.width - scenario.right));
      expect(tester.takeException(), isNull);
    }
  });
}
