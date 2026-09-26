import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Shared relief for badge artwork. The rim catches light while the quieter
/// inset keeps the supplied icon legible in either theme. Interaction and
/// accessible naming belong to the enclosing badge control.
class GfBadgeMedallion extends StatelessWidget {
  const GfBadgeMedallion({
    super.key,
    required this.icon,
    required this.color,
    this.size = 56,
  }) : assert(size > 0 && size < double.infinity);

  final Widget icon;
  final Color color;
  final double size;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    Color tint(double opacity) =>
        Color.alphaBlend(color.withValues(alpha: opacity), colors.base100);
    // Server artwork may contain fixed dark strokes. Keep a soft enamel face
    // in dark mode instead of recoloring or losing those original SVG details.
    final face = dark
        ? Color.lerp(colors.base100, GfColors.light.base200, .90)!
        : colors.base100;
    Color faceTint(double opacity) =>
        Color.alphaBlend(color.withValues(alpha: opacity), face);
    final rim = tint(dark ? .24 : .16);
    final inset = size * .075;
    return SizedBox.square(
      dimension: size,
      child: DecoratedBox(
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              Color.alphaBlend(
                Colors.white.withValues(alpha: dark ? .16 : .65),
                rim,
              ),
              rim,
              tint(dark ? .30 : .24),
            ],
          ),
          border: Border.all(
            color: Color.alphaBlend(
              color.withValues(alpha: dark ? .34 : .26),
              colors.line,
            ),
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: dark ? .20 : .08),
              blurRadius: size * .08,
              offset: Offset(0, size * .035),
            ),
          ],
        ),
        child: Padding(
          padding: EdgeInsets.all(inset),
          child: DecoratedBox(
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [
                  faceTint(dark ? .10 : .15),
                  faceTint(dark ? .04 : .06),
                  Color.alphaBlend(
                    Colors.white.withValues(alpha: dark ? .12 : .40),
                    faceTint(dark ? .06 : .08),
                  ),
                ],
              ),
              border: Border.all(
                color: Colors.white.withValues(alpha: dark ? .36 : .70),
                width: .8,
              ),
            ),
            child: Center(
              child: IconTheme.merge(
                data: IconThemeData(size: size * .5, color: color),
                child: SizedBox.square(
                  dimension: size * .60,
                  child: Center(child: icon),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
