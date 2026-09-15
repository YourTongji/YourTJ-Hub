import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfButton', () {
    testWidgets('small buttons have a 44px target and room for scaled labels', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          GfButton(label: 'OK', size: GfButtonSize.small, onPressed: () {}),
        ),
      );
      expect(
        tester.getSize(find.byType(ElevatedButton)).height,
        greaterThanOrEqualTo(44),
      );
      await tester.pumpWidget(
        gfApp(
          MediaQuery(
            data: const MediaQueryData(textScaler: TextScaler.linear(2)),
            child: SizedBox(
              width: 180,
              child: GfButton(
                label: 'Änderungen speichern',
                expanded: true,
                onPressed: () {},
              ),
            ),
          ),
        ),
      );
      final text = tester.getRect(find.text('Änderungen speichern'));
      final button = tester.getRect(find.byType(ElevatedButton));
      expect(button.contains(text.topLeft), isTrue);
      expect(button.contains(text.bottomRight), isTrue);
      expect(tester.takeException(), isNull);
    });

    testWidgets('builds all 7 variants in light and dark', (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        for (final GfButtonVariant variant in GfButtonVariant.values) {
          await tester.pumpWidget(
            gfApp(
              GfButton(label: variant.name, variant: variant, onPressed: () {}),
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

    testWidgets(
      'follow SVG inherits the primary button foreground in both themes',
      (tester) async {
        await forEachBrightness(tester, (tester, brightness) async {
          await tester.pumpWidget(
            gfApp(
              GfButton(
                label: 'Follow',
                icon: const GfSymbol('user-round-plus'),
                onPressed: () {},
              ),
              brightness: brightness,
            ),
          );
          await tester.pumpAndSettle();
          final svg = tester.widget<SvgPicture>(find.byType(SvgPicture));
          expect(
            svg.colorFilter,
            ColorFilter.mode(
              GfColors.forBrightness(brightness).primaryContent,
              BlendMode.srcIn,
            ),
          );
        });
      },
    );

    testWidgets('invokes onPressed when enabled', (tester) async {
      int taps = 0;
      await tester.pumpWidget(
        gfApp(GfButton(label: 'Tap', onPressed: () => taps++)),
      );
      await tester.tap(find.text('Tap'));
      expect(taps, 1);
    });

    testWidgets('disabled TDesign button ignores taps', (tester) async {
      int taps = 0;
      await tester.pumpWidget(gfApp(GfButton(label: 'No', onPressed: null)));
      expect(find.byType(td.TButton), findsOneWidget);
      expect(
        tester.widget<td.TButton>(find.byType(td.TButton)).onPressed,
        isNull,
      );
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
