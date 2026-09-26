import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  test('empty avatars do not schedule network prefetch', () {
    expect(
      GfAvatar.imageProviderFor('', size: 36, devicePixelRatio: 3),
      isNull,
    );
  });

  test('prefetch keys share URL, decoded dimensions and fit policy', () async {
    final first = GfAvatar.imageProviderFor(
      'https://example.test/avatar.png',
      size: 36,
      devicePixelRatio: 3,
    )!;
    final second = GfAvatar.imageProviderFor(
      'https://example.test/avatar.png',
      size: 36,
      devicePixelRatio: 3,
    )!;
    expect(
      await first.obtainKey(ImageConfiguration.empty),
      await second.obtainKey(ImageConfiguration.empty),
    );
    expect((first as ResizeImage).width, 108);
    expect(first.height, 108);
    expect(first.policy, ResizeImagePolicy.fit);
  });

  testWidgets('native avatar uses the prefetch key and keeps its fallback', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 2;
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      gfApp(const GfAvatar(src: 'https://example.test/avatar.png', size: 36)),
    );
    final image = tester.widget<Image>(find.byType(Image));
    final expected = GfAvatar.imageProviderFor(
      'https://example.test/avatar.png',
      size: 36,
      devicePixelRatio: 2,
    )!;
    expect(
      await image.image.obtainKey(ImageConfiguration.empty),
      await expected.obtainKey(ImageConfiguration.empty),
    );
    expect(image.excludeFromSemantics, isTrue);
    expect(image.errorBuilder, isNotNull);
    expect(image.frameBuilder, isNotNull);
    expect(find.byType(GfSymbol), findsOneWidget);
  });
}
