import 'package:flutter/widgets.dart';

/// Duration + ease pair (motion.ts `motionTransitions`).
@immutable
class GfMotionTransition {
  const GfMotionTransition(this.duration, this.ease);

  final Duration duration;
  final Curve ease;
}

/// Motion constants mirroring `apps/gooseforum/resource/src/runtime/motion.ts`.
///
/// The web motion system maps its cubic-bezier easing to Flutter curves:
/// - standard ease: `cubic-bezier(0.22, 1, 0.36, 1)`
/// - emphasized ease: `cubic-bezier(0.16, 1, 0.3, 1)`
abstract final class GfMotion {
  /// Instant interactive transition (motion.ts `motionDurations.instant`,
  /// 0.12s).
  static const Duration instant = Duration(milliseconds: 120);

  /// Fast overlay transition (motion.ts `motionDurations.fast`, 0.16s).
  static const Duration fast = Duration(milliseconds: 160);

  /// Content transition (motion.ts `motionDurations.content`, 0.18s).
  static const Duration content = Duration(milliseconds: 180);

  /// Standard interactive transition (motion.ts `motionDurations.standard`,
  /// 0.22s).
  static const Duration standard = Duration(milliseconds: 220);

  /// Comfortable / drawer transition (motion.ts
  /// `motionDurations.comfortable`, 0.28s).
  static const Duration comfortable = Duration(milliseconds: 280);

  /// Legacy alias for [instant] (pre-5-tier API, kept for compatibility).
  static const Duration standardDuration = instant;

  /// Legacy alias for [comfortable] (pre-5-tier API, kept for compatibility).
  static const Duration emphasizedDuration = comfortable;

  /// Standard ease curve: exact `cubic-bezier(0.22, 1, 0.36, 1)` from
  /// motion.ts `motionEase.standard`.
  static const Curve standardEase = Cubic(0.22, 1.0, 0.36, 1.0);

  /// Emphasized ease curve: exact `cubic-bezier(0.16, 1, 0.3, 1)` from
  /// motion.ts `motionEase.emphasized`.
  static const Curve emphasizedEase = Cubic(0.16, 1.0, 0.3, 1.0);

  /// `fast` transition: 0.16s standard ease (motion.ts
  /// `motionTransitions.fast`).
  static const GfMotionTransition fastMotion = GfMotionTransition(
    fast,
    standardEase,
  );

  /// `content` transition: 0.18s standard ease (motion.ts
  /// `motionTransitions.content`).
  static const GfMotionTransition contentMotion = GfMotionTransition(
    content,
    standardEase,
  );

  /// `standard` transition: 0.22s standard ease (motion.ts
  /// `motionTransitions.standard`).
  static const GfMotionTransition standardMotion = GfMotionTransition(
    standard,
    standardEase,
  );

  /// `comfortable` transition: 0.28s emphasized ease (motion.ts
  /// `motionTransitions.comfortable`).
  static const GfMotionTransition comfortableMotion = GfMotionTransition(
    comfortable,
    emphasizedEase,
  );

  /// Menu pop motion (motion.css `.gf-menu-*`: 0.14s, -4px rise).
  static const Duration menuDuration = Duration(milliseconds: 140);

  /// Modal motion (motion.css `.gf-modal-*`: 0.16s, child scale 0.98 + 6px).
  static const Duration modalDuration = Duration(milliseconds: 160);

  /// Flash / toast motion (motion.css `.gf-flash-*`: 0.28s, -14px + scale 0.98).
  static const Duration flashDuration = Duration(milliseconds: 280);

  /// Floating reply motion (motion.css `.floating-reply-*`: 0.18s,
  /// 10px rise + scale 0.98).
  static const Duration floatingReplyDuration = Duration(milliseconds: 180);

  /// Local expand motion (motion.css `.gf-local-expand-*`: 0.16s).
  static const Duration localExpandDuration = Duration(milliseconds: 160);

  /// Default cross-fade builder usable with [AnimatedSwitcher].
  static Widget standardTransition(Widget child, Animation<double> animation) {
    return FadeTransition(opacity: animation, child: child);
  }
}

/// Page transition matching the web page-enter motion (motion.css
/// `gf-page-enter`: 0.22s standard ease, subtle fade + 4px rise).
class GfPageTransitionsBuilder extends PageTransitionsBuilder {
  const GfPageTransitionsBuilder();

  @override
  Widget buildTransitions<T>(
    PageRoute<T> route,
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    return FadeTransition(
      opacity: animation,
      child: SlideTransition(
        position: Tween<Offset>(begin: const Offset(0, 0.004), end: Offset.zero)
            .animate(
              CurvedAnimation(parent: animation, curve: GfMotion.standardEase),
            ),
        child: child,
      ),
    );
  }
}
