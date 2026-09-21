import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  const String onePixelPng =
      'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=';

  test(
    'dark matrix uses unnormalized 255 translations and transforms pixels',
    () {
      expect(GfCaptchaImage.darkColorMatrix[4], 255);
      expect(GfCaptchaImage.darkColorMatrix[9], 255);
      expect(GfCaptchaImage.darkColorMatrix[14], 255);
      expect(GfCaptchaImage.transformDarkPixel(Colors.black), Colors.white);
      expect(GfCaptchaImage.transformDarkPixel(Colors.white), Colors.black);
    },
  );

  testWidgets('shared captcha image switches filter with the theme', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: ThemeData(brightness: Brightness.light),
        home: const GfCaptchaImage(
          key: ValueKey<String>('light-captcha'),
          imageData: onePixelPng,
        ),
      ),
    );
    expect(find.byType(ColorFiltered), findsNothing);

    await tester.pumpWidget(
      MaterialApp(
        theme: ThemeData(brightness: Brightness.dark),
        home: const GfCaptchaImage(
          key: ValueKey<String>('dark-captcha'),
          imageData: onePixelPng,
        ),
      ),
    );
    await tester.pumpAndSettle();
    final ColorFiltered filtered = tester.widget<ColorFiltered>(
      find.byType(ColorFiltered),
    );
    expect(filtered.colorFilter.toString(), contains('255.0'));
  });
}
