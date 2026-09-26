import 'dart:ui' show SemanticsAction, Tristate;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  for (final brightness in Brightness.values) {
    testWidgets('glass clips blur to its control in $brightness', (
      tester,
    ) async {
      for (final cover in [Colors.white, Colors.black]) {
        await tester.pumpWidget(
          gfApp(
            ColoredBox(
              color: cover,
              child: const SizedBox(
                width: 300,
                height: 240,
                child: Center(
                  child: GfGlassSurface(size: 20, child: GfSymbol('ellipsis')),
                ),
              ),
            ),
            brightness: brightness,
          ),
        );
        final surface = find.byType(GfGlassSurface);
        expect(tester.getSize(surface), const Size(44, 44));
        final clip = find.descendant(
          of: surface,
          matching: find.byType(ClipOval),
        );
        final blur = find.descendant(
          of: surface,
          matching: find.byType(BackdropFilter),
        );
        expect(tester.getRect(clip), tester.getRect(surface));
        expect(tester.getRect(blur), tester.getRect(surface));
        final decoration =
            tester
                    .widget<DecoratedBox>(
                      find.descendant(
                        of: surface,
                        matching: find.byType(DecoratedBox),
                      ),
                    )
                    .decoration
                as BoxDecoration;
        expect(decoration.shape, BoxShape.circle);
        expect(decoration.border, isNotNull);
        final effectiveFill = Color.alphaBlend(decoration.color!, cover);
        final contrast = 1.05 / (effectiveFill.computeLuminance() + .05);
        expect(contrast, greaterThanOrEqualTo(3));
        expect(
          IconTheme.of(tester.element(find.byType(GfSymbol))).color,
          Colors.white,
        );
        expect(tester.takeException(), isNull);
      }
    });
  }

  for (final enabled in [true, false]) {
    testWidgets(
      'glass action preserves name and enabled semantics ($enabled)',
      (tester) async {
        var taps = 0;
        final semantics = tester.ensureSemantics();
        try {
          await tester.pumpWidget(
            gfApp(
              GfGlassIconButton(
                symbol: 'chevron-left',
                tooltip: 'Back',
                onPressed: enabled ? () => taps++ : null,
              ),
            ),
          );
          final target = find.byType(GfGlassIconButton);
          expect(tester.getSize(target), const Size(44, 44));
          expect(find.byTooltip('Back'), findsOneWidget);
          final node = tester.getSemantics(find.byType(GfIconButton));
          expect(node.label, 'Back');
          expect(node.flagsCollection.isButton, isTrue);
          expect(
            node.flagsCollection.isEnabled,
            enabled ? Tristate.isTrue : Tristate.isFalse,
          );
          expect(
            node.getSemanticsData().hasAction(SemanticsAction.tap),
            enabled,
          );
          await tester.tap(target);
          await tester.pumpAndSettle();
          expect(taps, enabled ? 1 : 0);
        } finally {
          semantics.dispose();
        }
      },
    );
  }

  testWidgets('glass surface leaves popup gesture ownership to its child', (
    tester,
  ) async {
    var parentTaps = 0, openings = 0, selections = 0;
    await tester.pumpWidget(
      gfApp(
        GestureDetector(
          onTap: () => parentTaps++,
          child: GfGlassSurface(
            child: PopupMenuButton<int>(
              tooltip: 'More',
              icon: const GfSymbol('ellipsis'),
              onOpened: () => openings++,
              onSelected: (_) => selections++,
              itemBuilder: (_) => const [
                PopupMenuItem(value: 1, child: Text('Choice')),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.byTooltip('More'));
    await tester.pumpAndSettle();
    expect(openings, 1);
    expect(parentTaps, 0);
    await tester.tap(find.text('Choice'));
    await tester.pumpAndSettle();
    expect(selections, 1);
    expect(parentTaps, 0);
  });

  testWidgets('glass remains static with reduced motion', (tester) async {
    for (final reduceMotion in [false, true]) {
      await tester.pumpWidget(
        gfApp(
          MediaQuery(
            data: MediaQueryData(disableAnimations: reduceMotion),
            child: const GfGlassSurface(size: 52, child: GfSymbol('ellipsis')),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.getSize(find.byType(GfGlassSurface)), const Size(52, 52));
      expect(tester.binding.transientCallbackCount, 0);
      expect(tester.takeException(), isNull);
    }
  });
}
