import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

void main() {
  for (final unread in [true, false]) {
    testWidgets('actorless review aligns heading with status, unread=$unread', (
      tester,
    ) async {
      int reads = 0, opens = 0;
      const detail =
          '首******尾\nEdit this content in content management and resubmit for review.';
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            width: 280,
            child: GfNotificationRow(
              symbol: 'shield-alert',
              tone: GfNotificationTone.warning,
              title: 'Content was rejected',
              subtitle: detail,
              time: 'now',
              unread: unread,
              subtitleMaxLines: null,
              onMarkRead: () => reads++,
              onTap: () => opens++,
            ),
          ),
        ),
      );
      final row = tester.getRect(find.byType(GfNotificationRow));
      final heading = tester.getRect(find.text('Content was rejected · now'));
      final status = tester.getRect(
        find.byWidgetPredicate(
          (w) => w is GfSymbol && w.name == 'shield-alert',
        ),
      );
      expect(heading.top, row.top + 12);
      expect((status.center.dy - (heading.top + 12)).abs(), lessThan(2));
      expect(tester.getRect(find.text(detail)).left, heading.left);
      expect(tester.widget<Text>(find.text(detail)).maxLines, isNull);
      if (unread) {
        final read = tester.getRect(find.byTooltip('Mark as read'));
        expect(read.left, greaterThanOrEqualTo(heading.right));
        await tester.tap(find.byTooltip('Mark as read'));
        expect(reads, 1);
        expect(opens, 0);
      }
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('avatar, actor name and row open separate targets', (
    tester,
  ) async {
    var avatar = 0, actor = 0, opens = 0;
    Widget row({VoidCallback? onAvatarTap}) => gfApp(
      SizedBox(
        width: 320,
        child: GfNotificationRow(
          symbol: 'heart-filled',
          tone: GfNotificationTone.like,
          title: 'Alice liked your reply',
          subtitle: '',
          time: 'now',
          unread: false,
          actorName: 'Alice',
          avatarUrl: '',
          onAvatarTap: onAvatarTap,
          onActorTap: () => actor++,
          onTap: () => opens++,
        ),
      ),
    );
    await tester.pumpWidget(row(onAvatarTap: () => avatar++));
    await tester.tap(find.byType(GfAvatar));
    expect((avatar, actor, opens), (1, 0, 0));
    await tester.tapOnText(find.textRange.ofSubstring('Alice'));
    expect((avatar, actor, opens), (1, 1, 0));
    await tester.tapOnText(find.textRange.ofSubstring('your reply'));
    expect((avatar, actor, opens), (1, 1, 1));
    // Without a dedicated avatar target the avatar keeps the actor target.
    await tester.pumpWidget(row());
    await tester.tap(find.byType(GfAvatar));
    expect((avatar, actor, opens), (1, 2, 1));
    expect(tester.takeException(), isNull);
  });
}
