import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_badge.dart';
import '../gf_chip.dart';
import '../gf_topic_row.dart';

/// Draft list row mirroring web DraftsPage.vue mobile layout: `px-4 py-3`,
/// title (16px) + category chips + blocked badge, one-line description,
/// meta row (created / views / replies) and an edit button.
class GfDraftRow extends StatelessWidget {
  const GfDraftRow({
    super.key,
    required this.title,
    required this.description,
    required this.categories,
    required this.blocked,
    required this.meta,
    required this.updatedTime,
    this.onTap,
    this.onEdit,
  });

  final String title;
  final String description;
  final List<GfTopicCategory> categories;
  final bool blocked;

  /// One-line meta, e.g. `2026-08-01 · 120 浏览 · 5 回复`.
  final String meta;

  /// Updated time label (web `text-xs`).
  final String updatedTime;

  final VoidCallback? onTap;
  final VoidCallback? onEdit;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Wrap(
              spacing: 8,
              runSpacing: 4,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: <Widget>[
                Text(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                    color: colors.baseContent,
                  ),
                ),
                if (blocked)
                  const GfBadge(
                    label: 'blocked',
                    variant: GfBadgeVariant.error,
                    icon: Icon(Icons.shield_outlined, size: 12),
                  ),
              ],
            ),
            if (description.isNotEmpty) ...<Widget>[
              const SizedBox(height: 4),
              Text(
                description,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 13,
                  color: colors.baseContent.withValues(alpha: 0.55),
                ),
              ),
            ],
            const SizedBox(height: 6),
            Wrap(
              spacing: 6,
              runSpacing: 4,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: <Widget>[
                for (final GfTopicCategory category in categories)
                  GfChip(label: category.name, color: category.color),
                Text(
                  meta,
                  style: TextStyle(
                    fontSize: 12,
                    color: colors.baseContent.withValues(alpha: 0.55),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Row(
              children: <Widget>[
                Text(
                  updatedTime,
                  style: TextStyle(
                    fontSize: 12,
                    color: colors.baseContent.withValues(alpha: 0.55),
                  ),
                ),
                const Spacer(),
                if (onEdit != null)
                  TextButton(
                    onPressed: onEdit,
                    style: TextButton.styleFrom(
                      foregroundColor: colors.primary,
                      padding: const EdgeInsets.symmetric(
                        horizontal: 12,
                        vertical: 4,
                      ),
                      minimumSize: Size.zero,
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    ),
                    child: const Text('编辑'),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
