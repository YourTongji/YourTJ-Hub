import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

Widget _button({
  VoidCallback? onPressed,
  GfGlassTone tone = GfGlassTone.prominent,
  bool autofocus = false,
}) => GfGlassButton(
  key: const Key('glass'),
  tone: tone,
  tooltip: 'Publish',
  autofocus: autofocus,
  onPressed: onPressed,
  child: const GfSymbol('plus', size: 28),
);

Rect _surface(WidgetTester tester) => tester.getRect(
  find.descendant(
    of: find.byKey(const Key('glass')),
    matching: find.byType(GfNavigationSurface),
  ),
);

void main() {
  testWidgets('both tones are dock glass in both themes and activate', (
    tester,
  ) async {
    await forEachBrightness(tester, (tester, brightness) async {
      for (final tone in GfGlassTone.values) {
        var taps = 0;
        await tester.pumpWidget(
          gfApp(
            _button(tone: tone, onPressed: () => taps++),
            brightness: brightness,
          ),
        );
        expect(
          tester.getSize(find.byKey(const Key('glass'))),
          const Size(56, 56),
        );
        expect(
          find.descendant(
            of: find.byKey(const Key('glass')),
            matching: find.byType(BackdropFilter),
          ),
          findsOneWidget,
        );
        await tester.tap(find.byKey(const Key('glass')));
        await tester.pumpAndSettle();
        expect(taps, 1);
      }
    });
  });

  testWidgets('a press swells the glass, not the layout, and settles back', (
    tester,
  ) async {
    await tester.pumpWidget(gfApp(_button(onPressed: () {})));
    final rest = tester.getRect(find.byKey(const Key('glass')));
    expect(_surface(tester), rest);

    final gesture = await tester.startGesture(rest.center);
    await tester.pumpAndSettle();
    expect(_surface(tester).width, closeTo(rest.width + 6, 0.5));
    // Layout and hit testing keep the resting box.
    expect(tester.getRect(find.byKey(const Key('glass'))), rest);

    await gesture.up();
    await tester.pumpAndSettle();
    expect(_surface(tester), rest);
  });

  testWidgets('reduced motion and high contrast use the opaque surface', (
    tester,
  ) async {
    for (final data in const [
      MediaQueryData(disableAnimations: true),
      MediaQueryData(highContrast: true),
    ]) {
      await tester.pumpWidget(
        MediaQuery(
          data: data,
          child: gfApp(_button(onPressed: () {})),
        ),
      );
      expect(find.byType(BackdropFilter), findsNothing);
      final rest = tester.getRect(find.byKey(const Key('glass')));
      final gesture = await tester.startGesture(rest.center);
      await tester.pump();
      expect(_surface(tester), rest);
      await gesture.up();
      await tester.pump();
    }
  });

  testWidgets('keyboard focus activates with Enter and Space', (tester) async {
    var taps = 0;
    await tester.pumpWidget(
      gfApp(_button(autofocus: true, onPressed: () => taps++)),
    );
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.sendKeyEvent(LogicalKeyboardKey.space);
    expect(taps, 2);
  });

  testWidgets('it is one labelled button; disabled ignores taps', (
    tester,
  ) async {
    final handle = tester.ensureSemantics();
    await tester.pumpWidget(gfApp(_button(onPressed: () {})));
    expect(
      tester.getSemantics(find.byKey(const Key('glass'))),
      matchesSemantics(
        isButton: true,
        hasEnabledState: true,
        isEnabled: true,
        isFocusable: true,
        hasTapAction: true,
        hasFocusAction: true,
        tooltip: 'Publish',
      ),
    );
    await tester.pumpWidget(gfApp(_button()));
    final rest = tester.getRect(find.byKey(const Key('glass')));
    await tester.startGesture(rest.center);
    await tester.pumpAndSettle();
    expect(_surface(tester), rest);
    handle.dispose();
  });
}
