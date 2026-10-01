import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';
import 'gf_symbol.dart';

class GfBottomNavigationItem {
  const GfBottomNavigationItem({
    required this.label,
    required this.symbol,
    required this.selectedSymbol,
    this.badge = false,
    this.badgeSemanticLabel,
  });

  final String label;
  final String symbol;
  final String selectedSymbol;
  final bool badge;
  final String? badgeSemanticLabel;

  String get semanticsLabel => badge && badgeSemanticLabel != null
      ? '$label, $badgeSemanticLabel'
      : label;
}

/// Shared geometry for the compact navigation shell.
///
/// Keeping these offsets here prevents page content and floating actions from
/// drifting when the bar switches between icon-only and labeled modes.
class GfBottomNavigationMetrics {
  const GfBottomNavigationMetrics({
    required this.barHeight,
    required this.contentBottomInset,
    required this.actionBottomInset,
  });

  final double barHeight;
  final double contentBottomInset;
  final double actionBottomInset;
}

/// Four accessible navigation destinations with optional visible labels and a
/// compose slot.
class GfBottomNavigation extends StatelessWidget {
  const GfBottomNavigation({
    super.key,
    required this.currentIndex,
    required this.items,
    required this.onSelected,
    this.onAction,
    this.actionLabel = '发布',
    this.actionSymbol = 'plus',
    this.showLabels = true,
  });

  final int currentIndex;
  final List<GfBottomNavigationItem> items;
  final ValueChanged<int> onSelected;
  final VoidCallback? onAction;
  final String actionLabel;
  final String actionSymbol;
  final bool showLabels;

  static GfBottomNavigationMetrics metrics({
    required double safeAreaBottom,
    required double barHeight,
  }) {
    return GfBottomNavigationMetrics(
      barHeight: barHeight,
      contentBottomInset: barHeight + safeAreaBottom + 8,
      actionBottomInset: barHeight + 16,
    );
  }

  static double heightFor(
    BuildContext context, {
    required Iterable<String> labels,
    required double availableWidth,
    int slots = 4,
    bool showLabels = true,
    bool hasAction = false,
  }) {
    if (!showLabels) return 56;
    final TextStyle style = _navigationLabelStyle(
      context,
      color: GfTheme.colorsOf(context).baseContent,
      fontWeight: FontWeight.w600,
    );
    final TextScaler scaler = MediaQuery.textScalerOf(context);
    final double itemWidth = math.max(1, availableWidth / slots - 8);
    double maxLabelHeight = 0;
    for (final String label in labels) {
      final TextPainter painter = TextPainter(
        text: TextSpan(text: label, style: style),
        textDirection: Directionality.of(context),
        textScaler: scaler,
      )..layout(maxWidth: itemWidth);
      maxLabelHeight = math.max(maxLabelHeight, painter.height);
      painter.dispose();
    }
    // The top divider contributes one layout pixel before the destination
    // padding and label.
    return math.max(72, (hasAction ? 56 : 53) + maxLabelHeight);
  }

  @override
  Widget build(BuildContext context) {
    if (items.length != 4) {
      throw FlutterError(
        'GfBottomNavigation requires exactly four destinations; '
        'received ${items.length}.',
      );
    }
    final GfColors colors = GfTheme.colorsOf(context);
    final bool highContrast = MediaQuery.highContrastOf(context);
    final bool useGlass =
        !highContrast &&
        !MediaQuery.disableAnimationsOf(context) &&
        !MediaQuery.accessibleNavigationOf(context) &&
        Theme.of(context).platform == TargetPlatform.iOS;
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final double barHeight = heightFor(
          context,
          labels: [
            ...items.map((item) => item.label),
            if (onAction != null) actionLabel,
          ],
          availableWidth: math.max(
            1,
            constraints.maxWidth - MediaQuery.paddingOf(context).horizontal,
          ),
          slots: onAction == null ? 4 : 5,
          showLabels: showLabels,
          hasAction: onAction != null && showLabels,
        );
        return _buildBar(context, colors, useGlass, barHeight);
      },
    );
  }

  Widget _buildBar(
    BuildContext context,
    GfColors colors,
    bool useGlass,
    double barHeight,
  ) {
    final Widget bar = Material(
      color: useGlass ? colors.base100.withValues(alpha: 0.94) : colors.base100,
      child: SafeArea(
        top: false,
        child: Container(
          height: barHeight,
          decoration: BoxDecoration(
            border: Border(top: BorderSide(color: colors.line)),
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              _Destination(
                showLabel: showLabels,
                item: items[0],
                selected: currentIndex == 0,
                onTap: () => onSelected(0),
              ),
              _Destination(
                showLabel: showLabels,
                item: items[1],
                selected: currentIndex == 1,
                onTap: () => onSelected(1),
              ),
              if (onAction != null)
                _ComposeAction(
                  symbol: actionSymbol,
                  label: actionLabel,
                  onTap: onAction,
                  showLabel: showLabels,
                ),
              _Destination(
                showLabel: showLabels,
                item: items[2],
                selected: currentIndex == 2,
                onTap: () => onSelected(2),
              ),
              _Destination(
                showLabel: showLabels,
                item: items[3],
                selected: currentIndex == 3,
                onTap: () => onSelected(3),
              ),
            ],
          ),
        ),
      ),
    );
    if (!useGlass) return bar;
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 18, sigmaY: 18),
        child: bar,
      ),
    );
  }
}

class _Destination extends StatelessWidget {
  const _Destination({
    required this.item,
    required this.selected,
    required this.onTap,
    required this.showLabel,
  });

  final bool showLabel;
  final GfBottomNavigationItem item;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Color foreground = selected ? colors.primary : colors.iconMuted;
    final Color labelColor = selected
        ? colors.primary
        : colors.baseContent.withValues(alpha: 0.72);
    final bool reducedMotion = GfMotion.reducedOf(context);

    return Expanded(
      child: Semantics(
        button: true,
        selected: selected,
        label: item.semanticsLabel,
        onTap: onTap,
        child: ExcludeSemantics(
          child: InkWell(
            onTap: onTap,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(4, 8, 4, 6),
              child: KeyedSubtree(
                // A preference change must discard any in-flight decorative
                // interpolation so reduced motion takes effect immediately.
                key: ValueKey<String>(
                  'gf-bottom-navigation-destination-${item.symbol}-$reducedMotion',
                ),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: <Widget>[
                    AnimatedContainer(
                      // Keep the animated element stable when selection moves
                      // between destinations so the transition can be
                      // interrupted without jumping to its end state.
                      key: ValueKey<String>(
                        'gf-bottom-navigation-indicator-${item.symbol}',
                      ),
                      duration: GfMotion.duration(context, GfMotion.selection),
                      curve: GfMotion.layoutCurve,
                      width: 48,
                      height: 36,
                      decoration: BoxDecoration(
                        color: selected
                            ? colors.primary.withValues(alpha: 0.09)
                            : Colors.transparent,
                        borderRadius: BorderRadius.circular(14),
                      ),
                      child: Stack(
                        clipBehavior: Clip.none,
                        children: <Widget>[
                          Center(
                            child: TweenAnimationBuilder<Color?>(
                              tween: ColorTween(end: foreground),
                              duration: GfMotion.duration(
                                context,
                                GfMotion.selection,
                              ),
                              curve: GfMotion.layoutCurve,
                              builder: (context, color, child) => GfSymbol(
                                selected ? item.selectedSymbol : item.symbol,
                                size: 24,
                                color: color ?? foreground,
                              ),
                            ),
                          ),
                          if (item.badge)
                            Positioned(
                              top: 3,
                              right: 8,
                              child: Container(
                                width: 7,
                                height: 7,
                                decoration: BoxDecoration(
                                  color: colors.primary,
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                    color: colors.base100,
                                    width: 1,
                                  ),
                                ),
                              ),
                            ),
                        ],
                      ),
                    ),
                    if (showLabel) const SizedBox(height: 2),
                    if (showLabel)
                      AnimatedDefaultTextStyle(
                        duration: GfMotion.duration(
                          context,
                          GfMotion.selection,
                        ),
                        curve: GfMotion.layoutCurve,
                        style: _navigationLabelStyle(
                          context,
                          color: labelColor,
                          fontWeight: selected
                              ? FontWeight.w600
                              : FontWeight.w500,
                        ),
                        child: Text(item.label, textAlign: TextAlign.center),
                      ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _ComposeAction extends StatelessWidget {
  const _ComposeAction({
    required this.symbol,
    required this.label,
    required this.onTap,
    required this.showLabel,
  });

  final String symbol;
  final String label;
  final VoidCallback? onTap;
  final bool showLabel;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Expanded(
      child: Semantics(
        button: true,
        label: label,
        onTap: onTap,
        child: ExcludeSemantics(
          child: InkWell(
            onTap: onTap,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(4, 5, 4, 5),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: <Widget>[
                  DecoratedBox(
                    decoration: BoxDecoration(
                      color: colors.primary,
                      shape: BoxShape.circle,
                      boxShadow: shadows.card,
                    ),
                    child: SizedBox(
                      width: 44,
                      height: 44,
                      child: GfSymbol(
                        symbol,
                        size: 24,
                        color: colors.primaryContent,
                      ),
                    ),
                  ),
                  const SizedBox(height: 1),
                  if (showLabel)
                    Text(
                      label,
                      textAlign: TextAlign.center,
                      style: _navigationLabelStyle(
                        context,
                        color: colors.primary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

TextStyle _navigationLabelStyle(
  BuildContext context, {
  required Color color,
  required FontWeight fontWeight,
}) {
  final TextStyle materialStyle =
      Theme.of(context).textTheme.bodyMedium ?? const TextStyle();
  return materialStyle.merge(
    GfTheme.typographyOf(context).meta.copyWith(
      color: color,
      fontWeight: MediaQuery.boldTextOf(context) ? FontWeight.w700 : fontWeight,
    ),
  );
}
