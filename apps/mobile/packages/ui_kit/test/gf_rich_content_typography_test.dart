import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import 'helpers.dart';

void main() {
  test('reading profile derives every size from the design baseline', () {
    final profile = GfRichContentTypography.standard(
      typography: GfTypography.standard(GfColors.light.baseContent),
      colors: GfColors.light,
    );

    expect(profile.body.fontSize, GfRichContentTypography.readingBodySize);
    expect(profile.body.fontSize, 17);
    for (var index = 0; index < GfRichContentTypography.headingRatios.length; index++) {
      final ratio = GfRichContentTypography.headingRatios[index];
      expect(
        switch (index) {
          0 => profile.h1.fontSize,
          1 => profile.h2.fontSize,
          2 => profile.h3.fontSize,
          _ => profile.h4.fontSize,
        },
        closeTo(17 * ratio, .001),
      );
    }
    expect(profile.h1.fontSize, greaterThan(profile.h2.fontSize!));
    expect(profile.h2.fontSize, greaterThan(profile.h3.fontSize!));
    expect(profile.h3.fontSize, greaterThan(profile.h4.fontSize!));
    expect(profile.h4.fontSize, greaterThan(profile.body.fontSize!));
    expect(profile.code.fontSize, lessThan(profile.body.fontSize!));
    expect(profile.inlineCode.fontSize, profile.code.fontSize);
    expect(profile.tableBody.fontSize, profile.body.fontSize);
  });

  test('reader preference multiplies the baseline without clamping it away', () {
    final profile = GfRichContentTypography.standard(
      typography: GfTypography.standard(GfColors.light.baseContent),
      colors: GfColors.light,
      userScale: 1.3,
    );

    expect(profile.body.fontSize, closeTo(17 * 1.3, .001));
    expect(profile.h1.fontSize, closeTo(17 * 1.45 * 1.3, .001));
    expect(profile.code.fontSize, closeTo(16 * 1.3, .001));
  });

  test('reader preference is clamped to 80%-140%', () {
    expect(GfRichContentTypography.clampUserScale(.5), .8);
    expect(GfRichContentTypography.clampUserScale(0), .8);
    expect(GfRichContentTypography.clampUserScale(1.1), 1.1);
    expect(GfRichContentTypography.clampUserScale(2), 1.4);
  });

  test('list indent shrinks with the system text scale, clamped to 16-32', () {
    expect(GfRichContentTypography.listIndentFor(1), 32);
    expect(GfRichContentTypography.listIndentFor(1.3), closeTo(32 / 1.3, .001));
    expect(GfRichContentTypography.listIndentFor(2), 16);
    // Beyond the 1x-2x window the indent stops moving in both directions.
    expect(GfRichContentTypography.listIndentFor(3), 16);
    expect(GfRichContentTypography.listIndentFor(.5), 32);
  });

  test('compact profile shares the same ratios at a smaller baseline', () {
    final reading = GfRichContentTypography.standard(
      typography: GfTypography.standard(GfColors.light.baseContent),
      colors: GfColors.light,
    );
    final compact = GfRichContentTypography.standard(
      typography: GfTypography.standard(GfColors.light.baseContent),
      colors: GfColors.light,
      bodySize: GfRichContentTypography.compactBodySize,
    );

    expect(compact.body.fontSize, GfRichContentTypography.compactBodySize);
    expect(compact.body.fontSize, inInclusiveRange(15, 16));
    expect(
      compact.h1.fontSize! / compact.body.fontSize!,
      closeTo(reading.h1.fontSize! / reading.body.fontSize!, .001),
    );
    expect(compact.blockSpacing, reading.blockSpacing);
  });

  testWidgets('profile colors come from the active theme tokens', (
    tester,
  ) async {
    await forEachBrightness(tester, (tester, brightness) async {
      late GfRichContentTypography profile;
      late GfColors colors;
      await tester.pumpWidget(
        gfApp(
          Builder(
            builder: (context) {
              profile = GfRichContentTypography.of(context);
              colors = GfTheme.colorsOf(context);
              return const SizedBox.shrink();
            },
          ),
          brightness: brightness,
        ),
      );

      expect(profile.body.color, colors.baseContent);
      // Token colors must win over the block style, so `code` stays colorless
      // and the plain fallback adds the body color explicitly.
      expect(profile.code.color, isNull);
      expect(profile.inlineCode.color, colors.error);
      expect(profile.quote.color, isNot(colors.baseContent));
    });
  });
}
