import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

import '../widgets/sticker_message_span.dart';

/// Plain chat text with selectable HTTP(S) links and existing inline stickers.
/// It deliberately does not interpret arbitrary message text as Markdown/HTML.
class MessageContent extends StatefulWidget {
  const MessageContent({
    super.key,
    required this.text,
    required this.stickers,
    required this.onOpenLink,
  });

  final String text;
  final Map<String, String> stickers;
  final ValueChanged<String> onOpenLink;

  @override
  State<MessageContent> createState() => _MessageContentState();
}

class _MessageContentState extends State<MessageContent> {
  final List<TapGestureRecognizer> _recognizers = [];

  void _disposeRecognizers() {
    for (final recognizer in _recognizers) {
      recognizer.dispose();
    }
    _recognizers.clear();
  }

  @override
  void dispose() {
    _disposeRecognizers();
    super.dispose();
  }

  TextSpan _linkText(String text) {
    final spans = <InlineSpan>[];
    var offset = 0;
    for (final match in RegExp(
      r'''https?://[^\s<>"“”，。！？；：、]+''',
      caseSensitive: false,
    ).allMatches(text)) {
      var end = match.end;
      while (end > match.start && '.,!?;:，。！？；：'.contains(text[end - 1])) {
        end--;
      }
      // Preserve balanced URL parentheses (e.g. Wikipedia titles).
      for (final pair in ['()', '[]', '{}']) {
        while (end > match.start && text[end - 1] == pair[1]) {
          final candidate = text.substring(match.start, end);
          if (pair[0].allMatches(candidate).length >=
              pair[1].allMatches(candidate).length) {
            break;
          }
          end--;
        }
      }
      final url = text.substring(match.start, end);
      final uri = Uri.tryParse(url);
      if (uri == null ||
          uri.host.isEmpty ||
          uri.userInfo.isNotEmpty ||
          url.contains('\\')) {
        continue;
      }
      spans.add(TextSpan(text: text.substring(offset, match.start)));
      final recognizer = TapGestureRecognizer()
        ..onTap = () => widget.onOpenLink(url);
      _recognizers.add(recognizer);
      spans.add(
        TextSpan(
          text: url,
          style: const TextStyle(decoration: TextDecoration.underline),
          recognizer: recognizer,
          mouseCursor: SystemMouseCursors.click,
        ),
      );
      offset = end;
    }
    spans.add(TextSpan(text: text.substring(offset)));
    return TextSpan(children: spans);
  }

  InlineSpan _linkSpan(InlineSpan span) {
    if (span is! TextSpan) return span;
    return TextSpan(
      children: [
        if (span.text != null) _linkText(span.text!),
        for (final child in span.children ?? <InlineSpan>[]) _linkSpan(child),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    _disposeRecognizers();
    final span =
        buildStickerMessageSpan(widget.text, widget.stickers) ??
        TextSpan(text: widget.text);
    return Text.rich(_linkSpan(span));
  }
}
