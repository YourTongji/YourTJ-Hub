import 'package:flutter/material.dart';
import '../../theme/gf_theme.dart';

/// Web's achievement presentation: a coloured hexagon, title and description.
class GfAchievementCard extends StatelessWidget {
  const GfAchievementCard({
    super.key,
    required this.title,
    required this.description,
    required this.icon,
    required this.color,
  });
  final String title;
  final String description;
  final Widget icon;
  final Color color;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colors.line),
      ),
      child: Row(
        children: [
          ClipPath(
            clipper: _Hexagon(),
            child: Container(
              width: 48,
              height: 48,
              color: color.withValues(alpha: 0.14),
              alignment: Alignment.center,
              child: icon,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  title,
                  style: GfTheme.typographyOf(
                    context,
                  ).body.copyWith(fontWeight: FontWeight.w600),
                ),
                if (description.trim().isNotEmpty) ...[
                  const SizedBox(height: 4),
                  Text(
                    description,
                    style: GfTheme.typographyOf(
                      context,
                    ).caption.copyWith(color: colors.iconMuted),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _Hexagon extends CustomClipper<Path> {
  @override
  Path getClip(Size s) => Path()
    ..moveTo(s.width * .25, 0)
    ..lineTo(s.width * .75, 0)
    ..lineTo(s.width, s.height / 2)
    ..lineTo(s.width * .75, s.height)
    ..lineTo(s.width * .25, s.height)
    ..lineTo(0, s.height / 2)
    ..close();
  @override
  bool shouldReclip(_Hexagon oldClipper) => false;
}
