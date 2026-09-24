import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/messages/message_content.dart';
import 'package:ui_kit/ui_kit.dart';

Iterable<TextSpan> textSpans(InlineSpan root) sync* {
  if (root is TextSpan) {
    yield root;
    for (final child in root.children ?? <InlineSpan>[]) {
      yield* textSpans(child);
    }
  }
}

void main() {
  testWidgets(
    'message links preserve literal text and strip prose punctuation',
    (tester) async {
      const content =
          '看 "https://example.com/a" 和 '
          '(https://example.com/Title_(test)). javascript:alert(1)';
      final opened = <String>[];
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GfMessageBubble(
              text: content,
              mine: false,
              selectable: true,
              content: MessageContent(
                text: content,
                stickers: const {},
                onOpenLink: opened.add,
              ),
            ),
          ),
        ),
      );
      expect(find.byType(SelectionArea), findsOneWidget);
      final text = tester.widget<Text>(
        find.descendant(
          of: find.byType(MessageContent),
          matching: find.byType(Text),
        ),
      );
      expect(text.textSpan!.toPlainText(), content);
      final links = textSpans(
        text.textSpan!,
      ).where((span) => span.recognizer != null);
      expect(links.map((span) => span.text), [
        'https://example.com/a',
        'https://example.com/Title_(test)',
      ]);
      for (final link in links) {
        (link.recognizer! as TapGestureRecognizer).onTap!();
      }
      expect(opened, links.map((span) => span.text));
      await tester.pumpWidget(const SizedBox());
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('credentials and malformed links stay plain text', (
    tester,
  ) async {
    const content =
        'https://user:pass@example.com https:///bad '
        r'https://example.com\evil';
    await tester.pumpWidget(
      MaterialApp(
        home: MessageContent(
          text: content,
          stickers: const {},
          onOpenLink: (_) => fail('Unsafe link opened'),
        ),
      ),
    );
    final text = tester.widget<Text>(find.byType(Text));
    expect(text.textSpan!.toPlainText(), content);
    expect(
      textSpans(text.textSpan!).where((span) => span.recognizer != null),
      isEmpty,
    );
  });
}
