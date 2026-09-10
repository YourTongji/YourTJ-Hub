import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../navigation/reading_chrome.dart';
import 'account_drawer.dart';

/// Root content scrolls underneath an overlay header. The initial header inset
/// belongs inside the scroll view, so hiding controls cannot jump the content.
class RootSurface extends ConsumerWidget {
  const RootSurface({
    super.key,
    required this.title,
    required this.body,
    this.actions = const [],
    this.toolbar,
    this.toolbarHeight = 0,
    this.onAction,
    this.actionLabel,
    this.actionSymbol = 'plus',
  });
  final String title;
  final Widget Function(double topInset, double bottomInset) body;
  final List<Widget> actions;
  final Widget? toolbar;
  final double toolbarHeight;
  final VoidCallback? onAction;
  final String? actionLabel;
  final String actionSymbol;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final hidden = ref.watch(readingChromeProvider).hidden;
    final colors = GfTheme.colorsOf(context);
    final duration = MediaQuery.disableAnimationsOf(context)
        ? Duration.zero
        : const Duration(milliseconds: 200);
    final bottom = MediaQuery.paddingOf(context).bottom;
    return Scaffold(
      body: SafeArea(
        bottom: false,
        child: ClipRect(
          child: Stack(
            children: [
              Positioned.fill(child: body(56 + toolbarHeight, 80 + bottom)),
              Positioned(
                top: 0,
                left: 0,
                right: 0,
                child: AnimatedSlide(
                  offset: hidden ? const Offset(0, -1) : Offset.zero,
                  duration: duration,
                  curve: Curves.easeOut,
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
                                    onPressed: () =>
                                        Scaffold.of(context).openDrawer(),
                                    icon: const AccountAvatar(),
                                  ),
                                  Expanded(
                                    child: Center(
                                      child: Text(
                                        title,
                                        style: GfTheme.typographyOf(
                                          context,
                                        ).title2,
                                      ),
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
                            const Divider(height: 1),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ),
              AnimatedPositioned(
                duration: duration,
                curve: Curves.easeOut,
                right: 16,
                bottom: (hidden ? 16 : 72) + bottom,
                child: FloatingActionButton(
                  heroTag: null,
                  tooltip:
                      actionLabel ?? AppLocalizations.of(context).navPublish,
                  onPressed: onAction ?? () => context.push('/publish?type=2'),
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
  }
}
