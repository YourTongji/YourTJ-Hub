import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('server route fixture deserializes mention ranges', () {
    final json =
        jsonDecode(
              File(
                '../../../../packages/api-contract/fixtures/post-window-mention-success.json',
              ).readAsStringSync(),
            )
            as Map<String, dynamic>;
    final post = PostPayload.fromJson(
      json['result']['posts'][0] as Map<String, dynamic>,
    );
    expect(
      expandPostMentions(post.content, post.mentions),
      '😀 [@contract-window-author](/u/9301)',
    );
  });

  test(
    'UTF-16 mapping links only accepted occurrences without changing raw Markdown',
    () {
      const source =
          '😀 @alice_smith `@alice_smith` [@alice_smith](/) @unknown';
      const mention = PostMention(
        username: 'alice_smith',
        userId: 42,
        start: 3,
        end: 15,
      );
      expect(
        expandPostMentions(source, [mention]),
        r'😀 [@alice\_smith](/u/42) `@alice_smith` [@alice_smith](/) @unknown',
      );
      expect(expandPostMentions(source, []), source);
    },
  );
  test('invalid, stale and overlapping mappings safely stay text', () {
    const source = '@alice';
    expect(
      expandPostMentions(source, [
        const PostMention(username: 'alice', userId: 0, start: 0, end: 6),
        const PostMention(username: 'alice', userId: 1, start: -1, end: 6),
        const PostMention(username: 'alice', userId: 1, start: 0, end: 99),
        const PostMention(username: 'other', userId: 1, start: 0, end: 6),
      ]),
      source,
    );
    const valid = PostMention(username: 'alice', userId: 1, start: 0, end: 6);
    expect(expandPostMentions(source, [valid, valid]), '[@alice](/u/1)');
  });
}
