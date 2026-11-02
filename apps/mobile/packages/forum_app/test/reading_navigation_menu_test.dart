import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:forum_app/src/navigation/window_reserved_regions.dart';
import 'package:ui_kit/ui_kit.dart';

Finder _destination(int index) =>
    find.byKey(ValueKey('rail-destination-$index'));
Finder _row(int index) => find.byKey(ValueKey('gf-dock-menu-row-$index'));

void main() {
  for (final brightness in Brightness.values) {
    for (final side in ['wide', 'left pocket', 'right pocket']) {
      testWidgets('$side shortcuts slide into content ($brightness)', (
        tester,
      ) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = Size(side == 'wide' ? 1000 : 520, 740);
        addTearDown(tester.view.reset);
        final padding = EdgeInsets.fromLTRB(
          side == 'left pocket' ? 96 : 0,
          24,
          side == 'right pocket' ? 96 : 0,
          20,
        );
        final picked = <int>[];
        final selected = <int>[];
        final items = _items(picked);
        await tester.pumpWidget(
          MaterialApp(
            theme: gfThemeData(
              brightness,
            ).copyWith(platform: TargetPlatform.iOS),
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(padding: padding, viewPadding: padding),
              child: child!,
            ),
            home: Builder(
              builder: (context) => ReadingWindow(
                reservedRegions: side == 'wide'
                    ? null
                    : WindowReservedRegions(
                        size: MediaQuery.sizeOf(context),
                        safeArea: padding,
                        occlusions: const [],
                      ),
                navigationRailWidth: ReadingNavigationRail.widthFor(
                  context,
                  items,
                ),
                rail: ReadingNavigationRail(
                  items: items,
                  currentIndex: 0,
                  onSelected: selected.add,
                ),
                bottomNavigation: const SizedBox.shrink(),
                child: const Center(child: Text('Reading content')),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final source = tester.getRect(_destination(1));
        final gesture = await tester.startGesture(source.center);
        await tester.pump(const Duration(milliseconds: 400));
        await tester.pumpAndSettle();
        expect(_row(0), findsOneWidget);
        final panel = tester.getRect(
          find.byKey(const ValueKey('gf-dock-menu-panel')),
        );
        expect(panel.left, greaterThanOrEqualTo(padding.left + 16));
        expect(
          panel.right,
          lessThanOrEqualTo(
            tester.view.physicalSize.width - padding.right - 16,
          ),
        );
        expect(panel.top, greaterThanOrEqualTo(padding.top + 16));
        expect(panel.bottom, lessThanOrEqualTo(740 - padding.bottom - 16));
        if (side == 'right pocket') {
          expect(panel.right, lessThan(source.left));
        } else {
          expect(panel.left, greaterThan(source.right));
        }
        await gesture.moveTo(tester.getCenter(_row(2)));
        await tester.pump(const Duration(milliseconds: 150));
        await gesture.up();
        await tester.pumpAndSettle();
        expect(picked, [2]);
        expect(selected, isEmpty);
        expect(_row(0), findsNothing);
        expect(tester.takeException(), isNull);
      });
    }
  }

  testWidgets(
    'rail exposes shortcuts to screen readers without a second tooltip',
    (tester) async {
      final semantics = tester.ensureSemantics();
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(1000, 740);
      addTearDown(tester.view.reset);
      final picked = <int>[];
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Align(
            alignment: Alignment.topLeft,
            child: SizedBox(
              width: 96,
              child: ReadingNavigationRail(
                items: _items(picked),
                currentIndex: 0,
                onSelected: (_) {},
              ),
            ),
          ),
        ),
      );
      final node = tester.getSemantics(_destination(1));
      expect(node.hint, 'Long press for shortcuts');
      expect(
        node.getSemanticsData().hasAction(SemanticsAction.longPress),
        isTrue,
      );
      tester.semantics.longPress(find.semantics.byLabel('Campus'));
      await tester.pumpAndSettle();
      expect(_row(0), findsOneWidget);
      expect(find.text('Campus'), findsOneWidget);
      await tester.tap(_row(0));
      await tester.pumpAndSettle();
      expect(picked, [0]);
      semantics.dispose();
    },
  );

  testWidgets('removing the source rail cancels its held menu', (tester) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(1000, 740);
    addTearDown(tester.view.reset);
    var visible = true;
    late StateSetter update;
    final picked = <int>[];
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return Align(
              alignment: Alignment.topLeft,
              child: SizedBox(
                width: 96,
                child: visible
                    ? ReadingNavigationRail(
                        items: _items(picked),
                        currentIndex: 0,
                        onSelected: (_) {},
                      )
                    : const SizedBox.shrink(),
              ),
            );
          },
        ),
      ),
    );
    final gesture = await tester.startGesture(
      tester.getCenter(_destination(1)),
    );
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    await gesture.moveTo(tester.getCenter(_row(0)));
    update(() => visible = false);
    await tester.pumpAndSettle();
    expect(_row(0), findsNothing);
    await gesture.up();
    expect(picked, isEmpty);
    expect(tester.takeException(), isNull);
  });
}

List<GfBottomNavigationItem> _items(List<int> picked) => [
  for (final (index, label, symbol) in [
    (0, 'Home', 'house'),
    (1, 'Campus', 'graduation-cap'),
    (2, 'Notifications', 'bell'),
    (3, 'Messages', 'message-circle'),
  ])
    GfBottomNavigationItem(
      label: label,
      symbol: symbol,
      selectedSymbol: '$symbol-filled',
      menuHint: index == 1 ? 'Long press for shortcuts' : null,
      menu: index != 1
          ? null
          : () => [
              for (var entry = 0; entry < 8; entry++)
                GfDockMenuEntry(
                  label: 'Section $entry',
                  symbol: 'sun',
                  onSelected: () => picked.add(entry),
                ),
            ],
    ),
];
