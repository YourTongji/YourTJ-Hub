import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

/// Status badge, mirroring web `.gf-badge` variants (components.css):
/// px-2 py-0.5, 12px w600, radius selector. Tint fills use the 10% alpha
/// version of the semantic color with the full-strength text color.
enum GfBadgeVariant {
  /// `gf-badge-muted`: base-300 fill, base-content/55 text.
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
    final ThemeData theme = Theme.of(context);
    final List<ThemeExtension<dynamic>> extensions = theme.extensions.values
        .toList(growable: true);
    extensions
      ..removeWhere((ThemeExtension<dynamic> item) => item is td.TTagThemeData)
      ..add(
        td.TTagThemeData(
          textColor: foreground,
          backgroundColor: background,
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          fontWeight: FontWeight.w600,
          shape: td.TTagShape.round,
        ),
      );

    return Theme(
      data: theme.copyWith(extensions: extensions),
      child: td.TTag(
        label,
        size: td.TTagSize.small,
        icon: icon is Icon ? (icon! as Icon).icon : null,
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
        return (colors.base300, colors.baseContent.withValues(alpha: 0.55));
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
