import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  group('GfMotion', () {
    test('durations mirror web motion.ts', () {
      // motion.ts: standard 0.22s, emphasized 0.28s; the brief pins
      // standard 0.12s / emphasized 0.28s.
      expect(GfMotion.standardDuration, const Duration(milliseconds: 120));
      expect(GfMotion.emphasizedDuration, const Duration(milliseconds: 280));
    });

    test('standard curve is an ease-out cubic', () {
      expect(GfMotion.standard, Curves.easeOutCubic);
    });

    test('emphasized curve is a custom cubic bezier', () {
      expect(GfMotion.emphasized, isA<Cubic>());
    });
  });

  group('gfThemeData', () {
    test('registers Gf theme extensions for both brightnesses', () {
      for (final Brightness brightness in <Brightness>[Brightness.light, Brightness.dark]) {
        final data = gfThemeData(brightness);
        expect(data.extension<GfRadii>(), GfRadii.standard);
        expect(data.extension<GfBorders>(), GfBorders.standard);
        expect(data.extension<GfSizes>(), GfSizes.standard);
      }
    });

    test('surfaces map to the Gf base-100 color', () {
      expect(gfThemeData(Brightness.light).colorScheme.surface, GfColors.light.base100);
      expect(gfThemeData(Brightness.dark).colorScheme.surface, GfColors.dark.base100);
    });

    test('primary maps to the Gf primary color', () {
      expect(gfThemeData(Brightness.light).colorScheme.primary, GfColors.light.primary);
      expect(gfThemeData(Brightness.dark).colorScheme.primary, GfColors.dark.primary);
    });
  });
}
