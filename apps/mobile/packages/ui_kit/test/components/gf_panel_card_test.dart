import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfPanel', () {
    testWidgets('mobile form is full-width, borderless and unrounded',
        (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            SizedBox(
              width: 300,
              child: GfPanel(child: const Text('content')),
            ),
            brightness: brightness,
          ),
        );
        final DecoratedBox box = tester.widget<DecoratedBox>(
          find.byType(DecoratedBox).first,
        );
        final BoxDecoration decoration = box.decoration as BoxDecoration;
        expect(decoration.borderRadius, BorderRadius.zero);
        expect(decoration.border, isNull);
        expect(find.text('content'), findsOneWidget);
      });
    });

    testWidgets('emphasized form restores border and radius', (tester) async {
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 300,
            child: GfPanel(emphasized: true, child: const Text('box')),
          ),
        ),
      );
      final DecoratedBox box = tester.widget<DecoratedBox>(
        find.byType(DecoratedBox).first,
      );
      final BoxDecoration decoration = box.decoration as BoxDecoration;
      expect(decoration.borderRadius, BorderRadius.circular(8));
      expect(decoration.border, isA<Border>());
    });
  });

  group('GfCard', () {
    testWidgets('mobile form keeps bottom divider and no radius',
        (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            SizedBox(
              width: 300,
              child: GfCard(child: const Text('row')),
            ),
            brightness: brightness,
          ),
        );
        expect(find.text('row'), findsOneWidget);
        final Material material = tester.widget<Material>(
          find
              .ancestor(of: find.text('row'), matching: find.byType(Material))
              .first,
        );
        expect(material.shape, isNull);
        // Divider container present below the content.
        expect(
          find.descendant(
            of: find.byType(GfCard),
            matching: find.byType(Container),
          ),
          findsWidgets,
        );
      });
    });

    testWidgets('showDivider=false removes the divider', (tester) async {
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 300,
            child: GfCard(showDivider: false, child: const Text('row')),
          ),
        ),
      );
      // No divider container with margin remains.
      expect(
        find.descendant(
          of: find.byType(GfCard),
          matching: find.byWidgetPredicate(
            (Widget w) => w is Container && w.margin != null,
          ),
        ),
        findsNothing,
      );
    });

    testWidgets('emphasized card has border and radius', (tester) async {
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 300,
            child: GfCard(emphasized: true, child: const Text('box')),
          ),
        ),
      );
      final Material material = tester.widget<Material>(
        find
            .ancestor(of: find.text('box'), matching: find.byType(Material))
            .first,
      );
      expect(material.shape, isA<RoundedRectangleBorder>());
    });

    testWidgets('onTap fires', (tester) async {
      int taps = 0;
      await tester.pumpWidget(
        gfApp(
          GfCard(onTap: () => taps++, child: const Text('tap me')),
        ),
      );
      await tester.tap(find.text('tap me'));
      expect(taps, 1);
    });
  });
}
