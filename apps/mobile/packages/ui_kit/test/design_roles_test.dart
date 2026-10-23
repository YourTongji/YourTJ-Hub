import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  test('small semantic labels keep AA contrast on supported surfaces', () {
    for (final colors in [GfColors.light, GfColors.dark]) {
      for (final tone in GfTone.values.where((t) => t != GfTone.neutral)) {
        for (final surface in [
          colors.base100,
          colors.base200,
          colors.base300,
          colors.surfaceRaised,
        ]) {
          final background = Color.alphaBlend(colors.toneSoft(tone), surface);
          final foreground = colors.toneOnSoft(tone);
          final a = foreground.computeLuminance(),
              b = background.computeLuminance();
          final ratio = (math.max(a, b) + .05) / (math.min(a, b) + .05);
          expect(
            ratio,
            greaterThanOrEqualTo(4.5),
            reason: '${colors.isDark}/$tone',
          );
        }
      }
    }
  });
}
