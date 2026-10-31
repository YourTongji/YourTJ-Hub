import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

Finder _target(int index) =>
    find.byKey(ValueKey<String>('gf-bottom-navigation-target-$index'));

Widget _dock({
  required List<String> picked,
  List<int>? selected,
  String current = 'today',
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
      menu: () => <GfDockMenuEntry>[
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
  testWidgets('holding a destination opens its shortcuts; a slide picks one', (
    tester,
  ) async {
    final picked = <String>[];
    final selected = <int>[];
    await tester.pumpWidget(
      gfApp(_dock(picked: picked, selected: selected)),
    );

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
}
