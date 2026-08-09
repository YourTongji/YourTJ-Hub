import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../theme/gf_theme.dart';

class GfBottomNavigationItem {
  const GfBottomNavigationItem({
    required this.label,
    required this.icon,
    required this.selectedIcon,
    this.badge = false,
  });

  final String label;
  final IconData icon;
  final IconData selectedIcon;
  final bool badge;
}

/// Bottom navigation backed by TDesign, with YourTJ colors and unread dots.
class GfBottomNavigation extends StatelessWidget {
  const GfBottomNavigation({
    super.key,
    required this.currentIndex,
    required this.items,
    required this.onSelected,
  });

  final int currentIndex;
  final List<GfBottomNavigationItem> items;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return td.TTabBar(
      variant: td.TTabBarVariant.weakIconText,
      value: currentIndex,
      onChanged: onSelected,
      useSafeArea: true,
      placeholder: false,
      barHeight: 60,
      showTopBorder: true,
      topBorder: BorderSide(color: colors.line),
      backgroundColor: colors.base100,
      needInkWell: true,
      navigationTabs: <td.TTabBarItemConfig>[
        for (int index = 0; index < items.length; index++)
          td.TTabBarItemConfig(
            onTap: () {},
            tabText: items[index].label,
            selectedIcon: Icon(
              items[index].selectedIcon,
              size: 24,
              color: colors.primary,
            ),
            unselectedIcon: Icon(
              items[index].icon,
              size: 24,
              color: colors.iconMuted,
            ),
            badgeConfig: td.TTabBarBadgeConfig(showBadge: items[index].badge),
          ),
      ],
    );
  }
}
