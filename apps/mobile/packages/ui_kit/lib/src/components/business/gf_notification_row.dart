import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Tone of a notification row, mirroring web `notificationTone` in
/// NotificationsPage.vue (icon block color per type).
enum GfNotificationTone {
  /// Follow / user actions: success green.
  success,

  /// Badge awards: warning yellow.
  warning,

  /// System / info messages: info blue.
  info,

  /// Everything else (replies, mentions): primary.
  primary,
}

/// Notification row mirroring web NotificationsPage.vue mobile layout:
/// `grid-cols-[34px_1fr] gap-3 px-3 py-2.5` — a 32px tone-colored icon block
/// plus content (actor, action, topic link, time). Unread rows get
/// `bg-info/10` + a 2px primary left bar + a blue dot.
class GfNotificationRow extends StatelessWidget {
  const GfNotificationRow({
    super.key,
    required this.icon,
    required this.tone,
    required this.title,
    required this.subtitle,
    required this.time,
    required this.unread,
    this.onTap,
    this.onMarkRead,
  });

  final IconData icon;
  final GfNotificationTone tone;

  /// Primary content line (actor + verb + topic link).
  final String title;

  /// Secondary content line (notification payload preview).
  final String subtitle;

  /// Relative time, shown under the content on mobile.
  final String time;

  final bool unread;
  final VoidCallback? onTap;
  final VoidCallback? onMarkRead;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    final Color toneColor = switch (tone) {
      GfNotificationTone.success => colors.success,
      GfNotificationTone.warning => colors.warning,
      GfNotificationTone.info => colors.info,
      GfNotificationTone.primary => colors.primary,
    };

    return InkWell(
      onTap: onTap,
      child: Container(
        color: unread ? colors.info.withValues(alpha: 0.10) : colors.base100,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            // 2px primary unread bar on the far left.
            SizedBox(
              width: 2,
              child: unread ? ColoredBox(color: colors.primary) : null,
            ),
            const SizedBox(width: 8),
            // 32px icon block.
            Container(
              width: 32,
              height: 32,
              decoration: BoxDecoration(
                color: toneColor.withValues(alpha: 0.10),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Icon(icon, size: 16, color: toneColor),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          title,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 14,
                            fontWeight: unread
                                ? FontWeight.w600
                                : FontWeight.w400,
                            color: colors.baseContent,
                          ),
                        ),
                      ),
                      const SizedBox(width: 6),
                      if (unread)
                        Container(
                          width: 8,
                          height: 8,
                          margin: const EdgeInsets.only(top: 5),
                          decoration: BoxDecoration(
                            color: colors.primary,
                            shape: BoxShape.circle,
                          ),
                        ),
                      if (unread && onMarkRead != null) ...<Widget>[
                        const SizedBox(width: 6),
                        InkWell(
                          onTap: onMarkRead,
                          child: Icon(
                            Icons.check_circle_outline,
                            size: 16,
                            color: colors.iconMuted,
                          ),
                        ),
                      ],
                    ],
                  ),
                  if (subtitle.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 13,
                        color: colors.baseContent.withValues(alpha: 0.55),
                      ),
                    ),
                  ],
                  const SizedBox(height: 4),
                  Text(
                    time,
                    style: TextStyle(
                      fontSize: 12,
                      color: colors.baseContent.withValues(alpha: 0.55),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
