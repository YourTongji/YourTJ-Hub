import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('Android route uses the declared content cadence', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(
          Brightness.light,
        ).copyWith(platform: TargetPlatform.android),
        home: const Scaffold(body: Text('Home')),
      ),
    );
    final route = MaterialPageRoute<void>(
      builder: (_) => const Scaffold(body: Text('Detail')),
    );
    tester.state<NavigatorState>(find.byType(Navigator)).push(route);
    await tester.pump();
    expect(route.transitionDuration, const Duration(milliseconds: 220));
    await tester.pumpAndSettle();
  });

  testWidgets('feedback stays mounted during its short exit', (tester) async {
    late BuildContext host;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Builder(
          builder: (context) {
            host = context;
            return const Scaffold(body: Text('Home'));
          },
        ),
      ),
    );
    showGfToast(host, 'Saved');
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Close'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(find.text('Saved'), findsOneWidget);
    await tester.pumpAndSettle();
    expect(find.text('Saved'), findsNothing);
  });

  testWidgets('old feedback timeout cannot remove its replacement', (
    tester,
  ) async {
    late BuildContext host;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Builder(
          builder: (context) {
            host = context;
            return const Scaffold();
          },
        ),
      ),
    );
    showGfToast(host, 'First');
    await tester.pumpAndSettle();
    await tester.pump(const Duration(seconds: 3));
    showGfToast(host, 'Latest', error: true);
    await tester.pumpAndSettle();
    await tester.pump(const Duration(seconds: 2));
    expect(find.text('Latest'), findsOneWidget);
    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();
  });
}
