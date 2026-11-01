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

/// How strongly a glass surface is tinted.
enum GfGlassTone {
  /// The dock, the rail and secondary floating controls: a translucent
  /// raised-surface tint.
  regular,

  /// The floating primary action: the same glass tinted with the primary
  /// colour, dense enough to keep its glyph legible on any content.
  prominent,
}

/// Shared floating-layer glass: the dock, the rail, the compose action and its
/// menu. Place in a bounded box behind controls.
/// On glass, content under it is blurred and its colour
/// lifted, as frosted glass reads, under a translucent tint with a lit rim.
/// High contrast keeps the opaque surface.
class GfNavigationSurface extends StatelessWidget {
  const GfNavigationSurface({
    super.key,
    this.borderRadius = const BorderRadius.all(Radius.circular(GfRadius.full)),
    this.tone = GfGlassTone.regular,
  });

  final BorderRadius borderRadius;
  final GfGlassTone tone;

  static bool usesGlass(BuildContext context) =>
      !MediaQuery.highContrastOf(context) &&
      !MediaQuery.disableAnimationsOf(context) &&
      !MediaQuery.accessibleNavigationOf(context);

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final radius = borderRadius;
    final bool prominent = tone == GfGlassTone.prominent;
    if (!usesGlass(context)) {
      if (prominent) {
        return DecoratedBox(
          decoration: BoxDecoration(
            color: colors.primary,
            borderRadius: radius,
          ),
        );
      }
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
            painter: prominent
                ? _ProminentGlassPainter(colors, gfHairline(context), radius)
                : _GlassPainter(colors, gfHairline(context), radius),
            child: const SizedBox.expand(),
          ),
        ),
      ),
    );
  }
}

/// Edge-anchored glass for a full-width bar along the bottom of the screen.
/// Content scrolling under it is frosted under a thinner tint than the
/// floating dock carries, so the bar reads as a lighter material on the same
/// page. It needs no rim or shadow: a hairline marks the edge it shares with
/// the content, with a lit line just inside it where the glass catches the
/// light. Where glass is skipped, the bar is the opaque canvas under a hairline.
class GfNavigationBarSurface extends StatelessWidget {
  const GfNavigationBarSurface({super.key});

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final double hairline = gfHairline(context);
    if (!GfNavigationSurface.usesGlass(context)) {
      return DecoratedBox(
        decoration: BoxDecoration(
          color: colors.canvas,
          border: Border(
            top: BorderSide(color: colors.separator, width: hairline),
          ),
        ),
      );
    }
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.compose(
          outer: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
          inner: ColorFilter.matrix(_saturation(colors.isDark ? 1.5 : 1.8)),
        ),
        child: CustomPaint(
          painter: _BarGlassPainter(colors, hairline),
          child: const SizedBox.expand(),
        ),
      ),
    );
  }
}

/// The single surface [GfBottomNavigation] paints while it changes style:
/// the dock's glass at [barness] 0, the bar's at 1, and one material in
/// between. Its tint, rim, sheen and shadow blend over one shared backdrop,
/// so the blur never drops out partway.
class GfNavigationMorphSurface extends StatelessWidget {
  const GfNavigationMorphSurface({
    super.key,
    required this.barness,
    required this.radius,
  });

  final double barness;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final double hairline = gfHairline(context);
    final double t = barness.clamp(0.0, 1.0);
    final BorderRadius borderRadius = BorderRadius.circular(radius);
    if (!GfNavigationSurface.usesGlass(context)) {
      return DecoratedBox(
        decoration: BoxDecoration(
          color: Color.lerp(colors.surfaceRaised, colors.canvas, t),
          borderRadius: borderRadius,
          border: Border.all(
            color: Color.lerp(colors.outline, colors.separator, t)!,
            width: hairline,
          ),
        ),
      );
    }
    final List<BoxShadow> shadows = <BoxShadow>[
      for (final BoxShadow shadow in colors.elevation(GfElevation.mid))
        shadow.copyWith(
          color: shadow.color.withValues(alpha: shadow.color.a * (1 - t)),
        ),
    ];
    return CustomPaint(
      painter: _DockShadowPainter(shadows, borderRadius),
      child: ClipRRect(
        borderRadius: borderRadius,
        child: BackdropFilter(
          filter: ImageFilter.compose(
            outer: ImageFilter.blur(sigmaX: 18 + 6 * t, sigmaY: 18 + 6 * t),
            inner: ColorFilter.matrix(
              _saturation(colors.isDark ? 1.4 + 0.1 * t : 1.7 + 0.1 * t),
            ),
          ),
          child: CustomPaint(
            painter: _MorphGlassPainter(colors, hairline, borderRadius, t),
            child: const SizedBox.expand(),
          ),
        ),
      ),
    );
  }
}

class _MorphGlassPainter extends CustomPainter {
  const _MorphGlassPainter(this.colors, this.hairline, this.radius, this.t);

  final GfColors colors;
  final double hairline;
  final BorderRadius radius;
  final double t;

  @override
  void paint(Canvas canvas, Size size) {
    final Rect bounds = Offset.zero & size;
    void blend(CustomPainter painter, double amount) {
      if (amount <= 0) return;
      canvas.saveLayer(
        bounds,
        Paint()..color = Colors.black.withValues(alpha: amount),
      );
      painter.paint(canvas, size);
      canvas.restore();
    }

    blend(_GlassPainter(colors, hairline, radius), 1 - t);
    blend(_BarGlassPainter(colors, hairline), t);
  }

  @override
  bool shouldRepaint(_MorphGlassPainter oldDelegate) =>
      oldDelegate.colors != colors ||
      oldDelegate.hairline != hairline ||
      oldDelegate.radius != radius ||
      oldDelegate.t != t;
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

class _BarGlassPainter extends CustomPainter {
  const _BarGlassPainter(this.colors, this.hairline);

  final GfColors colors;
  final double hairline;

  @override
  void paint(Canvas canvas, Size size) {
    final bool dark = colors.isDark;
    final Rect bounds = Offset.zero & size;
    canvas.drawRect(
      bounds,
      Paint()..color = colors.canvas.withValues(alpha: dark ? 0.72 : 0.76),
    );
    // A faint sheen settles from the top edge, so the glass reads lit from
    // above rather than as a flat tint.
    canvas.drawRect(
      bounds,
      Paint()
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.center,
          colors: <Color>[
            Colors.white.withValues(alpha: dark ? 0.04 : 0.2),
            Colors.white.withValues(alpha: 0),
          ],
        ).createShader(bounds),
    );
    // The lit line sits just inside the edge...
    canvas.drawRect(
      Rect.fromLTWH(0, hairline, size.width, 1),
      Paint()..color = Colors.white.withValues(alpha: dark ? 0.06 : 0.55),
    );
    // ...and the hairline keeps the bar distinct on a matching page.
    canvas.drawRect(
      Rect.fromLTWH(0, 0, size.width, hairline),
      Paint()
        ..color = dark
            ? colors.line
            : colors.baseContent.withValues(alpha: 0.1),
    );
  }

  @override
  bool shouldRepaint(_BarGlassPainter oldDelegate) =>
      oldDelegate.colors != colors || oldDelegate.hairline != hairline;
}

/// Primary-tinted glass, read as a thick lens: light from above lights the
/// top of the rim and pools again along the bottom inside edge, where a real
/// lens focuses it.
class _ProminentGlassPainter extends CustomPainter {
  const _ProminentGlassPainter(this.colors, this.hairline, this.radius);

  final GfColors colors;
  final double hairline;
  final BorderRadius radius;

  @override
  void paint(Canvas canvas, Size size) {
    final bool dark = colors.isDark;
    final Rect bounds = Offset.zero & size;
    final RRect pill = radius.toRRect(bounds).scaleRadii();
    // Dense enough that the glyph keeps 3:1 over white or black content.
    canvas.drawRRect(
      pill,
      Paint()..color = colors.primary.withValues(alpha: dark ? 0.84 : 0.88),
    );
    canvas.drawRRect(
      pill,
      Paint()
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.center,
          colors: <Color>[
            Colors.white.withValues(alpha: dark ? 0.14 : 0.24),
            Colors.white.withValues(alpha: 0),
          ],
        ).createShader(bounds),
    );
    canvas.drawRRect(
      pill,
      Paint()
        ..shader = LinearGradient(
          begin: Alignment.bottomCenter,
          end: const Alignment(0, 0.3),
          colors: <Color>[
            Colors.white.withValues(alpha: dark ? 0.1 : 0.14),
            Colors.white.withValues(alpha: 0),
          ],
        ).createShader(bounds),
    );
    canvas.drawRRect(
      pill.deflate(0.75),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.5
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          stops: const <double>[0, 0.55, 1],
          colors: <Color>[
            Colors.white.withValues(alpha: dark ? 0.34 : 0.6),
            Colors.white.withValues(alpha: dark ? 0.04 : 0.1),
            Colors.white.withValues(alpha: dark ? 0.14 : 0.26),
          ],
        ).createShader(bounds),
    );
    canvas.drawRRect(
      pill.deflate(hairline / 2),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = hairline
        ..color = dark
            ? Colors.black.withValues(alpha: 0.4)
            : Color.lerp(
                colors.primary,
                Colors.black,
                0.35,
              )!.withValues(alpha: 0.4),
    );
  }

  @override
  bool shouldRepaint(_ProminentGlassPainter oldDelegate) =>
      oldDelegate.colors != colors ||
      oldDelegate.hairline != hairline ||
      oldDelegate.radius != radius;
}
