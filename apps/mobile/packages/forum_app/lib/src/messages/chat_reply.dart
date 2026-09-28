import 'package:core/core.dart';

/// Upper bound for the quoted excerpt embedded in a reply.
const int chatReplyExcerptMaxLength = 120;

/// Single-line, bounded excerpt of a quoted message.
///
/// Sticker tokens expand to their readable preview label; line breaks and runs
/// of whitespace collapse so the excerpt stays one line. Truncation counts runes
/// so a multi-byte character (including emoji) is never split.
String chatReplyExcerpt(
  String content, {
  int maxLength = chatReplyExcerptMaxLength,
}) {
  if (maxLength <= 0) return '';
  final collapsed = stickerPreviewLabel(
    content,
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
