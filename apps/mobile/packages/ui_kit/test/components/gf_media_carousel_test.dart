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
    final Finder badgeFinder = find.byKey(
      const Key('gf-media-carousel-counter-badge'),
    );
    final ClipRRect counterBadge = tester.widget(badgeFinder);
    final Size badgeSize = tester.getSize(badgeFinder);
    expect(badgeSize.height, 22);
    expect(badgeSize.width, greaterThan(badgeSize.height));
    expect(counterBadge.borderRadius, BorderRadius.circular(999));
    final BoxDecoration badgeDecoration =
        tester
                .widget<DecoratedBox>(
                  find.descendant(
                    of: badgeFinder,
                    matching: find.byType(DecoratedBox),
                  ),
                )
                .decoration
            as BoxDecoration;
    expect(badgeDecoration.color, Colors.black.withValues(alpha: 0.48));
    expect(badgeDecoration.border!.top.color.a, closeTo(0.18, 0.01));
    expect(find.byType(ImageFiltered), findsWidgets);
    expect(
      tester
          .widget<Image>(
            find.byWidgetPredicate(
              (widget) => widget is Image && widget.fit == BoxFit.contain,
            ),
          )
          .fit,
      BoxFit.contain,
    );
    expect(find.byType(GfGlassSurface), findsNothing);
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

  testWidgets('tapping a preview opens that image with its Hero tag', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const SizedBox(
          width: 300,
          child: GfMediaCarousel(
            images: [
              'https://example.test/one.png',
              'https://example.test/two.png',
            ],
          ),
        ),
      ),
    );
    await tester.pump();
    await tester.drag(find.byType(PageView).first, const Offset(-700, 0));
    await tester.pumpAndSettle();
    await tester.tapAt(tester.getCenter(find.byType(PageView).first));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 240));

    final GfImageViewer viewer = tester.widget(find.byType(GfImageViewer));
    expect(viewer.initialIndex, 1);
    expect(viewer.heroTag, isNotNull);
  });
  testWidgets(
    'viewer returns to its focused occurrence even with duplicate URLs',
    (tester) async {
      await tester.pumpWidget(
        gfApp(
          const SizedBox(
            width: 300,
            child: GfMediaCarousel(
              images: [
                'https://example.test/same.png',
                'https://example.test/same.png',
              ],
            ),
          ),
        ),
      );
      await tester.tapAt(tester.getCenter(find.byType(PageView)));
      await tester.pumpAndSettle();
      await tester.drag(
        find.byKey(const Key('gf-image-viewer-page-swipe-area')),
        const Offset(-700, 0),
      );
      await tester.pumpAndSettle();
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.text('2 / 2'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'replacing a source cannot return an old viewer to a reused cell',
    (tester) async {
      var images = ['https://example.test/old.png'];
      late StateSetter update;
      await tester.pumpWidget(
        gfApp(
          StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return SizedBox(
                width: 300,
                child: GfMediaCarousel(images: images),
              );
            },
          ),
        ),
      );
      await tester.tapAt(tester.getCenter(find.byType(PageView)));
      await tester.pumpAndSettle();
      final viewer = tester.widget<GfImageViewer>(find.byType(GfImageViewer));
      final original = viewer.heroTag;
      update(() => images = ['https://example.test/new.png']);
      await tester.pump();
      final source = tester.widget<Hero>(
        find.descendant(
          of: find.byType(GfMediaCarousel, skipOffstage: false),
          matching: find.byType(Hero, skipOffstage: false),
        ),
      );
      expect(source.tag, isNot((original, 0)));
      expect(viewer.images, ['https://example.test/old.png']);
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      expect(find.byType(GfImageViewer), findsNothing);
    },
  );
  for (final fraction in [.1, .3, .7, 1.0]) {
    testWidgets('back interrupts media entry at $fraction', (tester) async {
      await tester.pumpWidget(
        gfApp(
          const SizedBox(
            width: 300,
            child: GfMediaCarousel(images: ['https://example.test/photo.png']),
          ),
        ),
      );
      await tester.tapAt(tester.getCenter(find.byType(PageView)));
      await tester.pump();
      await tester.pump(Duration(microseconds: (280000 * fraction).round()));
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.byType(GfImageViewer), findsNothing);
      expect(find.byType(GfMediaCarousel), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets(
    'removing a source keeps the viewer usable and returns with no Hero',
    (tester) async {
      var present = true;
      late StateSetter update;
      await tester.pumpWidget(
        gfApp(
          StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return present
                  ? const SizedBox(
                      width: 300,
                      child: GfMediaCarousel(
                        images: ['https://example.test/photo.png'],
                      ),
                    )
                  : const Text('Source removed');
            },
          ),
        ),
      );
      await tester.tapAt(tester.getCenter(find.byType(PageView)));
      await tester.pumpAndSettle();
      update(() => present = false);
      await tester.pumpAndSettle();
      expect(find.byType(GfImageViewer), findsOneWidget);
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.text('Source removed'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
