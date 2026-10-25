import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../gf_symbol.dart';

/// Bounded native menu surface with soft corners and a subtle outline.
class GfMenuSurface extends StatelessWidget {
  const GfMenuSurface({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(6),
  });

  final Widget child;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(GfRadius.lg),
        boxShadow: colors.elevation(GfElevation.mid),
      ),
      child: Material(
        color: colors.surfaceRaised,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(GfRadius.lg),
          side: colors.isDark
              ? BorderSide(color: colors.separator, width: borders.width / 2)
              : BorderSide.none,
        ),
        clipBehavior: Clip.antiAlias,
        elevation: 0,
        child: Padding(padding: padding, child: child),
      ),
    );
  }
}

/// Menu actions have a 48-pixel minimum target and grow with their labels.
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
    this.symbol,
    this.iconSize = GfIconSize.md,
    this.onTap,
    this.variant = GfMenuItemVariant.normal,
  });

  final String label;
  final String? symbol;
  final double iconSize;
  final VoidCallback? onTap;
  final GfMenuItemVariant variant;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    final Color variantColor = switch (variant) {
      GfMenuItemVariant.normal => colors.textPrimary,
      GfMenuItemVariant.primary => colors.primary,
      GfMenuItemVariant.warning => colors.warning,
      GfMenuItemVariant.danger => colors.error,
    };

    final color = onTap == null ? colors.textDisabled : variantColor;

    final Widget? leadingIcon = symbol == null
        ? null
        : GfSymbol(
            symbol!,
            size: iconSize,
            color: variant == GfMenuItemVariant.normal && onTap != null
                ? colors.textSecondary
                : color,
          );

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(GfRadius.md),
        hoverColor: switch (variant) {
          GfMenuItemVariant.normal => colors.pressed,
          GfMenuItemVariant.primary => colors.info.withValues(alpha: 0.10),
          GfMenuItemVariant.warning => colors.warning.withValues(alpha: 0.10),
          GfMenuItemVariant.danger => colors.error.withValues(alpha: 0.10),
        },
        child: Container(
          constraints: const BoxConstraints(minHeight: GfControl.touch),
          padding: const EdgeInsets.symmetric(
            horizontal: GfSpace.md,
            vertical: 10,
          ),
          alignment: Alignment.centerLeft,
          child: Row(
            children: <Widget>[
              if (leadingIcon != null) ...<Widget>[
                leadingIcon,
                const SizedBox(width: GfSpace.md),
              ],
              Expanded(
                child: GfText(
                  label,
                  style: GfTheme.typographyOf(context).body.copyWith(
                    color: color,
                    fontWeight: variant == GfMenuItemVariant.primary
                        ? FontWeight.w600
                        : FontWeight.w400,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
