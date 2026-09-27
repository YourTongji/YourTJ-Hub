import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('SSE parser preserves split UTF-8, CRLF and multiline data', () async {
    final bytes = utf8.encode(
      ': ping\r\nevent: chat.changed\r\ndata: {"text":"同济"}\r\ndata: second\r\n\r\n',
    );
    final frames = await decodeForumSse(
      Stream.fromIterable([
        bytes.sublist(0, 33),
        bytes.sublist(33, 35),
        bytes.sublist(35),
      ]),
    ).toList();
    expect(frames, hasLength(1));
    expect(frames.single.event, 'chat.changed');
    expect(frames.single.data, '{"text":"同济"}\nsecond');
  });

  test(
    'SSE parser ignores incomplete EOF and rejects oversized frames',
    () async {
      expect(
        await decodeForumSse(
          Stream.value(utf8.encode('event: hello\ndata: {}')),
        ).toList(),
        isEmpty,
      );
      await expectLater(
        decodeForumSse(
          Stream.value(utf8.encode('data: ${'x' * 70000}\n\n')),
        ).toList(),
        throwsFormatException,
      );
    },
  );
}
