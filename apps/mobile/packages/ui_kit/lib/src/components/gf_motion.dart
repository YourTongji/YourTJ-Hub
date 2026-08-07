import 'package:flutter/widgets.dart';

/// Motion constants mirroring `apps/gooseforum/resource/src/runtime/motion.ts`.
///
/// The web motion system maps its cubic-bezier easing to Flutter curves:
/// - standard ease: `[0.22, 1, 0.36, 1]` ≈ easeOutCubic
/// - emphasized ease: `[0.16, 1, 0.3, 1]` ≈ a snappier easeOutQuint
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

  /// Standard ease curve: cubic-bezier(0.22, 1, 0.36, 1) ≈ easeOutCubic.
  static const Curve standardEase = Curves.easeOutCubic;

  /// Emphasized ease curve: cubic-bezier(0.16, 1, 0.3, 1) ≈ easeOutQuint.
  static const Curve emphasizedEase = Cubic(0.16, 1.0, 0.3, 1.0);

  /// Default cross-fade builder usable with [AnimatedSwitcher].
  static Widget standardTransition(Widget child, Animation<double> animation) {
    return FadeTransition(opacity: animation, child: child);
  }
}
