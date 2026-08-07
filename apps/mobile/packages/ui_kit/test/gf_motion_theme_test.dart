import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  group('GfMotion', () {
    test('five durations mirror web motion.ts', () {
      expect(GfMotion.instant, const Duration(milliseconds: 120));
      expect(GfMotion.fast, const Duration(milliseconds: 160));
      expect(GfMotion.content, const Duration(milliseconds: 180));
      expect(GfMotion.standard, const Duration(milliseconds: 220));
      expect(GfMotion.comfortable, const Duration(milliseconds: 280));
    });

    test('legacy aliases keep the old API', () {
      expect(GfMotion.standardDuration, GfMotion.instant);
      expect(GfMotion.emphasizedDuration, GfMotion.comfortable);
    });

    test('standard ease is an ease-out cubic', () {
      expect(GfMotion.standardEase, Curves.easeOutCubic);
    });

    test('emphasized ease is a custom cubic bezier', () {
      expect(GfMotion.emphasizedEase, isA<Cubic>());
    });
  });

  group('gfThemeData', () {
    test('registers Gf theme extensions for both brightnesses', () {
      for (final Brightness brightness in <Brightness>[
        Brightness.light,
        Brightness.dark,
      ]) {
        final data = gfThemeData(brightness);
        expect(data.extension<GfRadii>(), GfRadii.standard);
        expect(data.extension<GfBorders>(), GfBorders.standard);
        expect(data.extension<GfSizes>(), GfSizes.standard);
      }
    });

    test('surfaces map to the Gf base-100 color', () {
      expect(
        gfThemeData(Brightness.light).colorScheme.surface,
        GfColors.light.base100,
      );
      expect(
        gfThemeData(Brightness.dark).colorScheme.surface,
        GfColors.dark.base100,
      );
    });

    test('primary maps to the Gf primary color', () {
      expect(
        gfThemeData(Brightness.light).colorScheme.primary,
        GfColors.light.primary,
      );
      expect(
        gfThemeData(Brightness.dark).colorScheme.primary,
        GfColors.dark.primary,
      );
    });
  });
}
