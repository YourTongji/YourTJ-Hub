import 'package:flutter/material.dart';

/// Keeps Flutter's filled-field label layout inside the surface while drawing
/// a continuous rounded focus/error outline. An OutlineInputBorder would place
/// floating labels across the top edge, even when its border is invisible.
class GfFilledInputBorder extends UnderlineInputBorder {
  const GfFilledInputBorder({
    super.borderSide = BorderSide.none,
    super.borderRadius = const BorderRadius.all(Radius.circular(16)),
  });

  @override
  GfFilledInputBorder copyWith({
    BorderSide? borderSide,
    BorderRadius? borderRadius,
  }) => GfFilledInputBorder(
    borderSide: borderSide ?? this.borderSide,
    borderRadius: borderRadius ?? this.borderRadius,
  );

  @override
  GfFilledInputBorder scale(double t) => GfFilledInputBorder(
    borderSide: borderSide.scale(t),
    borderRadius: borderRadius * t,
  );

  @override
  ShapeBorder? lerpFrom(ShapeBorder? a, double t) {
    if (a is GfFilledInputBorder) {
      return GfFilledInputBorder(
        borderSide: BorderSide.lerp(a.borderSide, borderSide, t),
        borderRadius: BorderRadius.lerp(a.borderRadius, borderRadius, t)!,
      );
    }
    return super.lerpFrom(a, t);
  }

  @override
  ShapeBorder? lerpTo(ShapeBorder? b, double t) {
    if (b is GfFilledInputBorder) {
      return GfFilledInputBorder(
        borderSide: BorderSide.lerp(borderSide, b.borderSide, t),
        borderRadius: BorderRadius.lerp(borderRadius, b.borderRadius, t)!,
      );
    }
    return super.lerpTo(b, t);
  }

  @override
  void paint(
    Canvas canvas,
    Rect rect, {
    double? gapStart,
    double gapExtent = 0,
    double gapPercentage = 0,
    TextDirection? textDirection,
  }) {
    RoundedRectangleBorder(
      side: borderSide,
      borderRadius: borderRadius,
    ).paint(canvas, rect, textDirection: textDirection);
  }
}
