import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('replacement text discards the old selected range', (
    tester,
  ) async {
    final controller = TextEditingController(text: 'old selection');
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      gfApp(GfChatInput(controller: controller, onSend: (_) {})),
    );
    controller.selection = const TextSelection(baseOffset: 0, extentOffset: 3);
    controller.text = 'new draft';
    await tester.tap(find.byTooltip('Emoji'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('😀'));
    expect(controller.text, 'new draft😀');
  });

  testWidgets('pointer-opened emoji panel handles hardware Escape', (
    tester,
  ) async {
    await tester.pumpWidget(gfApp(GfChatInput(onSend: (_) {})));
    await tester.tap(find.byType(TextField));
    await tester.pump();
    await tester.tap(find.byTooltip('Emoji'));
    await tester.pumpAndSettle();
    expect(find.text('😀'), findsOneWidget);
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.text('😀'), findsNothing);
    expect(tester.testTextInput.isVisible, isFalse);
  });

  testWidgets('send keeps a new draft installed by the accepting callback', (
    tester,
  ) async {
    final controller = TextEditingController(text: 'first');
    addTearDown(controller.dispose);
    final sent = <String>[];
    await tester.pumpWidget(
      gfApp(
        GfChatInput(
          controller: controller,
          onSend: (text) {
            sent.add(text);
            controller.text = 'next draft';
          },
        ),
      ),
    );
    await tester.tap(find.text('Send'));
    expect(sent, ['first']);
    expect(controller.text, 'next draft');
  });

  testWidgets(
    'composer accepts replacement controller and preserves IME text',
    (tester) async {
      final first = TextEditingController(text: 'old');
      final next = TextEditingController.fromValue(
        const TextEditingValue(
          text: '输入内容',
          selection: TextSelection.collapsed(offset: 2),
          composing: TextRange(start: 0, end: 2),
        ),
      );
      addTearDown(first.dispose);
      addTearDown(next.dispose);
      await tester.pumpWidget(
        gfApp(GfChatInput(controller: first, onSend: (_) {})),
      );
      await tester.pumpWidget(
        gfApp(GfChatInput(controller: next, onSend: (_) {})),
      );
      await tester.tap(find.byIcon(Icons.emoji_emotions_outlined));
      await tester.pumpAndSettle();
      await tester.tap(find.text('😀'));
      expect(next.text, '输入😀内容');
      expect(first.text, 'old');
      expect(next.value.composing, TextRange.empty);
    },
  );

  testWidgets(
    'emoji controls fit short narrow windows and have touch targets',
    (tester) async {
      tester.view.physicalSize = const Size(320, 500);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        gfApp(
          MediaQuery(
            data: const MediaQueryData(
              size: Size(320, 500),
              textScaler: TextScaler.linear(2),
            ),
            child: GfChatInput(onSend: (_) {}),
          ),
        ),
      );
      final toggle = find.byTooltip('Emoji');
      expect(tester.getSize(toggle).shortestSide, greaterThanOrEqualTo(44));
      await tester.tap(toggle);
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      final emojiButton = find.widgetWithText(TextButton, '😀');
      expect(
        tester.getSize(emojiButton).shortestSide,
        greaterThanOrEqualTo(44),
      );
    },
  );

  testWidgets('emoji replaces selected text and leaves caret after insertion', (
    tester,
  ) async {
    final controller = TextEditingController(text: 'hello world');
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      gfApp(GfChatInput(controller: controller, onSend: (_) {})),
    );
    controller.selection = const TextSelection(baseOffset: 6, extentOffset: 11);
    await tester.tap(find.byIcon(Icons.emoji_emotions_outlined));
    await tester.pumpAndSettle();
    await tester.tap(find.text('😀'));
    expect(controller.text, 'hello 😀');
    expect(controller.selection, const TextSelection.collapsed(offset: 8));
    await tester.tap(find.text('👍'));
    expect(controller.text, 'hello 😀👍');
    expect(controller.selection, const TextSelection.collapsed(offset: 10));
  });

  testWidgets(
    'emoji panel releases text focus and keyboard toggle restores it',
    (tester) async {
      await tester.pumpWidget(gfApp(GfChatInput(onSend: (_) {})));
      await tester.tap(find.byType(TextField));
      await tester.pump();
      expect(tester.testTextInput.isVisible, isTrue);
      await tester.tap(find.byIcon(Icons.emoji_emotions_outlined));
      await tester.pumpAndSettle();
      expect(tester.testTextInput.isVisible, isFalse);
      await tester.tap(find.byIcon(Icons.keyboard_alt_outlined));
      await tester.pumpAndSettle();
      expect(tester.testTextInput.isVisible, isTrue);
      expect(find.text('😀'), findsNothing);
    },
  );

  testWidgets('disabled composer cannot change text with emoji', (
    tester,
  ) async {
    final controller = TextEditingController(text: 'draft');
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      gfApp(
        GfChatInput(controller: controller, enabled: false, onSend: (_) {}),
      ),
    );
    await tester.tap(find.byIcon(Icons.emoji_emotions_outlined));
    await tester.pumpAndSettle();
    expect(find.text('😀'), findsNothing);
    expect(controller.text, 'draft');
  });
}
