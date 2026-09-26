import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';
import '../atoms/gf_badge.dart';
import '../atoms/gf_badge_medallion.dart';
import '../gf_symbol.dart';

@immutable
class GfUserBadge {
  const GfUserBadge({
    required this.label,
    this.color,
    this.icon,
    this.description = '',
    this.onTap,
  });

  final String label;
  final Color? color;
  final Widget? icon;
  final String description;
  final VoidCallback? onTap;
}

/// Cover, overlapping avatar and actions kept in the same painting boundary.
///
/// A collapsing profile can place this whole header in its flexible space and
/// render the remaining identity details with [GfUserCard.showHeader] disabled.
class GfUserCardHeader extends StatelessWidget {
  const GfUserCardHeader({
    super.key,
    required this.avatarUrl,
    this.coverUrl,
    this.avatarBadge,
    this.actions,
    this.coverHeight,
    this.actionHeight,
  });

  /// Leaves 8px below the avatar's 48px overlap before identity content starts.
  static const double minimumActionHeight = 56;

  /// Keep the avatar clear while preserving the action's full touch target.
  static const EdgeInsets actionPadding = EdgeInsets.fromLTRB(128, 4, 16, 4);

  /// Reserves room for a two-line action label at larger text sizes.
  static double actionHeightFor(TextScaler textScaler) => math.max(
    minimumActionHeight,
    textScaler.scale(14) * 2 * 1.4 + actionPadding.vertical,
  );

  final String avatarUrl;
  final String? coverUrl;
  final Widget? avatarBadge;
  final Widget? actions;
  final double? coverHeight;

  /// A fixed action band for a sliver whose expanded extent is known upfront.
  /// Without it, the band grows naturally while retaining the avatar clearance.
  final double? actionHeight;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final actionBand = Padding(
      padding: actionPadding,
      child: Align(
        alignment: Alignment.topRight,
        heightFactor: 1,
        child: actions,
      ),
    );

    return LayoutBuilder(
      builder: (context, constraints) {
        final coverHeight =
            this.coverHeight ?? GfUserCard.coverHeightFor(constraints.maxWidth);
        return Stack(
          clipBehavior: Clip.none,
          children: [
            Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                SizedBox(
                  key: const Key('profile-cover-image'),
                  width: double.infinity,
                  height: coverHeight,
                  child: ColoredBox(
                    color: colors.base300,
                    child: coverUrl?.isNotEmpty == true
                        ? Image.network(
                            coverUrl!,
                            fit: BoxFit.cover,
                            excludeFromSemantics: true,
                            cacheWidth:
                                (constraints.maxWidth *
                                        MediaQuery.devicePixelRatioOf(context))
                                    .round(),
                            errorBuilder: (_, _, _) => const SizedBox.shrink(),
                          )
                        : null,
                  ),
                ),
                if (actionHeight case final height?)
                  SizedBox(height: height, child: actionBand)
                else
                  ConstrainedBox(
                    constraints: const BoxConstraints(
                      minHeight: minimumActionHeight,
                    ),
                    child: actionBand,
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
                  child: GfAvatar(src: avatarUrl, size: 88, badge: avatarBadge),
                ),
              ),
            ),
          ],
        );
      },
    );
  }
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
    this.showHeader = true,
    this.coverHeight,
  });

  /// The same image crop is used by the public header and its live editor.
  /// Leave room for system insets, 56px navigation and the avatar overlap.
  static double coverHeightFor(double width, {double topInset = 0}) =>
      math.max((width / 3).clamp(112.0, 200.0), topInset + 104);

  final String avatarUrl;
  final String name;
  final String username;
  final String? bio;
  final String? signature;
  final String? coverUrl;

  /// Omit the complete cover/avatar/action block when it is rendered elsewhere.
  final bool showHeader;
  final double? coverHeight;

  /// Supplemental badge labels below public details (e.g. Admin, online).
  final List<String> badges;

  /// Icon-only badges with source-defined artwork, color and optional details.
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
        if (showHeader)
          GfUserCardHeader(
            avatarUrl: avatarUrl,
            coverUrl: coverUrl,
            avatarBadge: avatarBadge,
            actions: actions,
            coverHeight: coverHeight,
          ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
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
              const SizedBox(height: 2),
              Text(
                '@$username',
                style: TextStyle(
                  fontSize: 16,
                  color: colors.baseContent.withValues(alpha: 0.72),
                ),
              ),
              if (bio != null && bio!.trim().isNotEmpty) ...<Widget>[
                const SizedBox(height: 8),
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
                const SizedBox(height: 6),
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
                const SizedBox(height: 6),
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
                      MergeSemantics(
                        child: Semantics(
                          label: badge.label,
                          button: badge.onTap != null,
                          child: Tooltip(
                            message: [
                              badge.label,
                              badge.description,
                            ].where((text) => text.isNotEmpty).join('\n'),
                            excludeFromSemantics: true,
                            child: TextButton(
                              onPressed: badge.onTap,
                              style: TextButton.styleFrom(
                                padding: EdgeInsets.zero,
                                minimumSize: const Size(44, 44),
                                maximumSize: const Size(44, 44),
                                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                                shape: const CircleBorder(),
                              ),
                              child: ExcludeSemantics(
                                child: GfBadgeMedallion(
                                  size: 40,
                                  color: badge.color ?? colors.primary,
                                  icon:
                                      badge.icon ??
                                      const GfSymbol('award', size: 22),
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                  ],
                ),
              ],
              if (stats.isNotEmpty) ...<Widget>[
                const SizedBox(height: 12),
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
