import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// Exercise the actual Impeller shader, including translated/scaled ancestors.
/// Widget tests cover the fallback; this test must run on an Impeller device.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'optical controls survive transforms, interruption and live fallback',
    (tester) async {
      expect(ui.ImageFilter.isShaderFilterSupported, isTrue);
      await ui.FragmentProgram.fromAsset(
        'packages/ui_kit/shaders/liquid_glass.frag',
      );
      final position = ValueNotifier(Offset.zero);
      final opaque = ValueNotifier(false);
      final controller = TextEditingController(text: '保留草稿');
      final focus = FocusNode();
      addTearDown(position.dispose);
      addTearDown(opaque.dispose);
      addTearDown(controller.dispose);
      addTearDown(focus.dispose);
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Stack(
              children: [
                const Positioned.fill(
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        colors: [Colors.blue, Colors.orange],
                      ),
                    ),
                  ),
                ),
                Center(
                  child: ValueListenableBuilder(
                    valueListenable: position,
                    builder: (_, offset, child) => Transform.translate(
                      offset: offset,
                      child: Transform.scale(scale: .97, child: child),
                    ),
                    child: ValueListenableBuilder(
                      valueListenable: opaque,
                      builder: (_, value, _) => GfGlassSettings(
                        reduceTransparency: value,
                        child: SizedBox(
                          width: 280,
                          child: GfLiquidSurface(
                            pressable: true,
                            child: TextField(
                              controller: controller,
                              focusNode: focus,
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      focus.requestFocus();
      await tester.pumpAndSettle();
      final editor = tester.state(find.byType(EditableText));
      for (final fraction in [.1, .3, .7, 1.0]) {
        position.value = Offset(40 * fraction, -60 * fraction);
        await tester.pump();
        expect(tester.takeException(), isNull);
        opaque.value = true;
        await tester.pump();
        expect(tester.state(find.byType(EditableText)), same(editor));
        expect(controller.text, '保留草稿');
        expect(focus.hasFocus, isTrue);
        opaque.value = false;
        position.value = Offset.zero;
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
      }
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );
}
