import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfChip', () {
    testWidgets(
      'interactive categories keep a 44px target at large text sizes',
      (tester) async {
        var taps = 0;
        await tester.pumpWidget(
          gfApp(
            MediaQuery(
              data: const MediaQueryData(textScaler: TextScaler.linear(2)),
              child: SizedBox(
                width: 160,
                child: GfChip(
                  label: 'Campusleben',
                  color: Colors.blue,
                  onTap: () => taps++,
                ),
              ),
            ),
          ),
        );
        final box = tester.getRect(find.byType(GfChip));
        expect(box.height, greaterThanOrEqualTo(44));
        await tester.tapAt(box.topCenter + const Offset(0, 2));
        expect(taps, 1);
        expect(tester.takeException(), isNull);
      },
    );

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
        gfApp(GfChip(label: 'tap', color: Colors.red, onTap: () => taps++)),
      );
      await tester.tap(find.text('tap'));
      expect(taps, 1);
    });
  });
}
