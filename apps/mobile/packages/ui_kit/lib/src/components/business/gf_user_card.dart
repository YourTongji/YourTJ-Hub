import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_avatar.dart';
import '../atoms/gf_badge.dart';
import '../gf_symbol.dart';

@immutable
class GfUserBadge {
  const GfUserBadge({required this.label, this.color});

  final String label;
  final Color? color;
}

/// User profile header card mirroring web UserPage.vue mobile layout:
/// a cover, overlapping avatar, identity, trimmed bio and a separate signature.
/// The overlap is part of layout, so no translated blank space remains below.
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

  /// Badge labels shown next to the name (e.g. Admin, online).
  final List<String> badges;

  /// Badges that preserve a source-defined color.
  final List<GfUserBadge> coloredBadges;

  /// (label, value) pairs rendered in a compact, equal-width stats row.
  final List<(String, String)> stats;

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
        Stack(
          children: [
            Container(
              height: 184,
              padding: const EdgeInsets.only(bottom: 44),
              child: Container(
                width: double.infinity,
                decoration: BoxDecoration(
                  color: colors.base300,
                  image: coverUrl == null || coverUrl!.isEmpty
                      ? null
                      : DecorationImage(
                          image: ResizeImage(
                            NetworkImage(coverUrl!),
                            policy: ResizeImagePolicy.fit,
                            width:
                                (MediaQuery.sizeOf(context).width *
                                        MediaQuery.devicePixelRatioOf(context))
                                    .round(),
                            height:
                                (184 * MediaQuery.devicePixelRatioOf(context))
                                    .round(),
                          ),
                          fit: BoxFit.cover,
                        ),
                ),
              ),
            ),
            Positioned(
              left: 20,
              top: 96,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: colors.base100,
                ),
                child: Padding(
                  padding: const EdgeInsets.all(4),
                  child: GfAvatar(src: avatarUrl, size: 80, badge: avatarBadge),
                ),
              ),
            ),
          ],
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 24),
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
                      fontSize: 24,
                      fontWeight: FontWeight.w700,
                      color: colors.baseContent,
                    ),
                  ),
                  for (final String badge in badges)
                    GfBadge(label: badge, variant: GfBadgeVariant.info),
                  for (final GfUserBadge badge in coloredBadges)
                    GfBadge(label: badge.label, color: badge.color),
                ],
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
                    color: colors.baseContent.withValues(alpha: 0.75),
                  ),
                ),
              ],
              if (signature?.trim().isNotEmpty == true) ...[
                const SizedBox(height: 8),
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfSymbol(
                      'feather',
                      size: 18,
                      color: colors.primary.withValues(alpha: 0.6),
                    ),
                    const SizedBox(width: 8),
                    Flexible(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(
                            signature!.trim(),
                            style: TextStyle(
                              fontSize: 16,
                              height: 1.45,
                              fontWeight: FontWeight.w500,
                              color: colors.iconMuted,
                            ),
                          ),
                          const SizedBox(height: 4),
                          CustomPaint(
                            size: const Size(160, 5),
                            painter: _SignatureLine(
                              colors.primary.withValues(alpha: 0.4),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ],
              if (details != null) ...<Widget>[
                const SizedBox(height: 8),
                details!,
              ],
              if (actions != null) ...<Widget>[
                const SizedBox(height: 12),
                actions!,
              ],
              if (stats.isNotEmpty) ...<Widget>[
                const SizedBox(height: 16),
                LayoutBuilder(
                  builder: (context, constraints) {
                    final scaledWidth = MediaQuery.textScalerOf(
                      context,
                    ).scale(112);
                    final columns =
                        MediaQuery.textScalerOf(context).scale(14) <= 20 &&
                            constraints.maxWidth >= 320
                        ? stats.length
                        : (constraints.maxWidth / scaledWidth).floor().clamp(
                            1,
                            stats.length,
                          );
                    return Wrap(
                      runSpacing: 16,
                      children: [
                        for (final (label, value) in stats)
                          SizedBox(
                            width: constraints.maxWidth / columns,
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  value,
                                  style: TextStyle(
                                    fontSize: 18,
                                    height: 1.35,
                                    fontWeight: FontWeight.w600,
                                    color: colors.baseContent,
                                    fontFeatures: const [
                                      FontFeature.tabularFigures(),
                                    ],
                                  ),
                                ),
                                const SizedBox(height: 4),
                                Text(
                                  label,
                                  style: TextStyle(
                                    fontSize: 14,
                                    height: 1.35,
                                    color: colors.baseContent.withValues(
                                      alpha: 0.72,
                                    ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                      ],
                    );
                  },
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

class _SignatureLine extends CustomPainter {
  const _SignatureLine(this.color);
  final Color color;
  @override
  void paint(Canvas canvas, Size size) {
    final path = Path()..moveTo(0, size.height / 2);
    for (double x = 0; x < size.width; x += 40) {
      path.cubicTo(x + 10, 0, x + 30, size.height, x + 40, size.height / 2);
    }
    canvas.drawPath(
      path,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.5
        ..strokeCap = StrokeCap.round,
    );
  }

  @override
  bool shouldRepaint(_SignatureLine oldDelegate) => oldDelegate.color != color;
}
