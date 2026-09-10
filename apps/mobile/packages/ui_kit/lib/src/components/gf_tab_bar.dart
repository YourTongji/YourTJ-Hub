import 'package:flutter/material.dart';
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
            constraints: const BoxConstraints(minHeight: 44),
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const SizedBox(height: 10),
                Text(
                  tab.label,
                  style: TextStyle(
                    fontSize: 14,
                    height: 22 / 14,
                    color: active ? colors.baseContent : colors.iconMuted,
                    fontWeight: active ? FontWeight.w700 : FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 8),
                Container(
                  width: 28,
                  height: 3,
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
