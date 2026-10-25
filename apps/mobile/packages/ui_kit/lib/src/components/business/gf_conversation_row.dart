import '../atoms/gf_decorated_name.dart';
import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../atoms/gf_avatar.dart';
import '../atoms/gf_badge.dart';

/// Conversation list row mirroring web MessagesPage.vue mobile layout:
/// `px-4 py-3` with a 40px avatar, unread red dot, nickname, time and the
/// last message preview. The active (selected) row gets `bg-info/10` with a
/// 3px primary inset bar on the left.
class GfConversationRow extends StatelessWidget {
  const GfConversationRow({
    super.key,
    required this.avatarUrl,
    required this.name,
    this.avatarFrameColors = const [],
    this.avatarFrameAssetUrl = '',
    this.avatarFrameScale = 1.25,
    this.nicknameColors = const [],
    required this.lastMessage,
    required this.time,
    required this.unreadCount,
    required this.unreadLabel,
    this.active = false,
    this.onTap,
  });

  final String avatarUrl;
  final String name;
  final List<Color> avatarFrameColors, nicknameColors;
  final String lastMessage;
  final String time;
  final int unreadCount;
  final String unreadLabel;
  final bool active;
  final VoidCallback? onTap;

  final String avatarFrameAssetUrl;
  final double avatarFrameScale;
  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);

    return InkWell(
      onTap: onTap,
      child: Container(
        decoration: BoxDecoration(
          color: active ? colors.primarySoft : colors.canvas,
        ),
        padding: const EdgeInsets.symmetric(
          horizontal: GfSpace.gutter,
          vertical: GfSpace.md,
        ),
        child: Row(
          children: <Widget>[
            Stack(
              clipBehavior: Clip.none,
              children: <Widget>[
                GfAvatar(
                  frameColors: avatarFrameColors,
                  frameAssetUrl: avatarFrameAssetUrl,
                  frameScale: avatarFrameScale,
                  src: avatarUrl,
                  size: 48,
                  ring: true,
                ),
              ],
            ),
            const SizedBox(width: 12),
            Expanded(
              child: LayoutBuilder(
                builder: (context, constraints) {
                  final timeStyle = typography.caption.copyWith(
                    color: colors.textSecondary,
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
                  final bool largeText =
                      MediaQuery.textScalerOf(context).scale(16) > 24;
                  timePainter.dispose();
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: GfDecoratedName(
                              name,
                              colors: nicknameColors,
                              maxLines: largeText ? 2 : 1,
                              overflow: TextOverflow.ellipsis,
                              style: typography.heading,
                            ),
                          ),
                          if (inlineTime) ...[
                            const SizedBox(width: 8),
                            GfText(time, style: timeStyle),
                          ],
                        ],
                      ),
                      const SizedBox(height: GfSpace.xxs),
                      Row(
                        children: [
                          Expanded(
                            child: GfText(
                              lastMessage,
                              maxLines: largeText ? 2 : 1,
                              overflow: TextOverflow.ellipsis,
                              style: typography.small.copyWith(
                                color: unreadCount > 0
                                    ? colors.textPrimary
                                    : colors.textSecondary,
                              ),
                            ),
                          ),
                          if (unreadCount > 0) ...[
                            const SizedBox(width: GfSpace.sm),
                            Semantics(
                              label: '$unreadLabel: $unreadCount',
                              child: ExcludeSemantics(
                                child: GfCountBadge(count: unreadCount),
                              ),
                            ),
                          ],
                        ],
                      ),
                      if (!inlineTime) ...[
                        const SizedBox(height: GfSpace.xs),
                        GfText(time, style: timeStyle),
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
