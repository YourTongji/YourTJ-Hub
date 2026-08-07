import 'package:flutter/material.dart';

import 'gf_colors.dart';
import 'gf_theme_extensions.dart';

export 'gf_colors.dart';
export 'gf_theme_extensions.dart';

/// Convenience accessors for the Gf design-token theme extensions.
///
/// Usage:
/// ```dart
/// final GfColors colors = GfTheme.colorsOf(context);
/// final GfRadii radii = GfTheme.radiiOf(context);
/// ```
abstract final class GfTheme {
  /// The [GfColors] palette active in [context].
  static GfColors colorsOf(BuildContext context) {
    return GfColors.forBrightness(Theme.of(context).brightness);
  }

  /// The [GfRadii] tokens registered by `gfThemeData`.
  static GfRadii radiiOf(BuildContext context) {
    return Theme.of(context).extension<GfRadii>() ?? GfRadii.standard;
  }

  /// The [GfBorders] tokens registered by `gfThemeData`.
  static GfBorders bordersOf(BuildContext context) {
    return Theme.of(context).extension<GfBorders>() ?? GfBorders.standard;
  }

  /// The [GfSizes] tokens registered by `gfThemeData`.
  static GfSizes sizesOf(BuildContext context) {
    return Theme.of(context).extension<GfSizes>() ?? GfSizes.standard;
  }
}
