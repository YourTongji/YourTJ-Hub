import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../navigation/tab_scroll_registry.dart';

/// The real navigation over a few rows of content, above the choice that
/// drives it, so changing style shows the change itself rather than a name.
class NavigationStylePicker extends StatelessWidget {
  const NavigationStylePicker({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final GfNavigationStyle value;
  final ValueChanged<GfNavigationStyle> onChanged;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Padding(
      padding: const EdgeInsets.all(GfSpace.md),
      child: Column(
        children: <Widget>[
          _Preview(style: value),
          const SizedBox(height: GfSpace.md),
          GfSegmented<GfNavigationStyle>(
            key: const ValueKey('settings-navigation-style'),
            segments: <(String, GfNavigationStyle)>[
              (l10n.settingsNavigationDock, GfNavigationStyle.dock),
              (l10n.settingsNavigationBar, GfNavigationStyle.bar),
            ],
            selected: value,
            onSelected: onChanged,
          ),
        ],
      ),
    );
  }
}

class _Preview extends StatelessWidget {
  const _Preview({required this.style});

  final GfNavigationStyle style;

  // Room for a home indicator under the navigation.
  static const double _inset = 14;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    return IgnorePointer(
      child: ExcludeSemantics(
        child: ClipRRect(
          key: const ValueKey('settings-navigation-preview'),
          borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
          child: ColoredBox(
            color: colors.canvas,
            child: SizedBox(
              height: 152,
              child: MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(padding: const EdgeInsets.only(bottom: _inset)),
                child: Stack(
                  children: <Widget>[
                    const Positioned.fill(child: _Content()),
                    Positioned(
                      left: 0,
                      right: 0,
                      bottom: 0,
                      child: GfBottomNavigation(
                        style: style,
                        currentIndex: 0,
                        onSelected: (_) {},
                        showLabels: shellNavigationShowsLabels,
                        items: <GfBottomNavigationItem>[
                          for (final GfShellDestination destination
                              in GfShellDestination.values)
                            GfBottomNavigationItem(
                              symbol: destination.symbol,
                              selectedSymbol: '${destination.symbol}-filled',
                              label: destination.label(l10n),
                            ),
                        ],
                      ),
                    ),
                    Positioned(
                      left: 0,
                      right: 0,
                      bottom: (_inset - 4) / 2,
                      child: Center(
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            color: colors.baseContent.withValues(alpha: 0.3),
                            borderRadius: BorderRadius.circular(GfRadius.full),
                          ),
                          child: const SizedBox(width: 96, height: 4),
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
    );
  }
}

/// A few feed rows running under the navigation, so its glass has something
/// to frost.
class _Content extends StatelessWidget {
  const _Content();

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Color line = colors.baseContent.withValues(
      alpha: colors.isDark ? 0.14 : 0.09,
    );
    final List<Color> avatars = <Color>[
      colors.primary,
      colors.accent,
      colors.info,
      colors.success,
    ];
    Widget bar(double width) => FractionallySizedBox(
      widthFactor: width,
      alignment: AlignmentDirectional.centerStart,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: line,
          borderRadius: BorderRadius.circular(GfRadius.full),
        ),
        child: const SizedBox(height: 8),
      ),
    );
    return OverflowBox(
      alignment: Alignment.topCenter,
      maxHeight: double.infinity,
      child: Padding(
        padding: const EdgeInsets.all(GfSpace.md),
        child: Column(
          children: <Widget>[
            for (int i = 0; i < avatars.length; i++)
              Padding(
                padding: const EdgeInsets.only(bottom: GfSpace.md),
                child: Row(
                  children: <Widget>[
                    DecoratedBox(
                      decoration: BoxDecoration(
                        color: avatars[i].withValues(alpha: 0.7),
                        shape: BoxShape.circle,
                      ),
                      child: const SizedBox.square(dimension: 28),
                    ),
                    const SizedBox(width: GfSpace.sm),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          bar(0.42 + 0.12 * (i % 2)),
                          const SizedBox(height: GfSpace.xs),
                          bar(0.86 - 0.1 * (i % 3)),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}
