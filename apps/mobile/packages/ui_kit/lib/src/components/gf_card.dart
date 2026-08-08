import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Tappable card surface used for list rows.
///
/// Mobile form (default): full width, no border, no radius, hairline divider
/// at the bottom — matching the web `<640px` breakpoint where `.gf-card`
/// keeps only its bottom border. Set [emphasized] to restore the desktop card
/// look (border + radius + `gf-shadows.card`).
///
/// Web adjacency rule (`components.css:50-55`): in a mobile shell, adjacent
/// cards share a single bottom hairline — every card *except the last* hides
/// its divider (`:has(+ .gf-card)` transparentizes the border). Use
/// [GfCardList] to apply this automatically.
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
    final GfShadows shadows = GfTheme.shadowsOf(context);

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
      shadowColor: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        child: Container(
          decoration: emphasized
              ? BoxDecoration(boxShadow: shadows.card)
              : null,
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
      ),
    );
  }
}

/// Renders a list of [GfCard]s applying the web adjacency rule: every card
/// except the last hides its bottom divider, so adjacent cards read as one
/// seamless mobile sheet with a single closing hairline (web
/// `:has(+ .gf-card)` transparentization).
class GfCardList extends StatelessWidget {
  const GfCardList({
    super.key,
    required this.children,
    this.cardPadding = EdgeInsets.zero,
    this.emphasized = false,
  });

  final List<Widget> children;
  final EdgeInsetsGeometry cardPadding;
  final bool emphasized;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        for (int i = 0; i < children.length; i++)
          GfCard(
            padding: cardPadding,
            emphasized: emphasized,
            showDivider: i == children.length - 1,
            child: children[i],
          ),
      ],
    );
  }
}
