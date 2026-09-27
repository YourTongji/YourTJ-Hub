import 'dart:math' as math;

import 'package:flutter/widgets.dart';

import 'gf_motion.dart';

/// A restrained, one-shot acknowledgement of an explicit toggle action.
/// Passive state updates never replay it; callbacks run immediately, and the
/// owner retains pending/rollback semantics. Only the visual scales, not its
/// focusable hit target or surrounding layout.
class GfActionFeedback extends StatefulWidget {
  const GfActionFeedback({
    super.key,
    required this.active,
    required this.onPressed,
    required this.child,
    required this.builder,
  });

  final bool active;
  final VoidCallback? onPressed;
  final Widget child;
  final Widget Function(VoidCallback? onPressed, Widget visual) builder;

  @override
  State<GfActionFeedback> createState() => _GfActionFeedbackState();
}

class _GfActionFeedbackState extends State<GfActionFeedback>
    with SingleTickerProviderStateMixin {
  late final _controller = AnimationController(
    vsync: this,
    duration: GfMotion.layout,
  );

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (GfMotion.reducedOf(context)) _controller.value = 0;
  }

  @override
  void didUpdateWidget(GfActionFeedback oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.active && !widget.active) _controller.value = 0;
  }

  void _activate() {
    if (widget.onPressed == null) return;
    if (!widget.active &&
        !GfMotion.reducedOf(context) &&
        !_controller.isAnimating) {
      _controller.forward(from: 0);
    }
    widget.onPressed!();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.builder(
    widget.onPressed == null ? null : _activate,
    AnimatedBuilder(
      animation: _controller,
      child: widget.child,
      builder: (_, child) => Transform.scale(
        scale: 1 + math.sin(_controller.value * math.pi) * .06,
        child: child,
      ),
    ),
  );
}
