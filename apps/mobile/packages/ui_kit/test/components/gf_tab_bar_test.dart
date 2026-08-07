import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfTabBar', () {
    const List<GfTab> tabs = <GfTab>[
      GfTab(label: '最新', value: 'latest'),
      GfTab(label: '热门', value: 'hot'),
      GfTab(label: '精华', value: 'digest'),
    ];

    testWidgets('renders all tabs and the active one in both themes',
        (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            GfTabBar(
              tabs: tabs,
              selected: 'hot',
              onSelected: (_) {},
            ),
            brightness: brightness,
          ),
        );
        for (final GfTab tab in tabs) {
          expect(find.text(tab.label), findsOneWidget);
        }
        // Active tab renders with the neutral fill: its AnimatedContainer
        // decoration color is non-transparent.
        final AnimatedContainer active = tester.widget<AnimatedContainer>(
          find.ancestor(
            of: find.text('热门'),
            matching: find.byType(AnimatedContainer),
          ),
        );
        final BoxDecoration decoration =
            active.decoration as BoxDecoration;
        expect(decoration.color, isNot(Colors.transparent));
      });
    });

    testWidgets('selecting a tab calls onSelected with its value',
        (tester) async {
      Object? selected;
      await tester.pumpWidget(
        gfApp(
          GfTabBar(
            tabs: tabs,
            selected: 'latest',
            onSelected: (Object value) => selected = value,
          ),
        ),
      );
      await tester.tap(find.text('热门'));
      expect(selected, 'hot');
    });

    testWidgets('mobile mode scrolls horizontally when overflowed',
        (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfTabBar(
            tabs: tabs,
            selected: 'latest',
            onSelected: (_) {},
          ),
        ),
      );
      expect(find.byType(SingleChildScrollView), findsOneWidget);
    });

    testWidgets('desktop mode wraps instead of scrolling', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfTabBar(
            tabs: tabs,
            selected: 'latest',
            onSelected: (_) {},
            mobile: false,
          ),
        ),
      );
      expect(find.byType(SingleChildScrollView), findsNothing);
      expect(find.byType(Wrap), findsOneWidget);
    });
  });
}
