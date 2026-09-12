import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  Future<void> openDetail(
    WidgetTester tester,
    Brightness brightness, {
    bool canPop = true,
  }) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(brightness).copyWith(platform: TargetPlatform.iOS),
        home: const Scaffold(body: Text('Home')),
      ),
    );
    final navigator = tester.state<NavigatorState>(find.byType(Navigator));
    navigator.push(
      MaterialPageRoute<void>(
        builder: (_) => PopScope(
          canPop: canPop,
          child: const Scaffold(body: Text('Detail')),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  for (final brightness in Brightness.values) {
    testWidgets('edge swipe pops the detail in $brightness', (tester) async {
      await openDetail(tester, brightness);
      await tester.dragFrom(const Offset(1, 200), const Offset(700, 0));
      await tester.pumpAndSettle();
      expect(find.text('Detail'), findsNothing);
      expect(find.text('Home'), findsOneWidget);
    });
  }

  testWidgets('short cancelled swipe and body swipe keep the detail', (
    tester,
  ) async {
    await openDetail(tester, Brightness.light);
    final gesture = await tester.startGesture(const Offset(1, 200));
    await gesture.moveBy(const Offset(100, 0));
    await tester.pump(const Duration(milliseconds: 500));
    await gesture.up();
    await tester.pumpAndSettle();
    expect(find.text('Detail'), findsOneWidget);
    await tester.dragFrom(const Offset(200, 200), const Offset(500, 0));
    await tester.pumpAndSettle();
    expect(find.text('Detail'), findsOneWidget);
  });

  testWidgets('unsaved content can veto the edge swipe', (tester) async {
    await openDetail(tester, Brightness.light, canPop: false);
    await tester.dragFrom(const Offset(1, 200), const Offset(700, 0));
    await tester.pumpAndSettle();
    expect(find.text('Detail'), findsOneWidget);
  });
}
