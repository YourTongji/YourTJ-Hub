import 'package:flutter/material.dart';

/// A text gradient preserves the Text widget's selection and accessibility semantics.
class GfDecoratedName extends StatelessWidget {
  const GfDecoratedName(
    this.data, {
    super.key,
    this.colors = const [],
    this.style,
    this.maxLines,
    this.overflow,
  });
  final String data;
  final List<Color> colors;
  final TextStyle? style;
  final int? maxLines;
  final TextOverflow? overflow;
  @override
  Widget build(BuildContext context) {
    final text = Text(
      data,
      style: style,
      maxLines: maxLines,
      overflow: overflow,
    );
    if (colors.length < 2) return text;
    return ShaderMask(
      blendMode: BlendMode.srcIn,
      shaderCallback: (bounds) =>
          LinearGradient(colors: colors).createShader(bounds),
      child: text,
    );
  }
}
