import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

Finder _target(int index) =>
    find.byKey(ValueKey<String>('gf-bottom-navigation-target-$index'));

Finder _indicator(int index) => find.descendant(
  of: _target(index),
  matching: find.byKey(
    const ValueKey<String>('gf-bottom-navigation-indicator'),
  ),
);

double _platter(WidgetTester tester, int index) => tester
    .widget<AnimatedOpacity>(
      find.descendant(
        of: _target(index),
        matching: find.byKey(
          const ValueKey<String>('gf-bottom-navigation-platter'),
        ),
      ),
    )
    .opacity;

Widget _bar({
  required int index,
  ValueChanged<int>? onSelected,
  VoidCallback? onAction,
  List<String>? picked,
  bool showLabels = false,
}) => GfBottomNavigation(
  currentIndex: index,
  onSelected: onSelected ?? (_) {},
  onAction: onAction,
  showLabels: showLabels,
  items: <GfBottomNavigationItem>[
    const GfBottomNavigationItem(
      label: '首页',
      symbol: 'house',
      selectedSymbol: 'house-filled',
    ),
    GfBottomNavigationItem(
      label: '校园',
      symbol: 'graduation-cap',
      selectedSymbol: 'graduation-cap-filled',
      menuHint: '长按打开快捷入口',
      menu: () => <GfDockMenuEntry>[
        for (final (value, label) in const [('today', '今天'), ('plan', '课表')])
          GfDockMenuEntry(
            label: label,
            symbol: 'sun',
            onSelected: () => picked?.add(value),
          ),
      ],
    ),
    const GfBottomNavigationItem(
      label: '通知',
      symbol: 'bell',
      selectedSymbol: 'bell-filled',
    ),
    const GfBottomNavigationItem(
      label: '私信',
      symbol: 'mail',
      selectedSymbol: 'mail-filled',
    ),
  ],
);

/// The bar along the bottom of a phone with a home indicator, as the shell
/// places it.
Widget _phone(
  Widget bar, {
  MediaQueryData data = const MediaQueryData(
    size: Size(390, 800),
    padding: EdgeInsets.only(bottom: 34),
  ),
}) => MaterialApp(
  theme: gfThemeData(Brightness.light).copyWith(platform: TargetPlatform.iOS),
  home: MediaQuery(
    data: data,
    child: GfNavigationStyleScope(
      style: GfNavigationStyle.bar,
      child: Stack(
        children: <Widget>[
          Positioned(left: 0, right: 0, bottom: 0, child: bar),
        ],
      ),
    ),
  ),
);

/// A phone whose navigation style can change while it is on screen.
Widget _switchable(
  ValueNotifier<GfNavigationStyle> style, {
  MediaQueryData data = const MediaQueryData(
    size: Size(390, 800),
    padding: EdgeInsets.only(bottom: 34),
  ),
  ValueChanged<int>? onSelected,
}) => MaterialApp(
  theme: gfThemeData(Brightness.light).copyWith(platform: TargetPlatform.iOS),
  home: MediaQuery(
    data: data,
    child: ValueListenableBuilder<GfNavigationStyle>(
      valueListenable: style,
      builder: (context, value, child) =>
          GfNavigationStyleScope(style: value, child: child!),
      child: Stack(
        children: <Widget>[
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: _bar(index: 0, onSelected: onSelected),
          ),
        ],
      ),
    ),
  ),
);

void main() {
  testWidgets('the bar runs edge to edge and pads itself for the safe area', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 800);
    addTearDown(tester.view.reset);
    final selected = <int>[];
    await tester.pumpWidget(_phone(_bar(index: 0, onSelected: selected.add)));

    expect(
      tester.getRect(find.byType(GfNavigationBarSurface)),
      const Rect.fromLTRB(0, 800 - 56 - 34, 390, 800),
    );
    final Rect target = tester.getRect(_target(0));
    expect(target.height, 56);
    expect(target.bottom, 800 - 34);
    expect(target.width, 390 / 4);
    expect(
      find.byKey(const ValueKey<String>('gf-bottom-navigation-lens')),
      findsNothing,
    );
    await tester.tap(_target(2));
    expect(selected, <int>[2]);
  });

  testWidgets('metrics and height follow the scope', (tester) async {
    late GfBottomNavigationMetrics bar;
    late GfBottomNavigationMetrics dock;
    late double barHeight;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(
          Brightness.light,
        ).copyWith(platform: TargetPlatform.iOS),
        home: MediaQuery(
          data: const MediaQueryData(padding: EdgeInsets.only(bottom: 34)),
          child: GfNavigationStyleScope(
            style: GfNavigationStyle.bar,
            child: Builder(
              builder: (context) {
                barHeight = GfBottomNavigation.heightFor(
                  context,
                  labels: const <String>['首页'],
                  availableWidth: 390,
                  showLabels: false,
                );
                bar = GfBottomNavigation.metricsOf(context, barHeight: 56);
                dock = GfBottomNavigation.metricsOf(
                  context,
                  barHeight: 56,
                  style: GfNavigationStyle.dock,
                );
                return const SizedBox();
              },
            ),
          ),
        ),
      ),
    );

    expect(barHeight, 56);
    expect(bar.contentBottomInset, 56 + 34 + 8);
    expect(bar.actionBottomInset, 56 + 16);
    // The dock floats over the home indicator instead.
    expect(dock.contentBottomInset, 22 + 56 + 12);
  });

  testWidgets('labels give the bar its measured height', (tester) async {
    await tester.pumpWidget(_phone(_bar(index: 0, showLabels: true)));

    expect(find.text('校园'), findsOneWidget);
    expect(tester.getSize(_target(0)).height, greaterThanOrEqualTo(64));
  });

  testWidgets('the selection pill grows out of the chosen glyph', (
    tester,
  ) async {
    int index = 0;
    late StateSetter update;
    await tester.pumpWidget(
      _phone(
        StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return _bar(index: index);
          },
        ),
      ),
    );
    expect(tester.getSize(_indicator(0)).width, 56);
    expect(_indicator(1), findsNothing);

    update(() => index = 1);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 40));
    final double growing = tester.getSize(_indicator(1)).width;
    expect(growing, inExclusiveRange(32, 56));
    await tester.pumpAndSettle();
    expect(tester.getSize(_indicator(1)).width, 56);
    expect(_indicator(0), findsNothing);
  });

  testWidgets('reduced motion moves the pill at once', (tester) async {
    int index = 0;
    late StateSetter update;
    await tester.pumpWidget(
      _phone(
        StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return _bar(index: index);
          },
        ),
        data: const MediaQueryData(
          size: Size(390, 800),
          disableAnimations: true,
        ),
      ),
    );
    update(() => index = 2);
    await tester.pump();
    await tester.pump();
    expect(tester.getSize(_indicator(2)).width, 56);
    expect(_indicator(0), findsNothing);
  });

  testWidgets('holding a destination lights it and opens its menu above', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 800);
    addTearDown(tester.view.reset);
    final picked = <String>[];
    final selected = <int>[];
    await tester.pumpWidget(
      _phone(_bar(index: 0, picked: picked, onSelected: selected.add)),
    );

    final gesture = await tester.startGesture(tester.getCenter(_target(1)));
    await tester.pump(const Duration(milliseconds: 100));
    expect(_platter(tester, 1), 1, reason: 'a press lights the platter');
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pumpAndSettle();

    final Rect panel = tester.getRect(
      find.byKey(const ValueKey<String>('gf-dock-menu-panel')),
    );
    final Rect bar = tester.getRect(find.byType(GfNavigationBarSurface));
    expect(panel.bottom, lessThanOrEqualTo(bar.top));
    expect(
      panel.center.dx,
      moreOrLessEquals(tester.getCenter(_target(1)).dx, epsilon: 60),
    );
    expect(_platter(tester, 1), 1, reason: 'the open menu keeps it lit');
    expect(_platter(tester, 0), 0);

    await gesture.moveTo(tester.getCenter(find.text('课表')));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();

    expect(picked, <String>['plan']);
    expect(selected, isEmpty, reason: 'the menu, not the hold, chose');
    expect(find.text('课表'), findsNothing);
    expect(_platter(tester, 1), 0);
  });

  testWidgets('screen readers open the menu through the long-press action', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    try {
      await tester.pumpWidget(_phone(_bar(index: 0)));
      final node = tester.getSemantics(find.bySemanticsLabel('校园'));
      expect(
        node.getSemanticsData().hasAction(SemanticsAction.longPress),
        isTrue,
      );
      tester.semantics.longPress(find.semantics.byLabel('校园'));
      await tester.pumpAndSettle();
      expect(find.bySemanticsLabel('今天'), findsOneWidget);
      expect(_platter(tester, 1), 1);
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      await tester.pumpAndSettle();
      expect(find.text('今天'), findsNothing);
      expect(_platter(tester, 1), 0);
    } finally {
      semantics.dispose();
    }
  });

  testWidgets('Tab follows the bar in visual order, compose included', (
    tester,
  ) async {
    int selected = 0;
    int composeTaps = 0;
    await tester.pumpWidget(
      _phone(
        StatefulBuilder(
          builder: (context, setState) => _bar(
            index: selected,
            onSelected: (value) => setState(() => selected = value),
            onAction: () => composeTaps++,
          ),
        ),
      ),
    );

    for (final int? index in <int?>[0, 1, null, 2, 3]) {
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      await tester.pumpAndSettle();
      if (index == null) {
        expect(composeTaps, 1);
      } else {
        expect(selected, index);
      }
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('changing style morphs one surface between dock and bar', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 800);
    addTearDown(tester.view.reset);
    final style = ValueNotifier(GfNavigationStyle.dock);
    addTearDown(style.dispose);
    final selected = <int>[];
    await tester.pumpWidget(_switchable(style, onSelected: selected.add));
    final Rect dock = tester.getRect(find.byType(GfNavigationSurface));
    final Finder morph = find.byType(GfNavigationMorphSurface);
    expect(find.byType(GfNavigationBarSurface), findsNothing);

    style.value = GfNavigationStyle.bar;
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    // One shared surface, partway between the pill and the edges.
    expect(morph, findsOneWidget);
    expect(find.byType(BackdropFilter), findsOneWidget);
    final Rect between = tester.getRect(morph);
    expect(between.width, inExclusiveRange(dock.width, 390));
    expect(between.bottom, greaterThan(dock.bottom));
    // Touches already belong to the bar.
    await tester.tapAt(tester.getCenter(_target(3).last));
    expect(selected, <int>[3]);

    await tester.pumpAndSettle();
    expect(morph, findsNothing);
    expect(find.byType(GfNavigationSurface), findsNothing);
    expect(
      tester.getRect(find.byType(GfNavigationBarSurface)),
      const Rect.fromLTRB(0, 800 - 34 - 56, 390, 800),
    );

    style.value = GfNavigationStyle.dock;
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    expect(morph, findsOneWidget);
    await tester.pumpAndSettle();
    expect(morph, findsNothing);
    expect(find.byType(GfNavigationBarSurface), findsNothing);
    expect(tester.getRect(find.byType(GfNavigationSurface)), dock);
    expect(tester.takeException(), isNull);
  });

  testWidgets('reduced motion changes style at once', (tester) async {
    final style = ValueNotifier(GfNavigationStyle.dock);
    addTearDown(style.dispose);
    await tester.pumpWidget(
      _switchable(
        style,
        data: const MediaQueryData(
          size: Size(390, 800),
          disableAnimations: true,
        ),
      ),
    );
    style.value = GfNavigationStyle.bar;
    await tester.pump();
    await tester.pump();
    expect(find.byType(GfNavigationMorphSurface), findsNothing);
    expect(find.byType(GfNavigationBarSurface), findsOneWidget);
    expect(_target(0), findsOneWidget);
  });

  for (final (String name, MediaQueryData data, bool glass)
      in <(String, MediaQueryData, bool)>[
        ('a normal bar', const MediaQueryData(), true),
        ('high contrast', const MediaQueryData(highContrast: true), false),
        (
          'reduced motion',
          const MediaQueryData(disableAnimations: true),
          false,
        ),
      ]) {
    testWidgets('$name ${glass ? 'is frosted' : 'keeps the opaque bar'}', (
      tester,
    ) async {
      await tester.pumpWidget(_phone(_bar(index: 0), data: data));
      expect(
        find.byType(BackdropFilter),
        glass ? findsOneWidget : findsNothing,
      );
    });
  }
}
