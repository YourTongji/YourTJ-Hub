import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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
  testWidgets('curly-quoted URLs preserve quotes outside the link', (
    tester,
  ) async {
    const content = '“https://example.com”';
    final opened = <String>[];
    await tester.pumpWidget(
      MaterialApp(
        home: MessageContent(
          text: content,
          stickers: const {},
          onOpenLink: opened.add,
        ),
      ),
    );
    final text = tester.widget<Text>(find.byType(Text));
    expect(text.textSpan!.toPlainText(), content);
    final links = textSpans(
      text.textSpan!,
    ).where((span) => span.recognizer != null).toList();
    expect(links.map((span) => span.text), ['https://example.com']);
    (links.single.recognizer as TapGestureRecognizer).onTap!();
    expect(opened, ['https://example.com']);
  });

  testWidgets('whole-message copy retains sticker tokens and URL source', (
    tester,
  ) async {
    String? copied;
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.setData') {
          copied = (call.arguments as Map)['text'] as String?;
        }
        if (call.method == 'Clipboard.hasStrings') {
          return {'value': copied != null};
        }
        if (call.method == 'Clipboard.getData') return {'text': copied};
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    const raw = 'Hello [:sticker:wave:] https://example.com';
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: GfMessageBubble(
            text: raw,
            mine: false,
            selectable: true,
            copyMessageLabel: 'Copy entire message',
            contentSpan: const TextSpan(
              children: [
                TextSpan(text: 'Hello '),
                WidgetSpan(child: SizedBox(width: 24, height: 24)),
                TextSpan(text: ' https://example.com'),
              ],
            ),
          ),
        ),
      ),
    );
    final selection = tester.state<SelectableRegionState>(
      find.byType(SelectableRegion),
    );
    await tester.longPress(find.textContaining('Hello'));
    selection.selectAll(SelectionChangedCause.toolbar);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Copy entire message'));
    await tester.pump();
    expect(copied, raw);
  });

  testWidgets('CJK prose after full-width punctuation is outside the URL', (
    tester,
  ) async {
    const content =
        'https://yourtj.de/p/post/123，明天讨论。https://example.com/中文？可以吗';
    await tester.pumpWidget(
      MaterialApp(
        home: MessageContent(
          text: content,
          stickers: const {},
          onOpenLink: (_) {},
        ),
      ),
    );
    final text = tester.widget<Text>(find.byType(Text));
    expect(text.textSpan!.toPlainText(), content);
    expect(
      textSpans(
        text.textSpan!,
      ).where((span) => span.recognizer != null).map((span) => span.text),
      ['https://yourtj.de/p/post/123', 'https://example.com/中文'],
    );
  });

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
