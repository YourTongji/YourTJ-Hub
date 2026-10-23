import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Card surface with optional tap interaction.
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
    final double hairline = gfHairline(context);

    final Widget content = Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Padding(padding: padding, child: child),
        if (!emphasized && showDivider)
          Container(
            height: hairline,
            margin: const EdgeInsets.symmetric(horizontal: GfSpace.gutter),
            color: colors.separator,
          ),
      ],
    );

    if (!emphasized) {
      return Material(
        color: colors.canvas,
        child: onTap == null ? content : InkWell(onTap: onTap, child: content),
      );
    }

    // Raised card: grouped surface, large radius, soft shadow in light mode
    // and a hairline in dark mode where shadows do not read.
    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(GfRadius.lg),
        boxShadow: colors.elevation(GfElevation.low),
      ),
      child: Material(
        color: colors.isDark ? colors.surfaceGrouped : colors.base100,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(GfRadius.lg),
          side: colors.isDark
              ? BorderSide(color: colors.separator, width: hairline)
              : BorderSide.none,
        ),
        clipBehavior: Clip.antiAlias,
        child: onTap == null ? content : InkWell(onTap: onTap, child: content),
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
