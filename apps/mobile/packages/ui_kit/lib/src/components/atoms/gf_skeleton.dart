import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Skeleton loading placeholder, mirroring the muted base-300 fill used for
/// web loading states. No dedicated web class exists; this is the mobile
/// loading primitive used by list rows and cards.
class GfSkeleton extends StatelessWidget {
  const GfSkeleton({super.key, this.width, this.height = 16, this.radius = 8});

  final double? width;
  final double height;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Container(
      width: width,
      height: height,
      decoration: BoxDecoration(
        color: colors.base300,
        borderRadius: BorderRadius.circular(radius),
      ),
    );
  }
}
