import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  for (final brightness in Brightness.values) {
    test('modal scrim dims content in $brightness', () {
      final scrim = gfThemeData(brightness).colorScheme.scrim;
      expect(scrim.computeLuminance(), 0);
      expect(scrim.a, greaterThan(0));
      expect(scrim.a, lessThan(1));
    });
    test('button themes distinguish disabled state in $brightness', () {
      final theme = gfThemeData(brightness);
      const enabled = <WidgetState>{};
      const disabled = <WidgetState>{WidgetState.disabled};
      for (final style in [
        theme.filledButtonTheme.style!,
        theme.outlinedButtonTheme.style!,
        theme.textButtonTheme.style!,
      ]) {
        expect(
          style.foregroundColor!.resolve(disabled),
          isNot(style.foregroundColor!.resolve(enabled)),
          reason: 'Unavailable actions must not look enabled.',
        );
      }
      final filled = theme.filledButtonTheme.style!;
      expect(
        filled.backgroundColor!.resolve(disabled),
        isNot(filled.backgroundColor!.resolve(enabled)),
      );
      final outlined = theme.outlinedButtonTheme.style!;
      expect(
        outlined.side!.resolve(disabled)?.color,
        isNot(outlined.side!.resolve(enabled)?.color),
      );
    });
  }
}
