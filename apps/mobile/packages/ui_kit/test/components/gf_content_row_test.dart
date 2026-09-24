import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

void main() {
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
                          contextIcon: Icons.favorite,
                          onTap: () => opened++,
                          onAuthorTap: () => actor++,
                        ),
                        GfNotificationRow(
                          icon: Icons.favorite,
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
