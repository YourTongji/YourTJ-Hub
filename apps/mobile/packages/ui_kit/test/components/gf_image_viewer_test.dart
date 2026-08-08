import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfImageViewer', () {
    testWidgets('single image builds viewer chrome in both themes',
        (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            GfImageViewer(images: const <String>['https://example.com/a.png']),
            brightness: brightness,
          ),
        );
        // Close button, no counter, no side navigation for a single image.
        expect(find.byIcon(Icons.close), findsOneWidget);
        expect(find.byIcon(Icons.chevron_left), findsNothing);
        expect(find.byIcon(Icons.chevron_right), findsNothing);
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
      expect(find.byIcon(Icons.chevron_left), findsOneWidget);
      expect(find.byIcon(Icons.chevron_right), findsOneWidget);
      expect(find.byIcon(Icons.zoom_in_map), findsOneWidget);
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
          GfImageViewer(
            images: const <String>['https://example.com/a.png'],
          ),
        ),
      );
      expect(find.byIcon(Icons.zoom_in_map), findsOneWidget);
      await tester.tap(find.byIcon(Icons.zoom_in_map));
      await tester.pump();
      expect(find.byIcon(Icons.zoom_out_map), findsOneWidget);
    });
  });
}
