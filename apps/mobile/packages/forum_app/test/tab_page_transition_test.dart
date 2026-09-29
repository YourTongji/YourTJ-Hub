import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/tab_page_transition.dart';
import 'package:forum_app/src/navigation/tab_swipe_surface.dart';
import 'package:ui_kit/ui_kit.dart';

class _Pager extends StatefulWidget {
  const _Pager();

  @override
  State<_Pager> createState() => _PagerState();
}

class _PagerState extends State<_Pager> {
  static const tabs = [
    GfTab(label: 'First', value: 0),
    GfTab(label: 'Second', value: 1),
  ];
  int _index = 0;

  @override
  Widget build(BuildContext context) => MaterialApp(
    home: Scaffold(
      body: TabSwipeSurface(
        index: _index,
        length: tabs.length,
        onChanged: (index) => setState(() => _index = index),
        child: Column(
          children: [
            GfTabBar(tabs: tabs, selected: _index, onSelected: (_) {}),
            Expanded(
              child: TabPageTransition(
                index: _index,
                length: tabs.length,
                pageKey: (index) => index,
                pageBuilder: (index, _) => ColoredBox(
                  key: ValueKey('page-$index'),
                  color: index == 0 ? Colors.red : Colors.blue,
                  child: Align(
                    alignment: Alignment.topLeft,
                    child: Text('Page $index'),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    ),
  );
}

class _HiddenChromePager extends StatefulWidget {
  const _HiddenChromePager();

  @override
  State<_HiddenChromePager> createState() => _HiddenChromePagerState();
}

class _HiddenChromePagerState extends State<_HiddenChromePager> {
  int _index = 0;
  bool _hidden = false;
  final _scrollControllers = List.generate(3, (_) => ScrollController());
  final _insetHeights = <int, double>{};

  @override
  void dispose() {
    for (final controller in _scrollControllers) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
    home: Scaffold(
      body: Column(
        children: [
          TextButton(
            onPressed: () => setState(() => _hidden = !_hidden),
            child: Text(_hidden ? 'Show chrome' : 'Hide chrome'),
          ),
          Expanded(
            child: TabSwipeSurface(
              index: _index,
              length: _scrollControllers.length,
              onChanged: (index) => setState(() => _index = index),
              child: TabPageTransition(
                index: _index,
                length: _scrollControllers.length,
                chromeHidden: _hidden,
                pageKey: (index) => index,
                pageBuilder: (index, chromeHidden) {
                  final inset = chromeHidden ? 0.0 : 48.0;
                  _insetHeights[index] = inset;
                  return ListView(
                    controller: _scrollControllers[index],
                    children: [
                      SizedBox(height: inset),
                      Text('Page $index'),
                      const SizedBox(height: 1000),
                    ],
                  );
                },
              ),
            ),
          ),
        ],
      ),
    ),
  );
}

void main() {
  testWidgets('tab taps animate the page with the selected tab', (
    tester,
  ) async {
    await tester.pumpWidget(const _Pager());
    await tester.tap(find.text('Second'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    final target = tester.getTopLeft(find.text('Page 1'));
    expect(target.dx, greaterThan(0));
    expect(target.dx, lessThan(tester.view.physicalSize.width));

    await tester.pumpAndSettle();
    expect(tester.getTopLeft(find.text('Page 1')).dx, closeTo(0, 1));
    expect(tester.takeException(), isNull);
  });

  testWidgets('horizontal swipes settle the adjacent page into place', (
    tester,
  ) async {
    await tester.pumpWidget(const _Pager());
    await tester.drag(find.text('Page 0'), const Offset(-280, 0));
    await tester.pumpAndSettle();

    expect(tester.getTopLeft(find.text('Page 1')).dx, closeTo(0, 1));
    expect(tester.takeException(), isNull);
  });

  testWidgets('hidden chrome removes cached header insets from visited tabs', (
    tester,
  ) async {
    await tester.pumpWidget(const _HiddenChromePager());
    expect(_insetHeight(tester, 0), 48);

    await tester.drag(find.text('Page 0'), const Offset(-280, 0));
    await tester.pumpAndSettle();
    expect(_insetHeight(tester, 1), 48);
    await tester.drag(find.text('Page 1'), const Offset(280, 0));
    await tester.pumpAndSettle();

    await tester.drag(find.text('Page 0'), const Offset(0, -160));
    await tester.pumpAndSettle();
    final pageZeroOffset = _scrollOffset(tester, 0);

    await tester.tap(find.text('Hide chrome'));
    await tester.pump();
    expect(_insetHeight(tester, 0), 0);
    expect(_scrollOffset(tester, 0), pageZeroOffset);

    await tester.drag(find.byType(TabPageTransition), const Offset(-280, 0));
    await tester.pumpAndSettle();
    expect(_insetHeight(tester, 1), 0);

    await tester.drag(find.text('Page 1'), const Offset(-280, 0));
    await tester.pumpAndSettle();
    expect(_insetHeight(tester, 2), 0);
    expect(_scrollOffset(tester, 2), 0);

    await tester.tap(find.text('Show chrome'));
    await tester.pump();
    expect(_insetHeight(tester, 2), 48);
    expect(tester.takeException(), isNull);
  });
}

double _scrollOffset(WidgetTester tester, int index) => tester
    .state<_HiddenChromePagerState>(find.byType(_HiddenChromePager))
    ._scrollControllers[index]
    .offset;

double _insetHeight(WidgetTester tester, int index) => tester
    .state<_HiddenChromePagerState>(find.byType(_HiddenChromePager))
    ._insetHeights[index]!;
