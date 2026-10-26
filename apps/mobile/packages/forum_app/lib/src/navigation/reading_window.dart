import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import 'window_reserved_regions.dart';

/// Uses visible labels in ordinary rails and tooltips in a narrow side pocket;
/// accessibility exposes the same named, selectable actions in both layouts.
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
    final textScale = MediaQuery.textScalerOf(context).scale(17) / 17;
    final usableWidth =
        MediaQuery.sizeOf(context).width -
        MediaQuery.paddingOf(context).horizontal;
    if (usableWidth >= 1100 && textScale <= 1.4) {
      return 220;
    }
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
    // Leave room for the floating surface's outer margins without shrinking
    // the localized labels or their touch targets.
    return width.clamp(72, 160) + 16;
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        final edge = ReadingWindowScope.edgeNavigationOf(context);
        final extended = !edge && constraints.maxWidth >= 200;
        return Padding(
          padding: const EdgeInsets.all(8),
          child: Align(
            alignment: edge ? Alignment.bottomCenter : Alignment.topCenter,
            child: SizedBox(
              width: edge ? 56 : null,
              child: Stack(
                children: [
                  const Positioned.fill(
                    child: GfNavigationSurface(
                      borderRadius: BorderRadius.all(Radius.circular(28)),
                    ),
                  ),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(28),
                    child: ListView(
                      shrinkWrap: true,
                      padding: edge
                          ? const EdgeInsets.symmetric(
                              horizontal: 4,
                              vertical: 8,
                            )
                          : const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 12,
                            ),
                      children: [
                        if (extended)
                          const Padding(
                            padding: EdgeInsets.fromLTRB(16, 8, 16, 28),
                            child: Align(
                              alignment: Alignment.centerLeft,
                              child: GfLogo(size: 36),
                            ),
                          ),
                        for (var i = 0; i < items.length; i++)
                          Padding(
                            padding: EdgeInsets.only(bottom: edge ? 0 : 8),
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
                                        padding: EdgeInsets.symmetric(
                                          horizontal: extended ? 16 : 4,
                                          vertical: edge
                                              ? 0
                                              : (extended ? 16 : 12),
                                        ),
                                        shape: RoundedRectangleBorder(
                                          borderRadius: BorderRadius.circular(
                                            GfRadius.lg,
                                          ),
                                        ),
                                        foregroundColor: currentIndex == i
                                            ? colors.primary
                                            : colors.iconMuted,
                                        backgroundColor: currentIndex == i
                                            ? colors.primary.withValues(
                                                alpha: .08,
                                              )
                                            : Colors.transparent,
                                      ),
                                      child: Flex(
                                        direction: extended
                                            ? Axis.horizontal
                                            : Axis.vertical,
                                        mainAxisSize: MainAxisSize.min,
                                        children: [
                                          Badge(
                                            isLabelVisible: items[i].badge,
                                            backgroundColor: colors.primary,
                                            child: GfSymbol(
                                              currentIndex == i
                                                  ? items[i].selectedSymbol
                                                  : items[i].symbol,
                                              size: extended ? 26 : 24,
                                              color: currentIndex == i
                                                  ? colors.primary
                                                  : colors.iconMuted,
                                            ),
                                          ),
                                          if (!edge)
                                            SizedBox(
                                              width: extended ? 16 : 0,
                                              height: extended ? 0 : 6,
                                            ),
                                          if (!edge)
                                            Flexible(
                                              fit: extended
                                                  ? FlexFit.tight
                                                  : FlexFit.loose,
                                              child: Text(
                                                items[i].label,
                                                textAlign: extended
                                                    ? TextAlign.start
                                                    : TextAlign.center,
                                                style:
                                                    (extended
                                                            ? GfTheme.typographyOf(
                                                                context,
                                                              ).title3
                                                            : GfTheme.typographyOf(
                                                                context,
                                                              ).caption)
                                                        .copyWith(
                                                          fontWeight:
                                                              currentIndex == i
                                                              ? FontWeight.w700
                                                              : FontWeight.w400,
                                                        ),
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
                    ),
                  ),
                ],
              ),
            ),
          ),
        );
      },
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
    this.reservedRegions,
    required this.child,
    required this.rail,
    required this.bottomNavigation,
    this.maxContentWidth = 720,
    this.navigationRailWidth = 72,
  });

  final WindowReservedRegions? reservedRegions;
  final Widget child;
  final Widget rail;
  final Widget bottomNavigation;
  final double maxContentWidth;
  final double navigationRailWidth;

  @override
  Widget build(BuildContext context) => ColoredBox(
    color: GfTheme.colorsOf(context).base200,
    child: LayoutBuilder(
      builder: (context, constraints) {
        final media = MediaQuery.of(context);
        final available = constraints.maxWidth - media.padding.horizontal;
        final wide = available >= 600;
        final edgeFrame = reservedRegions?.navigationFrame(
          media,
          constraints.biggest,
          // The root router fills the window, or its bottom-aligned remainder
          // below PreviewStatusHost. Translate native frames past that notice.
          topInset: media.size.height - constraints.maxHeight,
        );
        final hasRail = edgeFrame != null || wide;
        final colors = GfTheme.colorsOf(context);
        final railWidth = wide && edgeFrame == null ? navigationRailWidth : 0.0;
        final outerMargin = wide
            ? ((available - railWidth - 1120) / 2).clamp(0.0, double.infinity)
            : 0.0;
        return ReadingWindowScope(
          hasRail: hasRail,
          edgeNavigation: edgeFrame != null,
          edgeContentBottomInset: edgeFrame == null
              ? 0
              : constraints.maxHeight -
                    edgeFrame.bottom +
                    WindowReservedRegions.navigationClearance,
          navigationWidth: available,
          // Keep the same retained route subtree for every navigation placement.
          child: Stack(
            children: [
              Positioned.fill(
                child: Padding(
                  padding: EdgeInsets.only(
                    left:
                        (edgeFrame == null ? media.padding.left : 0) +
                        outerMargin +
                        railWidth,
                    right:
                        (edgeFrame == null ? media.padding.right : 0) +
                        outerMargin,
                  ),
                  child: MediaQuery.removePadding(
                    context: context,
                    removeLeft: edgeFrame == null,
                    removeRight: edgeFrame == null,
                    child: Align(
                      alignment: Alignment.topLeft,
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
                            if (!hasRail)
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
              ),
              if (hasRail)
                Positioned.fromRect(
                  rect:
                      edgeFrame ??
                      Rect.fromLTWH(
                        media.padding.left + outerMargin,
                        0,
                        railWidth,
                        constraints.maxHeight,
                      ),
                  child: NotificationListener<ScrollNotification>(
                    onNotification: (_) => true,
                    child: MediaQuery.removePadding(
                      context: context,
                      removeLeft: true,
                      removeRight: true,
                      removeTop: edgeFrame != null,
                      removeBottom: edgeFrame != null,
                      child: SafeArea(child: rail),
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    ),
  );
}

class ReadingWindowScope extends InheritedWidget {
  const ReadingWindowScope({
    super.key,
    required this.hasRail,
    required this.navigationWidth,
    this.edgeNavigation = false,
    this.edgeContentBottomInset = 0,
    required super.child,
  });
  final bool hasRail;
  final bool edgeNavigation;
  final double edgeContentBottomInset;
  static double edgeContentBottomInsetOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<ReadingWindowScope>()
          ?.edgeContentBottomInset ??
      0;
  static bool edgeNavigationOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<ReadingWindowScope>()
          ?.edgeNavigation ??
      false;
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
      edgeNavigation != oldWidget.edgeNavigation ||
      edgeContentBottomInset != oldWidget.edgeContentBottomInset ||
      navigationWidth != oldWidget.navigationWidth;
}
