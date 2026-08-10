import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Chat message bubble mirroring web MessagesPage.vue mobile layout:
/// max-width 88%, own messages right-aligned with a primary fill, others
/// left-aligned with a base-300 fill; radius box (8), `px-3 py-2`, 14px text.
class GfMessageBubble extends StatelessWidget {
  const GfMessageBubble({
    super.key,
    required this.text,
    required this.mine,
    this.time,
    this.maxWidthFactor = 0.88,
  });

  final String text;
  final bool mine;

  /// Optional time label under the bubble (web shows it next to the avatar).
  final String? time;

  /// Fraction of available width the bubble may occupy (web `max-w-[88%]`).
  final double maxWidthFactor;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    final Widget bubble = Container(
      constraints: BoxConstraints(
        maxWidth: MediaQuery.sizeOf(context).width * maxWidthFactor,
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: mine ? colors.primary : colors.base300,
        borderRadius: BorderRadius.circular(8),
        boxShadow: <BoxShadow>[
          BoxShadow(
            color: colors.neutral.withValues(alpha: 0.05),
            blurRadius: 4,
            offset: const Offset(0, 1),
          ),
        ],
      ),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 14,
          height: 1.4,
          color: mine ? colors.primaryContent : colors.baseContent,
        ),
      ),
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
                  fontSize: 11,
                  color: colors.baseContent.withValues(alpha: 0.55),
                ),
              ),
            ],
          );

    return Align(
      alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
      child: withTime,
    );
  }
}
