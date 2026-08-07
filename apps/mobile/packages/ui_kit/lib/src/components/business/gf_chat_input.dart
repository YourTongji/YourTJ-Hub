import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Chat input area mirroring web MessagesPage.vue mobile footer: a
/// `min-h-11 max-h-36` text area with an emoji button that opens a 4-column
/// emoji panel (web `w-48`), plus a send button. Enter sends, Shift+Enter
/// inserts a newline (handled by the caller via [onSend]).
class GfChatInput extends StatefulWidget {
  const GfChatInput({
    super.key,
    required this.onSend,
    this.controller,
    this.hintText,
    this.enabled = true,
  });

  final ValueChanged<String> onSend;
  final TextEditingController? controller;
  final String? hintText;
  final bool enabled;

  @override
  State<GfChatInput> createState() => _GfChatInputState();
}

class _GfChatInputState extends State<GfChatInput> {
  static const List<String> _emojis = <String>[
    '😀',
    '😄',
    '😂',
    '😊',
    '😍',
    '🤔',
    '😭',
    '😡',
    '👍',
    '👎',
    '👏',
    '🙏',
    '🎉',
    '❤️',
    '🔥',
    '✨',
  ];

  late final TextEditingController _controller =
      widget.controller ?? TextEditingController();
  bool _emojiOpen = false;

  @override
  void dispose() {
    if (widget.controller == null) _controller.dispose();
    super.dispose();
  }

  void _send() {
    final String text = _controller.text.trim();
    if (text.isEmpty) return;
    widget.onSend(text);
    _controller.clear();
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: colors.base100,
        border: Border(top: BorderSide(color: colors.line, width: 1)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          if (_emojiOpen)
            Container(
              margin: const EdgeInsets.only(bottom: 8),
              padding: const EdgeInsets.all(8),
              width: 192,
              decoration: BoxDecoration(
                color: colors.base200,
                borderRadius: BorderRadius.circular(radii.box),
              ),
              child: GridView.count(
                crossAxisCount: 4,
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                mainAxisSpacing: 4,
                crossAxisSpacing: 4,
                children: <Widget>[
                  for (final String emoji in _emojis)
                    InkWell(
                      borderRadius: BorderRadius.circular(6),
                      onTap: () {
                        _controller.text += emoji;
                        setState(() {});
                      },
                      child: Center(
                        child: Text(
                          emoji,
                          style: const TextStyle(fontSize: 20),
                        ),
                      ),
                    ),
                ],
              ),
            ),
          Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: <Widget>[
              IconButton(
                icon: Icon(
                  _emojiOpen
                      ? Icons.keyboard_alt_outlined
                      : Icons.emoji_emotions_outlined,
                  size: 22,
                  color: colors.iconMuted,
                ),
                onPressed: () => setState(() => _emojiOpen = !_emojiOpen),
                tooltip: 'Emoji',
              ),
              const SizedBox(width: 8),
              Expanded(
                child: TextField(
                  controller: _controller,
                  enabled: widget.enabled,
                  minLines: 1,
                  maxLines: 4,
                  style: const TextStyle(fontSize: 14),
                  decoration: InputDecoration(
                    hintText: widget.hintText,
                    filled: true,
                    fillColor: colors.base200,
                    isDense: true,
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 10,
                    ),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(radii.field),
                      borderSide: BorderSide.none,
                    ),
                    enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(radii.field),
                      borderSide: BorderSide.none,
                    ),
                    focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(radii.field),
                      borderSide: BorderSide(color: colors.primary, width: 1.5),
                    ),
                  ),
                  onSubmitted: (_) => _send(),
                ),
              ),
              const SizedBox(width: 8),
              IconButton(
                icon: Icon(Icons.send, size: 22, color: colors.primary),
                onPressed: widget.enabled ? _send : null,
                tooltip: 'Send',
              ),
            ],
          ),
        ],
      ),
    );
  }
}
