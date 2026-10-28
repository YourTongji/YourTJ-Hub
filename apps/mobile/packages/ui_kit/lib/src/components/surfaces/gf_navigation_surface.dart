import 'dart:ui' show ImageFilter;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

List<double> _saturation(double amount) {
  const double r = 0.2126, g = 0.7152, b = 0.0722;
  final double rest = 1 - amount;
  return <double>[
    r * rest + amount, g * rest, b * rest, 0, 0, //
    r * rest, g * rest + amount, b * rest, 0, 0, //
    r * rest, g * rest, b * rest + amount, 0, 0, //
    0, 0, 0, 1, 0, //
  ];
}

/// Shared dock/rail background. Place in a bounded box behind controls.
/// On glass, content under it is blurred and its colour
/// lifted, as frosted glass reads, under a translucent raised-surface tint
/// with a lit rim. High contrast keeps the opaque raised surface.
class GfNavigationSurface extends StatelessWidget {
  const GfNavigationSurface({
    super.key,
    this.borderRadius = const BorderRadius.all(Radius.circular(GfRadius.full)),
  });

  final BorderRadius borderRadius;

  static bool usesGlass(BuildContext context) =>
      !MediaQuery.highContrastOf(context) &&
      !MediaQuery.disableAnimationsOf(context) &&
      !MediaQuery.accessibleNavigationOf(context);

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final radius = borderRadius;
    if (!usesGlass(context)) {
      return DecoratedBox(
        decoration: BoxDecoration(
          color: colors.surfaceRaised,
          borderRadius: radius,
          border: Border.all(color: colors.outline),
          boxShadow: colors.elevation(GfElevation.mid),
        ),
      );
    }
    return CustomPaint(
      painter: _DockShadowPainter(colors.elevation(GfElevation.mid), radius),
      child: ClipRRect(
        borderRadius: radius,
        child: BackdropFilter(
          filter: ImageFilter.compose(
            outer: ImageFilter.blur(sigmaX: 18, sigmaY: 18),
            inner: ColorFilter.matrix(_saturation(colors.isDark ? 1.4 : 1.7)),
          ),
          child: CustomPaint(
            painter: _GlassPainter(colors, gfHairline(context), radius),
            child: const SizedBox.expand(),
          ),
        ),
      ),
    );
  }
}

/// Casts the floating elevation only outside the pill, so translucent glass
/// never shows its own shadow through itself.
class _DockShadowPainter extends CustomPainter {
  const _DockShadowPainter(this.shadows, this.radius);

  final List<BoxShadow> shadows;
  final BorderRadius radius;

  @override
  void paint(Canvas canvas, Size size) {
    if (shadows.isEmpty) return;
    final RRect pill = radius.toRRect(Offset.zero & size).scaleRadii();
    canvas
      ..save()
      ..clipPath(
        Path.combine(
          PathOperation.difference,
          Path()..addRect((Offset.zero & size).inflate(64)),
          Path()..addRRect(pill),
        ),
      );
    for (final BoxShadow shadow in shadows) {
      canvas.drawRRect(
        pill.shift(shadow.offset).inflate(shadow.spreadRadius),
        shadow.toPaint(),
      );
    }
    canvas.restore();
  }

  @override
  bool shouldRepaint(_DockShadowPainter oldDelegate) =>
      !listEquals(oldDelegate.shadows, shadows) || oldDelegate.radius != radius;
}

class _GlassPainter extends CustomPainter {
  const _GlassPainter(this.colors, this.hairline, this.radius);

  final GfColors colors;
  final double hairline;
  final BorderRadius radius;

  @override
  void paint(Canvas canvas, Size size) {
    final bool dark = colors.isDark;
    final Rect bounds = Offset.zero & size;
    final RRect pill = radius.toRRect(bounds).scaleRadii();
    canvas.drawRRect(
      pill,
      Paint()
        ..color = colors.surfaceRaised.withValues(alpha: dark ? 0.66 : 0.64),
    );
    // Light falls from above: a sheen across the upper half...
    canvas.drawRRect(
      pill,
      Paint()
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.center,
          colors: <Color>[
            Colors.white.withValues(alpha: dark ? 0.05 : 0.3),
            Colors.white.withValues(alpha: 0),
          ],
        ).createShader(bounds),
    );
    // ...a rim that is brightest where it faces the light...
    canvas.drawRRect(
      pill.deflate(0.75),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.5
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: <Color>[
            Colors.white.withValues(alpha: dark ? 0.16 : 0.9),
            Colors.white.withValues(alpha: dark ? 0.03 : 0.3),
          ],
        ).createShader(bounds),
    );
    // ...and an outer edge that keeps the pill distinct on a matching page.
    canvas.drawRRect(
      pill.deflate(hairline / 2),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = hairline
        ..color = dark
            ? colors.line
            : colors.baseContent.withValues(alpha: 0.07),
    );
  }

  @override
  bool shouldRepaint(_GlassPainter oldDelegate) =>
      oldDelegate.colors != colors ||
      oldDelegate.hairline != hairline ||
      oldDelegate.radius != radius;
}
