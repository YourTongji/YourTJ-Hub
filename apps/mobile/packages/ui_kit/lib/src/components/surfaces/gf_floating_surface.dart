import 'dart:ui';

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Floating surface, mirroring web `.gf-floating-surface` (components.css):
/// 1px line border, radius box, base-100 background, the two-layer
/// `gf-shadows.floating` shadow and an 8px backdrop blur (web
/// `backdrop-blur`). Used by the topic floating controls, floating reply
/// composer and bottom sheets.
class GfFloatingSurface extends StatelessWidget {
  const GfFloatingSurface({
    super.key,
    required this.child,
    this.padding = EdgeInsets.zero,
    this.radius,
    this.blur = true,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;

  /// Corner radius; defaults to `gf-radius-box` (8), callers may override
  /// for pill shapes (e.g. rounded-full floating bars).
  final double? radius;

  /// Whether to apply the 8px backdrop blur (web `backdrop-blur`).
  final bool blur;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    final double corner = radius ?? GfRadius.lg;

    Widget surface = Container(
      padding: padding,
      decoration: BoxDecoration(
        color: blur
            ? colors.surfaceRaised.withValues(alpha: 0.92)
            : colors.surfaceRaised,
        borderRadius: BorderRadius.circular(corner),
        border: Border.all(
          color: colors.separator.withValues(alpha: colors.isDark ? 1 : 0.6),
          width: borders.width / 2,
        ),
        boxShadow: colors.elevation(GfElevation.mid),
      ),
      child: child,
    );

    if (blur) {
      surface = ClipRRect(
        borderRadius: BorderRadius.circular(corner),
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 8, sigmaY: 8),
          child: surface,
        ),
      );
    }
    return surface;
  }
}

/// Drawer surface, mirroring web `.gf-drawer-surface` (components.css):
/// base-100 background with the strong `gf-shadows.drawer` edge shadow
/// (light source on the left, so the shadow falls to the right).
class GfDrawerSurface extends StatelessWidget {
  const GfDrawerSurface({super.key, required this.child, this.width});

  final Widget child;
  final double? width;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Container(
      width: width,
      decoration: BoxDecoration(
        color: colors.base100,
        boxShadow: shadows.drawer,
      ),
      child: child,
    );
  }
}
