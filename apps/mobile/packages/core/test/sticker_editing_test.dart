import 'dart:ui' show TextRange;

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('stickerTokenRanges', () {
    test(
      'Markdown excludes code, links and escapes; plain text only escapes',
      () {
        const content =
            '`[:sticker:a:]` [:sticker:b:] [x](/[:sticker:c:]) \\[:sticker:d:]';
        expect(stickerTokenRanges(content, markdown: true), const [
          StickerTokenRange(16, 29, 'b'),
        ]);
        expect(
          stickerTokenRanges(content, markdown: false).map((r) => r.name),
          ['a', 'b', 'c'],
        );
      },
    );
  });

  group('expandStickerDeletion', () {
    const text = '你好[:sticker:huaji:]！';
    test('one backspace after a token removes all of it', () {
      expect(
        expandStickerDeletion(text, 18, 19),
        const TextRange(start: 2, end: 19),
      );
    });
    test('a delete at the token start removes all of it', () {
      expect(
        expandStickerDeletion(text, 2, 3),
        const TextRange(start: 2, end: 19),
      );
    });
    test('half-typed fragments are atomic too', () {
      expect(
        expandStickerDeletion('图 [:sticker:na', 13, 14),
        const TextRange(start: 2, end: 14),
      );
      expect(
        expandStickerDeletion('[:sticker:huaji:', 15, 16),
        const TextRange(start: 0, end: 16),
      );
    });
    test('plain text keeps single-character deletion', () {
      expect(
        expandStickerDeletion(text, 19, 20),
        const TextRange(start: 19, end: 20),
      );
      expect(
        expandStickerDeletion('你好', 1, 2),
        const TextRange(start: 1, end: 2),
      );
    });
  });

  test('extractStickerQuery mirrors the web rules', () {
    expect(
      extractStickerQuery('看 [:sticker:hu'),
      const StickerQuery(length: 12, query: 'hu'),
    );
    expect(
      extractStickerQuery('[:sticker:'),
      const StickerQuery(length: 10, query: ''),
    );
    expect(extractStickerQuery('[:sticker:huaji:]'), isNull);
    expect(extractStickerQuery('[:sticker:hu aji'), isNull);
    expect(extractStickerQuery(r'\[:sticker:hu'), isNull);
  });

  test('rankStickerSuggestions orders exact, prefix, then substring', () {
    final items = [
      ('u_1', '我的滑稽'),
      ('huaji', 'huaji'),
      ('shuai', 'shuai'),
      ('hug', 'hug'),
      ('huaji', 'duplicate'),
    ];
    List<String> rank(String query, {int limit = 8}) => rankStickerSuggestions(
      items,
      query,
      name: (item) => item.$1,
      label: (item) => item.$2,
      limit: limit,
    ).map((item) => item.$1).toList();
    expect(rank(''), ['u_1', 'huaji', 'shuai', 'hug']);
    expect(rank('', limit: 2), ['u_1', 'huaji']);
    expect(rank('hu'), ['huaji', 'hug', 'shuai']);
    expect(rank('hug'), ['hug']);
    expect(rank('我的'), ['u_1']);
    expect(rank('zzz'), isEmpty);
  });
}
