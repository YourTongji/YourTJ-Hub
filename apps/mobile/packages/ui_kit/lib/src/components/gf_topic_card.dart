import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'atoms/gf_avatar.dart';
import 'gf_card.dart';
import 'gf_chip.dart';
import 'gf_topic_row.dart';

/// Mobile topic-feed card aligned with the web `TopicFeedPreview` surface.
class GfTopicCard extends StatelessWidget {
  const GfTopicCard({
    super.key,
    required this.title,
    required this.description,
    required this.authorName,
    required this.authorAvatarUrl,
    required this.categories,
    required this.imageUrls,
    required this.activityText,
    required this.replyCount,
    required this.viewCount,
    this.onTap,
    this.pinned = false,
    this.unseen = false,
    this.hot = false,
  });

  final String title;
  final String description;
  final String authorName;
  final String authorAvatarUrl;
  final List<GfTopicCategory> categories;
  final List<String> imageUrls;
  final String activityText;
  final int replyCount;
  final int viewCount;
  final VoidCallback? onTap;
  final bool pinned;
  final bool unseen;
  final bool hot;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final List<String> images = imageUrls
        .where((String url) => url.isNotEmpty)
        .take(imageUrls.length <= 2 ? 1 : 3)
        .toList();
    final bool singleImage = images.length == 1;

    final Widget textContent = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        _AuthorMeta(
          name: authorName,
          avatarUrl: authorAvatarUrl,
          activityText: activityText,
          categories: categories,
          hot: hot,
          pinned: pinned,
        ),
        const SizedBox(height: 12),
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
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
                            color: colors.baseContent,
                            fontSize: 16,
                            height: 1.45,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      if (unseen) ...<Widget>[
                        const SizedBox(width: 6),
                        Container(
                          width: 8,
                          height: 8,
                          margin: const EdgeInsets.only(top: 7),
                          decoration: BoxDecoration(
                            color: colors.primary,
                            shape: BoxShape.circle,
                          ),
                        ),
                      ],
                    ],
                  ),
                  if (description.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 6),
                    Text(
                      description,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.55),
                        fontSize: 14,
                        height: 1.55,
                      ),
                    ),
                  ],
                ],
              ),
            ),
            if (singleImage) ...<Widget>[
              const SizedBox(width: 12),
              _TopicImage(url: images.first, width: 112, height: 96),
            ],
          ],
        ),
        if (images.length > 1) ...<Widget>[
          const SizedBox(height: 12),
          Row(
            children: <Widget>[
              for (int index = 0; index < images.length; index++) ...<Widget>[
                if (index > 0) const SizedBox(width: 6),
                Expanded(child: _TopicImage(url: images[index], height: 104)),
              ],
            ],
          ),
        ],
        const SizedBox(height: 12),
        Divider(
          height: 1,
          thickness: 1,
          color: colors.line.withValues(alpha: 0.7),
        ),
        const SizedBox(height: 8),
        Row(
          children: <Widget>[
            _Metric(icon: Icons.chat_bubble_outline, value: '$replyCount'),
            const SizedBox(width: 6),
            _Metric(icon: Icons.visibility_outlined, value: '$viewCount'),
            const Spacer(),
            Text(
              activityText,
              style: TextStyle(
                color: colors.baseContent.withValues(alpha: 0.55),
                fontSize: 12,
              ),
            ),
          ],
        ),
      ],
    );

    return GfCard(
      emphasized: true,
      padding: const EdgeInsets.all(14),
      onTap: onTap,
      child: textContent,
    );
  }
}

class _AuthorMeta extends StatelessWidget {
  const _AuthorMeta({
    required this.name,
    required this.avatarUrl,
    required this.activityText,
    required this.categories,
    required this.hot,
    required this.pinned,
  });

  final String name;
  final String avatarUrl;
  final String activityText;
  final List<GfTopicCategory> categories;
  final bool hot;
  final bool pinned;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        GfAvatar(src: avatarUrl, size: 32, ring: true),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Row(
                children: <Widget>[
                  Flexible(
                    child: Text(
                      name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Text(
                    activityText,
                    style: TextStyle(
                      color: colors.baseContent.withValues(alpha: 0.55),
                      fontSize: 12,
                    ),
                  ),
                ],
              ),
              if (categories.isNotEmpty || hot) ...<Widget>[
                const SizedBox(height: 6),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: <Widget>[
                    for (final GfTopicCategory category in categories)
                      GfChip(label: category.name, color: category.color),
                    if (hot)
                      Container(
                        height: 20,
                        padding: const EdgeInsets.symmetric(horizontal: 7),
                        decoration: BoxDecoration(
                          color: colors.warning.withValues(alpha: 0.1),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: <Widget>[
                            Icon(
                              Icons.auto_awesome,
                              size: 12,
                              color: colors.warning,
                            ),
                            const SizedBox(width: 3),
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
                  ],
                ),
              ],
            ],
          ),
        ),
        if (pinned) Icon(Icons.push_pin, size: 16, color: colors.error),
      ],
    );
  }
}

class _TopicImage extends StatelessWidget {
  const _TopicImage({required this.url, this.width, required this.height});

  final String url;
  final double? width;
  final double height;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: SizedBox(
        width: width,
        height: height,
        child: Image.network(
          url,
          fit: BoxFit.cover,
          errorBuilder:
              (BuildContext context, Object error, StackTrace? stack) {
                return ColoredBox(
                  color: colors.base200,
                  child: Icon(Icons.image_outlined, color: colors.iconMuted),
                );
              },
        ),
      ),
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.icon, required this.value});

  final IconData icon;
  final String value;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Icon(icon, size: 16, color: colors.iconMuted),
          const SizedBox(width: 5),
          Text(
            value,
            style: TextStyle(
              color: colors.baseContent.withValues(alpha: 0.55),
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}
