import 'dart:math' as math;

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
    final duration = GfMotion.duration(context, GfMotion.layout);
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
    );
    final navMetrics = GfBottomNavigation.metrics(
      safeAreaBottom: bottom,
      barHeight: navigationHeight,
    );
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
                        pageBuilder: (index, chromeHidden) {
                          final top = chromeHidden ? 0.0 : 56 + toolbarHeight;
                          final pageBottom = hasRail
                              ? 24.0 + bottom
                              : chromeHidden
                              ? bottom
                              : navMetrics.contentBottomInset;
                          return index == swipeTabIndex
                              ? body(top, pageBottom)
                              : swipePageBuilder!(index, top, pageBottom);
                        },
                      )
                    : body(
                        56 + toolbarHeight,
                        hasRail ? 24.0 + bottom : navMetrics.contentBottomInset,
                      ),
              ),
              Positioned(
                top: 0,
                left: 0,
                right: 0,
                child: AnimatedSlide(
                  offset: hidden ? const Offset(0, -1) : Offset.zero,
                  duration: duration,
                  curve: GfMotion.layoutCurve,
                  child: IgnorePointer(
                    ignoring: hidden,
                    child: ExcludeSemantics(
                      excluding: hidden,
                      child: GfLiquidSurface(
                        radius: 0,
                        weight: GfGlassWeight.strong,
                        elevated: false,
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            SizedBox(
                              height: 56,
                              child: Row(
                                children: [
                                  const SizedBox(width: 8),
                                  GfLiquidSurface(
                                    radius: 26,
                                    elevated: false,
                                    child: IconButton(
                                      tooltip: AppLocalizations.of(
                                        context,
                                      ).navProfile,
                                      onPressed: () => accountDrawerLayerKey
                                          .currentState
                                          ?.open(),
                                      icon: const AccountAvatar(),
                                    ),
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
                                    GfLiquidSurface(
                                      radius: 26,
                                      elevated: false,
                                      pressable: true,
                                      child: Row(
                                        mainAxisSize: MainAxisSize.min,
                                        children: actions,
                                      ),
                                    ),
                                  const SizedBox(width: 8),
                                ],
                              ),
                            ),
                            if (toolbar != null)
                              SizedBox(height: toolbarHeight, child: toolbar),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ),
              if (showComposeAction)
                AnimatedPositioned(
                  duration: duration,
                  curve: GfMotion.layoutCurve,
                  right: 16,
                  bottom:
                      (hidden || hasRail ? 16 : navMetrics.actionBottomInset) +
                      bottom,
                  child: GfLiquidSurface(
                    radius: 30,
                    tint: colors.primary,
                    weight: GfGlassWeight.strong,
                    pressable: true,
                    child: FloatingActionButton(
                      backgroundColor: Colors.transparent,
                      elevation: 0,
                      highlightElevation: 0,
                      heroTag: null,
                      tooltip:
                          actionLabel ??
                          AppLocalizations.of(context).navPublish,
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
