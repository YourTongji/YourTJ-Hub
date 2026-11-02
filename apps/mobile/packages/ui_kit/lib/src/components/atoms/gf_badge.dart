import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';

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
    this.radius = GfRadius.xs,
  });

  final String label;
  final GfBadgeVariant variant;
  final Widget? icon;
  final double radius;

  /// Optional source color for domain badges that carry their own palette.
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final (Color background, Color foreground) = _palette(colors);
    return Container(
      constraints: const BoxConstraints(minHeight: 20),
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(radius),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            IconTheme.merge(
              data: IconThemeData(size: 12, color: foreground),
              child: SizedBox.square(dimension: 12, child: icon!),
            ),
            const SizedBox(width: 3),
          ],
          Flexible(
            child: GfText(
              label,
              style: TextStyle(
                color: foreground,
                fontSize: 12,
                height: 16 / 12,
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
      return (
        customColor.withValues(alpha: colors.isDark ? 0.18 : 0.1),
        customColor,
      );
    }
    final GfTone tone = switch (variant) {
      GfBadgeVariant.muted => GfTone.neutral,
      GfBadgeVariant.info => GfTone.primary,
      GfBadgeVariant.success => GfTone.success,
      GfBadgeVariant.warning => GfTone.warning,
      GfBadgeVariant.error => GfTone.error,
    };
    return (colors.toneSoft(tone), colors.toneOnSoft(tone));
  }
}

/// Unread counter: an error-coloured capsule that stays circular for one
/// digit and caps at [max]+.
class GfCountBadge extends StatelessWidget {
  const GfCountBadge({
    super.key,
    required this.count,
    this.max = 99,
    this.showCount = true,
    this.compact = false,
  });

  final int count;
  final int max;
  final bool showCount, compact;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    if (!showCount) {
      return Container(
        width: 8,
        height: 8,
        decoration: BoxDecoration(color: colors.error, shape: BoxShape.circle),
      );
    }
    return Container(
      constraints: BoxConstraints(
        minWidth: compact ? 16 : 20,
        minHeight: compact ? 16 : 20,
      ),
      padding: EdgeInsets.symmetric(horizontal: compact ? 4 : 6),
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: colors.error,
        borderRadius: BorderRadius.circular(GfRadius.full),
      ),
      child: GfText(
        count > max ? '$max+' : '$count',
        style: GfTheme.typographyOf(context).label.copyWith(
          color: colors.errorContent,
          height: 1,
          fontSize: compact ? 11 : null,
        ),
      ),
    );
  }
}
