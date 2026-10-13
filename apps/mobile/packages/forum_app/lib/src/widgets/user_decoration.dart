import '../asset_url.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:core/core.dart';
import 'package:flutter/material.dart';

List<Color> decorationColors(BuildContext context, DecorationStyle? style) {
  if (style == null || !style.isEnabled) return const [];
  final values = Theme.of(context).brightness == Brightness.dark
      ? style.darkColors
      : style.lightColors;
  if (values.isEmpty ||
      values.length > 2 ||
      values.any((c) => !RegExp(r'^#[0-9a-fA-F]{6}$').hasMatch(c))) {
    return const [];
  }
  final colors = values
      .map((c) => Color(int.parse(c.substring(1), radix: 16) | 0xff000000))
      .toList();
  if (style.kind == 'nicknameStyle') {
    final background = GfTheme.colorsOf(context).base100.computeLuminance();
    if (colors.any((color) {
      final luminance = color.computeLuminance();
      return (luminance > background
              ? (luminance + .05) / (background + .05)
              : (background + .05) / (luminance + .05)) <
          4.5;
    })) {
      return const [];
    }
  }
  // Flutter gradients require two stops; preserve the wire palette unchanged.
  return colors.length == 1 ? [colors.single, colors.single] : colors;
}

String decorationFrameAssetUrl(DecorationStyle? style) =>
    style?.isEnabled == true &&
        style?.kind == 'avatarFrame' &&
        style?.assetUrl != null &&
        RegExp(
          r'^/static/decorations/[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.(png|webp)$',
        ).hasMatch(style!.assetUrl!)
    ? resolveApiAssetUrl(style.assetUrl!)
    : '';
