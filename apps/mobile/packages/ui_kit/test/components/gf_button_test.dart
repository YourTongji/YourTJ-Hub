import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfButton', () {
    testWidgets('builds all 7 variants in light and dark', (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        for (final GfButtonVariant variant in GfButtonVariant.values) {
          await tester.pumpWidget(
            gfApp(
              GfButton(
                label: variant.name,
                variant: variant,
                onPressed: () {},
              ),
              brightness: brightness,
            ),
          );
          expect(find.text(variant.name), findsOneWidget);
          expect(
            find.byType(GfButton),
            findsOneWidget,
            reason: '${variant.name} must build in $brightness',
          );
        }
      });
    });

    testWidgets('invokes onPressed when enabled', (tester) async {
      int taps = 0;
      await tester.pumpWidget(
        gfApp(GfButton(label: 'Tap', onPressed: () => taps++)),
      );
      await tester.tap(find.text('Tap'));
      expect(taps, 1);
    });

    testWidgets('disabled button ignores taps and shows 60% opacity',
        (tester) async {
      int taps = 0;
      await tester.pumpWidget(
        gfApp(GfButton(label: 'No', onPressed: null)),
      );
      final Opacity opacity = tester.widget<Opacity>(
        find
            .ancestor(of: find.text('No'), matching: find.byType(Opacity))
            .first,
      );
      expect(opacity.opacity, 0.6);
      await tester.tap(find.text('No'));
      expect(taps, 0);
    });

    testWidgets('loading shows a spinner and blocks taps', (tester) async {
      int taps = 0;
      await tester.pumpWidget(
        gfApp(GfButton(label: 'Wait', loading: true, onPressed: () => taps++)),
      );
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      await tester.tap(find.text('Wait'));
      expect(taps, 0);
    });

    testWidgets('expanded stretches to full width', (tester) async {
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 200,
            child: GfButton(label: 'Wide', expanded: true, onPressed: () {}),
          ),
        ),
      );
      final Size size = tester.getSize(find.byType(GfButton));
      expect(size.width, 200);
    });

    testWidgets('renders optional icon', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfButton(
            label: 'Icon',
            icon: const Icon(Icons.add),
            onPressed: () {},
          ),
        ),
      );
      expect(find.byIcon(Icons.add), findsOneWidget);
    });
  });
}
