import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Tappable card surface used for list rows.
///
/// Mobile form (default): full width, no border, no radius, hairline divider
/// at the bottom — matching the web `<640px` breakpoint where `.gf-card`
/// keeps only its bottom border. Set [emphasized] to restore the desktop card
/// look (border + radius + subtle shadow).
class GfCard extends StatelessWidget {
  const GfCard({
    super.key,
    required this.child,
    this.onTap,
    this.padding = EdgeInsets.zero,
    this.emphasized = false,
    this.showDivider = true,
  });

  final Widget child;
  final VoidCallback? onTap;
  final EdgeInsetsGeometry padding;
  final bool emphasized;

  /// Whether the mobile hairline bottom divider is rendered.
  final bool showDivider;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    final BorderRadius radius = emphasized
        ? BorderRadius.circular(radii.box)
        : BorderRadius.zero;
    final Color dividerColor = colors.line.withValues(alpha: 0.7);

    return Material(
      color: colors.base100,
      shape: emphasized
          ? RoundedRectangleBorder(
              borderRadius: radius,
              side: BorderSide(color: colors.line, width: borders.width),
            )
          : null,
      clipBehavior: emphasized ? Clip.antiAlias : Clip.none,
      elevation: 0,
      child: InkWell(
        onTap: onTap,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            Padding(padding: padding, child: child),
            if (!emphasized && showDivider)
              Container(
                height: borders.width,
                margin: const EdgeInsets.symmetric(horizontal: 16),
                color: dividerColor,
              ),
          ],
        ),
      ),
    );
  }
}
