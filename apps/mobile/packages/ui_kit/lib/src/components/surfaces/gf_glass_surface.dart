import 'gf_liquid_surface.dart';

import 'package:flutter/material.dart';

/// Local cover-image glass for a single control. This surface adds no gesture
/// or semantics node: a menu or button child keeps ownership of its action.
/// Shares the optical material and accessibility fallback with app navigation.
class GfGlassSurface extends StatelessWidget {
  const GfGlassSurface({
    super.key,
    required this.child,
    this.size = 44,
    this.blurSigma = 12,
  }) : assert(size > 0 && size < double.infinity),
       assert(blurSigma >= 0 && blurSigma < double.infinity);

  final Widget child;
  final double size;
  final double blurSigma;

  @override
  Widget build(BuildContext context) => SizedBox.square(
    dimension: size < 44 ? 44 : size,
    child: GfLiquidSurface(
      radius: size,
      weight: GfGlassWeight.clear,
      blurSigma: blurSigma,
      pressable: true,
      child: IconTheme.merge(
        data: const IconThemeData(color: Colors.white),
        child: child,
      ),
    ),
  );
}
