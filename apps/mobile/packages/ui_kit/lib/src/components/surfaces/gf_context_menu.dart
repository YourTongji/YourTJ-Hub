import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';

import '../gf_motion.dart';
import 'gf_menu_surface.dart';
import 'gf_liquid_surface.dart';

/// The caller owns permissions and execution; the surface returns one value.
class GfContextAction<T> {
  const GfContextAction({
    required this.value,
    required this.label,
    this.symbol,
    this.enabled = true,
    this.selected = false,
    this.destructive = false,
  });

  final T value;
  final String label;
  final String? symbol;
  final bool enabled;
  final bool selected;
  final bool destructive;
}

/// An action-only menu anchored to the invoking control. Editors and content
/// sheets keep their own routes; only discrete actions use this surface.
Future<T?> showGfActionMenu<T>(
  BuildContext context, {
  required List<GfContextAction<T>> actions,
  String? semanticLabel,
  Rect? sourceRect,
  ValueListenable<bool>? sourceValid,
  Widget Function(Widget child)? routeWrapper,
  VoidCallback? onClosed,
}) => showGfContextMenu<T>(
  context,
  sourceRect: sourceRect ?? gfMenuSourceRectOf(context),
  semanticLabel:
      semanticLabel ?? MaterialLocalizations.of(context).showMenuTooltip,
  actions: actions,
  sourceValid: sourceValid ?? const AlwaysStoppedAnimation(true),
  routeWrapper: routeWrapper,
  onClosed: onClosed,
);

Rect gfMenuSourceRectOf(BuildContext context, {Offset? globalPosition}) {
  final overlay =
      Navigator.of(
            context,
            rootNavigator: true,
          ).overlay!.context.findRenderObject()!
          as RenderBox;
  if (globalPosition != null) {
    return overlay.globalToLocal(globalPosition) & const Size(1, 1);
  }
  final box = context.findRenderObject()! as RenderBox;
  return box.localToGlobal(Offset.zero, ancestor: overlay) & box.size;
}

class GfActionMenuButton<T> extends StatefulWidget {
  const GfActionMenuButton({
    super.key,
    required this.itemBuilder,
    required this.onSelected,
    required this.icon,
    this.tooltip,
    this.enabled = true,
    this.style,
  });
  final List<GfContextAction<T>> Function(BuildContext) itemBuilder;
  final ValueChanged<T> onSelected;
  final Widget icon;
  final String? tooltip;
  final bool enabled;
  final ButtonStyle? style;

  @override
  State<GfActionMenuButton<T>> createState() => _GfActionMenuButtonState<T>();
}

class _GfActionMenuButtonState<T> extends State<GfActionMenuButton<T>> {
  bool _open = false;
  ValueNotifier<bool>? _validity;

  @override
  void dispose() {
    final validity = _validity;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (identical(validity, _validity)) validity?.value = false;
    });
    super.dispose();
  }

  Future<void> _show() async {
    if (_open || !widget.enabled) return;
    final actions = widget.itemBuilder(context);
    if (actions.isEmpty) return;
    final onSelected = widget.onSelected;
    final validity = ValueNotifier(true);
    _validity = validity;
    setState(() => _open = true);
    try {
      final value = await showGfActionMenu<T>(
        context,
        semanticLabel: widget.tooltip,
        actions: actions,
        sourceValid: validity,
        onClosed: () {
          if (identical(_validity, validity)) _validity = null;
          validity.dispose();
        },
      );
      if (mounted &&
          widget.enabled &&
          value != null &&
          widget
              .itemBuilder(context)
              .any((action) => action.value == value && action.enabled)) {
        onSelected(value);
      }
    } finally {
      if (mounted) setState(() => _open = false);
    }
  }

  @override
  Widget build(BuildContext context) => IconButton(
    icon: widget.icon,
    tooltip: widget.tooltip,
    style: widget.style,
    onPressed: widget.enabled && !_open ? _show : null,
  );
}

/// Keeps the selected object beside its actions. Native modal focus, traversal,
/// back and barrier dismissal remain owned by Navigator. [sourceValid] must
/// remain alive through the exit transition; [onClosed] can release it.
Future<T?> showGfContextMenu<T>(
  BuildContext context, {
  required Rect sourceRect,
  Widget? preview,
  required String semanticLabel,
  required List<GfContextAction<T>> actions,
  required ValueListenable<bool> sourceValid,
  VoidCallback? onClosed,
  Widget Function(Widget child)? routeWrapper,
}) {
  final navigator = Navigator.of(context, rootNavigator: true);
  final route = _ContextMenuRoute<T>(
    sourceRect: sourceRect,
    preview: preview,
    routeWrapper: routeWrapper,
    semanticLabel: semanticLabel,
    actions: List.unmodifiable(actions),
    sourceValid: sourceValid,
    themes: InheritedTheme.capture(from: context, to: navigator.context),
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Theme.of(context).colorScheme.scrim.withValues(alpha: .16),
    reduced: GfMotion.reducedOf(context),
  );
  // Return selection immediately; resource cleanup waits for the exit frames.
  unawaited(route.completed.then((_) => onClosed?.call()));
  return navigator.push<T>(route);
}

class _ContextMenuRoute<T> extends PopupRoute<T> {
  _ContextMenuRoute({
    required this.sourceRect,
    required this.preview,
    required this.routeWrapper,
    required this.semanticLabel,
    required this.actions,
    required this.sourceValid,
    required this.themes,
    required this.barrierLabel,
    required this.barrierColor,
    required this.reduced,
  }) : super(
         requestFocus: true,
         traversalEdgeBehavior: TraversalEdgeBehavior.closedLoop,
       );

  final Rect sourceRect;
  final Widget? preview;
  final Widget Function(Widget child)? routeWrapper;
  final String semanticLabel;
  final List<GfContextAction<T>> actions;
  final ValueListenable<bool> sourceValid;
  final CapturedThemes themes;
  final bool reduced;

  @override
  final Color barrierColor;
  @override
  final String barrierLabel;
  @override
  bool get barrierDismissible => true;
  @override
  Duration get transitionDuration => reduced ? Duration.zero : GfMotion.layout;
  @override
  Duration get reverseTransitionDuration =>
      reduced ? Duration.zero : GfMotion.content;

  @override
  Widget buildModalBarrier() {
    final barrier = super.buildModalBarrier();
    return AnimatedBuilder(
      animation: animation!,
      child: barrier,
      builder: (context, child) => ClipRect(
        child: BackdropFilter(
          key: const Key('gf-context-backdrop'),
          enabled: !GfGlassSettings.opaqueOf(context),
          filter: ui.ImageFilter.blur(
            sigmaX: 12 * GfMotion.enterCurve.transform(animation!.value),
            sigmaY: 12 * GfMotion.enterCurve.transform(animation!.value),
          ),
          // The modal barrier is below the route content. Blur the whole app,
          // including chrome, without filtering the lifted object or actions.
          child: child,
        ),
      ),
    );
  }

  @override
  Widget buildTransitions(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) => child;

  @override
  Widget buildPage(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
  ) {
    final media = MediaQuery.of(context);
    final padding = EdgeInsets.fromLTRB(
      math.max(media.padding.left, media.viewInsets.left) + 12,
      math.max(media.padding.top, media.viewInsets.top) + 12,
      math.max(media.padding.right, media.viewInsets.right) + 12,
      math.max(media.padding.bottom, media.viewInsets.bottom) + 12,
    );
    final content = ValueListenableBuilder<bool>(
      valueListenable: sourceValid,
      builder: (context, valid, _) {
        if (!valid) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (isActive) navigator?.removeRoute(this);
          });
          return const SizedBox.shrink();
        }
        return themes.wrap(
          Shortcuts(
            shortcuts: const {
              SingleActivator(LogicalKeyboardKey.escape): DismissIntent(),
            },
            child: Actions(
              actions: {
                DismissIntent: CallbackAction<DismissIntent>(
                  onInvoke: (_) {
                    navigator?.pop();
                    return null;
                  },
                ),
              },
              child: Semantics(
                scopesRoute: true,
                namesRoute: true,
                explicitChildNodes: true,
                label: semanticLabel,
                child: Padding(
                  padding: padding,
                  child: CustomMultiChildLayout(
                    delegate: _ContextMenuLayout(
                      sourceRect.shift(-padding.topLeft),
                      maxPreviewHeight: math.min(160, media.size.height * .24),
                      animation: animation,
                    ),
                    children: [
                      // Paint the lifted object above the action panel while it
                      // travels through it; semantics still read the object first.
                      LayoutId(
                        id: _ContextPart.actions,
                        child: Semantics(
                          sortKey: const OrdinalSortKey(1),
                          child: FadeTransition(
                            opacity: animation.drive(
                              CurveTween(curve: GfMotion.enterCurve),
                            ),
                            child: IntrinsicWidth(
                              child: GfMenuSurface(
                                key: const Key('gf-context-menu'),
                                child: SingleChildScrollView(
                                  child: Column(
                                    mainAxisSize: MainAxisSize.min,
                                    crossAxisAlignment:
                                        CrossAxisAlignment.stretch,
                                    children: [
                                      for (final action in actions)
                                        GfMenuItem(
                                          label: action.label,
                                          symbol: action.symbol,
                                          selected: action.selected,
                                          variant: action.destructive
                                              ? GfMenuItemVariant.danger
                                              : GfMenuItemVariant.normal,
                                          onTap: !action.enabled
                                              ? null
                                              : () {
                                                  if (isCurrent &&
                                                      sourceValid.value) {
                                                    navigator?.pop(
                                                      action.value,
                                                    );
                                                  }
                                                },
                                        ),
                                    ],
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                      if (preview != null)
                        LayoutId(
                          id: _ContextPart.preview,
                          child: Semantics(
                            sortKey: const OrdinalSortKey(0),
                            child: Material(
                              key: const Key('gf-context-preview'),
                              type: MaterialType.transparency,
                              child: ExcludeFocus(
                                child: IgnorePointer(
                                  child: HeroMode(
                                    enabled: false,
                                    child: ClipRect(
                                      child: SingleChildScrollView(
                                        physics:
                                            const NeverScrollableScrollPhysics(),
                                        child: TickerMode(
                                          enabled: false,
                                          child: preview!,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );
    return routeWrapper?.call(content) ?? content;
  }
}

enum _ContextPart { preview, actions }

/// The message keeps its own width. The action panel sizes to its longest label
/// plus icon/insets, capped at the viewport so enlarged labels can wrap.
class _ContextMenuLayout extends MultiChildLayoutDelegate {
  _ContextMenuLayout(
    this.source, {
    required this.maxPreviewHeight,
    required this.animation,
  }) : super(relayout: animation);
  final Rect source;
  final double maxPreviewHeight;
  final Animation<double> animation;

  @override
  void performLayout(Size size) {
    if (!hasChild(_ContextPart.preview)) {
      final menu = layoutChild(
        _ContextPart.actions,
        BoxConstraints.loose(size),
      );
      final below = source.bottom + 8;
      final above = source.top - 8 - menu.height;
      final y = below + menu.height <= size.height ? below : above;
      final x = source.center.dx > size.width / 2
          ? source.right - menu.width
          : source.left;
      positionChild(
        _ContextPart.actions,
        Offset(
          x.clamp(0.0, math.max(0, size.width - menu.width)).toDouble(),
          y.clamp(0.0, math.max(0, size.height - menu.height)).toDouble(),
        ),
      );
      return;
    }
    final preview = layoutChild(
      _ContextPart.preview,
      BoxConstraints(
        minWidth: math.min(source.width, size.width),
        maxWidth: math.min(source.width, size.width),
        maxHeight: math.min(
          source.height,
          math.min(maxPreviewHeight, math.max(0, size.height - 64)),
        ),
      ),
    );
    final gap = preview.height > 0 ? 8.0 : 0.0;
    final menu = layoutChild(
      _ContextPart.actions,
      BoxConstraints(
        maxWidth: size.width,
        maxHeight: math.max(0, size.height - preview.height - gap),
      ),
    );
    final previewX = source.left
        .clamp(0.0, math.max(0.0, size.width - preview.width))
        .toDouble();
    final previewY = source.top
        .clamp(
          0.0,
          math.max(0.0, size.height - preview.height - gap - menu.height),
        )
        .toDouble();
    final menuX = source.center.dx > size.width / 2
        ? previewX + preview.width - menu.width
        : previewX;
    positionChild(
      _ContextPart.preview,
      Offset.lerp(
        source.topLeft,
        Offset(previewX, previewY),
        GfMotion.enterCurve.transform(animation.value),
      )!,
    );
    positionChild(
      _ContextPart.actions,
      Offset(
        menuX.clamp(0.0, math.max(0.0, size.width - menu.width)).toDouble(),
        previewY + preview.height + gap,
      ),
    );
  }

  @override
  bool shouldRelayout(_ContextMenuLayout oldDelegate) =>
      source != oldDelegate.source ||
      maxPreviewHeight != oldDelegate.maxPreviewHeight;
}
