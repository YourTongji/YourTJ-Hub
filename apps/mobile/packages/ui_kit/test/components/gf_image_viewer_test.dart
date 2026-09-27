import 'package:extended_image/extended_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfImageViewer', () {
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
        // Close button, no counter, no side navigation for a single image.
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
        isTrue,
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
