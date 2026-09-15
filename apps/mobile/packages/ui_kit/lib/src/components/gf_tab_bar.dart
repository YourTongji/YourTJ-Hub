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
    Widget item(GfTab tab) {
      final active = tab.value == selected;
      return Semantics(
        selected: active,
        button: true,
        child: InkWell(
          onTap: () => onSelected(tab.value),
          child: Container(
            height: heightFor(context),
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const SizedBox(height: 12),
                Text(
                  tab.label,
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.25,
                    color: active ? colors.baseContent : colors.iconMuted,
                    fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                  ),
                ),
                const Spacer(),
                Container(
                  width: 28,
                  height: 4,
                  decoration: BoxDecoration(
                    color: active ? colors.primary : Colors.transparent,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }

    if (!mobile) return Wrap(children: tabs.map(item).toList());
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(children: tabs.map(item).toList()),
    );
  }
}
