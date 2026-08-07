import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Static surface container.
///
/// Mobile form (default): full width, no border, no radius — matching the web
/// `<640px` breakpoint where `.gf-panel` loses its side borders and radius.
/// Set [emphasized] to restore the desktop panel look (border + radius).
class GfPanel extends StatelessWidget {
  const GfPanel({
    super.key,
    required this.child,
    this.padding = EdgeInsets.zero,
    this.emphasized = false,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;
  final bool emphasized;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    return DecoratedBox(
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: emphasized
            ? BorderRadius.circular(radii.box)
            : BorderRadius.zero,
        border: emphasized
            ? Border.all(color: colors.line, width: borders.width)
            : null,
      ),
      child: Padding(padding: padding, child: child),
    );
  }
}
