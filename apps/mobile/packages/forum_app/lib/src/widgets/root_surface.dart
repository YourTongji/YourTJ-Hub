import 'dart:math' as math;
import 'dart:ui' show lerpDouble;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../navigation/reading_chrome.dart';
import '../navigation/reading_window.dart';
import '../navigation/tab_scroll_registry.dart';
import '../navigation/tab_swipe_surface.dart';
import '../navigation/tab_page_transition.dart';
import 'account_drawer.dart';
import 'compose_menu.dart';

/// Root content scrolls underneath overlay chrome; swipe tabs follow its visibility.
class RootSurface extends ConsumerWidget {
  const RootSurface({
    super.key,
    this.title = '',
    this.titleWidget,
    this.showLogo = false,
    required this.body,
    this.actions = const [],
    this.toolbar,
    this.toolbarHeight = 0,
    this.onAction,
    this.showComposeAction = true,
    this.actionLabel,
    this.actionSymbol = 'plus',
    this.swipeTabIndex,
    this.swipeTabCount = 0,
    this.onSwipeTabChanged,
    this.swipePageBuilder,
    this.swipePageKey,
  });
  final String title;
  final Widget? titleWidget;
  final bool showLogo;
  final Widget Function(double topInset, double bottomInset) body;
  final List<Widget> actions;
  final Widget? toolbar;
  final double toolbarHeight;
  final VoidCallback? onAction;
  final bool showComposeAction;
  final String? actionLabel;
  final String actionSymbol;
  final int? swipeTabIndex;
  final int swipeTabCount;
  final ValueChanged<int>? onSwipeTabChanged;
  final Widget Function(int index, double topInset, double bottomInset)?
  swipePageBuilder;
  final Object Function(int index)? swipePageKey;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final hidden = ref.watch(readingChromeProvider).hidden;
    final colors = GfTheme.colorsOf(context);
    final bottom = MediaQuery.paddingOf(context).bottom;
    final hasRail = ReadingWindowScope.hasRailOf(context);
    final l10n = AppLocalizations.of(context);
    // Mirror the labels of the shell's own bottom bar so the precomputed
    // content insets stay in lockstep with the measured bar height.
    final navigationHeight = GfBottomNavigation.heightFor(
      context,
      labels: GfShellDestination.values.map(
        (destination) => destination.label(l10n),
      ),
      availableWidth: math.max(
        1,
        ReadingWindowScope.navigationWidthOf(context) -
            MediaQuery.paddingOf(context).horizontal,
      ),
      showLabels: shellNavigationShowsLabels,
    );
    final navMetrics = GfBottomNavigation.metrics(
      safeAreaBottom: bottom,
      barHeight: navigationHeight,
    );
    // Insets stay fixed while chrome slides, so content never jumps.
    final top = 56 + toolbarHeight;
    final contentBottom = hasRail
        ? 24.0 + bottom
        : navMetrics.contentBottomInset;
    final surface = Scaffold(
      body: SafeArea(
        bottom: false,
        child: ClipRect(
          child: Stack(
            children: [
              Positioned.fill(
                child:
                    swipeTabIndex != null &&
                        swipeTabCount > 1 &&
                        swipePageBuilder != null
                    ? TabPageTransition(
                        index: swipeTabIndex!,
                        length: swipeTabCount,
                        chromeHidden: hidden,
                        pageKey: swipePageKey,
                        pageBuilder: (index, chromeHidden) => ChromeAlignedPage(
                          topInset: top,
                          chromeHidden: chromeHidden,
                          current: index == swipeTabIndex,
                          child: index == swipeTabIndex
                              ? body(top, contentBottom)
                              : swipePageBuilder!(index, top, contentBottom),
                        ),
                      )
                    : body(top, contentBottom),
              ),
              Positioned(
                top: 0,
                left: 0,
                right: 0,
                child: ReadingChromeSlide(
                  direction: -1,
                  child: IgnorePointer(
                    ignoring: hidden,
                    child: ExcludeSemantics(
                      excluding: hidden,
                      child: ColoredBox(
                        color: colors.base100,
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            SizedBox(
                              height: 56,
                              child: Row(
                                children: [
                                  const SizedBox(width: 8),
                                  IconButton(
                                    tooltip: AppLocalizations.of(
                                      context,
                                    ).navProfile,
                                    onPressed: () => accountDrawerLayerKey
                                        .currentState
                                        ?.open(),
                                    icon: const AccountAvatar(),
                                  ),
                                  Expanded(
                                    child: Center(
                                      child:
                                          titleWidget ??
                                          (showLogo || title.isEmpty
                                              ? const GfLogo(size: 32)
                                              : Text(
                                                  title,
                                                  maxLines: 1,
                                                  overflow:
                                                      TextOverflow.ellipsis,
                                                  style: GfTheme.typographyOf(
                                                    context,
                                                  ).title2,
                                                )),
                                    ),
                                  ),
                                  if (actions.isEmpty)
                                    const SizedBox(width: 48)
                                  else
                                    ...actions,
                                  const SizedBox(width: 8),
                                ],
                              ),
                            ),
                            if (toolbar != null)
                              SizedBox(height: toolbarHeight, child: toolbar),
                            const Divider(height: 1, thickness: 0),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ),
              if (showComposeAction)
                ValueListenableBuilder<ChromeReveal>(
                  valueListenable: ref.watch(readingChromeProvider).reveal,
                  builder: (context, reveal, child) => AnimatedPositioned(
                    duration: readingChromeDuration(context, reveal),
                    curve: GfMotion.enterCurve,
                    right: 16,
                    bottom:
                        (hasRail
                            ? 16
                            : lerpDouble(
                                16,
                                navMetrics.actionBottomInset,
                                reveal.value,
                              )!) +
                        bottom,
                    child: child!,
                  ),
                  child: FloatingActionButton(
                    heroTag: null,
                    tooltip:
                        actionLabel ?? AppLocalizations.of(context).navPublish,
                    onPressed:
                        onAction ??
                        () => showComposeMenu(
                          context,
                          bottom: hidden || hasRail
                              ? 16
                              : navMetrics.actionBottomInset,
                        ),
                    child: GfSymbol(
                      actionSymbol,
                      color: colors.primaryContent,
                      size: 28,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
    return swipeTabIndex != null &&
            swipeTabCount > 1 &&
            onSwipeTabChanged != null
        ? TabSwipeSurface(
            index: swipeTabIndex!,
            length: swipeTabCount,
            onChanged: onSwipeTabChanged!,
            child: surface,
          )
        : surface;
  }
}
