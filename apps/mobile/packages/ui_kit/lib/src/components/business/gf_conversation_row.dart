import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';

/// Conversation list row mirroring web MessagesPage.vue mobile layout:
/// `px-4 py-3` with a 40px avatar, unread red dot, nickname, time and the
/// last message preview. The active (selected) row gets `bg-info/10` with a
/// 3px primary inset bar on the left.
class GfConversationRow extends StatelessWidget {
  const GfConversationRow({
    super.key,
    required this.avatarUrl,
    required this.name,
    required this.lastMessage,
    required this.time,
    required this.unreadCount,
    this.active = false,
    this.onTap,
  });

  final String avatarUrl;
  final String name;
  final String lastMessage;
  final String time;
  final int unreadCount;
  final bool active;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return InkWell(
      onTap: onTap,
      child: Container(
        decoration: BoxDecoration(
          color: active ? colors.info.withValues(alpha: 0.10) : colors.base100,
          border: active
              ? Border(left: BorderSide(color: colors.primary, width: 3))
              : null,
        ),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          children: <Widget>[
            Stack(
              clipBehavior: Clip.none,
              children: <Widget>[
                GfAvatar(src: avatarUrl, size: 40, ring: true),
                if (unreadCount > 0)
                  Positioned(
                    right: -1,
                    top: -1,
                    child: Semantics(
                      label: '$unreadCount unread',
                      child: Container(
                        width: 10,
                        height: 10,
                        decoration: BoxDecoration(
                          color: colors.error,
                          shape: BoxShape.circle,
                          border: Border.all(color: colors.base100, width: 2),
                        ),
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(width: 12),
            Expanded(
              child: LayoutBuilder(
                builder: (context, constraints) {
                  final timeStyle = TextStyle(
                    fontSize: 13,
                    height: 1.35,
                    color: colors.baseContent.withValues(alpha: 0.72),
                  );
                  final timePainter = TextPainter(
                    text: TextSpan(
                      text: time,
                      style: DefaultTextStyle.of(
                        context,
                      ).style.merge(timeStyle),
                    ),
                    textDirection: Directionality.of(context),
                    textScaler: MediaQuery.textScalerOf(context),
                  )..layout();
                  // Keep at least 120px for identity. Long dates move below the
                  // preview instead of taking width away from the sender.
                  final inlineTime =
                      constraints.maxWidth - timePainter.width >= 120 &&
                      MediaQuery.textScalerOf(context).scale(16) <= 24;
                  timePainter.dispose();
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              name,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                                color: colors.baseContent,
                              ),
                            ),
                          ),
                          if (inlineTime) ...[
                            const SizedBox(width: 8),
                            Text(time, style: timeStyle),
                          ],
                        ],
                      ),
                      const SizedBox(height: 8),
                      Text(
                        lastMessage,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 15,
                          fontWeight: unreadCount > 0
                              ? FontWeight.w600
                              : FontWeight.w400,
                          color: colors.baseContent.withValues(
                            alpha: unreadCount > 0 ? 0.85 : 0.72,
                          ),
                        ),
                      ),
                      if (!inlineTime) ...[
                        const SizedBox(height: 8),
                        Text(time, style: timeStyle),
                      ],
                    ],
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}
