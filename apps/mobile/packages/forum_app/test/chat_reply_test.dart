import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/messages/chat_reply.dart';

void main() {
  group('parseChatReplyQuote', () {
    test('reads the existing sender and excerpt prefix', () {
      final quote = parseChatReplyQuote('> @bob: 你好\n\n收到\n第二行');
      expect(quote?.sender, '@bob');
      expect(quote?.excerpt, '你好');
      expect(quote?.body, '收到\n第二行');
    });

    test(
      'supports self labels and CRLF messages without changing the body',
      () {
        final quote = parseChatReplyQuote(
          '> You: Original\r\n\r\nReply\r\n\r\nNext',
        );
        expect(quote?.sender, 'You');
        expect(quote?.excerpt, 'Original');
        expect(quote?.body, 'Reply\r\n\r\nNext');
      },
    );

    test('returns null for ordinary messages', () {
      expect(parseChatReplyQuote('ordinary\n\nmessage'), isNull);
    });

    test('renders sender-less quote lines without a sender row', () {
      // Legacy replies composed with an unknown sender send `> excerpt`; a
      // hand-typed blockquote is indistinguishable, so both keep the text
      // intact and render as a sender-less quote block.
      final quote = parseChatReplyQuote('> 逐字引用\n\n正文');
      expect(quote?.sender, '');
      expect(quote?.excerpt, '逐字引用');
      expect(quote?.body, '正文');
    });

    test('keeps unrecognized sender labels in the excerpt', () {
      final quote = parseChatReplyQuote('> bob: hi\n\nreply');
      expect(quote?.sender, '');
      expect(quote?.excerpt, 'bob: hi');
      expect(quote?.body, 'reply');
    });
  });

  group('chatReplyExcerpt', () {
    test('collapses line breaks and runs of whitespace', () {
      expect(chatReplyExcerpt('第一行\n\n第二行\t结束  '), '第一行 第二行 结束');
    });

    test('keeps short content untouched', () {
      expect(chatReplyExcerpt('消息 1'), '消息 1');
    });

    test('bounds long content with one trailing marker', () {
      final excerpt = chatReplyExcerpt('a' * 500, maxLength: 10);
      expect(excerpt, '${'a' * 10}…');
    });

    test('never splits a surrogate pair while truncating', () {
      final excerpt = chatReplyExcerpt('😀' * 50, maxLength: 5);
      expect(excerpt, '${'😀' * 5}…');
      expect(excerpt.runes.length, 6);
    });

    test('expands sticker tokens to their readable preview label', () {
      expect(chatReplyExcerpt('[:sticker:smile:] 早'), '[smile] 早');
    });

    test('a non-positive bound yields an empty excerpt', () {
      expect(chatReplyExcerpt('消息', maxLength: 0), '');
    });
  });

  group('composeChatReply', () {
    test('prefixes a plain-text quote and keeps the reply body', () {
      expect(
        composeChatReply(sender: '@bob', content: '你好', body: '收到'),
        '> @bob: 你好\n\n收到',
      );
    });

    test('omits a missing sender or empty excerpt', () {
      expect(
        composeChatReply(sender: '', content: '你好', body: '收到'),
        '> 你好\n\n收到',
      );
      expect(
        composeChatReply(sender: '@bob', content: '   ', body: '收到'),
        '> @bob:\n\n收到',
      );
    });

    test('without any quote context only the body is sent', () {
      expect(composeChatReply(sender: '', content: '', body: '收到'), '收到');
    });

    test('bounds the quoted excerpt of very long content', () {
      final content = 'a' * 5000;
      final text = composeChatReply(
        sender: '@bob',
        content: content,
        body: '收到',
      );
      final header = text.split('\n\n').first;
      expect(
        header.length,
        lessThanOrEqualTo(chatReplyExcerptMaxLength + '> @bob: '.length + 1),
      );
      expect(header, endsWith('…'));
    });

    test('keeps multi-line reply bodies intact', () {
      expect(
        composeChatReply(sender: '@bob', content: 'hi', body: 'a\nb'),
        '> @bob: hi\n\na\nb',
      );
    });
  });

  group('ChatReplyTarget', () {
    test(
      'derives a bounded excerpt and composes from the original content',
      () {
        final target = ChatReplyTarget(
          messageId: 7,
          sender: '@bob',
          content: '你好\n世界',
        );
        expect(target.messageId, 7);
        expect(target.excerpt, '你好 世界');
        expect(target.compose('收到'), '> @bob: 你好 世界\n\n收到');
      },
    );
  });
}
