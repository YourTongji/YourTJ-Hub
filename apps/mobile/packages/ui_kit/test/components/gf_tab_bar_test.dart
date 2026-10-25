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

    testWidgets('indicator follows the tab controller linearly', (
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
        TabIndicatorAnimation.linear,
      );
      expect(
        tester.widget<TabBar>(tabBar).indicator.toString(),
        contains('width: 28.0'),
      );
    });

    for (final (direction, distribute, commit) in [
      (TextDirection.ltr, false, true),
      (TextDirection.rtl, false, true),
      (TextDirection.ltr, true, true),
      (TextDirection.ltr, false, false),
    ]) {
      testWidgets('a held swipe stretches the underline without drifting '
          '(${direction.name}, distribute: $distribute, commit: $commit)', (
        tester,
      ) async {
        final progress = ValueNotifier(
          const GfTabSwipeProgress(originIndex: 0, offset: 0),
        );
        addTearDown(progress.dispose);
        Object selected = 'latest';
        late StateSetter update;
        await tester.pumpWidget(
          gfApp(
            Directionality(
              textDirection: direction,
              child: GfTabSwipeProgressScope(
                progress: progress,
                child: StatefulBuilder(
                  builder: (context, setState) {
                    update = setState;
                    return GfTabBar(
                      tabs: tabs,
                      selected: selected,
                      distribute: distribute,
                      onSelected: (_) {},
                    );
                  },
                ),
              ),
            ),
          ),
        );
        final primary = GfTheme.colorsOf(
          tester.element(find.byType(GfTabBar)),
        ).primary;
        final bar = find.byKey(const ValueKey('gf-tab-bar'));
        // Visual sign of travel toward the next tab, and the painted
        // underline as (trailing, leading) edges along that direction.
        final s = direction == TextDirection.rtl ? -1.0 : 1.0;
        (double, double) edges() {
          late Rect painted;
          expect(
            tester.renderObject(bar),
            paints..everything((method, arguments) {
              if (method == #drawRRect &&
                  (arguments[1] as Paint).color.toARGB32() ==
                      primary.toARGB32()) {
                painted = (arguments[0] as RRect).outerRect;
              }
              return true;
            }),
          );
          final rect = painted.shift(tester.getTopLeft(bar));
          return s > 0 ? (rect.left, rect.right) : (-rect.right, -rect.left);
        }

        double center(String label) =>
            tester.getCenter(find.text(label)).dx * s;
        final origin = center('最新');
        final target = center('热门');
        final (startTrail, startLead) = edges();
        expect(startTrail, closeTo(origin - 14, .5));
        expect(startLead, closeTo(origin + 14, .5));

        // Held: the trailing edge stays on the origin tab while the leading
        // edge follows the finger and never passes the next tab.
        var lead = startLead;
        for (final offset in [.2, .4, .6, .8, .95]) {
          progress.value = GfTabSwipeProgress(
            originIndex: 0,
            offset: offset,
            targetIndex: 1,
          );
          await tester.pump();
          final (trail, next) = edges();
          expect(trail, closeTo(startTrail, .5), reason: 'offset $offset');
          expect(next, greaterThan(lead), reason: 'offset $offset');
          expect(next, lessThanOrEqualTo(target + 14.5));
          lead = next;
        }

        if (!commit) {
          // Cancelled: the leading edge springs back while the trailing edge
          // stays on the origin tab.
          for (final offset in [.6, .3, .1, .02, 0.0]) {
            progress.value = GfTabSwipeProgress(
              originIndex: 0,
              offset: offset,
              targetIndex: offset == 0 ? null : 1,
            );
            await tester.pump(const Duration(milliseconds: 16));
            final (trail, next) = edges();
            expect(trail, closeTo(startTrail, .5), reason: 'back $offset');
            expect(next, lessThanOrEqualTo(lead + .5));
            lead = next;
          }
          await tester.pumpAndSettle();
          expect(edges().$1, closeTo(startTrail, .5));
          expect(edges().$2, closeTo(startLead, .5));
          return;
        }

        // Released: the page commits and the trailing edge catches up; no
        // edge moves backwards or overshoots the destination.
        update(() => selected = 'hot');
        var trail = startTrail;
        for (var frame = 0; frame < 24; frame++) {
          await tester.pump(const Duration(milliseconds: 16));
          final (nextTrail, nextLead) = edges();
          expect(nextTrail, greaterThanOrEqualTo(trail - .5));
          expect(nextLead, greaterThanOrEqualTo(lead - .5));
          expect(nextLead, lessThanOrEqualTo(target + 14.5));
          (trail, lead) = (nextTrail, nextLead);
        }
        progress.value = const GfTabSwipeProgress(originIndex: 1, offset: 0);
        await tester.pumpAndSettle();
        final (endTrail, endLead) = edges();
        expect(endTrail, closeTo(target - 14, .5));
        expect(endLead, closeTo(target + 14, .5));
      });
    }

    testWidgets('distributed tabs share the width and fall back to scrolling', (
      tester,
    ) async {
      const symbolTabs = <GfTab>[
        GfTab(label: '全部', value: 'all', symbol: 'layout-grid'),
        GfTab(label: '帖子', value: 'topics', symbol: 'file-text'),
        GfTab(label: '用户', value: 'users', symbol: 'users-round'),
        GfTab(label: '分类', value: 'categories', symbol: 'folder'),
      ];
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 800);
      addTearDown(tester.view.reset);
      Widget bar() => GfTabBar(
        tabs: symbolTabs,
        selected: 'all',
        distribute: true,
        onSelected: (_) {},
      );
      await tester.pumpWidget(gfApp(bar()));
      final tabBar = find.byKey(const ValueKey('gf-tab-bar'));
      expect(tester.widget<TabBar>(tabBar).isScrollable, isFalse);
      final widths = [
        for (final element in find.byType(Tab).evaluate())
          tester.getSize(find.byWidget(element.widget)).width,
      ];
      expect(widths.toSet(), hasLength(1));
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'folder',
        ),
        findsOneWidget,
      );

      tester.view.physicalSize = const Size(200, 800);
      await tester.pumpWidget(gfApp(bar()));
      expect(tester.widget<TabBar>(tabBar).isScrollable, isTrue);

      // Equal cells must fit the widest label, not just the combined width.
      tester.view.physicalSize = const Size(600, 800);
      await tester.pumpWidget(
        gfApp(
          GfTabBar(
            tabs: const [
              GfTab(label: '全部', value: 'all', symbol: 'layout-grid'),
              GfTab(label: '帖子', value: 'topics', symbol: 'file-text'),
              GfTab(label: '用户', value: 'users', symbol: 'users-round'),
              GfTab(label: '一个很长的分类名称', value: 'long', symbol: 'folder'),
            ],
            selected: 'all',
            distribute: true,
            onSelected: (_) {},
          ),
        ),
      );
      expect(tester.widget<TabBar>(tabBar).isScrollable, isTrue);
      expect(tester.takeException(), isNull);
    });

    testWidgets('selected-only glyphs hand off toward each other', (
      tester,
    ) async {
      const symbolTabs = <GfTab>[
        GfTab(label: '全部', value: 'all', symbol: 'layers-filled'),
        GfTab(label: '@我', value: 'mentions', symbol: 'at-sign-filled'),
        GfTab(label: '回复', value: 'replies', symbol: 'message-circle-filled'),
      ];
      late StateSetter update;
      Object selected = 'all';
      await tester.pumpWidget(
        gfApp(
          StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return GfTabBar(
                tabs: symbolTabs,
                selected: selected,
                selectedSymbolOnly: true,
                onSelected: (_) {},
              );
            },
          ),
        ),
      );
      Finder glyph(String name) => find.byWidgetPredicate(
        (widget) => widget is GfSymbol && widget.name == name,
      );
      double opacityOf(String name) => tester
          .widget<Opacity>(
            find
                .ancestor(of: glyph(name), matching: find.byType(Opacity))
                .first,
          )
          .opacity;
      // The nearest Transform scales the glyph; the next one moves it.
      double travelOf(String name) => tester
          .widgetList<Transform>(
            find.ancestor(of: glyph(name), matching: find.byType(Transform)),
          )
          .elementAt(1)
          .transform
          .getTranslation()
          .x;

      expect(glyph('layers-filled'), findsOneWidget);
      expect(glyph('at-sign-filled'), findsNothing);
      expect(glyph('message-circle-filled'), findsNothing);

      update(() => selected = 'replies');
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 60));
      // A skipped middle tab never flashes its glyph.
      expect(glyph('at-sign-filled'), findsNothing);
      expect(opacityOf('layers-filled'), inExclusiveRange(0, 1));
      expect(opacityOf('message-circle-filled'), inExclusiveRange(0, 1));
      expect(travelOf('layers-filled'), greaterThan(0));
      expect(travelOf('message-circle-filled'), lessThan(0));

      await tester.pumpAndSettle();
      expect(glyph('layers-filled'), findsNothing);
      expect(glyph('message-circle-filled'), findsOneWidget);
      expect(opacityOf('message-circle-filled'), 1);
      expect(travelOf('message-circle-filled'), 0);
      expect(tester.takeException(), isNull);
    });

    testWidgets('fades only the edges that still have tabs to scroll to', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(220, 800);
      addTearDown(tester.view.reset);
      final manyTabs = [
        for (var index = 0; index < 6; index++)
          GfTab(label: '分类 $index', value: index),
      ];
      await tester.pumpWidget(
        gfApp(GfTabBar(tabs: manyTabs, selected: 0, onSelected: (_) {})),
      );
      await tester.pumpAndSettle();
      Offset fade() => tester
          .widget<TweenAnimationBuilder<Offset>>(
            find.byType(TweenAnimationBuilder<Offset>),
          )
          .tween
          .end!;
      expect(find.byType(ShaderMask), findsOneWidget);
      expect(fade(), const Offset(0, 1));

      await tester.drag(find.text('分类 1'), const Offset(-2000, 0));
      await tester.pumpAndSettle();
      expect(fade(), const Offset(1, 0));

      await tester.drag(find.text('分类 4'), const Offset(80, 0));
      await tester.pumpAndSettle();
      expect(fade(), const Offset(1, 1));

      tester.view.physicalSize = const Size(1200, 800);
      await tester.pumpWidget(
        gfApp(GfTabBar(tabs: tabs, selected: 'latest', onSelected: (_) {})),
      );
      await tester.pumpAndSettle();
      expect(fade(), Offset.zero);
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
