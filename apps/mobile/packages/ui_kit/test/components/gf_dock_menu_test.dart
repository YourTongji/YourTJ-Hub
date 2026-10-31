import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

Finder _target(int index) =>
    find.byKey(ValueKey<String>('gf-bottom-navigation-target-$index'));

Finder get _highlight =>
    find.byKey(const ValueKey<String>('gf-dock-menu-highlight'));

/// Eight sections, more than fit above the dock in a short window.
List<GfDockMenuEntry> _sections(List<String> picked) => <GfDockMenuEntry>[
  for (int i = 0; i < 8; i++)
    GfDockMenuEntry(
      label: '分区$i',
      symbol: 'sun',
      onSelected: () => picked.add('s$i'),
    ),
];

/// A dock along the bottom of the screen, as the shell places it.
Widget _bottomApp(Widget dock, {double textScale = 1}) => MaterialApp(
  theme: gfThemeData(Brightness.light),
  builder: (BuildContext context, Widget? child) => MediaQuery(
    data: MediaQuery.of(
      context,
    ).copyWith(textScaler: TextScaler.linear(textScale)),
    child: child!,
  ),
  home: Scaffold(
    body: Align(
      alignment: Alignment.bottomCenter,
      child: Padding(padding: const EdgeInsets.only(bottom: 16), child: dock),
    ),
  ),
);

/// A router whose back button reaches the menu, as in the app.
class _Shell extends RouterDelegate<Object> with ChangeNotifier {
  _Shell(this.home);

  final Widget home;
  int pops = 0;

  @override
  Widget build(BuildContext context) => Navigator(
    pages: <Page<void>>[MaterialPage<void>(child: home)],
    onDidRemovePage: (_) {},
  );

  @override
  Future<bool> popRoute() async {
    pops++;
    return true;
  }

  @override
  Future<void> setNewRoutePath(Object configuration) async {}
}

Widget _dock({
  required List<String> picked,
  List<int>? selected,
  String current = 'today',
  List<GfDockMenuEntry>? entries,
}) => GfBottomNavigation(
  currentIndex: 0,
  onSelected: (index) => selected?.add(index),
  showLabels: false,
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
      menu: () =>
          entries ??
          <GfDockMenuEntry>[
            for (final (value, label, symbol) in const [
              ('today', '今天', 'sun'),
              ('timetable', '课表', 'layout-grid'),
            ])
              GfDockMenuEntry(
                label: label,
                symbol: symbol,
                selected: value == current,
                onSelected: () => picked.add(value),
              ),
            GfDockMenuEntry(
              label: 'Wiki',
              symbol: 'book-open',
              group: 1,
              opensPage: true,
              onSelected: () => picked.add('wiki'),
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

void main() {
  for (final change in ['fold', 'safe area', 'keyboard']) {
    testWidgets('$change cancels a held shortcut without selecting it', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(400, 740);
      addTearDown(tester.view.reset);
      final picked = <String>[];
      await tester.pumpWidget(_bottomApp(_dock(picked: picked)));
      final gesture = await tester.startGesture(tester.getCenter(_target(1)));
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpAndSettle();
      final row = find.byKey(const ValueKey<String>('gf-dock-menu-row-0'));
      await gesture.moveTo(tester.getCenter(row));
      await tester.pump();
      switch (change) {
        case 'fold':
          tester.view.physicalSize = const Size(840, 600);
        case 'safe area':
          tester.view.padding = FakeViewPadding(right: 96);
        case 'keyboard':
          tester.view.viewInsets = FakeViewPadding(bottom: 240);
      }
      await tester.pumpAndSettle();
      expect(row, findsNothing);
      await gesture.up();
      await tester.pumpAndSettle();
      expect(picked, isEmpty);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('holding a destination opens its shortcuts; a slide picks one', (
    tester,
  ) async {
    final picked = <String>[];
    final selected = <int>[];
    await tester.pumpWidget(gfApp(_dock(picked: picked, selected: selected)));

    final gesture = await tester.startGesture(tester.getCenter(_target(1)));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    expect(find.text('课表'), findsOneWidget);
    // The menu blooms above the dock, not over it.
    expect(
      tester.getRect(find.text('Wiki')).bottom,
      lessThan(tester.getRect(_target(1)).top),
    );

    await gesture.moveTo(tester.getCenter(find.text('课表')));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();

    expect(picked, <String>['timetable']);
    expect(selected, isEmpty, reason: 'holding never selects the destination');
    expect(find.text('课表'), findsNothing);
  });

  testWidgets('a hold released in place leaves the menu open for a tap', (
    tester,
  ) async {
    final picked = <String>[];
    await tester.pumpWidget(gfApp(_dock(picked: picked)));

    await tester.longPress(_target(1));
    await tester.pumpAndSettle();
    expect(find.text('今天'), findsOneWidget);

    await tester.tap(find.text('Wiki'));
    await tester.pumpAndSettle();
    expect(picked, <String>['wiki']);
    expect(find.text('今天'), findsNothing);
  });

  testWidgets('tapping outside the menu dismisses it without a choice', (
    tester,
  ) async {
    final picked = <String>[];
    await tester.pumpWidget(gfApp(_dock(picked: picked)));

    await tester.longPress(_target(1));
    await tester.pumpAndSettle();
    await tester.tapAt(const Offset(8, 8));
    await tester.pumpAndSettle();

    expect(find.text('今天'), findsNothing);
    expect(picked, isEmpty);
  });

  testWidgets('a tap still selects a destination that has a menu', (
    tester,
  ) async {
    final selected = <int>[];
    await tester.pumpWidget(
      gfApp(_dock(picked: <String>[], selected: selected)),
    );

    await tester.tap(_target(1));
    await tester.pumpAndSettle();

    expect(selected, <int>[1]);
    expect(find.text('今天'), findsNothing);
  });

  testWidgets('screen readers open the menu through the long-press action', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    try {
      await tester.pumpWidget(gfApp(_dock(picked: <String>[])));
      final node = tester.getSemantics(find.bySemanticsLabel('校园'));
      expect(
        node.getSemanticsData().hasAction(SemanticsAction.longPress),
        isTrue,
      );
      tester.semantics.longPress(find.semantics.byLabel('校园'));
      await tester.pumpAndSettle();
      expect(find.bySemanticsLabel('今天'), findsOneWidget);
      expect(find.bySemanticsLabel('首页'), findsOneWidget);
    } finally {
      semantics.dispose();
    }
  });

  testWidgets('the highlight stays on the rows a finger slides across', (
    tester,
  ) async {
    await tester.pumpWidget(gfApp(_dock(picked: <String>[])));
    final gesture = await tester.startGesture(tester.getCenter(_target(1)));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    double rowY(String label) => tester.getCenter(find.text(label)).dy;
    double highlightY() => tester.getCenter(_highlight).dy;
    double opacity() => tester.widget<AnimatedOpacity>(_highlight).opacity;
    final double x = tester.getCenter(find.text('课表')).dx;

    await gesture.moveTo(Offset(x, rowY('课表')));
    await tester.pumpAndSettle();
    expect(opacity(), 1);
    expect(highlightY(), moreOrLessEquals(rowY('课表'), epsilon: 1));

    // The gap between groups belongs to a row, so the highlight never drops
    // out on the way across it.
    await gesture.moveTo(Offset(x, (rowY('课表') + rowY('Wiki')) / 2));
    await tester.pump();
    expect(opacity(), 1);
    await gesture.moveTo(Offset(x, rowY('Wiki')));
    for (int frame = 0; frame < 8; frame++) {
      await tester.pump(const Duration(milliseconds: 16));
      expect(highlightY(), greaterThanOrEqualTo(rowY('课表') - 1));
      expect(highlightY(), lessThanOrEqualTo(rowY('Wiki') + 1));
    }
    await tester.pumpAndSettle();
    expect(highlightY(), moreOrLessEquals(rowY('Wiki'), epsilon: 1));

    // Leaving the rows fades it where it is...
    await gesture.moveTo(tester.getCenter(_target(1)));
    await tester.pumpAndSettle();
    expect(opacity(), 0);
    expect(highlightY(), moreOrLessEquals(rowY('Wiki'), epsilon: 1));

    // ...and it returns under the finger instead of sliding in.
    await gesture.moveTo(Offset(x, rowY('今天')));
    await tester.pump();
    expect(highlightY(), moreOrLessEquals(rowY('今天'), epsilon: 1));
    expect(opacity(), 1);

    await gesture.moveTo(tester.getCenter(_target(1)));
    await gesture.up();
    await tester.pumpAndSettle();
  });

  testWidgets('rows that do not fit above the dock scroll instead', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(400, 420);
    addTearDown(tester.view.reset);
    final picked = <String>[];
    await tester.pumpWidget(
      _bottomApp(
        _dock(picked: picked, entries: _sections(picked)),
        textScale: 1.6,
      ),
    );

    final gesture = await tester.startGesture(tester.getCenter(_target(1)));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    final Rect rows = tester.getRect(find.byType(SingleChildScrollView));
    expect(rows.top, greaterThanOrEqualTo(0));
    expect(rows.bottom, lessThan(tester.getRect(_target(1)).top));
    expect(
      tester.getRect(find.text('分区7')).top,
      greaterThan(rows.bottom),
      reason: 'the last shortcut starts out of view',
    );

    // Holding near the bottom edge scrolls the last shortcut into reach.
    await gesture.moveTo(Offset(rows.center.dx, rows.bottom - 8));
    for (int frame = 0; frame < 60; frame++) {
      await tester.pump(const Duration(milliseconds: 16));
    }
    expect(tester.getRect(find.text('分区7')).bottom, lessThan(rows.bottom));
    await gesture.moveTo(tester.getCenter(find.text('分区7')));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    expect(picked, <String>['s7']);
  });

  testWidgets('a drag that scrolls the rows picks nothing', (tester) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(400, 420);
    addTearDown(tester.view.reset);
    final picked = <String>[];
    await tester.pumpWidget(
      _bottomApp(
        _dock(picked: picked, entries: _sections(picked)),
        textScale: 1.6,
      ),
    );
    await tester.longPress(_target(1));
    await tester.pumpAndSettle();
    final Rect rows = tester.getRect(find.byType(SingleChildScrollView));

    await tester.dragFrom(
      Offset(rows.center.dx, rows.bottom - 24),
      const Offset(0, -200),
    );
    await tester.pumpAndSettle();
    expect(picked, isEmpty);
    expect(find.byType(SingleChildScrollView), findsOneWidget);

    await tester.tap(find.text('分区7'));
    await tester.pumpAndSettle();
    expect(picked, <String>['s7']);
  });

  testWidgets('Escape closes the menu without a choice', (tester) async {
    final picked = <String>[];
    await tester.pumpWidget(gfApp(_dock(picked: picked)));
    await tester.longPress(_target(1));
    await tester.pumpAndSettle();
    expect(find.text('今天'), findsOneWidget);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.text('今天'), findsNothing);
    expect(picked, isEmpty);
  });

  testWidgets('system back closes the menu before leaving the page', (
    tester,
  ) async {
    final picked = <String>[];
    final shell = _Shell(
      Scaffold(
        body: Center(child: _dock(picked: picked)),
      ),
    );
    addTearDown(shell.dispose);
    await tester.pumpWidget(
      MaterialApp.router(
        theme: gfThemeData(Brightness.light),
        routerConfig: RouterConfig<Object>(
          routerDelegate: shell,
          backButtonDispatcher: RootBackButtonDispatcher(),
        ),
      ),
    );
    await tester.longPress(_target(1));
    await tester.pumpAndSettle();
    expect(find.text('今天'), findsOneWidget);

    await tester.binding.defaultBinaryMessenger.handlePlatformMessage(
      SystemChannels.navigation.name,
      SystemChannels.navigation.codec.encodeMethodCall(
        const MethodCall('popRoute'),
      ),
      (_) {},
    );
    await tester.pumpAndSettle();
    expect(find.text('今天'), findsNothing);
    expect(shell.pops, 0);
    expect(picked, isEmpty);
  });

  testWidgets('a sideways move before the menu opens still scrubs the dock', (
    tester,
  ) async {
    final selected = <int>[];
    await tester.pumpWidget(
      gfApp(_dock(picked: <String>[], selected: selected)),
    );
    final Offset start = tester.getCenter(_target(1));
    final Offset end = tester.getCenter(_target(2));
    final gesture = await tester.startGesture(start);
    for (int step = 1; step <= 5; step++) {
      await gesture.moveTo(Offset.lerp(start, end, step / 5)!);
      await tester.pump(const Duration(milliseconds: 16));
    }
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('今天'), findsNothing);

    await gesture.up();
    await tester.pumpAndSettle();
    expect(selected, <int>[2]);
  });
}
