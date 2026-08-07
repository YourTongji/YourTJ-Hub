import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// The authoritative 29-token key set, mirrored from
/// `apps/gooseforum/resource/packages/client/src/contracts/payload.ts`
/// (`siteThemeTokenKeys`).
const List<String> siteThemeTokenKeys = <String>[
  'color-base-100',
  'color-base-200',
  'color-base-300',
  'color-base-content',
  'color-icon-muted',
  'color-line',
  'color-primary',
  'color-primary-content',
  'color-secondary',
  'color-secondary-content',
  'color-accent',
  'color-accent-content',
  'color-neutral',
  'color-neutral-content',
  'color-info',
  'color-info-content',
  'color-success',
  'color-success-content',
  'color-warning',
  'color-warning-content',
  'color-error',
  'color-error-content',
  'radius-selector',
  'radius-field',
  'radius-box',
  'size-selector',
  'size-field',
  'border',
  'depth',
];

String colorToHex(Color color) {
  return '#${(color.toARGB32() & 0xFFFFFF).toRadixString(16).padLeft(6, '0')}';
}

void main() {
  final Map<String, dynamic> tokens =
      jsonDecode(File('lib/src/theme/tokens.json').readAsStringSync())
          as Map<String, dynamic>;

  test('tokens.json exposes all 29 keys for light and dark', () {
    expect(tokens.keys.toSet(), <String>{'light', 'dark'});
    for (final String theme in <String>['light', 'dark']) {
      final Map<String, dynamic> themeTokens =
          tokens[theme] as Map<String, dynamic>;
      expect(
        themeTokens.keys.toSet(),
        siteThemeTokenKeys.toSet(),
        reason: '$theme must carry exactly the siteThemeTokenKeys',
      );
      for (final String key in siteThemeTokenKeys) {
        expect(
          themeTokens[key],
          isNotNull,
          reason: '$theme/$key must not be null',
        );
      }
    }
  });

  test('color tokens are valid sRGB hex strings', () {
    for (final String theme in <String>['light', 'dark']) {
      final Map<String, dynamic> themeTokens =
          tokens[theme] as Map<String, dynamic>;
      for (final String key in siteThemeTokenKeys) {
        if (!key.startsWith('color-')) continue;
        final Object value = themeTokens[key]!;
        expect(value, isA<String>(), reason: '$theme/$key');
        expect(
          RegExp(r'^#[0-9a-f]{6}$').hasMatch(value as String),
          isTrue,
          reason: '$theme/$key must be #rrggbb, got $value',
        );
      }
    }
  });

  test('GfColors matches tokens.json for both themes', () {
    for (final String theme in <String>['light', 'dark']) {
      final GfColors colors = theme == 'light' ? GfColors.light : GfColors.dark;
      final Map<String, dynamic> themeTokens =
          tokens[theme] as Map<String, dynamic>;
      colors.asMap.forEach((String name, Color color) {
        final String expected = themeTokens['color-$name']! as String;
        expect(
          colorToHex(color),
          expected,
          reason: 'GfColors.$theme.$name must equal tokens.json color-$name',
        );
      });
    }
  });

  test('numeric tokens match GfRadii / GfBorders / GfSizes', () {
    for (final String theme in <String>['light', 'dark']) {
      final Map<String, dynamic> t = tokens[theme] as Map<String, dynamic>;
      expect(GfRadii.standard.selector, t['radius-selector']);
      expect(GfRadii.standard.field, t['radius-field']);
      expect(GfRadii.standard.box, t['radius-box']);
      expect(GfSizes.standard.selector, t['size-selector']);
      expect(GfSizes.standard.field, t['size-field']);
      expect(GfBorders.standard.width, t['border']);
      expect(GfBorders.standard.depth, t['depth']);
    }
  });

  test('GfColors.forBrightness picks the matching palette', () {
    expect(GfColors.forBrightness(Brightness.light), GfColors.light);
    expect(GfColors.forBrightness(Brightness.dark), GfColors.dark);
  });

  group('GfTypography', () {
    test('registers a scale for both brightnesses', () {
      for (final Brightness brightness in <Brightness>[
        Brightness.light,
        Brightness.dark,
      ]) {
        final GfTypography t = gfThemeData(
          brightness,
        ).extension<GfTypography>()!;
        expect(t.display.fontSize, 22);
        expect(t.title1.fontSize, 20);
        expect(t.title2.fontSize, 18);
        expect(t.title3.fontSize, 17);
        expect(t.heading.fontSize, 16);
        expect(t.body.fontSize, 15);
        expect(t.bodyStrong.fontSize, 15);
        expect(t.small.fontSize, 13);
        expect(t.caption.fontSize, 12);
        expect(t.meta.fontSize, 11);
        expect(t.label.fontSize, 10);
      }
    });

    test('weights match the web conventions', () {
      final GfTypography t = GfTypography.standard(GfColors.light.baseContent);
      expect(t.display.fontWeight, FontWeight.w700);
      expect(t.title1.fontWeight, FontWeight.w700);
      expect(t.title2.fontWeight, FontWeight.w700);
      expect(t.title3.fontWeight, FontWeight.w700);
      expect(t.heading.fontWeight, FontWeight.w600);
      expect(t.body.fontWeight, FontWeight.w400);
      expect(t.bodyStrong.fontWeight, FontWeight.w600);
      expect(t.label.fontWeight, FontWeight.w700);
    });

    test('body uses relaxed line height like web leading-relaxed (1.625)', () {
      final GfTypography t = GfTypography.standard(GfColors.light.baseContent);
      expect(t.body.height, 1.625);
      expect(
        t.display.fontFeatures,
        contains(const FontFeature.tabularFigures()),
      );
    });

    testWidgets('GfTheme.typographyOf falls back when unregistered', (
      tester,
    ) async {
      await tester.pumpWidget(const MaterialApp(home: SizedBox()));
      final BuildContext context = tester.element(find.byType(SizedBox));
      expect(GfTheme.typographyOf(context), isA<GfTypography>());
    });
  });
}
