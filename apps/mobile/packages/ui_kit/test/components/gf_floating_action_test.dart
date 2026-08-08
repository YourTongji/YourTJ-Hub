import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfFloatingAction', () {
    testWidgets('circular variant builds and fires onPressed in both themes',
        (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        int taps = 0;
        await tester.pumpWidget(
          gfApp(
            GfFloatingAction(
              onPressed: () => taps++,
              bottomInset: 0,
            ),
            brightness: brightness,
          ),
        );
        expect(find.byIcon(Icons.edit), findsOneWidget);
        await tester.tap(find.byIcon(Icons.edit));
        expect(taps, 1);
      });
    });

    testWidgets('pill variant renders label with icon', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfFloatingAction(
            onPressed: () {},
            label: '参与讨论',
            icon: Icons.chat_bubble_outline,
          ),
        ),
      );
      expect(find.text('参与讨论'), findsOneWidget);
      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
    });

    testWidgets('disabled variant blocks taps', (tester) async {
      int taps = 0;
      await tester.pumpWidget(
        gfApp(
          GfFloatingAction(
            onPressed: () => taps++,
            enabled: false,
          ),
        ),
      );
      await tester.tap(find.byIcon(Icons.edit));
      expect(taps, 0);
    });
  });
}
