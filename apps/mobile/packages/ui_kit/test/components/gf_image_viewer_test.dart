import 'dart:ui' as ui;

import 'package:extended_image/extended_image.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

Future<void> cachePhoto(WidgetTester tester, String url) async {
  final image = await tester.runAsync(() async {
    final recorder = ui.PictureRecorder();
    ui.Canvas(recorder).drawRect(
      const Rect.fromLTWH(0, 0, 400, 800),
      ui.Paint()..color = Colors.blue,
    );
    final picture = recorder.endRecording();
    final image = await picture.toImage(400, 800);
    picture.dispose();
    return image;
  });
  final provider = GfMediaScope.imageProvider(null, url);
  final key = await provider.obtainKey(ImageConfiguration.empty);
  PaintingBinding.instance.imageCache.putIfAbsent(
    key,
    () => OneFrameImageStreamCompleter(
      SynchronousFuture(ImageInfo(image: image!)),
    ),
  );
  addTearDown(() {
    PaintingBinding.instance.imageCache.clear();
    PaintingBinding.instance.imageCache.clearLiveImages();
  });
}

void main() {
  group('GfImageViewer', () {
    for (final cancelPointer in [false, true]) {
      testWidgets(
        'vertical drag resets when ${cancelPointer ? 'cancelled' : 'a second finger joins'}',
        (tester) async {
          tester.view.devicePixelRatio = 1;
          tester.view.physicalSize = const Size(390, 844);
          addTearDown(tester.view.reset);
          const url = 'https://example.com/cancel-drag.png';
          await cachePhoto(tester, url);
          await tester.pumpWidget(gfApp(const SizedBox.shrink()));
          tester
              .state<NavigatorState>(find.byType(Navigator))
              .push(
                MaterialPageRoute<void>(
                  builder: (_) => const GfImageViewer(images: [url]),
                ),
              );
          await tester.pumpAndSettle();
          final first = await tester.startGesture(
            const Offset(190, 400),
            pointer: 1,
          );
          await first.moveBy(
            const Offset(0, 100),
            timeStamp: const Duration(milliseconds: 600),
          );
          await tester.pump(const Duration(milliseconds: 600));
          final slide = tester.state<ExtendedImageSlidePageState>(
            find.byType(ExtendedImageSlidePage),
          );
          expect(slide.offset.dy, greaterThan(slide.pageSize.height / 10));
          if (cancelPointer) {
            await first.cancel();
          } else {
            final second = await tester.startGesture(
              const Offset(290, 600),
              pointer: 2,
            );
            await tester.pumpAndSettle();
            expect(find.byType(GfImageViewer), findsOneWidget);
            await second.up();
            await first.up();
          }
          await tester.pumpAndSettle();
          expect(find.byType(GfImageViewer), findsOneWidget);
          expect(slide.offset, Offset.zero);
          expect(slide.isSliding, isFalse);
          // Cancellation must not disable a subsequent intentional dismissal.
          await tester.timedDragFrom(
            const Offset(190, 400),
            const Offset(0, 200),
            const Duration(milliseconds: 600),
          );
          await tester.pumpAndSettle();
          expect(find.byType(GfImageViewer), findsNothing);
        },
        variant: TargetPlatformVariant({
          TargetPlatform.android,
          TargetPlatform.iOS,
        }),
      );
    }

    testWidgets('single image builds viewer chrome in both themes', (
      tester,
    ) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            GfImageViewer(images: const <String>['https://example.com/a.png']),
            brightness: brightness,
          ),
        );
        // A single image keeps the counter and close button, without navigation.
        expect(
          find.byWidgetPredicate(
            (widget) => widget is GfSymbol && widget.name == 'x',
          ),
          findsOneWidget,
        );
        expect(
          find.byWidgetPredicate(
            (widget) => widget is GfSymbol && widget.name == 'chevron-left',
          ),
          findsNothing,
        );
        expect(
          find.byWidgetPredicate(
            (widget) => widget is GfSymbol && widget.name == 'chevron-right',
          ),
          findsNothing,
        );
        expect(find.text('1 / 1'), findsOneWidget);
        expect(
          find.byKey(const Key('gf-image-viewer-thumbnail-rail')),
          findsNothing,
        );
      });
    });

    testWidgets('multi-image shows counter and touch controls', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: const <String>[
              'https://example.com/a.png',
              'https://example.com/b.png',
              'https://example.com/c.png',
            ],
          ),
        ),
      );
      expect(find.text('1 / 3'), findsOneWidget);
      final ListView thumbnailRail = tester.widget(
        find.byKey(const Key('gf-image-viewer-thumbnail-rail')),
      );
      expect(thumbnailRail.childrenDelegate.estimatedChildCount, 3);
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'chevron-left',
        ),
        findsNothing,
      );
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'chevron-right',
        ),
        findsNothing,
      );
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'maximize',
        ),
        findsOneWidget,
      );
    });

    testWidgets('zoom disables page swipes and enables vertical dismissal', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: const <String>[
              'https://example.com/a.png',
              'https://example.com/b.png',
            ],
          ),
        ),
      );

      final ExtendedImageGesturePageView pageView = tester.widget(
        find.byType(ExtendedImageGesturePageView),
      );
      expect(pageView.canScrollPage(GestureDetails(totalScale: 1)), isTrue);
      expect(pageView.canScrollPage(GestureDetails(totalScale: 2)), isFalse);
      final ExtendedImageSlidePage slidePage = tester.widget(
        find.byType(ExtendedImageSlidePage),
      );
      expect(slidePage.slideAxis, SlideAxis.vertical);
      expect(slidePage.slideType, SlideType.wholePage);
      expect(
        tester
            .widget<ExtendedImage>(find.byType(ExtendedImage).first)
            .enableSlideOutPage,
        isFalse,
      );
    });

    testWidgets('short vertical drags spring back and long drags dismiss', (
      tester,
    ) async {
      await tester.pumpWidget(gfApp(const SizedBox.shrink()));
      final NavigatorState navigator = tester.state(
        find.byType(Navigator).first,
      );
      navigator.push<void>(
        MaterialPageRoute<void>(
          builder: (_) => GfImageViewer(
            images: const <String>['https://example.com/a.png'],
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 350));
      expect(find.byType(GfImageViewer), findsOneWidget);

      ExtendedImageSlidePageState slide = tester.state(
        find.byType(ExtendedImageSlidePage),
      );
      final double dismissalThreshold = slide.pageSize.height / 6;
      expect(dismissalThreshold, greaterThan(1));
      slide.slide(Offset(0, dismissalThreshold / 2));
      slide.endSlide(ScaleEndDetails());
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 600));
      expect(find.byType(GfImageViewer), findsOneWidget);
      expect(slide.offset.dy, closeTo(0, 1));

      slide = tester.state(find.byType(ExtendedImageSlidePage));
      slide.slide(Offset(0, dismissalThreshold + 1));
      slide.endSlide(ScaleEndDetails());
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 400));
      expect(find.byType(GfImageViewer), findsNothing);
    });

    testWidgets(
      'horizontal page swipe survives vertical wobble and upward flick dismisses',
      (tester) async {
        const first = 'https://example.com/swipe-first.png';
        const second = 'https://example.com/swipe-second.png';
        await cachePhoto(tester, first);
        await cachePhoto(tester, second);
        await tester.pumpWidget(gfApp(const SizedBox.shrink()));
        tester
            .state<NavigatorState>(find.byType(Navigator).first)
            .push<void>(
              MaterialPageRoute<void>(
                builder: (_) => const GfImageViewer(images: [first, second]),
              ),
            );
        await tester.pumpAndSettle();

        var center = tester.getCenter(
          find.byKey(const Key('gf-image-viewer-page-swipe-area')),
        );
        final TestGesture swipe = await tester.startGesture(center);
        await swipe.moveBy(
          const Offset(8, 12),
          timeStamp: const Duration(milliseconds: 16),
        );
        await tester.pump(const Duration(milliseconds: 16));
        await swipe.moveBy(
          const Offset(-32, -5),
          timeStamp: const Duration(milliseconds: 32),
        );
        await tester.pump(const Duration(milliseconds: 16));
        await swipe.moveBy(
          const Offset(-95, -6),
          timeStamp: const Duration(milliseconds: 48),
        );
        await tester.pump(const Duration(milliseconds: 16));
        await swipe.moveBy(
          const Offset(-90, -20),
          timeStamp: const Duration(milliseconds: 64),
        );
        await tester.pump(const Duration(milliseconds: 16));
        await swipe.up(timeStamp: const Duration(milliseconds: 80));
        await tester.pumpAndSettle();
        expect(find.text('2 / 2'), findsOneWidget);

        center = tester.getCenter(
          find.byKey(const Key('gf-image-viewer-page-swipe-area')),
        );
        await tester.timedDragFrom(
          center,
          const Offset(0, -50),
          const Duration(milliseconds: 80),
        );
        await tester.pumpAndSettle();
        expect(find.byType(GfImageViewer), findsNothing);
      },
    );

    testWidgets('rapid horizontal swipes interrupt page settling', (
      tester,
    ) async {
      const images = [
        'https://example.com/rapid-first.png',
        'https://example.com/rapid-second.png',
        'https://example.com/rapid-third.png',
        'https://example.com/rapid-fourth.png',
      ];
      for (final image in images) {
        await cachePhoto(tester, image);
      }
      await tester.pumpWidget(gfApp(const GfImageViewer(images: images)));
      await tester.pumpAndSettle();

      final area = find.byKey(const Key('gf-image-viewer-page-swipe-area'));
      var eventTime = Duration.zero;
      for (var index = 1; index < images.length; index++) {
        final gesture = await tester.createGesture();
        await gesture.down(tester.getCenter(area), timeStamp: eventTime);
        await gesture.moveBy(
          const Offset(-32, 3),
          timeStamp: eventTime + const Duration(milliseconds: 16),
        );
        await tester.pump(const Duration(milliseconds: 16));
        await gesture.moveBy(
          const Offset(-420, 3),
          timeStamp: eventTime + const Duration(milliseconds: 32),
        );
        await tester.pump(const Duration(milliseconds: 16));
        await gesture.up(
          timeStamp: eventTime + const Duration(milliseconds: 48),
        );
        for (var frame = 0; frame < 5; frame++) {
          await tester.pump(const Duration(milliseconds: 16));
        }
        eventTime += const Duration(milliseconds: 128);
      }

      await tester.pumpAndSettle();
      expect(find.text('4 / 4'), findsOneWidget);
    });

    testWidgets('motion preference changes settle and restore drag return', (
      tester,
    ) async {
      const url = 'https://example.com/return-photo.png';
      await cachePhoto(tester, url);
      await tester.pumpWidget(gfApp(const GfImageViewer(images: [url])));
      await tester.pumpAndSettle();
      final slide = tester.state<ExtendedImageSlidePageState>(
        find.byType(ExtendedImageSlidePage),
      );
      final controller = slide.backAnimationController;
      slide.slide(Offset(0, slide.pageSize.height / 12));
      slide.endSlide(ScaleEndDetails());
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 60));
      expect(controller.isAnimating, isTrue);
      expect(slide.offset.dy, greaterThan(0));

      tester.platformDispatcher.accessibilityFeaturesTestValue =
          const FakeAccessibilityFeatures(disableAnimations: true);
      addTearDown(
        tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
      );
      await tester.pump();
      expect(tester.takeException(), isNull);
      expect(slide.backAnimationController, same(controller));
      expect(controller.isAnimating, isFalse);
      expect(slide.isSliding, isFalse);
      expect(slide.offset, Offset.zero);

      tester.platformDispatcher.accessibilityFeaturesTestValue =
          const FakeAccessibilityFeatures(disableAnimations: false);
      await tester.pump();
      slide.slide(Offset(0, slide.pageSize.height / 12));
      slide.endSlide(ScaleEndDetails());
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 60));
      expect(controller.isAnimating, isTrue);
      expect(slide.offset.dy, greaterThan(0));
      await tester.pumpAndSettle();
      expect(slide.isSliding, isFalse);
      expect(slide.offset, Offset.zero);
    });

    for (final reduceDuringZoom in [false, true]) {
      testWidgets(
        'loaded photo zoom blocks dismissal until reset reduced=$reduceDuringZoom',
        (tester) async {
          const url = 'https://example.com/gesture-photo.png';
          await cachePhoto(tester, url);
          await tester.pumpWidget(gfApp(const SizedBox.shrink()));
          tester
              .state<NavigatorState>(find.byType(Navigator))
              .push(
                MaterialPageRoute<void>(
                  builder: (_) => const GfImageViewer(images: [url]),
                ),
              );
          await tester.pumpAndSettle();
          final image = tester.widget<ExtendedImage>(
            find.byType(ExtendedImage),
          );
          final state =
              (image.extendedImageGestureKey!
                      as GlobalKey<ExtendedImageGestureState>)
                  .currentState!;
          expect(state.gestureDetails!.totalScale, 1);
          final center = tester.getCenter(find.byType(ExtendedImage));
          await tester.tapAt(center);
          await tester.pump(const Duration(milliseconds: 50));
          await tester.tapAt(center);
          if (reduceDuringZoom) {
            await tester.pump();
            await tester.pump(const Duration(milliseconds: 60));
            final partialScale = state.gestureDetails!.totalScale!;
            tester.platformDispatcher.accessibilityFeaturesTestValue =
                const FakeAccessibilityFeatures(disableAnimations: true);
            addTearDown(
              tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
            );
            await tester.pump();
            expect(
              state.gestureDetails!.totalScale!,
              greaterThan(partialScale),
            );
          }
          await tester.pumpAndSettle();
          expect(state.gestureDetails!.totalScale, greaterThan(1));
          await tester.timedDragFrom(
            center,
            const Offset(0, 200),
            const Duration(milliseconds: 400),
          );
          await tester.pumpAndSettle();
          expect(find.byType(GfImageViewer), findsOneWidget);
          // Reset through the actual gesture surface, then drag to dismiss.
          await tester.tapAt(center);
          await tester.pump(const Duration(milliseconds: 50));
          await tester.tapAt(center);
          await tester.pumpAndSettle();
          expect(state.gestureDetails!.totalScale, 1);
          await tester.timedDragFrom(
            center,
            const Offset(0, 200),
            const Duration(milliseconds: 400),
          );
          await tester.pumpAndSettle();
          expect(find.byType(GfImageViewer), findsNothing);
        },
      );
    }

    testWidgets('initial index is respected', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: const <String>[
              'https://example.com/a.png',
              'https://example.com/b.png',
            ],
            initialIndex: 1,
          ),
        ),
      );
      expect(find.text('2 / 2'), findsOneWidget);
    });

    testWidgets('thumbnail selection recenters and resets viewer sizing', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(400, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: const <String>[
              'https://example.com/a.png',
              'https://example.com/b.png',
              'https://example.com/c.png',
            ],
            initialIndex: 1,
          ),
        ),
      );
      await tester.pump();

      final Semantics selectedThumbnail = tester.widget(
        find.byKey(const ValueKey('gf-image-viewer-thumbnail-semantic-1')),
      );
      expect(selectedThumbnail.properties.button, isTrue);
      expect(selectedThumbnail.properties.selected, isTrue);
      expect(selectedThumbnail.properties.label, 'Image 2 of 3');
      expect(
        tester
            .widget<Transform>(
              find.byKey(const ValueKey('gf-image-viewer-thumbnail-visual-1')),
            )
            .transform
            .getMaxScaleOnAxis(),
        closeTo(1, 0.001),
      );

      final ScrollableState railScroll = tester.state(
        find.descendant(
          of: find.byKey(const Key('gf-image-viewer-thumbnail-rail')),
          matching: find.byType(Scrollable),
        ),
      );
      expect(railScroll.position.pixels, closeTo(64, 1));

      await tester.tap(find.byTooltip('Original size'));
      await tester.pump();
      expect(find.byTooltip('Fit preview'), findsOneWidget);
      await tester.tap(
        find.byKey(const ValueKey('gf-image-viewer-thumbnail-1')),
      );
      await tester.pump();
      expect(find.byTooltip('Original size'), findsOneWidget);

      final Finder nextThumbnail = find.byKey(
        const ValueKey('gf-image-viewer-thumbnail-2'),
      );
      await tester.ensureVisible(nextThumbnail);
      await tester.tap(nextThumbnail);
      for (int attempt = 0; attempt < 10; attempt++) {
        await tester.pump(const Duration(milliseconds: 80));
        if (find.text('3 / 3').evaluate().isNotEmpty) break;
      }
      expect(find.text('3 / 3'), findsOneWidget);
      expect(railScroll.position.pixels, closeTo(128, 1));
    });

    testWidgets('distant thumbnail selection skips intervening images', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          const GfImageViewer(
            images: [
              'https://example.com/a.png',
              'https://example.com/b.png',
              'https://example.com/c.png',
            ],
          ),
        ),
      );
      await tester.pump();
      await tester.tap(
        find.byKey(const ValueKey('gf-image-viewer-thumbnail-2')),
      );
      await tester.pump();
      final pages = tester.widget<ExtendedImageGesturePageView>(
        find.byType(ExtendedImageGesturePageView),
      );
      expect(pages.controller.page, 2);
      expect(find.text('3 / 3'), findsOneWidget);
    });

    testWidgets('hidden thumbnail rail is faded and disabled', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: const <String>[
              'https://example.com/a.png',
              'https://example.com/b.png',
            ],
          ),
        ),
      );
      await tester.tapAt(const Offset(200, 300));
      await tester.pump(const Duration(milliseconds: 180));

      expect(
        tester
            .widget<AnimatedOpacity>(
              find.byKey(const Key('gf-image-viewer-thumbnail-opacity')),
            )
            .opacity,
        0,
      );
      expect(
        tester
            .widget<IgnorePointer>(
              find.byKey(const Key('gf-image-viewer-thumbnail-interaction')),
            )
            .ignoring,
        isTrue,
      );
      expect(
        tester
            .widget<ExcludeSemantics>(
              find.byKey(const Key('gf-image-viewer-thumbnail-semantics')),
            )
            .excluding,
        isTrue,
      );
    });

    testWidgets('thumbnail rail fits narrow portrait and landscape layouts', (
      tester,
    ) async {
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      tester.view.devicePixelRatio = 1;
      for (final Size size in <Size>[
        const Size(320, 640),
        const Size(640, 320),
      ]) {
        tester.view.physicalSize = size;
        await tester.pumpWidget(
          gfApp(
            GfImageViewer(
              images: const <String>[
                'https://example.com/a.png',
                'https://example.com/b.png',
                'https://example.com/c.png',
              ],
            ),
          ),
        );
        await tester.pump();
        expect(tester.takeException(), isNull);
      }
    });

    testWidgets('reduced motion skips viewer transitions', (tester) async {
      await tester.pumpWidget(
        gfApp(
          Builder(
            builder: (context) => MediaQuery(
              data: MediaQuery.of(context).copyWith(disableAnimations: true),
              child: GfImageViewer(
                images: const <String>[
                  'https://example.com/a.png',
                  'https://example.com/b.png',
                ],
              ),
            ),
          ),
        ),
      );

      expect(
        tester
            .widget<ExtendedImageSlidePage>(find.byType(ExtendedImageSlidePage))
            .resetPageDuration,
        Duration.zero,
      );
      expect(
        tester
            .widget<AnimatedOpacity>(
              find.byKey(const Key('gf-image-viewer-header-opacity')),
            )
            .duration,
        Duration.zero,
      );
      expect(
        tester
            .widget<AnimatedOpacity>(
              find.byKey(const Key('gf-image-viewer-thumbnail-opacity')),
            )
            .duration,
        Duration.zero,
      );
    });

    testWidgets('actual size toggle swaps icon', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(images: const <String>['https://example.com/a.png']),
        ),
      );
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'maximize',
        ),
        findsOneWidget,
      );
      await tester.tap(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'maximize',
        ),
      );
      await tester.pump();
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'minimize',
        ),
        findsOneWidget,
      );
    });

    testWidgets('tap toggles the viewer chrome', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(images: const <String>['https://example.com/a.png']),
        ),
      );
      expect(find.byTooltip('Close'), findsOneWidget);
      await tester.tap(
        find.byKey(const Key('gf-image-viewer-page-swipe-area')),
      );
      await tester.pump(const Duration(milliseconds: 180));
      expect(
        tester.widget<AnimatedOpacity>(find.byType(AnimatedOpacity)).opacity,
        0,
      );
      await tester.tap(
        find.byKey(const Key('gf-image-viewer-page-swipe-area')),
      );
      await tester.pump(const Duration(milliseconds: 180));
      expect(
        tester.widget<AnimatedOpacity>(find.byType(AnimatedOpacity)).opacity,
        1,
      );
    });

    testWidgets('long press exposes save image without copy image', (
      tester,
    ) async {
      String? savedUrl;
      const String imageUrl = 'https://example.com/a.png';
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: const <String>[imageUrl],
            onSaveImage: (String url) async => savedUrl = url,
            saveImageLabel: '保存图片',
          ),
        ),
      );

      await tester.longPress(find.byType(ExtendedImage));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));

      expect(find.text('保存图片'), findsOneWidget);
      expect(find.text('复制图片'), findsNothing);
      await tester.tap(find.text('保存图片'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(savedUrl, imageUrl);
    });

    testWidgets('share button exposes the currently focused image', (
      tester,
    ) async {
      String? sharedUrl;
      const List<String> images = <String>[
        'https://example.com/a.png',
        'https://example.com/b.png',
      ];
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: images,
            initialIndex: 1,
            onShareImage: (String url) async => sharedUrl = url,
            shareImageLabel: '分享图片',
          ),
        ),
      );

      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'share-2',
        ),
        findsOneWidget,
      );
      await tester.tap(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'share-2',
        ),
      );
      await tester.pump();
      expect(sharedUrl, images[1]);
    });

    testWidgets('double tap is handled by the image gesture surface', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          GfImageViewer(
            images: const <String>[
              'https://example.com/a.png',
              'https://example.com/b.png',
            ],
          ),
        ),
      );

      final Finder image = find.byType(ExtendedImage).first;
      await tester.tap(image);
      await tester.pump(const Duration(milliseconds: 50));
      await tester.tap(image);
      await tester.pump(const Duration(milliseconds: 300));

      expect(find.text('1 / 2'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  });
}
