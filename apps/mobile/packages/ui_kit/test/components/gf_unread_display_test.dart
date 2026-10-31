import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

List<GfBottomNavigationItem> items({int? count}) => [
  const GfBottomNavigationItem(
    label: 'Home',
    symbol: 'home',
    selectedSymbol: 'home',
  ),
  const GfBottomNavigationItem(
    label: 'Campus',
    symbol: 'school',
    selectedSymbol: 'school',
  ),
  GfBottomNavigationItem(
    label: 'Notices',
    symbol: 'bell',
    selectedSymbol: 'bell',
    badge: true,
    badgeCount: count,
    badgeSemanticLabel: 'Unread',
  ),
  const GfBottomNavigationItem(
    label: 'Messages',
    symbol: 'message-circle',
    selectedSymbol: 'message-circle',
  ),
];

void main() {
  for (final width in [320.0, 390.0]) {
    for (final scale in [1.0, 2.0]) {
      testWidgets(
        'navigation count caps and stays inside the dock at $width/$scale; dot removes quantity',
        (tester) async {
          tester.view.physicalSize = Size(width, 780);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.reset);
          final semantics = tester.ensureSemantics();
          try {
            Widget surface(int? count) => gfApp(
              MediaQuery(
                data: MediaQueryData(
                  size: Size(width, 780),
                  textScaler: TextScaler.linear(scale),
                ),
                child: Align(
                  alignment: Alignment.bottomCenter,
                  child: GfBottomNavigation(
                    currentIndex: 0,
                    items: items(count: count),
                    showLabels: false,
                    onSelected: (_) {},
                  ),
                ),
              ),
            );
            await tester.pumpWidget(surface(149));
            await tester.pumpAndSettle();
            // Both lens layers carry the mark so it travels with its glyph.
            expect(find.text('99+'), findsNWidgets(2));
            expect(
              find.bySemanticsLabel('Notices, Unread: 149'),
              findsOneWidget,
            );
            final badge = tester.getRect(find.text('99+').first);
            final glyph = tester.getRect(
              find
                  .descendant(
                    of: find.byKey(
                      const ValueKey<String>(
                        'gf-bottom-navigation-resting-layer',
                      ),
                    ),
                    matching: find.byType(GfSymbol),
                  )
                  .at(2),
            );
            // Tucked into the glyph's top-trailing corner, not the slot's.
            expect(badge.left, lessThan(glyph.right));
            expect(badge.left, greaterThan(glyph.center.dx));
            expect(badge.center.dy, lessThan(glyph.center.dy));
            final dock = tester.getRect(find.byType(GfBottomNavigation));
            expect(badge.left, greaterThanOrEqualTo(dock.left));
            expect(badge.right, lessThanOrEqualTo(dock.right));
            expect(badge.top, greaterThanOrEqualTo(dock.top));
            expect(badge.bottom, lessThanOrEqualTo(dock.bottom));
            expect(tester.takeException(), isNull);
            await tester.pumpWidget(surface(null));
            await tester.pumpAndSettle();
            expect(find.text('99+'), findsNothing);
            expect(find.bySemanticsLabel('Notices, Unread'), findsOneWidget);
          } finally {
            semantics.dispose();
          }
        },
      );
    }
  }
  testWidgets(
    'conversation dot keeps unread semantics without a hidden number',
    (tester) async {
      final semantics = tester.ensureSemantics();
      try {
        await tester.pumpWidget(
          gfApp(
            const GfConversationRow(
              avatarUrl: '',
              name: 'Peer',
              lastMessage: 'Hello',
              time: 'Now',
              unreadCount: 36,
              unreadLabel: 'Unread',
              showUnreadCount: false,
            ),
          ),
        );
        expect(find.text('36'), findsNothing);
        expect(find.bySemanticsLabel(RegExp(r'\bUnread\b')), findsWidgets);
        expect(find.bySemanticsLabel(RegExp('36')), findsNothing);
      } finally {
        semantics.dispose();
      }
    },
  );
  testWidgets('a saving switch row cannot invoke either toggle target', (
    tester,
  ) async {
    var changes = 0;
    await tester.pumpWidget(
      gfApp(
        GfSwitchRow(
          title: 'Unread counts',
          value: true,
          enabled: false,
          onChanged: (_) => changes++,
        ),
      ),
    );
    expect(tester.widget<Switch>(find.byType(Switch)).onChanged, isNull);
    await tester.tap(find.text('Unread counts'));
    await tester.pump();
    expect(changes, 0);
  });
}
