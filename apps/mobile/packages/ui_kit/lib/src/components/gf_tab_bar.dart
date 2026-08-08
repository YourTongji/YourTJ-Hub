import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

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
    final List<Widget> children = <Widget>[
      for (final GfTab tab in tabs)
        Padding(
          padding: const EdgeInsetsDirectional.only(end: 8),
          child: td.TSelectTag(
            tab.label,
            value: tab.value == selected,
            size: td.TTagSize.large,
            colorScheme: td.TTagColorScheme.primary,
            onChanged: (_) => onSelected(tab.value),
          ),
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
