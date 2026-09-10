import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('feed uses one timestamp and compact readable rows', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const SizedBox(
          width: 390,
          child: GfTopicCard(
            title: 'Campus',
            description: 'A short preview',
            authorName: 'Student',
            authorAvatarUrl: '',
            imageUrls: [],
            categories: [],
            activityText: 'now',
            replyCount: 4,
            viewCount: 12,
          ),
        ),
      ),
    );
    expect(find.text('now'), findsOneWidget);
    expect(
      tester.widget<Text>(find.text('A short preview')).style!.fontSize,
      greaterThanOrEqualTo(16),
    );
    expect(tester.getSize(find.byType(GfTopicCard)).height, lessThan(150));
  });

  testWidgets(
    'compact feed remains readable on narrow screens with large text',
    (tester) async {
      await tester.pumpWidget(
        gfApp(
          const MediaQuery(
            data: MediaQueryData(textScaler: TextScaler.linear(2)),
            child: SizedBox(
              width: 320,
              child: GfTopicCard(
                title: 'A longer campus discussion title',
                description: 'A longer preview that wraps onto several lines',
                authorName: 'A long author name',
                authorAvatarUrl: '',
                imageUrls: [],
                categories: [],
                activityText: '2 hours ago',
                replyCount: 10,
                viewCount: 100,
              ),
            ),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(find.text('2 hours ago'), findsOneWidget);
    },
  );
  for (final (name, count, ratio) in [
    ('text only', 0, 1.5),
    ('portrait single', 1, 0.75),
    ('landscape single', 1, 16 / 9),
    ('portrait gallery', 5, 0.75),
    ('landscape pair', 2, 16 / 9),
    ('landscape stack', 7, 16 / 9),
  ]) {
    testWidgets('$name preserves its intended feed placement', (tester) async {
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 390,
            child: GfTopicCard(
              title: 'Campus',
              description: 'A short preview',
              authorName: 'Student',
              authorAvatarUrl: '',
              categories: const [],
              activityText: 'now',
              replyCount: 0,
              viewCount: 1,
              imageAspectRatio: ratio,
              imageUrls: List.generate(
                count,
                (i) => 'https://example.test/$i.png',
              ),
            ),
          ),
        ),
      );
      final photos = find.byWidgetPredicate(
        (w) => w is Image && w.image is NetworkImage,
      );
      expect(photos, findsNWidgets(count > 3 ? 3 : count));
      if (count == 0) return;
      final first = tester.getRect(photos.first);
      final title = tester.getRect(find.text('Campus'));
      if (count == 1 && ratio < 1) {
        expect(first.left, greaterThan(title.right));
        expect(first.top, title.top);
      } else {
        expect(
          first.top,
          greaterThan(tester.getRect(find.text('A short preview')).bottom),
        );
      }
      for (final photo in tester.widgetList<Image>(photos)) {
        expect(photo.fit, ratio < 1 ? BoxFit.cover : BoxFit.contain);
      }
      if (count == 2 || (count > 1 && ratio < 1)) {
        final second = tester.getRect(photos.at(1));
        expect(second.top, first.top);
        expect(second.left, greaterThan(first.right));
      } else if (count > 2) {
        final rects = List.generate(3, (i) => tester.getRect(photos.at(i)))
          ..sort((a, b) => a.left.compareTo(b.left));
        expect(rects[1].left, lessThan(rects.first.right));
        expect(rects[1].top, greaterThan(rects.first.top));
        expect(find.text('$count'), findsOneWidget);
      }
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('a two-image topic retains both images', (tester) async {
    await tester.pumpWidget(
      gfApp(
        const SizedBox(
          width: 390,
          child: GfTopicCard(
            title: 'Campus',
            description: 'Two views',
            authorName: 'Student',
            authorAvatarUrl: '',
            categories: [],
            imageUrls: [
              'https://example.test/one.png',
              'https://example.test/two.png',
            ],
            activityText: 'now',
            replyCount: 0,
            viewCount: 1,
          ),
        ),
      ),
    );
    final images = tester
        .widgetList<Image>(find.byType(Image))
        .map((image) => image.image)
        .whereType<NetworkImage>()
        .map((image) => image.url);
    expect(images, contains('https://example.test/one.png'));
    expect(images, contains('https://example.test/two.png'));
  });
}
