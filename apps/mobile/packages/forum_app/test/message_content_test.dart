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
  for (final brightness in Brightness.values) {
    testWidgets('outgoing prose and links inherit white ($brightness)', (
      tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(brightness),
          home: Scaffold(
            body: GfMessageBubble(
              text: 'See https://example.com',
              mine: true,
              content: MessageContent(
                text: 'See https://example.com',
                stickers: const {},
                onOpenLink: (_) {},
              ),
            ),
          ),
        ),
      );
      final richText = tester.widget<RichText>(
        find.descendant(
          of: find.byType(MessageContent),
          matching: find.byType(RichText),
        ),
      );
      final root = richText.text as TextSpan;
      expect(root.style!.color, Colors.white);
      final link = textSpans(
        root,
      ).singleWhere((span) => span.recognizer != null);
      expect(link.style?.color ?? root.style!.color, Colors.white);
      expect(link.style!.decoration, TextDecoration.underline);
    });
  }

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
    final text = tester.widget<GfText>(find.byType(GfText));
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
    final text = tester.widget<GfText>(find.byType(GfText));
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
      final text = tester.widget<GfText>(
        find.descendant(
          of: find.byType(MessageContent),
          matching: find.byType(GfText),
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
    final text = tester.widget<GfText>(find.byType(GfText));
    expect(text.textSpan!.toPlainText(), content);
    expect(
      textSpans(text.textSpan!).where((span) => span.recognizer != null),
      isEmpty,
    );
  });

  testWidgets('chat links show the URL itself, never a Markdown label', (
    tester,
  ) async {
    // 聊天是纯文本渲染(MessageContent):Markdown 语法原样显示,可点 span 的文本
    // 就是被打开的 URL,因此聊天表面不会出现「标签与目标不符」,与同样不解释
    // Markdown 的 Web 聊天一致(issue #1200 review 对聊天表面的核对)。
    const content =
        '[https://bank.example/account](https://github.com/x) '
        'https://github.com/x';
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
    final text = tester.widget<GfText>(find.byType(GfText));
    expect(text.textSpan!.toPlainText(), content);
    final links = textSpans(
      text.textSpan!,
    ).where((span) => span.recognizer != null).toList();
    expect(links, isNotEmpty);
    for (final span in links) {
      // 每个可点 span 的可见文本就是它的目标:不存在显示 A、点开 B 的链接。
      final int before = opened.length;
      (span.recognizer as TapGestureRecognizer).onTap!();
      expect(opened.sublist(before), <String>[span.text!]);
    }
  });
}
