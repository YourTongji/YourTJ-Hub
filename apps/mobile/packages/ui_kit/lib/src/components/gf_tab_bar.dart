import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';

/// A single selectable tab in [GfTabBar].
class GfTab {
  const GfTab({required this.label, required this.value});

  final String label;
  final Object value;
}

/// Scrollable tab bar mirroring web `.gf-tab` semantics.
///
/// On mobile the bar scrolls horizontally when tabs overflow; the active tab
/// renders with the neutral fill (`gf-tab-active`), idle tabs with muted
/// text.
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
  /// tabs wrap, matching the desktop pill layout.
  final bool mobile;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    final List<Widget> children = <Widget>[
      for (final GfTab tab in tabs)
        _GfTabItem(
          tab: tab,
          active: tab.value == selected,
          colors: colors,
          radii: radii,
          onTap: () => onSelected(tab.value),
        ),
    ];

    if (mobile) {
      return SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(children: children),
      );
    }
    return Wrap(spacing: 8, children: children);
  }
}

class _GfTabItem extends StatelessWidget {
  const _GfTabItem({
    required this.tab,
    required this.active,
    required this.colors,
    required this.radii,
    required this.onTap,
  });

  final GfTab tab;
  final bool active;
  final GfColors colors;
  final GfRadii radii;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final Color foreground = active
        ? colors.neutralContent
        : colors.baseContent.withValues(alpha: 0.55);

    return AnimatedContainer(
      duration: GfMotion.instant,
      curve: GfMotion.standardEase,
      height: 32,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: active ? colors.neutral : Colors.transparent,
        borderRadius: BorderRadius.circular(radii.field),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(radii.field),
        child: Text(
          tab.label,
          style: TextStyle(
            color: foreground,
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}
