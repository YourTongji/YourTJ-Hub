import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';
import 'gf_image_viewer_test.dart' show cachePhoto;

void main() {
  testWidgets(
    'return source rejects clipped, scrolled and removed thumbnails',
    (tester) async {
      final key = GlobalKey();
      final scroll = ScrollController();
      addTearDown(scroll.dispose);
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            height: 200,
            child: SingleChildScrollView(
              controller: scroll,
              child: Column(
                children: [
                  SizedBox(key: key, height: 100, width: 100),
                  const SizedBox(height: 1000),
                ],
              ),
            ),
          ),
        ),
      );
      expect(gfImageSourceIsVisible(key), isTrue);
      scroll.jumpTo(50);
      await tester.pump();
      expect(
        gfImageSourceIsVisible(key),
        isFalse,
        reason: 'A partially clipped thumbnail cannot be a return target',
      );
      scroll.jumpTo(150);
      await tester.pump();
      expect(gfImageSourceIsVisible(key), isFalse);
      scroll.jumpTo(0);
      await tester.pump();
      expect(gfImageSourceIsVisible(key), isTrue);
      await tester.pumpWidget(const SizedBox.shrink());
      expect(gfImageSourceIsVisible(key), isFalse);
    },
  );

  for (final progress in [.1, .3, .7, 1.0]) {
    for (final valid in [true, false]) {
      testWidgets(
        'media entry can reverse at $progress with source valid=$valid',
        (tester) async {
          const url = 'https://example.com/source-return.png';
          await cachePhoto(tester, url);
          final key = GlobalKey();
          final identity = Object();
          await tester.pumpWidget(
            gfApp(
              Center(
                child: SizedBox(
                  key: key,
                  width: 100,
                  height: 100,
                  child: Hero(
                    tag: (identity, 0),
                    child: const ColoredBox(color: Colors.blue),
                  ),
                ),
              ),
            ),
          );
          final context = tester.element(find.byKey(key));
          final navigator = Navigator.of(context);
          navigator.push(
            gfImageViewerRoute(
              context,
              builder: (_) => GfImageViewer(
                images: const [url],
                heroTag: identity,
                canReturnToSource: (_) => valid && gfImageSourceIsVisible(key),
              ),
            ),
          );
          await tester.pump();
          await tester.pump(
            Duration(microseconds: (280000 * progress).round()),
          );
          navigator.pop();
          await tester.pump();
          await tester.pump(const Duration(milliseconds: 1));
          if (find.byType(GfImageViewer).evaluate().isNotEmpty) {
            final modes = find.descendant(
              of: find.byType(GfImageViewer),
              matching: find.byType(HeroMode),
            );
            expect(
              tester
                  .widgetList<HeroMode>(modes)
                  .every((mode) => mode.enabled == valid),
              isTrue,
            );
          }
          await tester.pumpAndSettle();
          expect(find.byType(GfImageViewer), findsNothing);
          expect(find.byKey(key), findsOneWidget);
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
}
