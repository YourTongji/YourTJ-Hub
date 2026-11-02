import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/navigation/reading_window.dart';
import 'package:ui_kit/ui_kit.dart';

const _items = [
  GfBottomNavigationItem(
    label: 'Startseite',
    symbol: 'home',
    selectedSymbol: 'home-filled',
  ),
  GfBottomNavigationItem(
    label: 'Campus',
    symbol: 'school',
    selectedSymbol: 'school-filled',
  ),
  GfBottomNavigationItem(
    label: 'Benachrichtigungen',
    symbol: 'bell',
    selectedSymbol: 'bell-filled',
    badge: true,
    badgeSemanticLabel: 'Ungelesen',
  ),
  GfBottomNavigationItem(
    label: 'Nachrichten',
    symbol: 'chat',
    selectedSymbol: 'chat-filled',
  ),
];

void main() {
  for (final brightness in Brightness.values) {
    testWidgets(
      'floating rail stays reachable in a short window ($brightness)',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = const Size(840, 320);
        addTearDown(tester.view.reset);
        final semantics = tester.ensureSemantics();
        var selected = 0;
        var highContrast = false;
        var reducedMotion = false;
        var accessibleNavigation = false;
        late StateSetter update;
        await tester.pumpWidget(
          MaterialApp(
            theme: gfThemeData(
              brightness,
            ).copyWith(platform: TargetPlatform.iOS),
            home: StatefulBuilder(
              builder: (context, setState) {
                update = setState;
                return MediaQuery(
                  data: MediaQuery.of(context).copyWith(
                    padding: const EdgeInsets.fromLTRB(28, 24, 76, 20),
                    viewPadding: const EdgeInsets.fromLTRB(28, 24, 76, 20),
                    textScaler: const TextScaler.linear(2),
                    highContrast: highContrast,
                    disableAnimations: reducedMotion,
                    accessibleNavigation: accessibleNavigation,
                  ),
                  child: Builder(
                    builder: (context) => ReadingWindow(
                      navigationRailWidth: ReadingNavigationRail.widthFor(
                        context,
                        _items,
                      ),
                      rail: ReadingNavigationRail(
                        items: _items,
                        currentIndex: selected,
                        onSelected: (index) => setState(() => selected = index),
                      ),
                      bottomNavigation: const SizedBox.shrink(),
                      child: const SizedBox.expand(),
                    ),
                  ),
                );
              },
            ),
          ),
        );
        await tester.pumpAndSettle();
        final rail = find.byType(ReadingNavigationRail);
        final glass = find.descendant(
          of: rail,
          matching: find.byType(GfNavigationSurface),
        );
        final railRect = tester.getRect(rail);
        final glassRect = tester.getRect(glass);
        expect(glassRect.left, railRect.left + 8);
        expect(glassRect.top, railRect.top + 8);
        expect(glassRect.right, railRect.right - 8);
        expect(glassRect.bottom, lessThanOrEqualTo(railRect.bottom - 8));
        expect(
          find.descendant(of: rail, matching: find.byType(BackdropFilter)),
          findsOneWidget,
        );
        await tester.scrollUntilVisible(find.text('Benachrichtigungen'), 120);
        final unread = tester.getSemantics(
          find.byKey(const ValueKey('rail-destination-2')),
        );
        expect(unread.label, contains('Ungelesen'));
        await tester.scrollUntilVisible(find.text('Nachrichten'), 120);
        await tester.tap(find.text('Nachrichten'));
        await tester.pumpAndSettle();
        expect(selected, 3);
        final button = find.widgetWithText(TextButton, 'Nachrichten');
        expect(tester.getSize(button).width, greaterThanOrEqualTo(44));
        expect(tester.getSize(button).height, greaterThanOrEqualTo(44));
        for (var mode = 0; mode < 3; mode++) {
          update(() {
            highContrast = mode == 0;
            reducedMotion = mode == 1;
            accessibleNavigation = mode == 2;
          });
          await tester.pumpAndSettle();
          expect(
            find.descendant(of: rail, matching: find.byType(BackdropFilter)),
            findsNothing,
          );
          expect(selected, 3);
        }
        semantics.dispose();
        expect(tester.takeException(), isNull);
      },
    );
  }
}
