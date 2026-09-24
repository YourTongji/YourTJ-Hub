import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/gf_theme.dart';

/// A selection-aware composer with mutually exclusive keyboard/emoji surfaces.
/// Mobile return inserts a newline; hardware Ctrl/Cmd+Enter sends.
class GfChatInput extends StatefulWidget {
  const GfChatInput({
    super.key,
    required this.onSend,
    this.controller,
    this.hintText,
    this.sendLabel,
    this.enterHint,
    this.emojiLabel = 'Emoji',
    this.keyboardLabel = 'Keyboard',
    this.enabled = true,
    this.canSend = true,
  });

  final ValueChanged<String> onSend;
  final TextEditingController? controller;
  final String? hintText;
  final String? sendLabel;
  final String? enterHint;
  final String emojiLabel;
  final String keyboardLabel;
  final bool enabled;

  /// Keep drafting available while the caller prepares the conversation.
  final bool canSend;

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

  late TextEditingController _controller =
      widget.controller ?? TextEditingController();
  final FocusNode _inputFocus = FocusNode();
  TextSelection? _lastSelection;
  bool _emojiOpen = false;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleTextChanged);
    _rememberSelection();
  }

  void _rememberSelection() {
    final selection = _controller.selection;
    if (selection.isValid && selection.end <= _controller.text.length) {
      _lastSelection = selection;
    }
  }

  void _handleTextChanged() {
    _rememberSelection();
    if (mounted) setState(() {});
  }

  @override
  void didUpdateWidget(covariant GfChatInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      _controller.removeListener(_handleTextChanged);
      if (oldWidget.controller == null) _controller.dispose();
      _controller = widget.controller ?? TextEditingController();
      _lastSelection = null;
      _rememberSelection();
      _controller.addListener(_handleTextChanged);
    }
    if (!widget.enabled) _emojiOpen = false;
  }

  @override
  void dispose() {
    _controller.removeListener(_handleTextChanged);
    if (widget.controller == null) _controller.dispose();
    _inputFocus.dispose();
    super.dispose();
  }

  void _send() {
    if (!widget.enabled || !widget.canSend) return;
    final String text = _controller.text.trim();
    if (text.isEmpty) return;
    final submittedValue = _controller.value;
    widget.onSend(text);
    // A synchronous caller may install another draft after accepting the send.
    if (_controller.value == submittedValue) _controller.clear();
  }

  void _toggleInputSurface() {
    if (!widget.enabled) return;
    if (_emojiOpen) {
      setState(() => _emojiOpen = false);
      _inputFocus.requestFocus();
    } else {
      _rememberSelection();
      _inputFocus.unfocus();
      setState(() => _emojiOpen = true);
    }
  }

  void _insertEmoji(String emoji) {
    if (!widget.enabled) return;
    final text = _controller.text;
    final selection = _lastSelection;
    final start = (selection?.start ?? text.length).clamp(0, text.length);
    final end = (selection?.end ?? text.length).clamp(start, text.length);
    _controller.value = TextEditingValue(
      text: text.replaceRange(start, end, emoji),
      selection: TextSelection.collapsed(offset: start + emoji.length),
      // Opening an accessory commits the visible IME composition, never drops it.
      composing: TextRange.empty,
    );
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    return PopScope(
      canPop: !_emojiOpen,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && _emojiOpen) setState(() => _emojiOpen = false);
      },
      child: CallbackShortcuts(
        bindings: {
          const SingleActivator(LogicalKeyboardKey.enter, control: true): _send,
          const SingleActivator(LogicalKeyboardKey.enter, meta: true): _send,
          const SingleActivator(LogicalKeyboardKey.escape): () {
            if (_emojiOpen) setState(() => _emojiOpen = false);
            _inputFocus.unfocus();
          },
        },
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: colors.base100,
            border: Border(top: BorderSide(color: colors.line, width: 1)),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
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
                      focusNode: _inputFocus,
                      enabled: widget.enabled,
                      textInputAction: TextInputAction.newline,
                      minLines: 1,
                      maxLines: 4,
                      style: const TextStyle(fontSize: 16, height: 1.45),
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
                      onTap: () {
                        if (_emojiOpen) setState(() => _emojiOpen = false);
                      },
                    ),
                    Divider(
                      height: 1,
                      color: colors.line.withValues(alpha: 0.7),
                    ),
                    const SizedBox(height: 7),
                    Row(
                      children: <Widget>[
                        SizedBox(
                          width: 48,
                          height: 48,
                          child: IconButton(
                            padding: EdgeInsets.zero,
                            icon: Icon(
                              _emojiOpen
                                  ? Icons.keyboard_alt_outlined
                                  : Icons.emoji_emotions_outlined,
                              size: 20,
                              color: colors.iconMuted,
                            ),
                            onPressed: widget.enabled
                                ? _toggleInputSurface
                                : null,
                            tooltip: _emojiOpen
                                ? widget.keyboardLabel
                                : widget.emojiLabel,
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
                                color: colors.baseContent.withValues(
                                  alpha: 0.55,
                                ),
                                fontSize: 11,
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                          ),
                        ] else
                          const Spacer(),
                        FilledButton.icon(
                          onPressed:
                              widget.enabled &&
                                  widget.canSend &&
                                  _controller.text.trim().isNotEmpty
                              ? _send
                              : null,
                          icon: const Icon(Icons.send, size: 16),
                          label: Text(widget.sendLabel ?? 'Send'),
                          style: FilledButton.styleFrom(
                            minimumSize: const Size(48, 44),
                            padding: const EdgeInsets.symmetric(horizontal: 12),
                            textStyle: GfTheme.typographyOf(context).caption
                                .copyWith(
                                  fontSize: 14,
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
              // Let the keyboard finish closing before occupying its space. The
              // accessory is bounded and scrollable on short/resized windows.
              if (_emojiOpen && MediaQuery.viewInsetsOf(context).bottom == 0)
                ConstrainedBox(
                  constraints: BoxConstraints(
                    maxHeight: (MediaQuery.sizeOf(context).height * 0.3).clamp(
                      96.0,
                      240.0,
                    ),
                    maxWidth: 480,
                  ),
                  child: GridView.extent(
                    maxCrossAxisExtent: 64,
                    childAspectRatio: 1,
                    shrinkWrap: true,
                    padding: const EdgeInsets.only(top: 8),
                    children: [
                      for (final emoji in _emojis)
                        TextButton(
                          onPressed: () => _insertEmoji(emoji),
                          style: TextButton.styleFrom(
                            minimumSize: const Size(44, 44),
                            padding: EdgeInsets.zero,
                          ),
                          child: Text(
                            emoji,
                            style: const TextStyle(fontSize: 24),
                          ),
                        ),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
