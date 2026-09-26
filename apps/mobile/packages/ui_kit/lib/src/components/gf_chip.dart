import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Compact category fill with a separate 44px target when interactive.
class GfChip extends StatelessWidget {
  const GfChip({
    super.key,
    required this.label,
    required this.color,
    this.onTap,
    this.tapTargetAlignment = Alignment.center,
  });

  final String label;

  /// Category color; rendered as the leading dot.
  final Color color;

  final VoidCallback? onTap;

  /// Positions the visible chip within its unchanged accessible tap target.
  final Alignment tapTargetAlignment;

  static const _fontSize = 12.0;
  static const _lineHeight = 1.25;
  static const _verticalPadding = 2.0;

  /// Horizontal rails and their content insets share this scalable hit height.
  static double tapTargetHeightFor(BuildContext context) => math.max(
    44,
    (MediaQuery.textScalerOf(context).scale(_fontSize) * _lineHeight)
            .ceilToDouble() +
        _verticalPadding * 2,
  );

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    final Widget chip = Container(
      constraints: const BoxConstraints(minHeight: 24),
      padding: const EdgeInsets.symmetric(
        horizontal: 8,
        vertical: _verticalPadding,
      ),
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
                fontSize: _fontSize,
                height: _lineHeight,
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
          constraints: BoxConstraints(
            minWidth: 44,
            minHeight: tapTargetHeightFor(context),
          ),
          child: Align(
            alignment: tapTargetAlignment,
            widthFactor: 1,
            heightFactor: 1,
            child: chip,
          ),
        ),
      ),
    );
  }
}
