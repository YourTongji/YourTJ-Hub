import 'package:flutter/material.dart';
import '../theme/gf_theme.dart';
import 'gf_symbol.dart';

/// Lucide marks shared with Web, placed on a soft semantic colour surface.
class GfIconTile extends StatelessWidget {
  const GfIconTile(this.symbol, {super.key, this.color, this.size = 36});
  final String symbol;
  final Color? color;
  final double size;
  @override
  Widget build(BuildContext context) {
    final tone = color ?? GfTheme.colorsOf(context).primary;
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: tone.withValues(alpha: 0.10),
        borderRadius: BorderRadius.circular(11),
      ),
      alignment: Alignment.center,
      child: GfSymbol(symbol, size: size * 0.55, color: tone),
    );
  }
}
