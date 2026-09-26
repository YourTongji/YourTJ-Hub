import 'package:flutter_test/flutter_test.dart';
import 'package:extended_image/extended_image.dart';
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
        expect(find.textContaining('/ 1'), findsNothing);
      });
    });

    testWidgets('multi-image shows counter and navigation', (tester) async {
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
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'chevron-left',
        ),
        findsOneWidget,
      );
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'chevron-right',
        ),
        findsOneWidget,
      );
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'maximize',
        ),
        findsOneWidget,
      );
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
