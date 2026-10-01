import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import 'gf_liquid_surface.dart';

/// Shared optical surface for floating controls and reply composers.
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

  /// Native floating controls use 28px corners; callers may request capsules.
  final double? radius;

  /// False requests a solid surface, including on optical-capable renderers.
  final bool blur;

  @override
  Widget build(BuildContext context) {
    return GfLiquidSurface(
      radius: radius ?? 28,
      padding: padding,
      forceOpaque: !blur,
      child: child,
    );
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
