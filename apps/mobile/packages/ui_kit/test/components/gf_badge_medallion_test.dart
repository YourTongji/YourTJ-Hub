import 'dart:ui' show SemanticsAction;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  for (final brightness in Brightness.values) {
    testWidgets('medallion keeps its layered circular size in $brightness', (
      tester,
    ) async {
      for (final size in [40.0, 56.0, 64.0]) {
        await tester.pumpWidget(
          gfApp(
            GfBadgeMedallion(
              icon: const GfSymbol('award', size: 24),
              color: const Color(0xFF2563EB),
              size: size,
            ),
            brightness: brightness,
          ),
        );
        await tester.pumpAndSettle();
        expect(
          tester.getSize(find.byType(GfBadgeMedallion)),
          Size.square(size),
        );
        final layers = tester
            .widgetList<DecoratedBox>(
              find.descendant(
                of: find.byType(GfBadgeMedallion),
                matching: find.byType(DecoratedBox),
              ),
            )
            .map((layer) => layer.decoration as BoxDecoration)
            .toList();
        expect(layers, hasLength(2));
        for (final layer in layers) {
          expect(layer.shape, BoxShape.circle);
          expect(layer.gradient, isA<LinearGradient>());
          expect(layer.border, isNotNull);
        }
        expect(layers.first.boxShadow, hasLength(1));
        // Fixed dark strokes in server SVGs retain a light backing in either
        // theme; only the rim adapts to the surrounding dark surface.
        final face = layers.last.gradient! as LinearGradient;
        for (final color in face.colors) {
          expect(color.computeLuminance(), greaterThan(.60));
        }
        expect(
          tester.getCenter(find.byType(GfSymbol)),
          tester.getCenter(find.byType(GfBadgeMedallion)),
        );
        expect(tester.takeException(), isNull);
      }
    });

    testWidgets(
      'achievement cards grow with narrow and scaled $brightness text',
      (tester) async {
        for (final (width, scale) in [(136.0, 1.0), (288.0, 2.0)]) {
          await tester.pumpWidget(
            gfApp(
              MediaQuery(
                data: MediaQueryData(textScaler: TextScaler.linear(scale)),
                child: SizedBox(
                  width: width,
                  child: SingleChildScrollView(
                    child: GfAchievementCard(
                      title: 'Community contributor with a long title',
                      description:
                          'Recognizes helpful participation and thoughtful contributions across the community.',
                      icon: const GfSymbol('award', size: 32),
                      color: const Color(0xFF0D9488),
                      onTap: () {},
                    ),
                  ),
                ),
              ),
              brightness: brightness,
            ),
          );
          await tester.pumpAndSettle();
          final badge = tester.getRect(find.byType(GfBadgeMedallion));
          final title = find.text('Community contributor with a long title');
          final titleBounds = tester.getRect(title);
          expect(titleBounds.top, greaterThan(badge.bottom));
          expect((badge.center.dx - titleBounds.center.dx).abs(), lessThan(.1));
          final description = tester.widget<Text>(
            find.textContaining('Recognizes'),
          );
          expect(description.maxLines, 2);
          expect(description.overflow, TextOverflow.ellipsis);
          expect(tester.takeException(), isNull);
        }
      },
    );
  }

  for (final interactive in [false, true]) {
    testWidgets(
      'achievement has one complete accessible label ($interactive)',
      (tester) async {
        var taps = 0;
        final semantics = tester.ensureSemantics();
        try {
          await tester.pumpWidget(
            gfApp(
              SizedBox(
                width: 160,
                child: GfAchievementCard(
                  title: 'First post',
                  description: 'Published your first community discussion.',
                  icon: Semantics(
                    label: 'duplicate artwork label',
                    child: const GfSymbol('square-pen', size: 32),
                  ),
                  color: const Color(0xFF2563EB),
                  onTap: interactive ? () => taps++ : null,
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          final node = tester.getSemantics(find.byType(GfAchievementCard));
          expect(
            node.label,
            'First post\nPublished your first community discussion.',
          );
          expect(node.flagsCollection.isButton, interactive);
          expect(
            node.getSemanticsData().hasAction(SemanticsAction.tap),
            interactive,
          );
          expect(
            find.bySemanticsLabel('duplicate artwork label'),
            findsNothing,
          );
          if (interactive) {
            await tester.tap(find.byType(GfBadgeMedallion));
            expect(taps, 1);
          }
          expect(tester.takeException(), isNull);
        } finally {
          semantics.dispose();
        }
      },
    );
  }
}
