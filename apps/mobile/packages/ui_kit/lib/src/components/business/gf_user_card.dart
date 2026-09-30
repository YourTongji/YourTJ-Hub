import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';
import '../atoms/gf_badge.dart';
import '../atoms/gf_badge_medallion.dart';
import '../gf_horizontal_scroll_view.dart';
import '../gf_media_image.dart';
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
                        ? GfNetworkImage(
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
    this.nameBadges = const <Widget>[],
    this.usernameAction,
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
    this.compact = false,
  });

  /// The same image crop is used by the public header and its live editor.
  /// Leave room for system insets, 56px navigation and the avatar overlap.
  static double coverHeightFor(double width, {double topInset = 0}) =>
      math.max((width / 3).clamp(112.0, 200.0), topInset + 104);

  final String avatarUrl;
  final String name;
  final String username;
  final List<Widget> nameBadges;

  /// Compact action kept beside the handle, away from the name badges.
  final Widget? usernameAction;
  final String? bio;
  final String? signature;
  final String? coverUrl;

  /// Omit the complete cover/avatar/action block when it is rendered elsewhere.
  final bool showHeader;
  final double? coverHeight;

  /// Tightens text grouping for bounded profile previews without changing the
  /// public profile header's established rhythm.
  final bool compact;

  /// Supplemental badge labels below public details (e.g. Admin, online).
  final List<String> badges;

  /// Icon-only badges with source-defined artwork, color and optional details.
  final List<GfUserBadge> coloredBadges;

  /// (label, value) pairs kept in one adaptive statistics row.
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
    final sectionGap = compact ? 6.0 : 8.0;
    final nameText = Text(
      name,
      maxLines: 1,
      overflow: TextOverflow.ellipsis,
      style: TextStyle(
        fontSize: 22,
        fontWeight: FontWeight.w700,
        color: colors.baseContent,
      ),
    );
    final nameRow = compact
        ? Row(
            key: const ValueKey('profile-name-row'),
            children: [
              Expanded(child: nameText),
              for (final badge in nameBadges) ...[
                const SizedBox(width: 6),
                badge,
              ],
            ],
          )
        : Wrap(
            key: const ValueKey('profile-name-row'),
            spacing: compact ? 6 : 8,
            runSpacing: 4,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [nameText, ...nameBadges],
          );
    final usernameText = Text(
      '@$username',
      maxLines: compact ? 1 : null,
      overflow: compact ? TextOverflow.ellipsis : null,
      style: TextStyle(
        fontSize: 13,
        fontWeight: FontWeight.w500,
        color: colors.baseContent.withValues(alpha: 0.55),
      ),
    );
    final usernameRow = compact || usernameAction != null
        ? Row(
            children: [
              Expanded(child: usernameText),
              ?usernameAction,
            ],
          )
        : usernameText;
    final profileBadges = <Widget>[
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
                child: Align(
                  // Balance visible gaps around the taller stat target.
                  alignment: Alignment.bottomCenter,
                  child: ExcludeSemantics(
                    child: GfBadgeMedallion(
                      size: 34,
                      color: badge.color ?? colors.primary,
                      icon: badge.icon ?? const GfSymbol('award', size: 22),
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
    ];

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
          padding: EdgeInsets.fromLTRB(16, 0, 16, showHeader ? 12 : 0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              nameRow,
              const SizedBox(height: 2),
              usernameRow,
              if (bio != null && bio!.trim().isNotEmpty) ...<Widget>[
                SizedBox(height: sectionGap),
                Text(
                  bio!.trim(),
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.4,
                    color: colors.baseContent,
                  ),
                ),
              ],
              if (signature?.trim().isNotEmpty == true) ...[
                SizedBox(height: sectionGap),
                Align(
                  alignment: Alignment.centerLeft,
                  child: IntrinsicWidth(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Transform.flip(
                              flipX: true,
                              child: GfSymbol(
                                'feather',
                                size: 14,
                                color: colors.primary.withValues(alpha: .62),
                              ),
                            ),
                            const SizedBox(width: 6),
                            Flexible(
                              child: Text(
                                signature!.trim(),
                                style: TextStyle(
                                  fontSize: 14,
                                  height: compact ? 1.45 : 1.55,
                                  fontWeight: FontWeight.w500,
                                  color: colors.baseContent.withValues(
                                    alpha: .62,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 2),
                        Padding(
                          padding: const EdgeInsets.only(left: 20),
                          child: CustomPaint(
                            size: const Size(0, 8),
                            painter: _SignatureSquigglePainter(
                              colors.primary.withValues(alpha: .45),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
              if (details != null) ...<Widget>[
                SizedBox(height: sectionGap),
                details!,
              ],
              if (badges.isNotEmpty || coloredBadges.isNotEmpty) ...[
                const SizedBox(height: 3),
                SizedBox(
                  key: const ValueKey('profile-badges-row'),
                  width: double.infinity,
                  child: compact
                      ? GfHorizontalScrollView(
                          scrollViewKey: const ValueKey(
                            'profile-badges-scroll',
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              for (
                                int i = 0;
                                i < profileBadges.length;
                                i++
                              ) ...[
                                if (i > 0) const SizedBox(width: 2),
                                profileBadges[i],
                              ],
                            ],
                          ),
                        )
                      : Wrap(
                          alignment: WrapAlignment.start,
                          spacing: 2,
                          runSpacing: 4,
                          children: profileBadges,
                        ),
                ),
              ],
              if (stats.isNotEmpty) ...<Widget>[
                const SizedBox(height: 3),
                GfHorizontalScrollView(
                  scrollViewKey: const ValueKey('profile-stats-row'),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      for (int i = 0; i < stats.length; i++) ...[
                        if (i > 0) SizedBox(width: compact ? 6 : 8),
                        ConstrainedBox(
                          constraints: BoxConstraints(
                            minWidth: statActions[i] == null ? 0 : 48,
                          ),
                          child: MergeSemantics(
                            child: Semantics(
                              button: statActions[i] != null,
                              child: InkWell(
                                onTap: statActions[i],
                                borderRadius: BorderRadius.circular(8),
                                child: SizedBox(
                                  height: 48,
                                  child: Padding(
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 4,
                                    ),
                                    child: Row(
                                      mainAxisSize: MainAxisSize.min,
                                      children: [
                                        Text(
                                          stats[i].$2,
                                          maxLines: 1,
                                          softWrap: false,
                                          style: TextStyle(
                                            fontSize: 13,
                                            fontWeight: FontWeight.w700,
                                            fontFeatures: const [
                                              FontFeature.tabularFigures(),
                                            ],
                                            color: colors.baseContent,
                                          ),
                                        ),
                                        const SizedBox(width: 3),
                                        Text(
                                          stats[i].$1,
                                          maxLines: 1,
                                          softWrap: false,
                                          style: TextStyle(
                                            fontSize: 12,
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
                        ),
                      ],
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

class _SignatureSquigglePainter extends CustomPainter {
  const _SignatureSquigglePainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final x = size.width / 100;
    final y = size.height / 8;
    final path = Path()
      ..moveTo(2 * x, 5 * y)
      ..cubicTo(10 * x, 0, 18 * x, 8 * y, 26 * x, 5 * y)
      ..cubicTo(34 * x, 2 * y, 42 * x, 8 * y, 50 * x, 5 * y)
      ..cubicTo(58 * x, 2 * y, 66 * x, 8 * y, 74 * x, 5 * y)
      ..cubicTo(82 * x, 2 * y, 90 * x, 8 * y, 98 * x, 5 * y);
    canvas.drawPath(
      path,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.8
        ..strokeCap = StrokeCap.round,
    );
  }

  @override
  bool shouldRepaint(covariant _SignatureSquigglePainter oldDelegate) =>
      oldDelegate.color != color;
}
