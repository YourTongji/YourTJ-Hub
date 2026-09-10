import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Dropdown menu surface, mirroring web `.gf-menu-surface` (components.css):
/// 1px line border, radius box, base-100 background, `gf-shadows.menu`.
class GfMenuSurface extends StatelessWidget {
  const GfMenuSurface({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(4),
  });

  final Widget child;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Material(
      color: colors.base100,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.box),
        side: BorderSide(color: colors.line, width: borders.width),
      ),
      clipBehavior: Clip.antiAlias,
      elevation: 0,
      child: Container(
        padding: padding,
        decoration: BoxDecoration(boxShadow: shadows.menu),
        child: child,
      ),
    );
  }
}

/// Menu item row, mirroring web `.gf-menu-item*` (components.css): h-9
/// (36px), gap-2.5 (10px), px-3 (12px), 14px text.
enum GfMenuItemVariant {
  /// `gf-menu-item`: base-content/75, hover base-200.
  normal,

  /// `gf-menu-item-primary`: primary text, hover info/10.
  primary,

  /// `gf-menu-item-warning`: warning text, hover warning/10.
  warning,

  /// `gf-menu-item-danger`: error text, hover error/10.
  danger,
}

class GfMenuItem extends StatelessWidget {
  const GfMenuItem({
    super.key,
    required this.label,
    this.icon,
    this.onTap,
    this.variant = GfMenuItemVariant.normal,
  });

  final String label;
  final IconData? icon;
  final VoidCallback? onTap;
  final GfMenuItemVariant variant;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    final Color color = switch (variant) {
      GfMenuItemVariant.normal => colors.baseContent.withValues(alpha: 0.75),
      GfMenuItemVariant.primary => colors.primary,
      GfMenuItemVariant.warning => colors.warning,
      GfMenuItemVariant.danger => colors.error,
    };

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        hoverColor: switch (variant) {
          GfMenuItemVariant.normal => colors.base200,
          GfMenuItemVariant.primary => colors.info.withValues(alpha: 0.10),
          GfMenuItemVariant.warning => colors.warning.withValues(alpha: 0.10),
          GfMenuItemVariant.danger => colors.error.withValues(alpha: 0.10),
        },
        child: Container(
          height: 36,
          padding: const EdgeInsets.symmetric(horizontal: 12),
          alignment: Alignment.centerLeft,
          child: Row(
            children: <Widget>[
              if (icon != null) ...<Widget>[
                Icon(icon, size: 18, color: color),
                const SizedBox(width: 10),
              ],
              Text(
                label,
                style: TextStyle(
                  color: color,
                  fontSize: 14,
                  fontWeight: variant == GfMenuItemVariant.primary
                      ? FontWeight.w600
                      : FontWeight.w500,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
