import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/gf_theme.dart';

/// Rounded chat content bounded by its conversation pane. Native text selection
/// and rich inline stickers share the same message body.
class GfMessageBubble extends StatelessWidget {
  const GfMessageBubble({
    super.key,
    required this.text,
    required this.mine,
    this.time,
    this.maxWidthFactor = 0.88,
    this.contentSpan,
    this.bubbleKey,
    this.content,
    this.selectable = false,
    this.showBubble = true,
    this.copyMessageLabel = 'Copy message',
  });

  /// Optional key on the message body, excluding alignment and timestamp.
  /// Retained when [showBubble] is false for context-menu and gesture bounds.
  final GlobalKey? bubbleKey;

  final String text;
  final bool mine;

  /// Optional time label under the bubble (web shows it next to the avatar).
  final String? time;

  /// Fraction of available width the bubble may occupy (web `max-w-[88%]`).
  final double maxWidthFactor;

  /// Optional rich content segments replacing the plain [text] body
  /// (sticker message rendering); null keeps the plain-text path.
  final InlineSpan? contentSpan;
  final Widget? content;
  final bool selectable;

  /// Paint and pad the message surface. Standalone stickers can opt out while
  /// retaining message alignment, width constraints, selection and timestamp.
  final bool showBubble;
  final String copyMessageLabel;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    final TextStyle contentStyle = TextStyle(
      fontSize: 16,
      height: 1.4,
      color: mine && showBubble ? colors.primaryContent : colors.baseContent,
    );
    Widget body = DefaultTextStyle(
      style: contentStyle,
      child:
          content ??
          (contentSpan == null
              ? Text(text)
              : Text.rich(TextSpan(children: <InlineSpan>[contentSpan!]))),
    );
    if (selectable) {
      body = SelectionArea(
        contextMenuBuilder: (context, selection) =>
            AdaptiveTextSelectionToolbar.buttonItems(
              anchors: selection.contextMenuAnchors,
              buttonItems: [
                ...selection.contextMenuButtonItems,
                ContextMenuButtonItem(
                  label: copyMessageLabel,
                  onPressed: () {
                    // Whole-message copy preserves sticker tokens that native
                    // partial text selection cannot represent as images.
                    Clipboard.setData(ClipboardData(text: text));
                    selection.hideToolbar();
                  },
                ),
              ],
            ),
        child: body,
      );
    }
    return LayoutBuilder(
      builder: (context, constraints) {
        final availableWidth = constraints.hasBoundedWidth
            ? constraints.maxWidth
            : MediaQuery.sizeOf(context).width;
        final Widget bubble = Container(
          key: bubbleKey,
          constraints: BoxConstraints(
            maxWidth: availableWidth * maxWidthFactor,
          ),
          padding: showBubble
              ? const EdgeInsets.symmetric(horizontal: 14, vertical: 10)
              : null,
          decoration: showBubble
              ? BoxDecoration(
                  color: mine ? colors.primary : colors.base300,
                  borderRadius: BorderRadius.circular(20),
                )
              : null,
          child: body,
        );

        final Widget withTime = time == null
            ? bubble
            : Column(
                crossAxisAlignment: mine
                    ? CrossAxisAlignment.end
                    : CrossAxisAlignment.start,
                children: <Widget>[
                  bubble,
                  const SizedBox(height: 4),
                  Text(
                    time!,
                    style: TextStyle(
                      fontSize: 12,
                      color: colors.baseContent.withValues(alpha: 0.55),
                    ),
                  ),
                ],
              );

        return Align(
          alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
          child: withTime,
        );
      },
    );
  }
}
