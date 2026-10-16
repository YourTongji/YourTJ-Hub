import 'package:flutter/material.dart';

import '../../twemoji/twemoji_text.dart';

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
    final text = GfText(
      data,
      style: style,
      maxLines: maxLines,
      overflow: overflow,
    );
    final hasTwemoji = parseTwemojiText(data).any((segment) => segment.isEmoji);
    if (colors.length < 2 || hasTwemoji) return text;
    return ShaderMask(
      blendMode: BlendMode.srcIn,
      shaderCallback: (bounds) =>
          LinearGradient(colors: colors).createShader(bounds),
      child: text,
    );
  }
}
