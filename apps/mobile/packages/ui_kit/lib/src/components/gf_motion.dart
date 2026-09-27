import 'package:flutter/material.dart';

/// Native mobile motion policy. Geometry/color tokens remain shared with Web;
/// motion belongs to this platform and preserves native direct manipulation.
abstract final class GfMotion {
  static const Duration press = Duration(milliseconds: 120);
  static const Duration selection = Duration(milliseconds: 160);
  static const Duration content = Duration(milliseconds: 180);
  static const Duration layout = Duration(milliseconds: 220);
  static const Duration overlay = Duration(milliseconds: 280);

  static const Curve enterCurve = Cubic(0.22, 1, 0.36, 1);
  static const Curve layoutCurve = Curves.easeInOutCubic;
  static const Curve exitCurve = Curves.easeInCubic;
  static const double rise = 6;

  static bool reducedOf(BuildContext context) =>
      MediaQuery.disableAnimationsOf(context);

  static Duration duration(BuildContext context, Duration duration) =>
      reducedOf(context) ? Duration.zero : duration;

  static AnimationStyle dialogStyle(BuildContext context) => reducedOf(context)
      ? AnimationStyle.noAnimation
      : const AnimationStyle(
          duration: layout,
          curve: enterCurve,
          reverseCurve: Curves.easeOutCubic,
        );

  static AnimationStyle sheetStyle(BuildContext context) => reducedOf(context)
      ? AnimationStyle.noAnimation
      : const AnimationStyle(
          duration: overlay,
          reverseDuration: layout,
          curve: enterCurve,
          reverseCurve: Curves.easeOutCubic,
        );

  /// Short, interruptible moves; long jumps avoid sweeping through unread text.
  static Future<void> scrollTo(
    BuildContext context,
    ScrollController controller,
    double offset,
  ) async {
    if (!controller.hasClients) return;
    final position = controller.position;
    final target = offset.clamp(
      position.minScrollExtent,
      position.maxScrollExtent,
    );
    if (reducedOf(context) ||
        (target - position.pixels).abs() > position.viewportDimension * 3) {
      controller.jumpTo(target);
      return;
    }
    await controller.animateTo(target, duration: overlay, curve: enterCurve);
  }
}

/// A single progress drives opacity and logical-pixel displacement. The static
/// child is reused per frame; enabling reduced motion removes the decoration.
class GfFadeTransition extends StatefulWidget {
  const GfFadeTransition({
    super.key,
    required this.animation,
    required this.child,
    this.offset = const Offset(0, GfMotion.rise),
  });

  final Animation<double> animation;
  final Widget child;
  final Offset offset;

  @override
  State<GfFadeTransition> createState() => _GfFadeTransitionState();
}

class _GfFadeTransitionState extends State<GfFadeTransition> {
  late CurvedAnimation _progress = _createProgress();

  CurvedAnimation _createProgress() => CurvedAnimation(
    parent: widget.animation,
    curve: GfMotion.enterCurve,
    reverseCurve: GfMotion.exitCurve.flipped,
  );

  @override
  void didUpdateWidget(GfFadeTransition oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.animation != widget.animation) {
      _progress.dispose();
      _progress = _createProgress();
    }
  }

  @override
  void dispose() {
    _progress.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final reduced = GfMotion.reducedOf(context);
    // Keep the element tree stable when accessibility changes at runtime, so
    // route-local drafts, focus and scroll positions survive the preference.
    return AnimatedBuilder(
      animation: _progress,
      child: widget.child,
      builder: (context, child) => IgnorePointer(
        ignoring: !reduced && _progress.value == 0,
        child: Opacity(
          opacity: reduced ? 1 : _progress.value,
          child: Transform.translate(
            offset: reduced
                ? Offset.zero
                : widget.offset * (1 - _progress.value),
            child: child,
          ),
        ),
      ),
    );
  }
}

/// Android page navigation. iOS retains Cupertino's interactive back gesture.
class GfPageTransitionsBuilder extends PageTransitionsBuilder {
  const GfPageTransitionsBuilder();

  @override
  Duration get transitionDuration => GfMotion.layout;

  @override
  Duration get reverseTransitionDuration => GfMotion.content;

  @override
  Widget buildTransitions<T>(
    PageRoute<T> route,
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) => GfFadeTransition(animation: animation, child: child);
}
