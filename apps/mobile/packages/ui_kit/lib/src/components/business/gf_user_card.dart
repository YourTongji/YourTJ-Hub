import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';
import '../atoms/gf_badge.dart';

/// User profile header card mirroring web UserPage.vue mobile layout:
/// a cover (h-20 = 80px), a 96px avatar overlapping it by 36px, name +
/// badges, `@username`, bio, signature (`border-l-2`), action row and a
/// 4-column stats grid (`grid-cols-4`).
class GfUserCard extends StatelessWidget {
  const GfUserCard({
    super.key,
    required this.avatarUrl,
    required this.name,
    required this.username,
    this.bio,
    this.signature,
    this.coverUrl,
    this.badges = const <String>[],
    this.stats = const <(String, String)>[],
    this.actions,
  });

  final String avatarUrl;
  final String name;
  final String username;
  final String? bio;
  final String? signature;
  final String? coverUrl;

  /// Badge labels shown next to the name (e.g. Admin, online).
  final List<String> badges;

  /// (label, value) pairs rendered in the 4-column stats grid.
  final List<(String, String)> stats;

  /// Optional action buttons row (e.g. follow / message / edit).
  final Widget? actions;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        // Cover.
        Container(
          height: 80,
          width: double.infinity,
          decoration: BoxDecoration(
            color: colors.base300,
            image: coverUrl == null || coverUrl!.isEmpty
                ? null
                : DecorationImage(
                    image: NetworkImage(coverUrl!),
                    fit: BoxFit.cover,
                  ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              // Avatar overlapping the cover by 36px.
              Transform.translate(
                offset: const Offset(0, -36),
                child: GfAvatar(src: avatarUrl, size: 96),
              ),
              Transform.translate(
                offset: const Offset(0, -32),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Wrap(
                      spacing: 8,
                      runSpacing: 4,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: <Widget>[
                        Text(
                          name,
                          style: TextStyle(
                            fontSize: 17,
                            fontWeight: FontWeight.w700,
                            color: colors.baseContent,
                          ),
                        ),
                        for (final String badge in badges)
                          GfBadge(label: badge, variant: GfBadgeVariant.info),
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '@$username',
                      style: TextStyle(
                        fontSize: 13,
                        color: colors.baseContent.withValues(alpha: 0.55),
                      ),
                    ),
                    if (bio != null && bio!.isNotEmpty) ...<Widget>[
                      const SizedBox(height: 6),
                      Text(
                        bio!,
                        style: TextStyle(
                          fontSize: 14,
                          color: colors.baseContent.withValues(alpha: 0.75),
                        ),
                      ),
                    ],
                    if (signature != null && signature!.isNotEmpty) ...<Widget>[
                      const SizedBox(height: 6),
                      Container(
                        padding: const EdgeInsets.only(left: 10),
                        decoration: BoxDecoration(
                          border: Border(
                            left: BorderSide(color: colors.line, width: 2),
                          ),
                        ),
                        child: Text(
                          signature!,
                          style: TextStyle(
                            fontSize: 13,
                            color: colors.baseContent.withValues(alpha: 0.55),
                          ),
                        ),
                      ),
                    ],
                    if (actions != null) ...<Widget>[
                      const SizedBox(height: 12),
                      actions!,
                    ],
                    if (stats.isNotEmpty) ...<Widget>[
                      const SizedBox(height: 16),
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          for (final (String label, String value) in stats)
                            Expanded(
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: <Widget>[
                                  Text(
                                    value,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                      fontSize: 16,
                                      fontWeight: FontWeight.w700,
                                      color: colors.baseContent,
                                    ),
                                  ),
                                  const SizedBox(height: 2),
                                  Text(
                                    label,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    textAlign: TextAlign.center,
                                    style: TextStyle(
                                      fontSize: 12,
                                      color: colors.baseContent.withValues(
                                        alpha: 0.55,
                                      ),
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
            ],
          ),
        ),
      ],
    );
  }
}
