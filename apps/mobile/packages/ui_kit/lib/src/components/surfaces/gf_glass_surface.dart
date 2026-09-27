import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/material.dart';

/// Local cover-image glass for a single control. This surface adds no gesture
/// or semantics node: a menu or button child keeps ownership of its action.
/// The glass is static, including when reduced motion is requested.
class GfGlassSurface extends StatelessWidget {
  const GfGlassSurface({super.key, required this.child, this.size = 44})
    : assert(size > 0 && size < double.infinity);

  final Widget child;
  final double size;

  @override
  Widget build(BuildContext context) => SizedBox.square(
    dimension: math.max(44, size),
    child: ClipOval(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 12, sigmaY: 12),
        child: DecoratedBox(
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            // The dark scrim keeps white controls legible even on white covers.
            color: const Color(0x8C141B28),
            border: Border.all(color: const Color(0x42FFFFFF)),
          ),
          child: IconTheme.merge(
            data: const IconThemeData(color: Colors.white),
            child: child,
          ),
        ),
      ),
    ),
  );
}
