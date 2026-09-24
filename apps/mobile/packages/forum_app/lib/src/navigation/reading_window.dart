import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

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
      return ReadingWindowScope(
        hasRail: wide,
        child: ColoredBox(
          color: colors.base200,
          child: Row(
            children: [
              SizedBox(
                width: wide ? 72 + MediaQuery.paddingOf(context).left : 0,
                child: wide
                    ? ColoredBox(
                        color: colors.base100,
                        child: SafeArea(right: false, child: rail),
                      )
                    : null,
              ),
              Expanded(
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
