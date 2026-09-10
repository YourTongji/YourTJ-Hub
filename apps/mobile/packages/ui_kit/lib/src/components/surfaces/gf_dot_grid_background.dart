import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Radially faded dot field used by the web login surface.
///
/// Mobile keeps the visual layer static: it preserves the web identity while
/// avoiding a perpetual ticker behind text fields and keyboard transitions.
class GfDotGridBackground extends StatelessWidget {
  const GfDotGridBackground({super.key});

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return IgnorePointer(
      child: LayoutBuilder(
        builder: (BuildContext context, BoxConstraints constraints) {
          const double spacing = 24;
          return RepaintBoundary(
            child: CustomPaint(
              size: Size(constraints.maxWidth, constraints.maxHeight),
              painter: _DotGridPainter(color: colors.primary, spacing: spacing),
            ),
          );
        },
      ),
    );
  }
}

class _DotGridPainter extends CustomPainter {
  const _DotGridPainter({required this.color, required this.spacing});

  final Color color;
  final double spacing;

  @override
  void paint(Canvas canvas, Size size) {
    final Paint paint = Paint()..style = PaintingStyle.fill;
    final Offset center = Offset(size.width * 0.5, size.height * 0.44);
    final double fadeRadius = size.longestSide * 0.66;
    for (double y = 0; y <= size.height; y += spacing) {
      for (double x = 0; x <= size.width; x += spacing) {
        final Offset point = Offset(x, y);
        final double progress = (1 - (point - center).distance / fadeRadius)
            .clamp(0.0, 1.0);
        final double opacity = 0.04 + progress * 0.14;
        paint.color = color.withValues(alpha: opacity);
        canvas.drawCircle(point, 1.45, paint);
      }
    }
  }

  @override
  bool shouldRepaint(covariant _DotGridPainter oldDelegate) {
    return oldDelegate.color != color || oldDelegate.spacing != spacing;
  }
}
