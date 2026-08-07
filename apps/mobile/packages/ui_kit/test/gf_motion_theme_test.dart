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

    test('standard ease is the exact web cubic-bezier(0.22,1,0.36,1)', () {
      expect(GfMotion.standardEase, const Cubic(0.22, 1.0, 0.36, 1.0));
    });

    test('emphasized ease is the exact web cubic-bezier(0.16,1,0.3,1)', () {
      expect(GfMotion.emphasizedEase, const Cubic(0.16, 1.0, 0.3, 1.0));
    });

    test('motion transitions pair durations with the web easing', () {
      expect(GfMotion.fastMotion.duration, GfMotion.fast);
      expect(GfMotion.fastMotion.ease, GfMotion.standardEase);
      expect(GfMotion.contentMotion.duration, GfMotion.content);
      expect(GfMotion.contentMotion.ease, GfMotion.standardEase);
      expect(GfMotion.standardMotion.duration, GfMotion.standard);
      expect(GfMotion.standardMotion.ease, GfMotion.standardEase);
      expect(GfMotion.comfortableMotion.duration, GfMotion.comfortable);
      expect(GfMotion.comfortableMotion.ease, GfMotion.emphasizedEase);
    });

    test('scene durations mirror motion.css', () {
      expect(GfMotion.menuDuration, const Duration(milliseconds: 140));
      expect(GfMotion.modalDuration, const Duration(milliseconds: 160));
      expect(GfMotion.flashDuration, const Duration(milliseconds: 280));
      expect(GfMotion.floatingReplyDuration, const Duration(milliseconds: 180));
      expect(GfMotion.localExpandDuration, const Duration(milliseconds: 160));
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
        expect(data.extension<GfShadows>(), GfShadows.standard);
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
