import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfChip', () {
    testWidgets('renders label with color dot in both themes', (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            GfChip(label: '公告', color: const Color(0xFF2563EB)),
            brightness: brightness,
          ),
        );
        expect(find.text('公告'), findsOneWidget);
        // The leading color dot: a circle-shaped container holding the
        // category color.
        final Container dot = tester.widget<Container>(
          find.descendant(
            of: find.byType(GfChip),
            matching: find.byWidgetPredicate(
              (Widget w) =>
                  w is Container &&
                  w.decoration is BoxDecoration &&
                  (w.decoration as BoxDecoration).shape == BoxShape.circle,
            ),
          ),
        );
        final BoxDecoration decoration = dot.decoration as BoxDecoration;
        expect(decoration.color, const Color(0xFF2563EB));
      });
    });

    testWidgets('onTap fires when provided', (tester) async {
      int taps = 0;
      await tester.pumpWidget(
        gfApp(
          GfChip(
            label: 'tap',
            color: Colors.red,
            onTap: () => taps++,
          ),
        ),
      );
      await tester.tap(find.text('tap'));
      expect(taps, 1);
    });
  });
}
