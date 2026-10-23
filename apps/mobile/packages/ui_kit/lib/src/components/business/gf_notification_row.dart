import 'package:flutter/material.dart';
import 'package:flutter/gestures.dart';
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
    this.onAvatarTap,
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
  final VoidCallback? onTap, onMarkRead, onActorTap, onAvatarTap;
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
                height: 24,
                child: GfSymbol(symbol, size: 24, color: toneColor),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if (avatarUrl != null)
                      Semantics(
                        label: actorName,
                        button: (onAvatarTap ?? onActorTap) != null,
                        child: InkWell(
                          onTap: onAvatarTap ?? onActorTap,
                          customBorder: const CircleBorder(),
                          child: Padding(
                            padding: const EdgeInsets.symmetric(
                              vertical: 4,
                              horizontal: 4,
                            ),
                            child: GfAvatar(
                              frameColors: avatarFrameColors,
                              frameAssetUrl: avatarFrameAssetUrl,
                              frameScale: avatarFrameScale,
                              src: avatarUrl!,
                              size: 36,
                            ),
                          ),
                        ),
                      ),
                    if (avatarUrl != null) const SizedBox(height: 4),
                    _ActorHeading(
                      title: title,
                      actorName: actorName,
                      time: time,
                      nicknameColors: nicknameColors,
                      onActorTap: onActorTap,
                    ),
                    if (subtitle.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 5),
                        child: GfText(
                          subtitle,
                          maxLines: subtitleMaxLines,
                          overflow: subtitleMaxLines == null
                              ? TextOverflow.visible
                              : TextOverflow.ellipsis,
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
              if (unread)
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 9,
                  ),
                  child: Container(
                    width: 7,
                    height: 7,
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
                    size: 18,
                    color: colors.iconMuted,
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Own the text recognizer so rebuilds and recycled notification rows dispose it.
class _ActorHeading extends StatefulWidget {
  const _ActorHeading({
    required this.title,
    required this.actorName,
    required this.time,
    required this.nicknameColors,
    this.onActorTap,
  });
  final String title, actorName, time;
  final List<Color> nicknameColors;
  final VoidCallback? onActorTap;
  @override
  State<_ActorHeading> createState() => _ActorHeadingState();
}

class _ActorHeadingState extends State<_ActorHeading> {
  final _actorTap = TapGestureRecognizer();
  @override
  void dispose() {
    _actorTap.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final title = widget.title,
        actorName = widget.actorName,
        time = widget.time;
    final nicknameColors = widget.nicknameColors;
    final actorStart = actorName.isEmpty ? -1 : title.indexOf(actorName);
    return GfText.rich(
      TextSpan(
        children: [
          if (actorStart < 0)
            TextSpan(text: title)
          else ...[
            TextSpan(text: title.substring(0, actorStart)),
            TextSpan(
              text: actorName,
              recognizer: _actorTap..onTap = widget.onActorTap,
              style: TextStyle(
                foreground: nicknameColors.length < 2
                    ? null
                    : (Paint()
                        ..shader = LinearGradient(colors: nicknameColors)
                            .createShader(
                              Rect.fromLTWH(0, 0, actorName.length * 8.0, 20),
                            )),
                fontWeight: FontWeight.w700,
              ),
            ),
            TextSpan(text: title.substring(actorStart + actorName.length)),
          ],
          TextSpan(
            text: ' · $time',
            style: TextStyle(color: colors.iconMuted, fontSize: 13),
          ),
        ],
      ),
      maxLines: 3,
      overflow: TextOverflow.ellipsis,
      style: TextStyle(fontSize: 15, height: 1.4, color: colors.baseContent),
    );
  }
}
