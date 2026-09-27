import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

void main() {
  testWidgets('activity metadata shares a compact line and body alignment', (
    tester,
  ) async {
    for (final brightness in Brightness.values) {
      await tester.pumpWidget(
        gfApp(
          const SizedBox(
            width: 390,
            child: GfContentRow(
              author: 'Alice',
              avatarUrl: '',
              contextLabel: '发表回复',
              contextSymbol: 'message-circle',
              time: '2 天前',
              text: 'A readable preview',
            ),
          ),
          brightness: brightness,
        ),
      );
      await tester.pumpAndSettle();
      final name = tester.getRect(find.text('Alice'));
      final time = tester.getRect(find.text('2 天前'));
      final body = tester.getRect(find.text('A readable preview'));
      expect((name.center.dy - time.center.dy).abs(), lessThan(1));
      expect(body.left, name.left);
      expect(body.top - name.bottom, inInclusiveRange(4, 6));
      expect(tester.getSize(find.byType(GfContentRow)).height, lessThan(80));
      expect(tester.takeException(), isNull);
    }
  });

  testWidgets('preview and footer consume their own taps with 44px targets', (
    tester,
  ) async {
    int opened = 0, liked = 0;
    await tester.pumpWidget(
      gfApp(
        SizedBox(
          width: 390,
          child: GfContentRow(
            author: 'Alice',
            avatarUrl: '',
            time: 'now',
            text: 'Open the discussion',
            onTap: () => opened++,
            footer: Row(
              children: [
                GfIconButton(
                  symbol: 'heart',
                  tooltip: 'Like',
                  onPressed: () => liked++,
                ),
              ],
            ),
          ),
        ),
      ),
    );
    expect(tester.getSize(find.byType(GfIconButton)), const Size(44, 44));
    await tester.tap(find.byTooltip('Like'));
    expect(liked, 1);
    expect(opened, 0);
    await tester.tap(find.text('Open the discussion'));
    expect(opened, 1);
  });

  testWidgets(
    'social rows fit narrow large-text layouts and separate actor taps',
    (tester) async {
      for (final brightness in Brightness.values) {
        for (final label in [
          '发表回复',
          'Liked your post',
          'いいねしました',
          'Gefällt-mir-Beiträge',
        ]) {
          int opened = 0, actor = 0, reads = 0;
          await tester.pumpWidget(
            gfApp(
              MediaQuery(
                data: const MediaQueryData(textScaler: TextScaler.linear(2)),
                child: SizedBox(
                  width: 280,
                  child: SingleChildScrollView(
                    child: Column(
                      children: [
                        GfContentRow(
                          author: 'A very long author name',
                          avatarUrl: '',
                          time: 'September 25',
                          text:
                              'A multiline excerpt with readable content and a long link https://example.test/very/long/path',
                          title: 'Original topic',
                          contextLabel: label,
                          contextSymbol: 'heart',
                          onTap: () => opened++,
                          onAuthorTap: () => actor++,
                        ),
                        GfNotificationRow(
                          symbol: 'heart-filled',
                          tone: GfNotificationTone.like,
                          title: 'Alice $label',
                          actorName: 'Alice',
                          avatarUrl: '',
                          subtitle: 'The original post excerpt',
                          time: '3 days ago',
                          unread: true,
                          onTap: () => opened++,
                          onActorTap: () => actor++,
                          onMarkRead: () => reads++,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              brightness: brightness,
            ),
          );
          await tester.pumpAndSettle();
          expect(tester.takeException(), isNull);
          final avatars = find.byType(GfAvatar);
          await tester.ensureVisible(avatars.first);
          await tester.tap(avatars.first);
          await tester.ensureVisible(avatars.last);
          await tester.tap(avatars.last);
          expect(actor, 2);
          expect(opened, 0);
          expect(reads, 0);
        }
      }
    },
  );
}
