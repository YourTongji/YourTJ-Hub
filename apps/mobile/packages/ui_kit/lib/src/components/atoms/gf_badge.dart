import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Status badge, mirroring web `.gf-badge` variants (components.css):
/// px-2 py-0.5, 12px w500, capsule radius. Tint fills use the 10% alpha
/// version of the semantic color with the full-strength text color.
enum GfBadgeVariant {
  /// Quiet grey fill with muted text.
  muted,

  /// `gf-badge-info`: info/10 fill, primary text.
  info,

  /// `gf-badge-success`: success/10 fill, success text.
  success,

  /// `gf-badge-warning`: warning/10 fill, warning text.
  warning,

  /// `gf-badge-error`: error/10 fill, error text.
  error,
}

/// Small pill badge aligned with web `gf-badge` semantics.
class GfBadge extends StatelessWidget {
  const GfBadge({
    super.key,
    required this.label,
    this.variant = GfBadgeVariant.muted,
    this.icon,
    this.color,
  });

  final String label;
  final GfBadgeVariant variant;
  final Widget? icon;

  /// Optional source color for domain badges that carry their own palette.
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final (Color background, Color foreground) = _palette(colors);
    return Container(
      constraints: const BoxConstraints(minHeight: 22),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            IconTheme.merge(
              data: IconThemeData(size: 14, color: foreground),
              child: SizedBox.square(dimension: 14, child: icon!),
            ),
            const SizedBox(width: 4),
          ],
          Flexible(
            child: Text(
              label,
              style: TextStyle(
                color: foreground,
                fontSize: 12,
                height: 1.25,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }

  (Color, Color) _palette(GfColors colors) {
    final Color? customColor = color;
    if (customColor != null) {
      return (customColor.withValues(alpha: 0.12), customColor);
    }
    switch (variant) {
      case GfBadgeVariant.muted:
        return (colors.base200, colors.iconMuted);
      case GfBadgeVariant.info:
        return (colors.info.withValues(alpha: 0.10), colors.primary);
      case GfBadgeVariant.success:
        return (colors.success.withValues(alpha: 0.10), colors.success);
      case GfBadgeVariant.warning:
        return (colors.warning.withValues(alpha: 0.10), colors.warning);
      case GfBadgeVariant.error:
        return (colors.error.withValues(alpha: 0.10), colors.error);
    }
  }
}
