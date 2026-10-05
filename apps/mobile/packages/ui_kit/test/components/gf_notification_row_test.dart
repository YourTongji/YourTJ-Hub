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
}
