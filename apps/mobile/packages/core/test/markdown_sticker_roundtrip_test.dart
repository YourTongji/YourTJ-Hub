import 'package:core/core.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'rich composition keeps sticker references recognizable after saving',
    () {
      final converter = MarkdownConverter();
      final document = Document()
        ..insert(0, 'before [:sticker:user_42:] after');
      final markdown = converter.documentToMarkdown(document);
      expect(markdown, contains('[:sticker:user_42:]'));
      expect(
        expandStickerTokens(markdown, {'user_42': '/personal.png'}),
        contains('![sticker:user_42](/personal.png)'),
      );
      expect(
        converter.documentToMarkdown(converter.mdToDocument(markdown)),
        markdown,
      );
    },
  );

  test('rich composition continues escaping ordinary Markdown punctuation', () {
    final converter = MarkdownConverter();
    final document = Document()
      ..insert(0, '[ordinary] *text* [:sticker:smile:]');
    final markdown = converter.documentToMarkdown(document);
    expect(markdown, contains(r'\[ordinary\] \*text\*'));
  });

  test('inline code stickers stay opaque after a rich-editor round trip', () {
    final converter = MarkdownConverter();
    final document = converter.mdToDocument('`[:sticker:smile:]`');
    final markdown = converter.documentToMarkdown(document);
    expect(expandStickerTokens(markdown, {'smile': '/smile.png'}), markdown);
  });
  test('an explicitly escaped sticker remains literal prose', () {
    final converter = MarkdownConverter();
    final document = Document()..insert(0, r'\[:sticker:smile:]');
    final markdown = converter.documentToMarkdown(document);
    expect(expandStickerTokens(markdown, {'smile': '/smile.png'}), markdown);
  });
}
