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

    testWidgets('renders all tabs and the active one in both themes', (
      tester,
    ) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            GfTabBar(tabs: tabs, selected: 'hot', onSelected: (_) {}),
            brightness: brightness,
          ),
        );
        for (final GfTab tab in tabs) {
          expect(find.text(tab.label), findsOneWidget);
        }
        final tabBar = tester.widget<TabBar>(
          find.byKey(const ValueKey('gf-tab-bar')),
        );
        expect(tabBar.controller!.index, 1);
      });
    });

    testWidgets('selecting a tab calls onSelected with its value', (
      tester,
    ) async {
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

    testWidgets('mobile mode scrolls horizontally when overflowed', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(GfTabBar(tabs: tabs, selected: 'latest', onSelected: (_) {})),
      );
      expect(
        tester
            .widget<TabBar>(find.byKey(const ValueKey('gf-tab-bar')))
            .isScrollable,
        isTrue,
      );
    });

    testWidgets('reveals a newly selected tab outside the viewport', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(220, 800);
      addTearDown(tester.view.reset);
      final tabs = [
        for (var index = 0; index < 5; index++)
          GfTab(label: '分类 $index', value: index),
      ];
      late StateSetter update;
      Object selected = 0;
      await tester.pumpWidget(
        gfApp(
          StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return GfTabBar(
                tabs: tabs,
                selected: selected,
                onSelected: (_) {},
              );
            },
          ),
        ),
      );

      update(() => selected = 4);
      await tester.pumpAndSettle();
      final viewport = tester.getRect(find.byKey(const ValueKey('gf-tab-bar')));
      final selectedTab = tester.getRect(find.text('分类 4'));
      expect(selectedTab.left, greaterThanOrEqualTo(viewport.left));
      expect(selectedTab.right, lessThanOrEqualTo(viewport.right));
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

    testWidgets('indicator follows the native elastic tab animation', (
      tester,
    ) async {
      late StateSetter update;
      Object selected = 'latest';
      await tester.pumpWidget(
        gfApp(
          StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return GfTabBar(
                tabs: tabs,
                selected: selected,
                onSelected: (_) {},
              );
            },
          ),
        ),
      );
      final tabBar = find.byKey(const ValueKey('gf-tab-bar'));
      final controller = tester.widget<TabBar>(tabBar).controller!;
      final start = controller.animation!.value;
      update(() => selected = 'digest');
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 110));
      final middle = controller.animation!.value;
      await tester.pumpAndSettle();
      final end = controller.animation!.value;
      expect(middle, greaterThan(start));
      expect(middle, lessThan(end));
      expect(
        tester.widget<TabBar>(tabBar).indicatorAnimation,
        TabIndicatorAnimation.elastic,
      );
      expect(
        tester.widget<TabBar>(tabBar).indicator.toString(),
        contains('width: 28.0'),
      );
    });

    testWidgets('reduced motion disables the underline transition', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          MediaQuery(
            data: const MediaQueryData(disableAnimations: true),
            child: GfTabBar(tabs: tabs, selected: 'hot', onSelected: (_) {}),
          ),
        ),
      );
      final tabBar = tester.widget<TabBar>(
        find.byKey(const ValueKey('gf-tab-bar')),
      );
      expect(tabBar.controller!.animationDuration, Duration.zero);
      expect(tester.takeException(), isNull);
    });
  });
}
