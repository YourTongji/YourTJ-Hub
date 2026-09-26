import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../atoms/gf_badge_medallion.dart';

/// An earned badge with centered artwork and a readable, naturally sized title.
/// The full description remains accessible when its visual preview is clipped.
class GfAchievementCard extends StatelessWidget {
  const GfAchievementCard({
    super.key,
    required this.title,
    required this.description,
    required this.icon,
    required this.color,
    this.onTap,
  });

  final String title;
  final String description;
  final Widget icon;
  final Color color;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    const radius = BorderRadius.all(Radius.circular(18));
    return MergeSemantics(
      child: Semantics(
        button: onTap != null,
        child: Material(
          color: colors.base100,
          shape: RoundedRectangleBorder(
            borderRadius: radius,
            side: BorderSide(color: colors.line),
          ),
          clipBehavior: Clip.antiAlias,
          child: InkWell(
            onTap: onTap,
            borderRadius: radius,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 18),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  ExcludeSemantics(
                    child: Center(
                      child: GfBadgeMedallion(icon: icon, color: color),
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    title,
                    textAlign: TextAlign.center,
                    style: type.body.copyWith(
                      fontSize: 15,
                      height: 1.4,
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                  if (description.trim().isNotEmpty) ...[
                    const SizedBox(height: 6),
                    Text(
                      description,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      textAlign: TextAlign.center,
                      style: type.caption.copyWith(
                        fontSize: 13,
                        height: 1.4,
                        color: colors.iconMuted,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
