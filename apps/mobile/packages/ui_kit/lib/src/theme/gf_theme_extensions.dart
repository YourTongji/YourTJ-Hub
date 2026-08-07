import 'dart:ui';

import 'package:flutter/material.dart';

/// Radius tokens, mirrored from `tokens.css` `--gf-radius-*`.
@immutable
class GfRadii extends ThemeExtension<GfRadii> {
  const GfRadii({required this.selector, required this.field, required this.box});

  /// Pill-like radii for selectors / chips. `--gf-radius-selector` (8).
  final double selector;

  /// Radii for form fields and buttons. `--gf-radius-field` (8).
  final double field;

  /// Radii for panels and cards. `--gf-radius-box` (8).
  final double box;

  /// Values shared by both themes (tokens.css defines them once in `:root`).
  static const GfRadii standard = GfRadii(selector: 8, field: 8, box: 8);

  @override
  GfRadii copyWith({double? selector, double? field, double? box}) {
    return GfRadii(
      selector: selector ?? this.selector,
      field: field ?? this.field,
      box: box ?? this.box,
    );
  }

  @override
  GfRadii lerp(GfRadii? other, double t) {
    if (other == null) return this;
    return GfRadii(
      selector: lerpDouble(selector, other.selector, t)!,
      field: lerpDouble(field, other.field, t)!,
      box: lerpDouble(box, other.box, t)!,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is GfRadii &&
      other.selector == selector &&
      other.field == field &&
      other.box == box;

  @override
  int get hashCode => Object.hash(selector, field, box);
}

/// Border and depth tokens, mirrored from `tokens.css` `--gf-border` /
/// `--gf-depth`.
@immutable
class GfBorders extends ThemeExtension<GfBorders> {
  const GfBorders({required this.width, required this.depth});

  /// Hairline border width in logical pixels. `--gf-border` (1px).
  final double width;

  /// Elevation multiplier controlling shadow strength. `--gf-depth` (1).
  final double depth;

  /// Values shared by both themes (tokens.css defines them once in `:root`).
  static const GfBorders standard = GfBorders(width: 1, depth: 1);

  @override
  GfBorders copyWith({double? width, double? depth}) {
    return GfBorders(width: width ?? this.width, depth: depth ?? this.depth);
  }

  @override
  GfBorders lerp(GfBorders? other, double t) {
    if (other == null) return this;
    return GfBorders(
      width: lerpDouble(width, other.width, t)!,
      depth: lerpDouble(depth, other.depth, t)!,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is GfBorders && other.width == width && other.depth == depth;

  @override
  int get hashCode => Object.hash(width, depth);
}

/// Spacing tokens, mirrored from `tokens.css` `--gf-size-*`.
@immutable
class GfSizes extends ThemeExtension<GfSizes> {
  const GfSizes({required this.selector, required this.field});

  /// Base spacing step for selectors. `--gf-size-selector` (4).
  final double selector;

  /// Base spacing step for fields. `--gf-size-field` (4).
  final double field;

  /// Values shared by both themes (tokens.css defines them once in `:root`).
  static const GfSizes standard = GfSizes(selector: 4, field: 4);

  @override
  GfSizes copyWith({double? selector, double? field}) {
    return GfSizes(selector: selector ?? this.selector, field: field ?? this.field);
  }

  @override
  GfSizes lerp(GfSizes? other, double t) {
    if (other == null) return this;
    return GfSizes(
      selector: lerpDouble(selector, other.selector, t)!,
      field: lerpDouble(field, other.field, t)!,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is GfSizes && other.selector == selector && other.field == field;

  @override
  int get hashCode => Object.hash(selector, field);
}
