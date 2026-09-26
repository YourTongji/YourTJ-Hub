import 'package:flutter/material.dart';
import 'dart:math' as math;
import '../theme/gf_theme.dart';

/// A single selectable tab in [GfTabBar].
class GfTab {
  const GfTab({required this.label, required this.value});

  final String label;
  final Object value;
}

/// Scrollable tab bar mirroring web `.gf-tab` semantics.
///
/// On mobile the bar scrolls horizontally when tabs overflow; the active tab
/// renders with a short brand underline; idle tabs use muted text.
class GfTabBar extends StatelessWidget {
  /// Use this for overlay toolbars so their content inset grows with text.
  static double heightFor(BuildContext context) =>
      math.max(48, MediaQuery.textScalerOf(context).scale(16) * 1.25 + 28);
  const GfTabBar({
    super.key,
    required this.tabs,
    required this.selected,
    required this.onSelected,
    this.mobile = true,
  });

  final List<GfTab> tabs;
  final Object selected;
  final ValueChanged<Object> onSelected;

  /// When true (default) the bar scrolls horizontally on overflow; when false
  /// tabs wrap.
  final bool mobile;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    const duration = Duration(milliseconds: 220);
    const curve = Curves.easeInOutCubic;
    final noAnimation = MediaQuery.disableAnimationsOf(context);
    final height = heightFor(context);
    final selectedIndex = tabs.indexWhere((tab) => tab.value == selected);
    final measureStyle = DefaultTextStyle.of(
      context,
    ).style.copyWith(fontSize: 16, height: 1.25, fontWeight: FontWeight.w600);
    final itemWidths = <double>[];
    for (final tab in tabs) {
      final painter = TextPainter(
        text: TextSpan(text: tab.label, style: measureStyle),
        maxLines: 1,
        textDirection: Directionality.of(context),
        textScaler: MediaQuery.textScalerOf(context),
      )..layout();
      itemWidths.add(painter.width + 32);
      painter.dispose();
    }
    var indicatorLeft = 0.0;
    if (selectedIndex >= 0) {
      for (var i = 0; i < selectedIndex; i++) {
        indicatorLeft += itemWidths[i];
      }
      indicatorLeft += (itemWidths[selectedIndex] - 28) / 2;
    }

    Widget item(int index, {required bool wrapIndicator}) {
      final active = index == selectedIndex;
      return Semantics(
        selected: active,
        button: true,
        child: InkWell(
          onTap: () => onSelected(tabs[index].value),
          child: SizedBox(
            width: itemWidths[index],
            height: height,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const SizedBox(height: 12),
                AnimatedDefaultTextStyle(
                  duration: noAnimation ? Duration.zero : duration,
                  curve: curve,
                  style: measureStyle.copyWith(
                    color: active ? colors.baseContent : colors.iconMuted,
                    fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                  ),
                  child: Text(tabs[index].label, maxLines: 1, softWrap: false),
                ),
                const Spacer(),
                if (wrapIndicator)
                  AnimatedContainer(
                    duration: noAnimation ? Duration.zero : duration,
                    curve: curve,
                    width: 28,
                    height: 4,
                    decoration: BoxDecoration(
                      color: active ? colors.primary : Colors.transparent,
                      borderRadius: BorderRadius.circular(2),
                    ),
                  )
                else
                  const SizedBox(height: 4),
              ],
            ),
          ),
        ),
      );
    }

    if (!mobile) {
      return Wrap(
        children: [
          for (var i = 0; i < tabs.length; i++) item(i, wrapIndicator: true),
        ],
      );
    }

    final row = SizedBox(
      height: height,
      child: Stack(
        children: [
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              for (var i = 0; i < tabs.length; i++)
                item(i, wrapIndicator: false),
            ],
          ),
          if (selectedIndex >= 0)
            AnimatedPositionedDirectional(
              key: const ValueKey('gf-tab-indicator'),
              duration: noAnimation ? Duration.zero : duration,
              curve: curve,
              start: indicatorLeft,
              bottom: 0,
              width: 28,
              height: 4,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: colors.primary,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
        ],
      ),
    );

    return SingleChildScrollView(scrollDirection: Axis.horizontal, child: row);
  }
}
