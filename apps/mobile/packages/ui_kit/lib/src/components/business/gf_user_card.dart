import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';
import '../atoms/gf_badge.dart';

@immutable
class GfUserBadge {
  const GfUserBadge({required this.label, this.color});

  final String label;
  final Color? color;
}

/// Social profile header: cover and overlapping avatar, trailing actions,
/// identity, public details and compact inline statistics.
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
    this.coloredBadges = const <GfUserBadge>[],
    this.stats = const <(String, String)>[],
    this.statActions = const {},
    this.actions,
    this.details,
    this.avatarBadge,
  });

  final String avatarUrl;
  final String name;
  final String username;
  final String? bio;
  final String? signature;
  final String? coverUrl;

  /// Supplemental badge labels below public details (e.g. Admin, online).
  final List<String> badges;

  /// Badges that preserve a source-defined color.
  final List<GfUserBadge> coloredBadges;

  /// (label, value) pairs rendered inline, wrapping with available width.
  final List<(String, String)> stats;

  /// Optional navigation actions keyed by the zero-based statistic index.
  /// Actionable statistics have a full-width, keyboard-accessible 48px target.
  final Map<int, VoidCallback> statActions;

  /// Optional action buttons row (e.g. follow / message / edit).
  final Widget? actions;

  /// Public metadata such as website and social links, below the bio.
  final Widget? details;

  /// The selected, worn badge attached to the profile avatar.
  final Widget? avatarBadge;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        LayoutBuilder(
          builder: (context, constraints) {
            final coverHeight = (constraints.maxWidth / 3).clamp(112.0, 200.0);
            return Stack(
              children: [
                Column(
                  children: [
                    SizedBox(
                      width: double.infinity,
                      height: coverHeight,
                      child: ColoredBox(
                        color: colors.base300,
                        child: coverUrl?.isNotEmpty == true
                            ? Image.network(
                                coverUrl!,
                                fit: BoxFit.cover,
                                cacheWidth:
                                    (constraints.maxWidth *
                                            MediaQuery.devicePixelRatioOf(
                                              context,
                                            ))
                                        .round(),
                                errorBuilder: (_, _, _) =>
                                    const SizedBox.shrink(),
                              )
                            : null,
                      ),
                    ),
                    ConstrainedBox(
                      constraints: const BoxConstraints(minHeight: 64),
                      child: Padding(
                        padding: const EdgeInsets.fromLTRB(128, 12, 16, 4),
                        child: Align(
                          alignment: Alignment.topRight,
                          child: actions,
                        ),
                      ),
                    ),
                  ],
                ),
                Positioned(
                  left: 16,
                  top: coverHeight - 48,
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: colors.base100,
                    ),
                    child: Padding(
                      padding: const EdgeInsets.all(4),
                      child: GfAvatar(
                        src: avatarUrl,
                        size: 88,
                        badge: avatarBadge,
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Text(
                name,
                style: TextStyle(
                  fontSize: 22,
                  fontWeight: FontWeight.w700,
                  color: colors.baseContent,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                '@$username',
                style: TextStyle(
                  fontSize: 16,
                  color: colors.baseContent.withValues(alpha: 0.72),
                ),
              ),
              if (bio != null && bio!.trim().isNotEmpty) ...<Widget>[
                const SizedBox(height: 12),
                Text(
                  bio!.trim(),
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.4,
                    color: colors.baseContent,
                  ),
                ),
              ],
              if (signature?.trim().isNotEmpty == true &&
                  signature!.trim() != bio?.trim()) ...[
                const SizedBox(height: 8),
                Text(
                  signature!.trim(),
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.4,
                    color: colors.iconMuted,
                  ),
                ),
              ],
              if (details != null) ...<Widget>[
                const SizedBox(height: 8),
                details!,
              ],
              if (badges.isNotEmpty || coloredBadges.isNotEmpty) ...[
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: [
                    for (final badge in badges)
                      GfBadge(label: badge, variant: GfBadgeVariant.info),
                    for (final badge in coloredBadges)
                      GfBadge(label: badge.label, color: badge.color),
                  ],
                ),
              ],
              if (stats.isNotEmpty) ...<Widget>[
                const SizedBox(height: 16),
                Wrap(
                  spacing: 20,
                  runSpacing: 0,
                  children: [
                    for (int i = 0; i < stats.length; i++)
                      MergeSemantics(
                        child: Semantics(
                          button: statActions[i] != null,
                          child: InkWell(
                            onTap: statActions[i],
                            borderRadius: BorderRadius.circular(8),
                            child: ConstrainedBox(
                              constraints: const BoxConstraints(
                                minHeight: 48,
                                minWidth: 48,
                              ),
                              child: Padding(
                                padding: const EdgeInsets.symmetric(
                                  vertical: 12,
                                ),
                                child: Wrap(
                                  spacing: 4,
                                  crossAxisAlignment: WrapCrossAlignment.center,
                                  children: [
                                    Text(
                                      stats[i].$2,
                                      style: TextStyle(
                                        fontSize: 15,
                                        height: 1.4,
                                        fontWeight: FontWeight.w700,
                                        color: colors.baseContent,
                                      ),
                                    ),
                                    Text(
                                      stats[i].$1,
                                      style: TextStyle(
                                        fontSize: 15,
                                        color: colors.iconMuted,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                  ],
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }
}
