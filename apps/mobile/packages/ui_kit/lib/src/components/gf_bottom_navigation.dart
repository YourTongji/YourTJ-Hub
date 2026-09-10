import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_symbol.dart';

class GfBottomNavigationItem {
  const GfBottomNavigationItem({
    required this.label,
    required this.icon,
    required this.selectedIcon,
    this.badge = false,
    this.symbol,
  });

  final String label;
  final IconData icon;
  final IconData selectedIcon;
  final bool badge;
  final String? symbol;
}

/// Four accessible navigation destinations. The unified mobile shell uses
/// icon-only items; labels and an optional compose slot remain available to
/// other callers of this shared component.
class GfBottomNavigation extends StatelessWidget {
  const GfBottomNavigation({
    super.key,
    required this.currentIndex,
    required this.items,
    required this.onSelected,
    this.onAction,
    this.actionLabel = '发布',
    this.actionIcon = Icons.add,
    this.showLabels = true,
  });

  final int currentIndex;
  final List<GfBottomNavigationItem> items;
  final ValueChanged<int> onSelected;
  final VoidCallback? onAction;
  final String actionLabel;
  final IconData actionIcon;
  final bool showLabels;

  @override
  Widget build(BuildContext context) {
    if (items.length != 4) {
      throw FlutterError(
        'GfBottomNavigation requires exactly four destinations; '
        'received ${items.length}.',
      );
    }
    final GfColors colors = GfTheme.colorsOf(context);

    return Material(
      color: colors.base100,
      child: SafeArea(
        top: false,
        child: Container(
          height: showLabels ? 72 : 56,
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
                  icon: actionIcon,
                  label: actionLabel,
                  onTap: onAction,
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
    final Color foreground = colors.baseContent;

    return Expanded(
      child: Semantics(
        button: true,
        selected: selected,
        label: item.label,
        child: InkWell(
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(4, 8, 4, 6),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: <Widget>[
                SizedBox(
                  width: 28,
                  height: 28,
                  child: Stack(
                    clipBehavior: Clip.none,
                    children: <Widget>[
                      Center(
                        child: item.symbol != null
                            ? GfSymbol(
                                item.symbol!,
                                size: 26,
                                color: foreground,
                              )
                            : Icon(
                                selected ? item.selectedIcon : item.icon,
                                size: 24,
                                color: foreground,
                              ),
                      ),
                      if (item.badge)
                        Positioned(
                          top: 0,
                          right: 0,
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
                if (!showLabel)
                  Container(
                    width: selected ? 4 : 0,
                    height: 3,
                    decoration: BoxDecoration(
                      color: foreground,
                      borderRadius: BorderRadius.circular(2),
                    ),
                  ),
                if (showLabel) const SizedBox(height: 2),
                if (showLabel)
                  Text(
                    item.label,
                    maxLines: 1,
                    overflow: TextOverflow.fade,
                    style: GfTheme.typographyOf(context).meta.copyWith(
                      color: foreground,
                      fontWeight: selected ? FontWeight.w600 : FontWeight.w500,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _ComposeAction extends StatelessWidget {
  const _ComposeAction({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Expanded(
      child: Semantics(
        button: true,
        label: label,
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
                    child: Icon(icon, size: 25, color: colors.primaryContent),
                  ),
                ),
                const SizedBox(height: 1),
                Text(
                  label,
                  maxLines: 1,
                  style: GfTheme.typographyOf(context).meta.copyWith(
                    color: colors.primary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
