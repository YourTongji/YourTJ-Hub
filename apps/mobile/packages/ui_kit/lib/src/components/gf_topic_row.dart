import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_chip.dart';

/// Category metadata for [GfTopicRow].
class GfTopicCategory {
  const GfTopicCategory({required this.name, required this.color});

  final String name;
  final Color color;
}

/// Topic list row mirroring web `TopicRow.vue` / `.gf-topic-row`.
///
/// Mobile layout: title, category chips, one-line description, then a meta
/// line with participants, relative time and reply count. A hairline divider
/// separates rows; the whole row is tappable.
class GfTopicRow extends StatelessWidget {
  const GfTopicRow({
    super.key,
    required this.title,
    required this.description,
    required this.categories,
    required this.participantAvatars,
    required this.activityText,
    required this.replyCount,
    this.onTap,
    this.pinned = false,
    this.unseen = false,
    this.viewCount,
    this.hot = false,
  });

  final String title;
  final String description;
  final List<GfTopicCategory> categories;
  final List<Widget> participantAvatars;
  final String activityText;
  final int replyCount;
  final VoidCallback? onTap;
  final bool pinned;
  final bool unseen;

  /// View count shown in the desktop meta layout; optional on mobile.
  final int? viewCount;

  /// Whether the row is trending; shows the `hot` badge (web
  /// `showHot && viewCount > 500`).
  final bool hot;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return InkWell(
      onTap: onTap,
      child: Container(
        color: colors.base100,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            // Title row: pin mark, title, unseen dot, category chips.
            Wrap(
              spacing: 8,
              runSpacing: 4,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: <Widget>[
                if (pinned)
                  Icon(
                    Icons.push_pin,
                    size: 14,
                    color: colors.error,
                    semanticLabel: 'pinned',
                  ),
                ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 280),
                  child: Text(
                    title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: colors.baseContent,
                      fontSize: 15,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
                if (hot)
                  Container(
                    height: 20,
                    padding: const EdgeInsets.symmetric(horizontal: 6),
                    decoration: BoxDecoration(
                      color: colors.warning.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: <Widget>[
                        Icon(
                          Icons.local_fire_department,
                          size: 12,
                          color: colors.warning,
                        ),
                        const SizedBox(width: 2),
                        Text(
                          'hot',
                          style: TextStyle(
                            color: colors.warning,
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                  ),
                if (unseen)
                  Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(
                      color: colors.primary,
                      shape: BoxShape.circle,
                    ),
                  ),
                for (final GfTopicCategory category in categories)
                  GfChip(label: category.name, color: category.color),
              ],
            ),
            if (description.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  description,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: colors.baseContent.withValues(alpha: 0.55),
                    fontSize: 13,
                  ),
                ),
              ),
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Row(
                children: <Widget>[
                  if (participantAvatars.isNotEmpty) ...<Widget>[
                    SizedBox(
                      width:
                          (participantAvatars.length > 4
                              ? 4
                              : participantAvatars.length) *
                          14.0,
                      height: 20,
                      child: Stack(
                        clipBehavior: Clip.none,
                        children: <Widget>[
                          for (
                            int i = 0;
                            i < participantAvatars.length && i < 4;
                            i++
                          )
                            Positioned(
                              left: i * 14,
                              child: participantAvatars[i],
                            ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 8),
                  ],
                  Text(
                    activityText,
                    style: TextStyle(
                      color: colors.baseContent.withValues(alpha: 0.55),
                      fontSize: 12,
                    ),
                  ),
                  const Spacer(),
                  Icon(
                    Icons.chat_bubble_outline,
                    size: 14,
                    color: colors.baseContent.withValues(alpha: 0.55),
                  ),
                  const SizedBox(width: 4),
                  Text(
                    '$replyCount',
                    style: TextStyle(
                      color: colors.baseContent.withValues(alpha: 0.55),
                      fontSize: 12,
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
