import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('ring preserves the image edges instead of zooming the crop', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 3;
    addTearDown(tester.view.resetDevicePixelRatio);
    final cache = PaintingBinding.instance.imageCache;
    cache.clear();
    cache.clearLiveImages();
    addTearDown(() {
      cache.clear();
      cache.clearLiveImages();
    });

    // Mark the outer 8% of a square portrait. Painting the image underneath
    // a 2px ring hides these markers at chat sizes; scaling it inside the
    // ring preserves them. No network or platform golden baseline is needed.
    final recorder = ui.PictureRecorder();
    final canvas = Canvas(recorder);
    canvas.drawColor(const Color(0xff00ff00), BlendMode.src);
    canvas.drawRect(
      const Rect.fromLTWH(0, 0, 24, 300),
      Paint()..color = Colors.blue,
    );
    canvas.drawRect(
      const Rect.fromLTWH(276, 0, 24, 300),
      Paint()..color = Colors.red,
    );
    final picture = recorder.endRecording();
    final source = (await tester.runAsync(() => picture.toImage(300, 300)))!;
    picture.dispose();
    addTearDown(source.dispose);

    for (final brightness in Brightness.values) {
      for (final size in <double>[24, 32, 36, 40]) {
        final url = 'https://example.test/edge-markers-$size.png';
        final provider = GfAvatar.imageProviderFor(
          url,
          size: size,
          devicePixelRatio: 3,
        )!;
        final key = await provider.obtainKey(ImageConfiguration.empty);
        cache.putIfAbsent(
          key,
          () => OneFrameImageStreamCompleter(
            Future.value(ImageInfo(image: source.clone())),
          ),
        );
        final boundaryKey = GlobalKey();
        await tester.pumpWidget(
          gfApp(
            RepaintBoundary(
              key: boundaryKey,
              child: GfAvatar(src: url, size: size, ring: true),
            ),
            brightness: brightness,
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.getSize(find.byType(GfAvatar)), Size.square(size));
        final boundary = tester.renderObject<RenderRepaintBoundary>(
          find.byKey(boundaryKey),
        );
        final rendered = (await tester.runAsync(
          () => boundary.toImage(pixelRatio: 4),
        ))!;
        final pixels = (await tester.runAsync(
          () => rendered.toByteData(format: ui.ImageByteFormat.rawRgba),
        ))!;
        final x = ((2 + (size - 4) * 0.04) * 4).floor();
        final y = (size * 2).floor();
        final left = (y * rendered.width + x) * 4;
        final right = (y * rendered.width + rendered.width - 1 - x) * 4;
        expect(
          pixels.getUint32(left),
          0x2196f3ff,
          reason: '$size px $brightness: the left image edge remains visible',
        );
        expect(
          pixels.getUint32(right),
          0xf44336ff,
          reason: '$size px $brightness: the right image edge remains visible',
        );
        rendered.dispose();
      }
    }
  });
}
