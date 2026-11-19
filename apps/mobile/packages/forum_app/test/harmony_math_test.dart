import 'package:flutter/material.dart';
import 'package:flutter_math_fork/flutter_math.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'selectable formulas retain rendering and touch selection on every platform',
    (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(child: SelectableMath.tex(r'\frac{a}{b}+x^2')),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(SelectableMath), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.tap(find.byType(SelectableMath));
      await tester.pumpAndSettle();
      await tester.longPress(find.byType(SelectableMath));
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
    },
    variant: TargetPlatformVariant.all(),
  );
}
