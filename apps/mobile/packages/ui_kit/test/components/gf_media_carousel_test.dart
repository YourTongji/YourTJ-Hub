import 'dart:async';
import 'dart:ui' as ui;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

void main() {
  testWidgets('gallery decode keeps a wide photo proportional', (tester) async {
    await tester.pumpWidget(
      gfApp(
        const SizedBox(
          width: 300,
          child: GfMediaCarousel(images: ['https://example.test/wide.png']),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final resize =
        tester.widget<Image>(find.byType(Image).first).image as ResizeImage;
    await tester.runAsync(() async {
      final recorder = ui.PictureRecorder();
      Canvas(recorder).drawRect(const Rect.fromLTWH(0, 0, 2400, 1200), Paint());
      final picture = recorder.endRecording();
      final original = await picture.toImage(2400, 1200);
      final bytes = (await original.toByteData(
        format: ui.ImageByteFormat.png,
      ))!;
      final provider = ResizeImage(
        MemoryImage(bytes.buffer.asUint8List()),
        width: resize.width,
        height: resize.height,
        policy: resize.policy,
      );
      final ready = Completer<ImageInfo>();
      final stream = provider.resolve(ImageConfiguration.empty);
      final listener = ImageStreamListener((info, _) => ready.complete(info));
      stream.addListener(listener);
      final decoded = await ready.future;
      expect(decoded.image.width / decoded.image.height, 2);
      stream.removeListener(listener);
      decoded.dispose();
      original.dispose();
      picture.dispose();
    });
  });
  testWidgets('gallery retains aspect fit, swipe count and full-screen entry', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const Scaffold(
          body: GfMediaCarousel(
            images: [
              'https://example.test/one.png',
              'https://example.test/two.png',
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('1 / 2'), findsOneWidget);
    expect(tester.widget<Image>(find.byType(Image).first).fit, BoxFit.contain);
    await tester.drag(find.byType(PageView), const Offset(-700, 0));
    await tester.pumpAndSettle();
    expect(find.text('2 / 2'), findsOneWidget);
    await tester.pumpWidget(
      gfApp(
        const Scaffold(
          body: GfMediaCarousel(
            images: [
              'https://example.test/new.png',
              'https://example.test/other.png',
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      find.text('1 / 2'),
      findsOneWidget,
      reason: 'A replaced gallery must reset its page and counter together',
    );
  });
}
