import 'package:flutter/material.dart';

import 'gf_icon_button.dart';
import 'surfaces/gf_glass_surface.dart';

/// A named white icon action on locally clipped cover-image glass. Reuses the
/// shared button's focus, disabled state, tooltip and merged accessible label.
class GfGlassIconButton extends StatelessWidget {
  const GfGlassIconButton({
    super.key,
    required this.symbol,
    required this.tooltip,
    required this.onPressed,
    this.size = 44,
  }) : assert(size > 0 && size < double.infinity);

  final String symbol;
  final String tooltip;
  final VoidCallback? onPressed;
  final double size;

  @override
  Widget build(BuildContext context) => GfGlassSurface(
    size: size,
    child: GfIconButton(
      symbol: symbol,
      tooltip: tooltip,
      onPressed: onPressed,
      size: size,
      color: Colors.white,
    ),
  );
}
