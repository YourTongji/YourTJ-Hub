import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Category chip with a leading color dot and a 44px target when interactive.
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
      constraints: const BoxConstraints(minHeight: 24),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
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
          Flexible(
            child: Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: colors.baseContent.withValues(alpha: 0.72),
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );

    if (onTap == null) return chip;
    return Semantics(
      button: true,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(radii.selector),
        child: ConstrainedBox(
          constraints: const BoxConstraints(minWidth: 44, minHeight: 44),
          child: Center(widthFactor: 1, heightFactor: 1, child: chip),
        ),
      ),
    );
  }
}
