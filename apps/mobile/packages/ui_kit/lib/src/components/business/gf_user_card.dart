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
    this.nameBadges = const <Widget>[],
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
  final List<Widget> nameBadges;
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
          padding: EdgeInsets.fromLTRB(16, 0, 16, showHeader ? 12 : 8),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Wrap(
                spacing: 8,
                runSpacing: 4,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: [
                  Text(
                    name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 22,
                      fontWeight: FontWeight.w700,
                      color: colors.baseContent,
                    ),
                  ),
                  ...nameBadges,
                ],
              ),
              const SizedBox(height: 2),
              Text(
                '@$username',
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: colors.baseContent.withValues(alpha: 0.55),
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
              if (signature?.trim().isNotEmpty == true) ...[
                const SizedBox(height: 12),
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
                                  height: 1.55,
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
                const SizedBox(height: 12),
                details!,
              ],
              if (badges.isNotEmpty || coloredBadges.isNotEmpty) ...[
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: Wrap(
                    alignment: WrapAlignment.spaceEvenly,
                    spacing: 0,
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
                                  tapTargetSize:
                                      MaterialTapTargetSize.shrinkWrap,
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
                ),
              ],
              if (stats.isNotEmpty) ...<Widget>[
                const SizedBox(height: 8),
                Row(
                  children: [
                    for (int i = 0; i < stats.length; i++)
                      Expanded(
                        child: MergeSemantics(
                          child: Semantics(
                            button: statActions[i] != null,
                            child: InkWell(
                              onTap: statActions[i],
                              borderRadius: BorderRadius.circular(8),
                              child: SizedBox(
                                height: 48,
                                child: Center(
                                  child: Padding(
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 2,
                                    ),
                                    child: FittedBox(
                                      fit: BoxFit.scaleDown,
                                      child: Row(
                                        mainAxisSize: MainAxisSize.min,
                                        children: [
                                          Text(
                                            stats[i].$2,
                                            maxLines: 1,
                                            softWrap: false,
                                            style: TextStyle(
                                              fontSize: 14,
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
                                              fontSize: 13,
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
