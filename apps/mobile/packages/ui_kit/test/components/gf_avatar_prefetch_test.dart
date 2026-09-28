import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

/// Cache entry stand-in: the tests only assert which keys start a load.
class _StubCompleter extends ImageStreamCompleter {}

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
    // 36 snaps up to the 40 decode step, times the 3x device pixel ratio.
    expect((first as ResizeImage).width, 120);
    expect(first.height, 120);
    expect(first.policy, ResizeImagePolicy.fit);
  });

  test('display sizes of one class share a decoded cache key', () async {
    // The same author avatar is shown at 36 in feed cards, 40 in topic
    // detail headers and 32 in reply rows. Those sizes must collapse to one
    // key, otherwise every page switch downloads and decodes the picture
    // again (issue #879).
    const String url = 'https://example.test/avatar.png';
    final Set<Object> keys = <Object>{};
    for (final double size in <double>[32, 36, 40]) {
      final ImageProvider<Object> provider = GfAvatar.imageProviderFor(
        url,
        size: size,
        devicePixelRatio: 3,
      )!;
      keys.add(await provider.obtainKey(ImageConfiguration.empty));
    }
    expect(keys, hasLength(1));
  });

  test('oversized avatars round the decode size up to a 16-pixel step', () {
    final provider = GfAvatar.imageProviderFor(
      'https://example.test/avatar.png',
      size: 100,
      devicePixelRatio: 3,
    )!;
    // Past the ladder the request rounds up (100 -> 112) instead of being
    // downscaled to the 96 step.
    expect((provider as ResizeImage).width, 336);
    expect(provider.height, 336);
  });

  test('non-positive sizes snap to the smallest decode step', () {
    for (final double size in <double>[0, -8]) {
      final provider = GfAvatar.imageProviderFor(
        'https://example.test/avatar.png',
        size: size,
        devicePixelRatio: 2,
      )!;
      // A degenerate size still lands on the shared 24 step instead of
      // decoding a one-pixel image.
      expect((provider as ResizeImage).width, 48);
      expect(provider.height, 48);
    }
  });

  test('home prefetch seeds the entries the following pages display', () async {
    const String url = 'https://example.test/avatar.png';
    final ImageProvider<Object> prefetch = GfAvatar.imageProviderFor(
      url,
      size: 36,
      devicePixelRatio: 3,
    )!;
    final Object prefetchKey = await prefetch.obtainKey(
      ImageConfiguration.empty,
    );
    for (final double size in <double>[32, 40]) {
      final ImageProvider<Object> display = GfAvatar.imageProviderFor(
        url,
        size: size,
        devicePixelRatio: 3,
      )!;
      expect(
        await display.obtainKey(ImageConfiguration.empty),
        prefetchKey,
        reason: 'size $size must reuse the precached entry',
      );
    }
  });

  testWidgets('repeated page visits load a shared avatar once', (tester) async {
    final ImageCache cache = PaintingBinding.instance.imageCache;
    cache.clear();
    cache.clearLiveImages();
    addTearDown(() {
      cache.clear();
      cache.clearLiveImages();
    });
    const String url = 'https://example.test/avatar.png';
    int loads = 0;
    // Mirrors ImageProvider.resolve: the first visit for a key runs the
    // loader, later visits reuse the completer the cache already holds.
    Future<bool> visit(double size) async {
      final ImageProvider<Object> provider = GfAvatar.imageProviderFor(
        url,
        size: size,
        devicePixelRatio: 2,
      )!;
      final Object key = await provider.obtainKey(ImageConfiguration.empty);
      if (cache.containsKey(key)) return false;
      cache.putIfAbsent(key, () {
        loads += 1;
        return _StubCompleter();
      });
      return true;
    }

    expect(await visit(36), isTrue); // feed card
    expect(await visit(40), isFalse, reason: 'topic detail header');
    expect(await visit(32), isFalse, reason: 'reply row');
    expect(loads, 1);
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
