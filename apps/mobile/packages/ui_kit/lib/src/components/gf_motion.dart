import 'package:flutter/widgets.dart';

/// Motion constants mirroring `apps/gooseforum/resource/src/runtime/motion.ts`.
///
/// The web motion system maps its cubic-bezier easing to Flutter curves:
/// - standard: `[0.22, 1, 0.36, 1]` ≈ easeOutCubic
/// - emphasized: `[0.16, 1, 0.3, 1]` ≈ a snappier easeOutQuint
abstract final class GfMotion {
  /// Standard interactive transition duration (0.12s in motion.ts).
  static const Duration standardDuration = Duration(milliseconds: 120);

  /// Emphasized / content transition duration (0.28s in motion.ts).
  static const Duration emphasizedDuration = Duration(milliseconds: 280);

  /// Standard ease curve: cubic-bezier(0.22, 1, 0.36, 1) ≈ easeOutCubic.
  static const Curve standard = Curves.easeOutCubic;

  /// Emphasized ease curve: cubic-bezier(0.16, 1, 0.3, 1) ≈ easeOutQuint.
  static const Curve emphasized = Cubic(0.16, 1.0, 0.3, 1.0);

  /// Default cross-fade builder usable with [AnimatedSwitcher].
  static Widget standardTransition(Widget child, Animation<double> animation) {
    return FadeTransition(opacity: animation, child: child);
  }
}
