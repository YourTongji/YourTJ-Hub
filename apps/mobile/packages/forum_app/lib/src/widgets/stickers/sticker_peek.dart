import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';

import 'sticker_image.dart';

/// Hold-to-preview for sticker grids (issue #1429, after FluxDO's picker).
///
/// Holding a cell opens an enlarged bubble anchored to it. Sliding the finger
/// over other cells switches the preview; releasing or cancelling closes it
/// without inserting anything, so a plain tap still inserts. When [onCollect]
/// accepts the sticker, the bubble offers a collect target: release over it
/// to save the sticker. The bubble lives in the root overlay and is removed
/// when the gesture ends, the grid leaves the tree, or its route is covered.
class StickerPeekScope extends StatefulWidget {
  const StickerPeekScope({
    super.key,
    required this.child,
    required this.labelOf,
    this.collectLabel,
    this.canCollect,
    this.onCollect,
  });
  final Widget child;
  final String Function(StickerItemPayload item) labelOf;
  final String? collectLabel;
  final bool Function(StickerItemPayload item)? canCollect;
  final ValueChanged<StickerItemPayload>? onCollect;

  static StickerPeekScopeState? maybeOf(BuildContext context) =>
      context.findAncestorStateOfType<StickerPeekScopeState>();

  @override
  State<StickerPeekScope> createState() => StickerPeekScopeState();
}

/// Hit-test markers: grid cells, the bubble body and its collect target.
class _PeekCell {
  const _PeekCell(this.item);
  final StickerItemPayload item;
}

class _PeekBubble {
  const _PeekBubble();
}

class _PeekCollect {
  const _PeekCollect();
}

class _PeekTarget {
  const _PeekTarget(this.item, this.anchor, {this.collectArmed = false});
  final StickerItemPayload item;
  final Rect anchor;
  final bool collectArmed;
}

class StickerPeekScopeState extends State<StickerPeekScope> {
  final ValueNotifier<_PeekTarget?> _target = ValueNotifier(null);
  OverlayEntry? _entry;

  bool get active => _entry != null;

  void _start(BuildContext cell, StickerItemPayload item) {
    final box = cell.findRenderObject();
    if (box is! RenderBox || !box.attached) return;
    HapticFeedback.mediumImpact();
    _target.value = _PeekTarget(
      item,
      box.localToGlobal(Offset.zero) & box.size,
    );
    if (_entry == null) {
      _entry = OverlayEntry(builder: _buildBubble);
      Overlay.of(context, rootOverlay: true).insert(_entry!);
    }
  }

  /// Finds the marker under [global] in the view hosting the grid.
  Object? _hit(Offset global) {
    final result = HitTestResult();
    WidgetsBinding.instance.hitTestInView(
      result,
      global,
      View.of(context).viewId,
    );
    for (final entry in result.path) {
      final target = entry.target;
      if (target is RenderMetaData) {
        final data = target.metaData;
        if (data is _PeekCell || data is _PeekBubble || data is _PeekCollect) {
          if (data is _PeekCell) return (data, target);
          return data;
        }
      }
    }
    return null;
  }

  void _move(Offset global) {
    final current = _target.value;
    if (current == null) return;
    final hit = _hit(global);
    if (hit is _PeekCollect) {
      if (!current.collectArmed) {
        HapticFeedback.selectionClick();
        _target.value = _PeekTarget(
          current.item,
          current.anchor,
          collectArmed: true,
        );
      }
      return;
    }
    if (hit is (_PeekCell, RenderMetaData)) {
      final (cell, box) = hit;
      if (cell.item.name == current.item.name && !current.collectArmed) return;
      if (cell.item.name != current.item.name) HapticFeedback.selectionClick();
      _target.value = _PeekTarget(
        cell.item,
        box.localToGlobal(Offset.zero) & box.size,
      );
      return;
    }
    // Over the bubble body or empty space: keep the current preview.
    if (current.collectArmed) {
      _target.value = _PeekTarget(current.item, current.anchor);
    }
  }

  void _end() {
    final current = _target.value;
    _close();
    if (current != null && current.collectArmed) {
      widget.onCollect?.call(current.item);
    }
  }

  void _close() {
    _entry?.remove();
    _entry = null;
    _target.value = null;
  }

  @override
  void deactivate() {
    _close();
    super.deactivate();
  }

  @override
  void dispose() {
    _close();
    _target.dispose();
    super.dispose();
  }

  Widget _buildBubble(BuildContext overlayContext) =>
      ValueListenableBuilder<_PeekTarget?>(
        valueListenable: _target,
        builder: (context, target, _) {
          if (target == null) return const SizedBox.shrink();
          final collectable =
              widget.onCollect != null &&
              widget.collectLabel != null &&
              (widget.canCollect?.call(target.item) ?? true);
          // A root overlay entry has no Material above it: without one its
          // text falls back to the debug error style (double underline).
          return Material(
            type: MaterialType.transparency,
            child: _PeekBubbleLayout(
              anchor: target.anchor,
              child: _PeekBubbleView(
                item: target.item,
                label: widget.labelOf(target.item),
                collectLabel: collectable ? widget.collectLabel : null,
                collectArmed: target.collectArmed,
              ),
            ),
          );
        },
      );

  @override
  Widget build(BuildContext context) => widget.child;
}

/// Wraps one grid cell. Taps stay with [child]; holding opens the bubble.
class StickerPeekCell extends StatelessWidget {
  const StickerPeekCell({
    super.key,
    required this.item,
    required this.child,
    this.onSecondaryTap,
  });
  final StickerItemPayload item;
  final Widget child;

  /// Right click or stylus button: the pointer route to the cell's actions.
  final VoidCallback? onSecondaryTap;

  @override
  Widget build(BuildContext context) {
    final scope = StickerPeekScope.maybeOf(context);
    if (scope == null) return child;
    return MetaData(
      metaData: _PeekCell(item),
      behavior: HitTestBehavior.opaque,
      child: GestureDetector(
        // The cell's own Semantics exposes an accessible long-press action;
        // a synthesized drag gesture would open a bubble with nothing to hold.
        excludeFromSemantics: true,
        onLongPressStart: (_) => scope._start(context, item),
        onLongPressMoveUpdate: (details) => scope._move(details.globalPosition),
        onLongPressEnd: (_) => scope._end(),
        onLongPressCancel: scope._close,
        onSecondaryTap: onSecondaryTap,
        child: child,
      ),
    );
  }
}

/// Places the bubble above its anchor, or below when the top is too tight,
/// inside the safe area and above the keyboard.
class _PeekBubbleLayout extends StatelessWidget {
  const _PeekBubbleLayout({required this.anchor, required this.child});
  final Rect anchor;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final media = MediaQuery.of(context);
    return CustomSingleChildLayout(
      delegate: _PeekBubbleDelegate(
        anchor: anchor,
        padding: media.padding + media.viewInsets + const EdgeInsets.all(8),
      ),
      child: child,
    );
  }
}

class _PeekBubbleDelegate extends SingleChildLayoutDelegate {
  _PeekBubbleDelegate({required this.anchor, required this.padding});
  final Rect anchor;
  final EdgeInsets padding;
  static const double gap = 8;

  @override
  BoxConstraints getConstraintsForChild(BoxConstraints constraints) =>
      BoxConstraints.loose(constraints.biggest).deflate(padding);

  @override
  Offset getPositionForChild(Size size, Size child) {
    final minX = padding.left;
    final maxX = size.width - padding.right - child.width;
    final x = (anchor.center.dx - child.width / 2).clamp(
      minX,
      maxX < minX ? minX : maxX,
    );
    final above = anchor.top - gap - child.height;
    final below = anchor.bottom + gap;
    final maxY = size.height - padding.bottom - child.height;
    final y = above >= padding.top
        ? above
        : below.clamp(padding.top, maxY < padding.top ? padding.top : maxY);
    return Offset(x, y);
  }

  @override
  bool shouldRelayout(_PeekBubbleDelegate oldDelegate) =>
      oldDelegate.anchor != anchor || oldDelegate.padding != padding;
}

class _PeekBubbleView extends StatelessWidget {
  const _PeekBubbleView({
    required this.item,
    required this.label,
    required this.collectLabel,
    required this.collectArmed,
  });
  final StickerItemPayload item;
  final String label;
  final String? collectLabel;
  final bool collectArmed;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final reduceMotion = MediaQuery.of(context).disableAnimations;
    final bubble = MetaData(
      metaData: const _PeekBubble(),
      behavior: HitTestBehavior.opaque,
      child: Container(
        width: 156,
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: colors.surfaceRaised,
          borderRadius: BorderRadius.circular(GfRadius.lg),
          border: Border.all(
            color: colors.separator,
            width: gfHairline(context),
          ),
          boxShadow: colors.elevation(GfElevation.mid),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            IgnorePointer(
              child: StickerImage(
                key: ValueKey(item.name),
                name: item.name,
                url: item.url,
                label: label,
                size: 120,
                collectible: false,
                excludeSemantics: true,
              ),
            ),
            const SizedBox(height: 6),
            GfText(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(fontSize: 12, color: colors.textSecondary),
            ),
            if (collectLabel != null) ...[
              const SizedBox(height: 8),
              MetaData(
                metaData: const _PeekCollect(),
                behavior: HitTestBehavior.opaque,
                child: AnimatedContainer(
                  duration: reduceMotion
                      ? Duration.zero
                      : const Duration(milliseconds: 120),
                  curve: Curves.easeOut,
                  constraints: const BoxConstraints(minHeight: 36),
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  decoration: BoxDecoration(
                    color: collectArmed ? colors.primary : colors.fill,
                    borderRadius: BorderRadius.circular(GfRadius.full),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      GfSymbol(
                        'bookmark',
                        size: 16,
                        color: collectArmed
                            ? colors.primaryContent
                            : colors.textPrimary,
                      ),
                      const SizedBox(width: 6),
                      Flexible(
                        child: GfText(
                          collectLabel!,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                            color: collectArmed
                                ? colors.primaryContent
                                : colors.textPrimary,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
    if (reduceMotion) return bubble;
    return TweenAnimationBuilder<double>(
      tween: Tween(begin: 0, end: 1),
      duration: const Duration(milliseconds: 120),
      curve: Curves.easeOut,
      builder: (context, value, child) => Opacity(
        opacity: value,
        child: Transform.scale(scale: 0.96 + 0.04 * value, child: child),
      ),
      child: bubble,
    );
  }
}
