import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Small category chip with a leading color dot, mirroring web `.gf-topic-chip`
/// (base-300 pill, muted 11px text).
class GfChip extends StatelessWidget {
  const GfChip({
    super.key,
    required this.label,
    required this.color,
    this.onTap,
  });

  final String label;

  /// Category color; rendered as the leading dot.
  final Color color;

  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    final Widget chip = Container(
      height: 20,
      padding: const EdgeInsets.symmetric(horizontal: 6),
      decoration: BoxDecoration(
        color: colors.base300,
        borderRadius: BorderRadius.circular(radii.selector),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 4),
          Text(
            label,
            style: TextStyle(
              color: colors.baseContent.withValues(alpha: 0.55),
              fontSize: 11,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );

    if (onTap == null) return chip;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: chip,
    );
  }
}
