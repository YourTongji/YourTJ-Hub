import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';

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
    this.clearOnSend = true,
    this.accessoryBuilder,
    this.onAttach,
    this.attachLabel,
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

  /// Let an owning draft controller clear only after server acknowledgement.
  final bool clearOnSend;

  /// App-owned stickers use the same selection-preserving insertion callback.
  final Widget Function(ValueChanged<String> insert)? accessoryBuilder;
  final VoidCallback? onAttach;
  final String? attachLabel;

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
  final FocusNode _accessoryFocus = FocusNode();
  TextSelection? _lastSelection;
  bool _emojiOpen = false;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleTextChanged);
    _inputFocus.addListener(_handleFocusChanged);
    _rememberSelection();
  }

  void _handleFocusChanged() {
    if (mounted) setState(() {});
  }

  void _rememberSelection() {
    final selection = _controller.selection;
    if (selection.isValid && selection.end <= _controller.text.length) {
      _lastSelection = selection;
    } else {
      _lastSelection = null;
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
    _inputFocus.removeListener(_handleFocusChanged);
    _inputFocus.dispose();
    _accessoryFocus.dispose();
    super.dispose();
  }

  void _send() {
    if (!widget.enabled || !widget.canSend) return;
    final String text = _controller.text.trim();
    if (text.isEmpty) return;
    final submittedValue = _controller.value;
    widget.onSend(text);
    // A synchronous caller may install another draft after accepting the send.
    if (widget.clearOnSend && _controller.value == submittedValue) {
      _controller.clear();
    }
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
      _accessoryFocus.requestFocus();
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
    final colors = GfTheme.colorsOf(context);
    final ready =
        widget.enabled && widget.canSend && _controller.text.trim().isNotEmpty;
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
        child: Focus(
          focusNode: _accessoryFocus,
          skipTraversal: true,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: colors.base100,
              border: Border(top: BorderSide(color: colors.line)),
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    if (widget.onAttach != null) ...[
                      IconButton(
                        key: const Key('chat-attach'),
                        tooltip: widget.attachLabel,
                        icon: const GfSymbol('plus', size: 24),
                        onPressed: widget.enabled ? widget.onAttach : null,
                        style: IconButton.styleFrom(
                          fixedSize: const Size.square(44),
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                          backgroundColor: colors.base200,
                          foregroundColor: colors.baseContent,
                          disabledForegroundColor: colors.iconMuted.withValues(
                            alpha: .45,
                          ),
                          shape: const CircleBorder(),
                          padding: EdgeInsets.zero,
                        ),
                      ),
                      const SizedBox(width: 8),
                    ],
                    Expanded(
                      child: AnimatedContainer(
                        key: const Key('chat-input-surface'),
                        duration: const Duration(milliseconds: 140),
                        decoration: BoxDecoration(
                          color: colors.base200,
                          borderRadius: BorderRadius.circular(24),
                          border: Border.all(
                            color: _inputFocus.hasFocus
                                ? colors.primary.withValues(alpha: .32)
                                : Colors.transparent,
                          ),
                        ),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.end,
                          children: [
                            Expanded(
                              child: TextField(
                                controller: _controller,
                                focusNode: _inputFocus,
                                enabled: widget.enabled,
                                textInputAction: TextInputAction.newline,
                                minLines: 1,
                                maxLines: 4,
                                style: TextStyle(
                                  fontSize: 16,
                                  height: 1.4,
                                  color: widget.enabled
                                      ? colors.baseContent
                                      : colors.iconMuted,
                                ),
                                cursorColor: colors.primary,
                                decoration: InputDecoration(
                                  hintText: widget.hintText,
                                  hintStyle: TextStyle(color: colors.iconMuted),
                                  filled: false,
                                  isDense: true,
                                  constraints: const BoxConstraints(
                                    minHeight: 44,
                                  ),
                                  contentPadding: const EdgeInsets.fromLTRB(
                                    16,
                                    11,
                                    4,
                                    11,
                                  ),
                                  border: InputBorder.none,
                                  enabledBorder: InputBorder.none,
                                  focusedBorder: InputBorder.none,
                                  disabledBorder: InputBorder.none,
                                  errorBorder: InputBorder.none,
                                  focusedErrorBorder: InputBorder.none,
                                ),
                                onTap: () {
                                  if (_emojiOpen) {
                                    setState(() => _emojiOpen = false);
                                  }
                                },
                              ),
                            ),
                            IconButton(
                              key: const Key('chat-accessory-toggle'),
                              icon: GfSymbol(
                                _emojiOpen ? 'keyboard' : 'smile',
                                size: 23,
                              ),
                              onPressed: widget.enabled
                                  ? _toggleInputSurface
                                  : null,
                              tooltip: _emojiOpen
                                  ? widget.keyboardLabel
                                  : widget.emojiLabel,
                              style: IconButton.styleFrom(
                                fixedSize: const Size.square(44),
                                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                                padding: EdgeInsets.zero,
                                foregroundColor: colors.iconMuted,
                                disabledForegroundColor: colors.iconMuted
                                    .withValues(alpha: .45),
                                shape: const CircleBorder(),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    IconButton(
                      key: const Key('chat-send'),
                      tooltip: widget.sendLabel ?? 'Send',
                      onPressed: ready ? _send : null,
                      icon: const GfSymbol('arrow-up', size: 23),
                      style: IconButton.styleFrom(
                        fixedSize: const Size.square(44),
                        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        padding: EdgeInsets.zero,
                        shape: const CircleBorder(),
                        backgroundColor: colors.primary,
                        foregroundColor: colors.primaryContent,
                        disabledBackgroundColor: colors.base200,
                        disabledForegroundColor: colors.iconMuted.withValues(
                          alpha: .45,
                        ),
                      ),
                    ),
                  ],
                ),
                if (_emojiOpen && MediaQuery.viewInsetsOf(context).bottom == 0)
                  SizedBox(
                    height: (MediaQuery.sizeOf(context).height * .38).clamp(
                      180.0,
                      340.0,
                    ),
                    child:
                        widget.accessoryBuilder?.call(_insertEmoji) ??
                        GridView.extent(
                          maxCrossAxisExtent: 64,
                          childAspectRatio: 1,
                          padding: const EdgeInsets.only(top: 8),
                          children: [
                            for (final emoji in _emojis)
                              TextButton(
                                onPressed: () => _insertEmoji(emoji),
                                style: TextButton.styleFrom(
                                  minimumSize: const Size(48, 48),
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
      ),
    );
  }
}
