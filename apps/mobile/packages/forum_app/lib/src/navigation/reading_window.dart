import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

/// Uses ordinary icon buttons so native accessibility exposes the same named,
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
                label: items[i].label,
                button: true,
                enabled: true,
                selected: currentIndex == i,
                child: Tooltip(
                  message: items[i].label,
                  excludeFromSemantics: true,
                  child: IconButton(
                    onPressed: () => onSelected(i),
                    style: IconButton.styleFrom(
                      minimumSize: const Size(48, 48),
                      foregroundColor: currentIndex == i
                          ? colors.primary
                          : colors.iconMuted,
                      backgroundColor: currentIndex == i
                          ? colors.primary.withValues(alpha: .08)
                          : Colors.transparent,
                    ),
                    icon: Badge(
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
                  ),
                ),
              ),
            ),
          ),
      ],
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
  });

  final Widget child;
  final Widget rail;
  final Widget bottomNavigation;
  final double maxContentWidth;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final wide = constraints.maxWidth >= 600;
      final colors = GfTheme.colorsOf(context);
      final railWidth = wide ? 72 + MediaQuery.paddingOf(context).left : 0.0;
      return ReadingWindowScope(
        hasRail: wide,
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
                              child: child,
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
    required super.child,
  });
  final bool hasRail;
  static bool hasRailOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<ReadingWindowScope>()
          ?.hasRail ??
      false;
  @override
  bool updateShouldNotify(ReadingWindowScope oldWidget) =>
      hasRail != oldWidget.hasRail;
}
