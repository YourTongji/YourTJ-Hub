import 'package:core/core.dart';
import 'package:flutter/services.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_test/flutter_test.dart';

List<Object?> embeds(Document document) => [
  for (final op in document.toDelta().toList())
    if (op.data is Map) op.data,
];

void main() {
  final converter = MarkdownConverter();

  test('prose tokens load as embeds and save back as tokens', () {
    const markdown = '今天 [:sticker:huaji:] 下课见 [:sticker:doge:]';
    final document = converter.mdToDocument(markdown);
    expect(embeds(document), [
      {stickerEmbedType: 'huaji'},
      {stickerEmbedType: 'doge'},
    ]);
    expect(document.toPlainText(), '今天 ￼ 下课见 ￼\n');
    expect(converter.documentToMarkdown(document).trim(), markdown);
  });

  test('code, links, math and escaped tokens stay text', () {
    const markdown =
        '`[:sticker:a:]` [x](/u) [:sticker:b:]\n\n'
        '```\n[:sticker:c:]\n```\n\n'
        r'$[:sticker:d:]$';
    final document = converter.mdToDocument(markdown);
    expect(embeds(document), [
      {stickerEmbedType: 'b'},
    ]);
    final saved = converter.documentToMarkdown(document);
    for (final name in ['a', 'b', 'c', 'd']) {
      expect(saved, contains('[:sticker:$name:]'));
    }
  });

  test('a backslash typed before a token keeps it literal', () {
    final document = Document()..insert(0, r'\[:sticker:e:] [:sticker:f:]');
    expect(documentStickerTokens(document), const [
      StickerTokenRange(15, 28, 'f'),
    ]);
  });

  test('a sticker inside emphasis keeps the emphasis whole', () {
    const markdown = '**粗体 [:sticker:huaji:] 粗体**';
    final document = converter.mdToDocument(markdown);
    expect(embeds(document), [
      {stickerEmbedType: 'huaji'},
    ]);
    expect(converter.documentToMarkdown(document).trim(), markdown);
  });

  test('typed tokens become embeds and the caret follows', () {
    final controller = QuillController(
      document: Document()..insert(0, 'a [:sticker:huaji:] b'),
      selection: const TextSelection.collapsed(offset: 19),
    );
    expect(embedStickerTokensIn(controller), isTrue);
    expect(embeds(controller.document), [
      {stickerEmbedType: 'huaji'},
    ]);
    expect(controller.document.toPlainText(), 'a ￼ b\n');
    expect(controller.selection, const TextSelection.collapsed(offset: 3));
    expect(embedStickerTokensIn(controller), isFalse);
  });

  test('undo after a typed token returns to the unclosed fragment', () {
    final controller = QuillController.basic();
    controller.replaceText(
      0,
      0,
      '[:sticker:huaji:',
      const TextSelection.collapsed(offset: 16),
    );
    // A later keystroke, outside the history merge window of the first.
    controller.document.history.lastRecorded = 0;
    controller.replaceText(
      16,
      0,
      ']',
      const TextSelection.collapsed(offset: 17),
    );
    embedStickerTokensIn(controller);
    expect(controller.document.toPlainText(), '￼\n');
    controller.undo();
    expect(controller.document.toPlainText(), '[:sticker:huaji:\n');
  });
}
