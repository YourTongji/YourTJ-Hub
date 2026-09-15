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
/// a 40px tone-colored icon block with 16px content insets
/// plus content (actor, action, topic link, time). Unread rows get
/// a subtle tint, a primary left border and a blue dot.
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
        decoration: BoxDecoration(
          color: unread
              ? colors.primary.withValues(alpha: 0.04)
              : colors.base100,
          border: Border(
            left: BorderSide(
              color: unread ? colors.primary : Colors.transparent,
              width: 2,
            ),
          ),
        ),
        padding: const EdgeInsets.all(16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: toneColor.withValues(alpha: 0.10),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(icon, size: 20, color: toneColor),
            ),
            const SizedBox(width: 16),
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
                            fontSize: 16,
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
                          child: SizedBox(
                            width: 44,
                            height: 44,
                            child: Icon(
                              Icons.check_circle_outline,
                              size: 20,
                              color: colors.iconMuted,
                            ),
                          ),
                        ),
                      ],
                    ],
                  ),
                  if (subtitle.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 8),
                    Text(
                      subtitle,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 15,
                        color: colors.baseContent.withValues(alpha: 0.72),
                      ),
                    ),
                  ],
                  const SizedBox(height: 8),
                  Text(
                    time,
                    style: TextStyle(
                      fontSize: 13,
                      color: colors.baseContent.withValues(alpha: 0.72),
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
