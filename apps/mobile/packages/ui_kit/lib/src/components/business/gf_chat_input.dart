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
    this.sendLabel,
    this.enterHint,
    this.enabled = true,
  });

  final ValueChanged<String> onSend;
  final TextEditingController? controller;
  final String? hintText;
  final String? sendLabel;
  final String? enterHint;
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
  void initState() {
    super.initState();
    _controller.addListener(_handleTextChanged);
  }

  void _handleTextChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _controller.removeListener(_handleTextChanged);
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
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: colors.base200.withValues(alpha: 0.8),
              borderRadius: BorderRadius.circular(radii.box),
              border: Border.all(color: colors.line),
            ),
            child: Column(
              children: <Widget>[
                TextField(
                  controller: _controller,
                  enabled: widget.enabled,
                  minLines: 1,
                  maxLines: 4,
                  style: const TextStyle(fontSize: 15, height: 1.45),
                  decoration: InputDecoration(
                    hintText: widget.hintText,
                    filled: false,
                    isDense: false,
                    constraints: const BoxConstraints(minHeight: 44),
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 9,
                    ),
                    border: InputBorder.none,
                    enabledBorder: InputBorder.none,
                    focusedBorder: InputBorder.none,
                  ),
                  onSubmitted: (_) => _send(),
                ),
                Divider(height: 1, color: colors.line.withValues(alpha: 0.7)),
                const SizedBox(height: 7),
                Row(
                  children: <Widget>[
                    SizedBox(
                      width: 32,
                      height: 32,
                      child: IconButton(
                        padding: EdgeInsets.zero,
                        icon: Icon(
                          _emojiOpen
                              ? Icons.keyboard_alt_outlined
                              : Icons.emoji_emotions_outlined,
                          size: 20,
                          color: colors.iconMuted,
                        ),
                        onPressed: () =>
                            setState(() => _emojiOpen = !_emojiOpen),
                        tooltip: 'Emoji',
                      ),
                    ),
                    if (widget.enterHint != null) ...<Widget>[
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          widget.enterHint!,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: colors.baseContent.withValues(alpha: 0.55),
                            fontSize: 11,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ),
                    ] else
                      const Spacer(),
                    FilledButton.icon(
                      onPressed:
                          widget.enabled && _controller.text.trim().isNotEmpty
                          ? _send
                          : null,
                      icon: const Icon(Icons.send, size: 16),
                      label: Text(widget.sendLabel ?? 'Send'),
                      style: FilledButton.styleFrom(
                        minimumSize: const Size(0, 32),
                        padding: const EdgeInsets.symmetric(horizontal: 12),
                        textStyle: const TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                        ),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(radii.field),
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
