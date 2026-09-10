import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfTopicRow', () {
    const GfTopicCategory category = GfTopicCategory(
      name: '校园生活',
      color: Color(0xFF00BC7D),
    );

    Widget buildRow({
      VoidCallback? onTap,
      bool pinned = false,
      bool unseen = false,
    }) {
      return gfApp(
        GfTopicRow(
          title: '同济大学樱花大道拍照攻略',
          description: '三月末的樱花大道,适合清晨人少时去…',
          categories: const <GfTopicCategory>[category],
          participantAvatarUrls: const <String>[
            'https://example.com/a.png',
            'https://example.com/b.png',
          ],
          activityText: '3 小时前',
          replyCount: 42,
          onTap: onTap,
          pinned: pinned,
          unseen: unseen,
        ),
      );
    }

    testWidgets('renders title, description, chip, meta in both themes', (
      tester,
    ) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(buildRow());
        expect(find.text('同济大学樱花大道拍照攻略'), findsOneWidget);
        expect(find.text('三月末的樱花大道,适合清晨人少时去…'), findsOneWidget);
        expect(find.text('校园生活'), findsOneWidget);
        expect(find.text('3 小时前'), findsOneWidget);
        expect(find.text('42'), findsOneWidget);
        expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      });
    });

    testWidgets('shows pin mark and unseen dot when flagged', (tester) async {
      await tester.pumpWidget(buildRow(pinned: true, unseen: true));
      expect(find.byIcon(Icons.push_pin), findsOneWidget);
      expect(
        find.byWidgetPredicate(
          (Widget w) =>
              w is Container &&
              w.decoration is BoxDecoration &&
              (w.decoration as BoxDecoration).shape == BoxShape.circle &&
              (w.decoration as BoxDecoration).color == GfColors.light.primary,
        ),
        findsOneWidget,
      );
    });

    testWidgets('shows the hot badge when flagged', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfTopicRow(
            title: 'hot topic',
            description: '',
            categories: const <GfTopicCategory>[],
            participantAvatarUrls: const <String>[],
            activityText: '1 小时前',
            replyCount: 999,
            hot: true,
          ),
        ),
      );
      expect(find.byIcon(Icons.local_fire_department), findsOneWidget);
      expect(find.text('hot'), findsOneWidget);
    });

    testWidgets('row tap fires onTap', (tester) async {
      int taps = 0;
      await tester.pumpWidget(buildRow(onTap: () => taps++));
      await tester.tap(find.text('同济大学樱花大道拍照攻略'));
      expect(taps, 1);
    });
  });

  group('GfTopicCard', () {
    testWidgets('renders web-aligned author, content and metrics', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 360,
            child: GfTopicCard(
              title: '校园卡片话题',
              description: '卡片摘要内容',
              authorName: 'Alice',
              authorAvatarUrl: '',
              categories: const <GfTopicCategory>[
                GfTopicCategory(name: '校园生活', color: Color(0xFF00BC7D)),
              ],
              imageUrls: const <String>[],
              activityText: '3 小时前',
              replyCount: 42,
              viewCount: 128,
              onTap: () {},
            ),
          ),
        ),
      );

      expect(find.text('Alice'), findsOneWidget);
      expect(find.text('校园卡片话题'), findsOneWidget);
      expect(find.text('卡片摘要内容'), findsOneWidget);
      expect(find.text('校园生活'), findsOneWidget);
      expect(find.text('42'), findsOneWidget);
      expect(find.text('128'), findsOneWidget);
      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.byIcon(Icons.visibility_outlined), findsOneWidget);
    });
  });
}
