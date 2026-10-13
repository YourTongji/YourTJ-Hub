import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

/// Shows destination names while accessibility exposes the same named,
/// selectable actions as the compact bottom navigation, including in iOS.
class ReadingNavigationRail extends StatelessWidget {
  const ReadingNavigationRail({
    super.key,
    required this.items,
    required this.currentIndex,
    required this.onSelected,
  });
  final List<GfBottomNavigationItem> items;
  final int currentIndex;
  final ValueChanged<int> onSelected;

  static double widthFor(
    BuildContext context,
    List<GfBottomNavigationItem> items,
  ) {
    var width = 72.0;
    for (final item in items) {
      final painter = TextPainter(
        text: TextSpan(
          text: item.label,
          style: GfTheme.typographyOf(
            context,
          ).caption.copyWith(fontWeight: FontWeight.w600),
        ),
        textDirection: Directionality.of(context),
        textScaler: MediaQuery.textScalerOf(context),
      )..layout();
      if (painter.width + 24 > width) width = painter.width + 24;
      painter.dispose();
    }
    // Longer labels wrap without taking the compact content's usable width.
    return width.clamp(72, 160);
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return ListView(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 12),
      children: [
        for (var i = 0; i < items.length; i++)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: MergeSemantics(
              key: ValueKey('rail-destination-$i'),
              child: Semantics(
                label: items[i].semanticsLabel,
                button: true,
                enabled: true,
                selected: currentIndex == i,
                onTap: () => onSelected(i),
                child: Tooltip(
                  message: items[i].label,
                  excludeFromSemantics: true,
                  child: ExcludeSemantics(
                    child: TextButton(
                      onPressed: () => onSelected(i),
                      style: TextButton.styleFrom(
                        minimumSize: const Size(48, 48),
                        padding: const EdgeInsets.symmetric(
                          horizontal: 4,
                          vertical: 12,
                        ),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(16),
                        ),
                        foregroundColor: currentIndex == i
                            ? colors.primary
                            : colors.iconMuted,
                        backgroundColor: currentIndex == i
                            ? colors.primary.withValues(alpha: .08)
                            : Colors.transparent,
                      ),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Badge(
                            isLabelVisible: items[i].badge,
                            backgroundColor: colors.primary,
                            child: GfSymbol(
                              currentIndex == i
                                  ? items[i].selectedSymbol
                                  : items[i].symbol,
                              size: 24,
                              color: currentIndex == i
                                  ? colors.primary
                                  : colors.iconMuted,
                            ),
                          ),
                          const SizedBox(height: 6),
                          Text(
                            items[i].label,
                            textAlign: TextAlign.center,
                            style: GfTheme.typographyOf(context).caption
                                .copyWith(
                                  fontWeight: currentIndex == i
                                      ? FontWeight.w600
                                      : FontWeight.w400,
                                ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}

/// Bounds pushed reading and form pages without replacing their route or state.
class ReadingPageWindow extends StatelessWidget {
  const ReadingPageWindow({
    super.key,
    required this.child,
    this.maxContentWidth = 720,
  });

  final Widget child;
  final double maxContentWidth;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return ColoredBox(
      color: colors.base200,
      child: SafeArea(
        top: false,
        bottom: false,
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: BoxConstraints(maxWidth: maxContentWidth),
            child: DecoratedBox(
              decoration: BoxDecoration(
                color: colors.base100,
                border: Border.symmetric(
                  vertical: BorderSide(color: colors.line),
                ),
              ),
              child: LayoutBuilder(
                builder: (context, constraints) => MediaQuery(
                  // Page layout uses the column, while dialogs still open in
                  // the full root overlay with its original system insets.
                  data: MediaQuery.of(context).copyWith(
                    size: Size(constraints.maxWidth, constraints.maxHeight),
                  ),
                  child: child,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Window geometry only; the retained branch navigator never changes parents
/// when the window crosses a breakpoint or a software keyboard opens.
class ReadingWindow extends StatelessWidget {
  const ReadingWindow({
    super.key,
    required this.child,
    required this.rail,
    required this.bottomNavigation,
    this.maxContentWidth = 720,
    this.navigationRailWidth = 72,
  });

  final Widget child;
  final Widget rail;
  final Widget bottomNavigation;
  final double maxContentWidth;
  final double navigationRailWidth;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final wide = constraints.maxWidth >= 600;
      final colors = GfTheme.colorsOf(context);
      final railWidth = wide
          ? navigationRailWidth + MediaQuery.paddingOf(context).left
          : 0.0;
      return ReadingWindowScope(
        hasRail: wide,
        navigationWidth: constraints.maxWidth,
        child: ColoredBox(
          color: colors.base200,
          // The route comes before persistent controls in paint/semantics
          // order, as it does for the compact bottom bar. Otherwise the active
          // Navigator route can exclude an earlier rail from accessibility.
          child: Stack(
            children: [
              Positioned.fill(
                child: Padding(
                  padding: EdgeInsets.only(left: railWidth),
                  child: Align(
                    alignment: Alignment.topCenter,
                    child: Container(
                      constraints: BoxConstraints(maxWidth: maxContentWidth),
                      decoration: BoxDecoration(
                        color: colors.base100,
                        border: wide
                            ? Border.symmetric(
                                vertical: BorderSide(color: colors.line),
                              )
                            : null,
                      ),
                      child: Stack(
                        children: [
                          Positioned.fill(
                            child: MediaQuery.removePadding(
                              context: context,
                              removeLeft: wide,
                              child: LayoutBuilder(
                                builder: (context, contentConstraints) =>
                                    MediaQuery(
                                      data: MediaQuery.of(context).copyWith(
                                        size: Size(
                                          contentConstraints.maxWidth,
                                          contentConstraints.maxHeight,
                                        ),
                                      ),
                                      child: child,
                                    ),
                              ),
                            ),
                          ),
                          if (!wide)
                            Positioned(
                              left: 0,
                              right: 0,
                              bottom: 0,
                              child: bottomNavigation,
                            ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
              if (wide)
                Positioned(
                  left: 0,
                  top: 0,
                  bottom: 0,
                  width: railWidth,
                  child: ColoredBox(
                    color: colors.base100,
                    child: NotificationListener<ScrollNotification>(
                      // Scrolling a short rail is navigation, not reading
                      // progress. Keep it out of the shell chrome listener.
                      onNotification: (_) => true,
                      child: SafeArea(right: false, child: rail),
                    ),
                  ),
                ),
            ],
          ),
        ),
      );
    },
  );
}

class ReadingWindowScope extends InheritedWidget {
  const ReadingWindowScope({
    super.key,
    required this.hasRail,
    required this.navigationWidth,
    required super.child,
  });
  final bool hasRail;
  final double navigationWidth;
  static bool hasRailOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<ReadingWindowScope>()
          ?.hasRail ??
      false;
  static double navigationWidthOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<ReadingWindowScope>()
          ?.navigationWidth ??
      MediaQuery.sizeOf(context).width;
  @override
  bool updateShouldNotify(ReadingWindowScope oldWidget) =>
      hasRail != oldWidget.hasRail ||
      navigationWidth != oldWidget.navigationWidth;
}
