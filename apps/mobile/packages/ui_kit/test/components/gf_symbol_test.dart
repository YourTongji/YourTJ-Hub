import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  for (final brightness in Brightness.values) {
    testWidgets('input icons keep their glyph size in $brightness', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          const GfInput(
            labelText: 'Account',
            prefixIcon: GfSymbol('user-round', size: 20),
            suffixIcon: GfSymbol('circle-help', size: 20),
          ),
          brightness: brightness,
        ),
      );
      for (final element in find.byType(SvgPicture).evaluate()) {
        expect(
          tester.getSize(find.byWidget(element.widget)),
          const Size(20, 20),
        );
      }
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('glyph stays centered inside a larger touch target', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const SizedBox(
          width: 48,
          height: 56,
          child: GfSymbol('github', size: 22),
        ),
      ),
    );
    expect(tester.getSize(find.byType(SvgPicture)), const Size(22, 22));
    expect(
      tester.getCenter(find.byType(SvgPicture)),
      tester.getCenter(find.byType(GfSymbol)),
    );
  });

  testWidgets('loose row glyphs do not grow or lose their provider colors', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            GfSymbol('google', size: 22),
            GfSymbol('github', size: 20),
          ],
        ),
      ),
    );
    expect(tester.getSize(find.byType(Row)), const Size(42, 22));
    expect(
      tester.widgetList<SvgPicture>(find.byType(SvgPicture)).first.colorFilter,
      isNull,
    );
    expect(
      tester.widgetList<SvgPicture>(find.byType(SvgPicture)).last.colorFilter,
      isNotNull,
    );
  });

  testWidgets('notification and section marks accept the shared SVG family', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        Column(
          children: [
            const GfSectionHeader(title: 'People', symbol: 'users-round'),
            GfNotificationRow(
              symbol: 'heart-filled',
              tone: GfNotificationTone.like,
              title: 'A liked your post',
              subtitle: 'Hello',
              time: 'Now',
              unread: true,
              onMarkRead: () {},
            ),
          ],
        ),
      ),
    );
    final names = tester
        .widgetList<GfSymbol>(find.byType(GfSymbol))
        .map((s) => s.name);
    expect(names, containsAll(['users-round', 'heart-filled', 'circle-check']));
    expect(find.byType(Icon), findsNothing);
    expect(tester.takeException(), isNull);
  });
}
