import 'package:flutter/gestures.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';

/// Touch-only direct manipulation. Vertical scrolling, mouse text selection and
/// the iOS back edge remain owned by their existing surfaces.
class SwipeReply extends StatefulWidget {
  const SwipeReply({
    super.key,
    required this.child,
    required this.label,
    this.onReply,
    this.onActions,
    this.actionsLabel,
  });
  final Widget child;
  final String label;
  final VoidCallback? onReply;
  final VoidCallback? onActions;
  final String? actionsLabel;
  @override
  State<SwipeReply> createState() => _SwipeReplyState();
}

class _SwipeReplyState extends State<SwipeReply>
    with SingleTickerProviderStateMixin {
  static const threshold = 64.0;
  late final AnimationController _offset;
  @override
  void initState() {
    super.initState();
    _offset = AnimationController(vsync: this, lowerBound: 0, upperBound: 88);
  }

  bool _crossed = false;
  void _reset() {
    if (GfMotion.reducedOf(context)) {
      _offset.value = 0;
      return;
    }
    _offset.animateBack(
      0,
      duration: GfMotion.content,
      curve: Curves.easeOutCubic,
    );
  }

  @override
  void didUpdateWidget(SwipeReply oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.onReply == null) _offset.value = 0;
  }

  @override
  void dispose() {
    _offset.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.onReply == null) return widget.child;
    return Semantics(
      customSemanticsActions: {
        CustomSemanticsAction(label: widget.label): widget.onReply!,
        if (widget.onActions != null && widget.actionsLabel != null)
          CustomSemanticsAction(label: widget.actionsLabel!): widget.onActions!,
      },
      child: GestureDetector(
        // Include the first accepted move: a fast iOS swipe may arrive as one
        // coalesced event, with no later delta before pointer-up.
        dragStartBehavior: DragStartBehavior.down,
        supportedDevices: const {
          PointerDeviceKind.touch,
          PointerDeviceKind.stylus,
        },
        onHorizontalDragStart: (_) {
          _offset.stop();
          _crossed = false;
        },
        onHorizontalDragUpdate: (details) {
          _offset.value = (_offset.value - details.delta.dx).clamp(0, 88);
          if (_offset.value >= threshold && !_crossed) {
            _crossed = true;
            HapticFeedback.selectionClick();
          }
        },
        onHorizontalDragEnd: (_) {
          final reply = _offset.value >= threshold;
          _reset();
          if (reply) widget.onReply?.call();
        },
        onHorizontalDragCancel: _reset,
        child: AnimatedBuilder(
          animation: _offset,
          child: widget.child,
          builder: (context, child) => Stack(
            alignment: Alignment.centerRight,
            children: [
              ExcludeSemantics(
                child: Opacity(
                  opacity: (_offset.value / threshold).clamp(0, 1),
                  child: Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: IconTheme(
                      data: IconThemeData(
                        color: GfTheme.colorsOf(context).primary,
                      ),
                      child: const GfSymbol('quote', size: 24),
                    ),
                  ),
                ),
              ),
              Transform.translate(
                offset: Offset(-_offset.value, 0),
                child: child,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
