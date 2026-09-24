import 'package:flutter/material.dart';
import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';

enum GfNotificationTone { success, warning, info, primary, like }

/// Social notification: a small event mark, actor avatar, heading and excerpt.
/// Actor navigation is independent of opening/acknowledging the notification.
class GfNotificationRow extends StatelessWidget {
  const GfNotificationRow({
    super.key,
    required this.icon,
    required this.tone,
    required this.title,
    required this.subtitle,
    required this.time,
    required this.unread,
    this.actorName = '',
    this.avatarUrl,
    this.onActorTap,
    this.onTap,
    this.onMarkRead,
    this.markReadLabel = 'Mark as read',
  });
  final IconData icon;
  final GfNotificationTone tone;
  final String title, subtitle, time, actorName;
  final String? avatarUrl;
  final bool unread;
  final VoidCallback? onTap, onMarkRead, onActorTap;
  final String markReadLabel;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final toneColor = switch (tone) {
      GfNotificationTone.success => colors.success,
      GfNotificationTone.warning => colors.warning,
      GfNotificationTone.info => colors.info,
      GfNotificationTone.primary => colors.primary,
      GfNotificationTone.like => const Color(0xFFF91880),
    };
    final actorStart = actorName.isEmpty ? -1 : title.indexOf(actorName);
    return Material(
      color: unread
          ? Color.alphaBlend(
              colors.primary.withValues(alpha: .04),
              colors.base100,
            )
          : colors.base100,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SizedBox(
                width: 28,
                height: 44,
                child: Icon(icon, size: 25, color: toneColor),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        if (avatarUrl != null)
                          Semantics(
                            label: actorName,
                            button: onActorTap != null,
                            child: InkWell(
                              onTap: onActorTap,
                              customBorder: const CircleBorder(),
                              child: Padding(
                                padding: const EdgeInsets.symmetric(
                                  vertical: 4,
                                  horizontal: 4,
                                ),
                                child: GfAvatar(src: avatarUrl!, size: 36),
                              ),
                            ),
                          ),
                        const Spacer(),
                        if (unread)
                          Container(
                            width: 7,
                            height: 7,
                            decoration: BoxDecoration(
                              color: colors.primary,
                              shape: BoxShape.circle,
                            ),
                          ),
                        if (unread && onMarkRead != null)
                          IconButton(
                            onPressed: onMarkRead,
                            tooltip: markReadLabel,
                            constraints: const BoxConstraints(
                              minWidth: 44,
                              minHeight: 44,
                            ),
                            icon: Icon(
                              Icons.check_circle_outline,
                              size: 18,
                              color: colors.iconMuted,
                            ),
                          ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text.rich(
                      TextSpan(
                        children: [
                          if (actorStart < 0)
                            TextSpan(text: title)
                          else ...[
                            TextSpan(text: title.substring(0, actorStart)),
                            TextSpan(
                              text: actorName,
                              style: const TextStyle(
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            TextSpan(
                              text: title.substring(
                                actorStart + actorName.length,
                              ),
                            ),
                          ],
                          TextSpan(
                            text: ' · $time',
                            style: TextStyle(
                              color: colors.iconMuted,
                              fontSize: 13,
                            ),
                          ),
                        ],
                      ),
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 15,
                        height: 1.4,
                        color: colors.baseContent,
                      ),
                    ),
                    if (subtitle.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 5),
                        child: Text(
                          subtitle,
                          maxLines: 3,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 15,
                            height: 1.4,
                            color: colors.baseContent.withValues(alpha: .65),
                          ),
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
