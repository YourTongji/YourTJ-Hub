import 'package:core/core.dart';

/// Upper bound for the quoted excerpt embedded in a reply.
const int chatReplyExcerptMaxLength = 120;

/// Stable wire excerpt, independent of the sender's display language.
const String chatImageReplyMarker = '[Image]';

String localizedChatReplyExcerpt(
  String excerpt, {
  required String imageLabel,
}) => excerpt == chatImageReplyMarker ? '[$imageLabel]' : excerpt;

class ChatReplyQuote {
  const ChatReplyQuote({
    required this.sender,
    required this.excerpt,
    required this.body,
  });

  final String sender;
  final String excerpt;
  final String body;
}

/// Reads the inline quote format used by current and already-sent replies.
///
/// Sender-less legacy headers (`> excerpt`) are indistinguishable from
/// hand-typed blockquotes, so both render as a sender-less quote block; the
/// message text itself is never altered.
ChatReplyQuote? parseChatReplyQuote(String content) {
  final separator = RegExp(r'\r?\n\r?\n').firstMatch(content);
  if (separator == null) return null;
  final firstLine = content
      .substring(0, separator.start)
      .replaceFirst(RegExp(r'\r$'), '');
  if (!firstLine.startsWith('>')) return null;

  final quote = firstLine.substring(1).trimLeft();
  final colon = quote.indexOf(':');
  final sender = colon < 0 ? '' : quote.substring(0, colon);
  final hasSender =
      colon >= 0 &&
      (sender.startsWith('@') ||
          const {'我', 'You', 'Ich', '自分'}.contains(sender)) &&
      (colon == quote.length - 1 || quote[colon + 1] == ' ');
  return ChatReplyQuote(
    sender: hasSender ? quote.substring(0, colon) : '',
    excerpt: hasSender ? quote.substring(colon + 1).trimLeft() : quote,
    body: content.substring(separator.end),
  );
}

/// Single-line, bounded excerpt of a quoted message.
///
/// Sticker tokens expand to their readable preview label; line breaks and runs
/// of whitespace collapse so the excerpt stays one line. Truncation counts runes
/// so a multi-byte character (including emoji) is never split.
String chatReplyExcerpt(
  String content, {
  int maxLength = chatReplyExcerptMaxLength,
  String? stickerLabel,
}) {
  if (maxLength <= 0) return '';
  final collapsed = stickerPreviewLabel(
    content,
    stickerLabel: stickerLabel,
  ).replaceAll(RegExp(r'\s+'), ' ').trim();
  final runes = collapsed.runes.toList(growable: false);
  if (runes.length <= maxLength) return collapsed;
  return '${String.fromCharCodes(runes.take(maxLength))}…';
}

/// Reply to one message of a conversation.
///
/// The chat send contract carries no reply/quote field, so the quote travels
/// inside the message text as a plain-text blockquote header. Chat bodies are
/// deliberately not Markdown-rendered, so the header reads as literal text on
/// every client while matching the conventional quoted-line form.
class ChatReplyTarget {
  const ChatReplyTarget({
    required this.messageId,
    required this.sender,
    required this.content,
  });

  final int messageId;

  /// Quoted author label, e.g. `@alice`; empty when the name is unknown.
  final String sender;

  /// Original message content the quote is derived from.
  final String content;

  String get excerpt => chatReplyExcerpt(content);

  String compose(String body) =>
      composeChatReply(sender: sender, content: content, body: body);
}

/// One message body: a `> sender: excerpt` quote line, a blank line, then the reply.
String composeChatReply({
  required String sender,
  required String content,
  required String body,
}) {
  final excerpt = chatReplyExcerpt(content);
  if (sender.isEmpty && excerpt.isEmpty) return body;
  final header = sender.isEmpty ? '>' : '> $sender:';
  return excerpt.isEmpty ? '$header\n\n$body' : '$header $excerpt\n\n$body';
}
