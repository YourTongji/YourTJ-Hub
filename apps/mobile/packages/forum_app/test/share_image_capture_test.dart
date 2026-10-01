import 'dart:async';
import 'dart:ui' as ui;
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/markdown_view.dart';
import 'package:forum_app/src/widgets/share/share_image_preview.dart';
import 'package:forum_app/src/widgets/share/share_image_readiness.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';
import 'test_font_helpers.dart';

class _PendingImageProvider extends ImageProvider<int> {
  const _PendingImageProvider(this.image);
  final Future<ImageInfo> image;

  @override
  Future<int> obtainKey(ImageConfiguration configuration) async => 0;

  @override
  ImageStreamCompleter loadImage(int key, ImageDecoderCallback decode) =>
      OneFrameImageStreamCompleter(image);
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'image readiness freezes after timeout and can reset for recovery',
    () async {
      final tracker = ShareImageReadinessTracker(
        timeout: const Duration(milliseconds: 5),
      );
      addTearDown(tracker.dispose);
      tracker.begin('pending');
      await tracker.wait();
      expect(tracker.isFrozen('pending'), isTrue);
      tracker.reset();
      expect(tracker.isFrozen('pending'), isFalse);
      tracker.begin('pending');
      tracker.finish('pending');
      await tracker.wait();
      expect(tracker.isFrozen('pending'), isFalse);
    },
  );

  testWidgets('share image loading placeholder recovers after reset', (
    tester,
  ) async {
    final tracker = ShareImageReadinessTracker(
      timeout: const Duration(milliseconds: 5),
    );
    addTearDown(tracker.dispose);
    final image = Completer<ImageInfo>();
    await tester.pumpWidget(
      GfMediaScope(
        identity: 'pending-share-image',
        factory:
            (
              _, {
              int? width,
              int? height,
              Set<String>? allowedOrigins,
              ResizeImagePolicy policy = ResizeImagePolicy.exact,
            }) => _PendingImageProvider(image.future),
        child: MaterialApp(
          home: ShareImageReadiness(
            tracker: tracker,
            child: const SizedBox(
              width: 40,
              height: 40,
              child: ShareImageNetworkImage('pending-image', width: 40, height: 40),
            ),
          ),
        ),
      ),
    );
    await tester.runAsync(tracker.wait);
    await tester.pump();
    expect(tracker.isFrozen('pending-image'), isTrue);
    expect(
      find.byWidgetPredicate(
        (widget) => widget is GfSymbol && widget.name == 'image-off',
      ),
      findsOneWidget,
    );

    final recorder = ui.PictureRecorder();
    ui.Canvas(recorder).drawRect(
      const ui.Rect.fromLTWH(0, 0, 1, 1),
      ui.Paint()..color = Colors.blue,
    );
    final pixel = await recorder.endRecording().toImage(1, 1);
    tracker.reset();
    image.complete(ImageInfo(image: pixel));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 20));
    expect(tracker.isFrozen('pending-image'), isFalse);
    expect(find.byType(Image), findsOneWidget);
    pixel.dispose();
  });

  testWidgets('direct capture returns a 2x PNG', (tester) async {
    final key = GlobalKey();
    await tester.pumpWidget(
      MaterialApp(
        home: Center(
          child: RepaintBoundary(
            key: key,
            child: const SizedBox(
              width: 120,
              height: 80,
              child: ColoredBox(color: Colors.red),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final bytes = (await tester.runAsync(
      () => captureShareImageForTesting(key),
    ))!;
    final decoded = img.decodePng(bytes)!;
    expect((decoded.width, decoded.height), (240, 160));
    expect(decoded.getPixel(10, 10).r, greaterThan(240));

    final rejectedOversize = await tester.runAsync(() async {
      try {
        await captureShareImageForTesting(key, maxImageBytes: 1);
        return false;
      } on Exception {
        return true;
      }
    });
    expect(rejectedOversize, isTrue);

    final captureLockWorks = await tester.runAsync(() async {
      final firstCapture = captureShareImageForTesting(key);
      try {
        await captureShareImageForTesting(key);
        return false;
      } on StateError {
        await firstCapture;
        return true;
      }
    });
    expect(captureLockWorks, isTrue);
  });

  testWidgets('tiled capture stitches pixels across the texture boundary', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(375, 2500);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPhysicalSize);
    final key = GlobalKey();
    await tester.pumpWidget(
      MaterialApp(
        home: Center(
          child: RepaintBoundary(
            key: key,
            child: SizedBox(
              width: 375,
              height: 2200,
              child: const Stack(
                fit: StackFit.expand,
                children: [
                  ColoredBox(color: Colors.red),
                  Align(
                    alignment: Alignment.bottomCenter,
                    child: SizedBox(
                      width: double.infinity,
                      height: 100,
                      child: ColoredBox(color: Colors.blue),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final bytes = (await tester.runAsync(
      () => captureShareImageForTesting(key),
    ))!;
    final decoded = img.decodePng(bytes)!;
    expect((decoded.width, decoded.height), (750, 4400));
    expect(decoded.getPixel(10, 10).r, greaterThan(240));
    expect(decoded.getPixel(10, 4390).b, greaterThan(240));
  });

  testWidgets(
    'fixed card keeps Markdown wrapping and exports light and dark examples',
    (tester) async {
      final tracker = ShareImageReadinessTracker(
        timeout: const Duration(seconds: 2),
      );
      addTearDown(tracker.dispose);
      final key = GlobalKey();
      final markdown = '''
## Share card preview

Bold **course notes** with an inline `code` sample.

- First item
- Second item

> A quoted passage uses the same Markdown renderer.

| Part | Result |
| --- | --- |
| Lab | Useful |

[:sticker:not-loaded:]

![Mock image](https://mock.invalid/course.png)
''';
      await loadTestFonts(tester);
      final mockImage = img.Image(width: 400, height: 200, numChannels: 4);
      img.fill(mockImage, color: img.ColorRgba8(255, 0, 255, 255));
      final mockPng = Uint8List.fromList(img.encodePng(mockImage));
      final palettes = [
        ShareImageTheme('paper', 'Paper', GfColors.light),
        ShareImageTheme('dark', 'Dark', GfColors.dark),
      ];
      for (final theme in palettes) {
        await tester.pumpWidget(
          ProviderScope(
            child: GfMediaScope(
              identity: 'share-mock',
              factory:
                  (
                    url, {
                    int? width,
                    int? height,
                    Set<String>? allowedOrigins,
                    ResizeImagePolicy policy = ResizeImagePolicy.exact,
                  }) => MemoryImage(mockPng),
              child: MaterialApp(
                theme: gfThemeData(Brightness.light),
                home: Scaffold(
                  body: SingleChildScrollView(
                    child: Center(
                      child: ShareImageReadiness(
                        tracker: tracker,
                        child: RepaintBoundary(
                          key: key,
                          child: ShareImageCard(
                            theme: theme,
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                GfMarkdownView(
                                  data: markdown,
                                  screenshot: true,
                                  colors: theme.colors,
                                ),
                                const SizedBox(height: 8),
                                const SizedBox(
                                  width: 200,
                                  height: 100,
                                  child: ShareImageNetworkImage(
                                    'https://mock.invalid/course.png',
                                    width: 200,
                                    height: 100,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pump();
        await tester.runAsync(
          () => precacheImage(
            const AssetImage('assets/splash/brand_mark.png'),
            tester.element(find.byType(ShareImageCard)),
          ),
        );
        expect(find.byType(ShareImageNetworkImage), findsOneWidget);
        expect(find.textContaining('Sticker unavailable'), findsOneWidget);
        expect(find.textContaining('[:sticker:not-loaded:]'), findsNothing);
        final bytes = (await tester.runAsync(
          () => captureShareImageForTesting(key),
        ))!;
        final decoded = img.decodePng(bytes)!;
        expect(bytes, isNotEmpty);
        expect(decoded.width, 750);
        expect(decoded.height, greaterThan(500));
      }
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(seconds: 1));
    },
  );

  testWidgets(
    'share preview keeps a 375 logical pixel card at supported view widths',
    (tester) async {
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetDevicePixelRatio);
      final cardKey = GlobalKey();
      final scaledTextKey = GlobalKey();
      final navigatorKey = GlobalKey<NavigatorState>();
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: navigatorKey,
          theme: gfThemeData(Brightness.light),
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: TextScaler.linear(1.5),
            ),
            child: child!,
          ),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: Center(
                child: ElevatedButton(
                  onPressed: () => showShareImagePreview(
                    context,
                    fileName: 'test.png',
                    cardBuilder: (theme) => ShareImageCard(
                      key: cardKey,
                      theme: theme,
                      child: Builder(
                        key: scaledTextKey,
                        builder: (context) => Text(
                          'scale ${MediaQuery.textScalerOf(context).scale(100)}',
                        ),
                      ),
                    ),
                  ),
                  child: const Text('open'),
                ),
              ),
            ),
          ),
        ),
      );
      for (final width in [320.0, 375.0, 430.0, 600.0]) {
        tester.view.physicalSize = Size(width, 900);
        await tester.pump();
        await tester.tap(find.text('open'));
        await tester.pumpAndSettle();
        expect(tester.getSize(find.byKey(cardKey)).width, 375);
        expect(
          MediaQuery.textScalerOf(tester.element(find.byKey(scaledTextKey)))
              .scale(100),
          100,
        );
        await navigatorKey.currentState!.maybePop();
        await tester.pumpAndSettle();
      }
    },
  );
}
