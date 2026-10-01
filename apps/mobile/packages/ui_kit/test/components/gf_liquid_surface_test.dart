import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('live solid fallback preserves editing, focus and selection', (
    tester,
  ) async {
    final solid = ValueNotifier(false);
    final controller = TextEditingController(text: '尚未发送的草稿');
    final focus = FocusNode();
    addTearDown(solid.dispose);
    addTearDown(controller.dispose);
    addTearDown(focus.dispose);
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: ValueListenableBuilder(
            valueListenable: solid,
            builder: (_, value, child) => GfGlassSettings(
              reduceTransparency: value,
              child: GfLiquidSurface(
                child: TextField(controller: controller, focusNode: focus),
              ),
            ),
          ),
        ),
      ),
    );
    focus.requestFocus();
    controller.selection = const TextSelection(baseOffset: 1, extentOffset: 4);
    await tester.pump();
    final field = tester.state(find.byType(EditableText));
    expect(tester.layers.whereType<BackdropFilterLayer>(), isNotEmpty);
    solid.value = true;
    await tester.pump();
    expect(tester.layers.whereType<BackdropFilterLayer>(), isEmpty);
    expect(tester.state(find.byType(EditableText)), same(field));
    expect(focus.hasFocus, isTrue);
    expect(controller.text, '尚未发送的草稿');
    expect(
      controller.selection,
      const TextSelection(baseOffset: 1, extentOffset: 4),
    );
    solid.value = false;
    await tester.pump();
    expect(tester.layers.whereType<BackdropFilterLayer>(), isNotEmpty);
    expect(tester.state(find.byType(EditableText)), same(field));
  });

  for (final fraction in [.1, .3, .7, 1.0]) {
    testWidgets('press cancellation at $fraction settles without activating', (
      tester,
    ) async {
      var taps = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Center(
              child: GfLiquidSurface(
                pressable: true,
                child: IconButton(
                  tooltip: 'Action',
                  onPressed: () => taps++,
                  icon: const Icon(Icons.add),
                ),
              ),
            ),
          ),
        ),
      );
      final gesture = await tester.startGesture(
        tester.getCenter(find.byTooltip('Action')),
      );
      await tester.pump();
      await tester.pump(
        Duration(
          microseconds: (GfMotion.press.inMicroseconds * fraction).round(),
        ),
      );
      await gesture.cancel();
      await tester.pumpAndSettle();
      expect(taps, 0);
      expect(
        tester
            .widget<Transform>(
              find.byKey(const Key('gf-liquid-press-transform')),
            )
            .transform
            .storage[0],
        1,
      );
      await tester.tap(find.byTooltip('Action'));
      await tester.pumpAndSettle();
      expect(taps, 1);
      expect(tester.binding.transientCallbackCount, 0);
    });
  }

  testWidgets('contrast and reduced motion remove all backdrop filtering', (
    tester,
  ) async {
    for (final media in [
      const MediaQueryData(highContrast: true),
      const MediaQueryData(disableAnimations: true),
      const MediaQueryData(accessibleNavigation: true),
    ]) {
      await tester.pumpWidget(
        MaterialApp(
          home: MediaQuery(
            data: media,
            child: const Scaffold(
              body: GfLiquidSurface(child: Text('Readable')),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.layers.whereType<BackdropFilterLayer>(), isEmpty);
      expect(find.text('Readable'), findsOneWidget);
    }
  });
}
