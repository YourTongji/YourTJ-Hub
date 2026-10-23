import 'package:flutter/material.dart';
import '../../twemoji/twemoji_text.dart';
import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';
import '../gf_symbol.dart';

enum GfNotificationTone { success, warning, info, primary, like }

/// Social notification: a small event mark, actor avatar, heading and excerpt.
/// Actor navigation is independent of opening/acknowledging the notification.
class GfNotificationRow extends StatelessWidget {
  const GfNotificationRow({
    super.key,
    required this.symbol,
    required this.tone,
    required this.title,
    required this.subtitle,
    required this.time,
    required this.unread,
    this.actorName = '',
    this.avatarFrameColors = const [],
    this.avatarFrameAssetUrl = '',
    this.avatarFrameScale = 1.25,
    this.nicknameColors = const [],
    this.avatarUrl,
    this.onActorTap,
    this.onTap,
    this.onMarkRead,
    this.markReadLabel = 'Mark as read',
    this.subtitleMaxLines = 3,
  });
  final List<Color> avatarFrameColors, nicknameColors;
  final String symbol;
  final GfNotificationTone tone;
  final String title, subtitle, time;

  /// The real actor's name; empty suppresses actor-name emphasis for system notices.
  final String actorName;
  final String? avatarUrl;
  final bool unread;
  final VoidCallback? onTap, onMarkRead, onActorTap;
  final String markReadLabel;

  /// Null shows complete guidance rather than truncating a content preview.
  final int? subtitleMaxLines;

  final String avatarFrameAssetUrl;
  final double avatarFrameScale;
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
    final Color background = unread
        ? Color.alphaBlend(
            colors.primary.withValues(alpha: colors.isDark ? 0.08 : 0.04),
            colors.canvas,
          )
        : colors.canvas;
    final typography = GfTheme.typographyOf(context);
    final Widget toneMark = Container(
      width: 40,
      height: 40,
      decoration: BoxDecoration(
        color: toneColor.withValues(alpha: colors.isDark ? 0.18 : 0.1),
        shape: BoxShape.circle,
      ),
      alignment: Alignment.center,
      child: GfSymbol(symbol, size: GfIconSize.md, color: toneColor),
    );
    final Widget leading = avatarUrl == null
        ? SizedBox(
            width: 44,
            child: Align(alignment: Alignment.topLeft, child: toneMark),
          )
        : Semantics(
            label: actorName,
            button: onActorTap != null,
            child: InkWell(
              onTap: onActorTap,
              customBorder: const CircleBorder(),
              child: SizedBox.square(
                dimension: 44,
                child: Stack(
                  clipBehavior: Clip.none,
                  children: [
                    GfAvatar(
                      frameColors: avatarFrameColors,
                      frameAssetUrl: avatarFrameAssetUrl,
                      frameScale: avatarFrameScale,
                      src: avatarUrl!,
                      size: 40,
                    ),
                    // Event type rides on the actor as a small tone badge.
                    Positioned(
                      right: 0,
                      bottom: 0,
                      child: Container(
                        width: 22,
                        height: 22,
                        decoration: BoxDecoration(
                          color: toneColor,
                          shape: BoxShape.circle,
                          border: Border.all(color: background, width: 2),
                        ),
                        alignment: Alignment.center,
                        child: GfSymbol(symbol, size: 11, color: Colors.white),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );

    return Material(
      color: background,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(
            horizontal: GfSpace.gutter,
            vertical: GfSpace.md,
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              leading,
              const SizedBox(width: GfSpace.md),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfText.rich(
                      TextSpan(
                        children: [
                          if (actorStart < 0)
                            TextSpan(text: title)
                          else ...[
                            TextSpan(text: title.substring(0, actorStart)),
                            TextSpan(
                              text: actorName,
                              style: TextStyle(
                                foreground: nicknameColors.length < 2
                                    ? null
                                    : (Paint()
                                        ..shader =
                                            LinearGradient(
                                              colors: nicknameColors,
                                            ).createShader(
                                              Rect.fromLTWH(
                                                0,
                                                0,
                                                actorName.length * 8.0,
                                                20,
                                              ),
                                            )),
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                            TextSpan(
                              text: title.substring(
                                actorStart + actorName.length,
                              ),
                            ),
                          ],
                        ],
                      ),
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                      style: typography.small.copyWith(
                        color: colors.textPrimary,
                      ),
                    ),
                    if (subtitle.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: GfSpace.xs),
                        child: GfText(
                          subtitle,
                          maxLines: subtitleMaxLines,
                          overflow: subtitleMaxLines == null
                              ? TextOverflow.visible
                              : TextOverflow.ellipsis,
                          style: typography.small.copyWith(
                            color: colors.textSecondary,
                          ),
                        ),
                      ),
                    Padding(
                      padding: const EdgeInsets.only(top: GfSpace.xs),
                      child: GfText(
                        time,
                        style: typography.caption.copyWith(
                          color: colors.textSecondary,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              if (unread)
                Padding(
                  padding: const EdgeInsets.only(
                    left: GfSpace.sm,
                    right: GfSpace.xs,
                    top: 7,
                  ),
                  child: Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(
                      color: colors.primary,
                      shape: BoxShape.circle,
                    ),
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
                  icon: GfSymbol(
                    'circle-check',
                    size: GfIconSize.md,
                    color: colors.textSecondary,
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
