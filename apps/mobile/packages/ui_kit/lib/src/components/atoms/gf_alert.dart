import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Alert surface, mirroring web `.gf-alert` (components.css): 1px line border,
/// radius box, base-100 background, `gf-shadows.alert`.
class GfAlert extends StatelessWidget {
  const GfAlert({
    super.key,
    required this.child,
    this.icon,
    this.padding = const EdgeInsets.all(16),
  });

  final Widget child;
  final Widget? icon;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(radii.box),
        border: Border.all(color: colors.line, width: borders.width),
        boxShadow: shadows.alert,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          if (icon != null) ...<Widget>[icon!, const SizedBox(width: 12)],
          Expanded(child: child),
        ],
      ),
    );
  }
}
