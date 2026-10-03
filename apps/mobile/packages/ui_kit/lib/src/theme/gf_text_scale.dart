import 'package:flutter/material.dart';

/// App-wide text scaling layered over the system [TextScaler].
///
/// The effective size of a style is `system(fontSize × baseline × userScale)`:
/// [baseline] is the device-adapted default from [baselineFor] and
/// [userScale] the app text size preference (100% by default). The system
/// scaler — including Android 14's nonlinear curve — still applies last and
/// is never replaced or clamped.
///
/// Every text in the app follows this scaler, rich content included; the
/// reading preference in `GfRichContentTypography` then adjusts long-form
/// body text on top of it.
@immutable
class GfAppTextScaler extends TextScaler {
  const GfAppTextScaler({
    required this.system,
    this.baseline = 1,
    this.userScale = 1,
  });

  /// The platform scaler this one wraps.
  final TextScaler system;

  /// Device-adapted default; see [baselineFor].
  final double baseline;

  /// App text size preference.
  final double userScale;

  /// App preference bounds (90%–130%); 100% is the adapted default.
  static const double minUserScale = .9;
  static const double maxUserScale = 1.3;

  static double clampUserScale(double value) =>
      value.clamp(minUserScale, maxUserScale);

  /// The default text size for this device before any user preference.
  ///
  /// Design sizes are iOS points: body text is 17, the iOS body size.
  /// Android's reading body is 16sp, so the same 17 reads one step large
  /// there and posts looked like an accessibility "elder mode" on common
  /// 360–393dp phones. Android therefore starts at 16/17. Windows whose
  /// shortest side is under 360 logical px take one more small step so a
  /// line keeps about 20 CJK characters. The shortest side is used so
  /// rotating the device never changes the text size.
  static double baselineFor({
    required TargetPlatform platform,
    required double shortestSide,
  }) {
    final double base = platform == TargetPlatform.android ? 16 / 17 : 1;
    return shortestSide > 0 && shortestSide < 360 ? base * .95 : base;
  }

  /// The same scaler at the 100% preference, for controls that must stay
  /// still while that preference is being adjusted.
  static TextScaler defaultOf(BuildContext context) {
    final TextScaler scaler = MediaQuery.textScalerOf(context);
    return scaler is GfAppTextScaler
        ? GfAppTextScaler(system: scaler.system, baseline: scaler.baseline)
        : scaler;
  }

  double get _factor => baseline * userScale;

  @override
  double scale(double fontSize) => system.scale(fontSize * _factor);

  // Kept for widgets that still read the deprecated linear factor.
  @override
  // ignore: deprecated_member_use
  double get textScaleFactor => system.textScaleFactor * _factor;

  @override
  bool operator ==(Object other) =>
      other is GfAppTextScaler &&
      other.system == system &&
      other.baseline == baseline &&
      other.userScale == userScale;

  @override
  int get hashCode => Object.hash(system, baseline, userScale);

  @override
  String toString() =>
      'GfAppTextScaler($system × ${baseline.toStringAsFixed(3)} × '
      '${userScale.toStringAsFixed(2)})';
}
