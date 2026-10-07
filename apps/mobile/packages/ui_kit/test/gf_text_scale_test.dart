import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// Captures the rich-content profile under [scaler] at [width].
Future<GfRichContentTypography> _profileUnder(
  WidgetTester tester, {
  required TextScaler scaler,
  double readingScale = 1,
  double width = 390,
}) async {
  late GfRichContentTypography profile;
  await tester.pumpWidget(
    MaterialApp(
      theme: gfThemeData(Brightness.light),
      home: MediaQuery(
        data: MediaQueryData(size: Size(width, 800), textScaler: scaler),
        child: Builder(
          builder: (context) {
            profile = GfRichContentTypography.of(
              context,
              userScale: readingScale,
            );
            return const SizedBox.shrink();
          },
        ),
      ),
    ),
  );
  return profile;
}

void main() {
  test('app scale multiplies before the system scaler', () {
    const scaler = GfAppTextScaler(
      system: TextScaler.linear(1.5),
      baseline: 16 / 17,
      userScale: 1.1,
    );
    expect(scaler.scale(17), closeTo(16 * 1.1 * 1.5, .001));
    // A nonlinear system scaler receives the pre-multiplied size.
    final uneven = _HalvingAbove20();
    expect(
      const GfAppTextScaler(
        system: TextScaler.noScaling,
        userScale: 1.3,
      ).scale(20),
      closeTo(26, .001),
    );
    expect(
      GfAppTextScaler(system: uneven, userScale: 1.3).scale(20),
      uneven.scale(26),
    );
  });

  test('device baseline adapts Android and narrow windows only', () {
    double baseline(TargetPlatform platform, double side) =>
        GfAppTextScaler.baselineFor(platform: platform, shortestSide: side);
    expect(baseline(TargetPlatform.iOS, 390), 1);
    expect(baseline(TargetPlatform.android, 393), closeTo(16 / 17, .0001));
    expect(
      baseline(TargetPlatform.android, 320),
      closeTo(16 / 17 * .95, .0001),
    );
    expect(baseline(TargetPlatform.iOS, 320), closeTo(.95, .0001));
    // Tablets keep the platform baseline.
    expect(baseline(TargetPlatform.iOS, 820), 1);
  });

  test('app preference is clamped to 90%-130%', () {
    expect(GfAppTextScaler.clampUserScale(.5), .9);
    expect(GfAppTextScaler.clampUserScale(1.2), 1.2);
    expect(GfAppTextScaler.clampUserScale(2), 1.3);
  });

  testWidgets('reading text follows app text and adjusts on top of it', (
    tester,
  ) async {
    const app = GfAppTextScaler(system: TextScaler.noScaling, userScale: 1.3);
    final profile = await _profileUnder(tester, scaler: app, readingScale: 1.2);
    // Painted body = design body × reading preference × app preference.
    expect(app.scale(profile.body.fontSize!), closeTo(17 * 1.2 * 1.3, .001));
    final plain = await _profileUnder(tester, scaler: app);
    expect(app.scale(plain.body.fontSize!), closeTo(17 * 1.3, .001));
  });

  test('heading emphasis narrows as body text grows', () {
    double emphasis(double scale, [double width = 390]) =>
        GfRichContentTypography.headingEmphasisFor(
          renderedScale: scale,
          width: width,
        );
    expect(emphasis(.94), 1);
    expect(emphasis(1), 1);
    expect(emphasis(1.5), closeTo(.7, .001));
    expect(emphasis(2), closeTo(.4, .001));
    expect(emphasis(3), closeTo(.4, .001));
    expect(emphasis(1, 320), closeTo(.85, .001));
  });

  testWidgets('large system text keeps headings ordered but closer', (
    tester,
  ) async {
    final normal = await _profileUnder(tester, scaler: TextScaler.noScaling);
    final large = await _profileUnder(
      tester,
      scaler: const TextScaler.linear(2),
    );
    expect(normal.h1.fontSize, closeTo(17 * 1.45, .001));
    expect(large.body.fontSize, 17);
    expect(large.h1.fontSize, closeTo(17 * (1 + .45 * .4), .001));
    expect(large.h1.fontSize, greaterThan(large.h2.fontSize!));
    expect(large.h2.fontSize, greaterThan(large.h3.fontSize!));
    expect(large.h3.fontSize, greaterThan(large.h4.fontSize!));
    expect(large.h4.fontSize, greaterThan(large.body.fontSize!));
  });

  testWidgets('defaultOf drops only the user preference', (tester) async {
    late TextScaler fallback;
    await tester.pumpWidget(
      MediaQuery(
        data: const MediaQueryData(
          textScaler: GfAppTextScaler(
            system: TextScaler.linear(1.2),
            baseline: .9,
            userScale: 1.3,
          ),
        ),
        child: Builder(
          builder: (context) {
            fallback = GfAppTextScaler.defaultOf(context);
            return const SizedBox.shrink();
          },
        ),
      ),
    );
    expect(
      fallback,
      const GfAppTextScaler(system: TextScaler.linear(1.2), baseline: .9),
    );
  });
}

/// Nonlinear stand-in: sizes above 20 grow at half rate.
class _HalvingAbove20 extends TextScaler {
  @override
  double scale(double fontSize) =>
      fontSize <= 20 ? fontSize : 20 + (fontSize - 20) / 2;

  @override
  // ignore: deprecated_member_use
  double get textScaleFactor => 1;
}
