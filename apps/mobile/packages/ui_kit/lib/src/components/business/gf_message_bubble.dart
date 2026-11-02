import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';

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
    this.onLongPress,
    this.interactionWrapper,
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

  /// Opens the message action menu. The gesture layer sits inside
  /// [SelectionArea] so a long press wins over selection; dragging still
  /// scrolls or selects text.
  final VoidCallback? onLongPress;

  /// Message gestures sit inside selection so touch actions can win their
  /// gesture arena without intercepting mouse selection or vertical scrolling.
  final Widget Function(Widget child)? interactionWrapper;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final inheritedStyle = DefaultTextStyle.of(context).style;

    final TextStyle contentStyle = TextStyle(
      // Preserve the reading font and its CJK fallback while owning bubble
      // size and color; replacing these loses glyphs in bundled-font themes.
      fontFamily: inheritedStyle.fontFamily,
      fontFamilyFallback: inheritedStyle.fontFamilyFallback,
      fontSize: 16,
      height: 22 / 16,
      color: mine && showBubble
          ? colors.messageOutgoingContent
          : colors.baseContent,
    );
    final Widget body = DefaultTextStyle(
      style: contentStyle,
      child:
          content ??
          (contentSpan == null
              ? GfText(text)
              : GfText.rich(TextSpan(children: <InlineSpan>[contentSpan!]))),
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        final availableWidth = constraints.hasBoundedWidth
            ? constraints.maxWidth
            : MediaQuery.sizeOf(context).width;
        Widget bubble = Container(
          key: bubbleKey,
          constraints: BoxConstraints(
            maxWidth: availableWidth * maxWidthFactor,
          ),
          padding: showBubble
              ? const EdgeInsets.symmetric(horizontal: 14, vertical: 9)
              : null,
          decoration: showBubble
              ? BoxDecoration(
                  color: mine ? colors.messageOutgoing : colors.fill,
                  // The corner nearest the sender tightens into a tail.
                  borderRadius: BorderRadiusDirectional.only(
                    topStart: const Radius.circular(GfRadius.lg),
                    topEnd: const Radius.circular(GfRadius.lg),
                    bottomStart: Radius.circular(
                      mine ? GfRadius.lg : GfRadius.xs,
                    ),
                    bottomEnd: Radius.circular(
                      mine ? GfRadius.xs : GfRadius.lg,
                    ),
                  ),
                )
              : null,
          child: body,
        );
        if (onLongPress != null) {
          bubble = GestureDetector(
            behavior: HitTestBehavior.opaque,
            onLongPress: onLongPress,
            child: bubble,
          );
        }
        if (interactionWrapper != null) {
          bubble = interactionWrapper!(bubble);
        }
        if (selectable) {
          bubble = GfSelectionArea(
            // `text` is the original message source, including sticker tokens
            // represented by rich image spans inside `content`.
            selectionSource: text,
            contextMenuBuilder: (context, selection, buttonItems) =>
                AdaptiveTextSelectionToolbar.buttonItems(
                  anchors: selection.contextMenuAnchors,
                  buttonItems: [
                    ...buttonItems,
                    ContextMenuButtonItem(
                      label: copyMessageLabel,
                      onPressed: () {
                        // Whole-message copy preserves sticker tokens that
                        // native partial text selection cannot represent as
                        // images.
                        Clipboard.setData(ClipboardData(text: text));
                        selection.hideToolbar();
                      },
                    ),
                  ],
                ),
            child: bubble,
          );
        }

        final Widget withTime = time == null
            ? bubble
            : Column(
                crossAxisAlignment: mine
                    ? CrossAxisAlignment.end
                    : CrossAxisAlignment.start,
                children: <Widget>[
                  bubble,
                  const SizedBox(height: GfSpace.xs),
                  GfText(
                    time!,
                    style: GfTheme.typographyOf(context).meta.copyWith(
                      fontWeight: FontWeight.w400,
                      color: colors.textSecondary,
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
