import 'dart:convert';

import 'package:flutter/material.dart';

/// Renders the server-generated captcha image in the current app theme.
///
/// The server image uses a light background. Keep the light-theme pixels
/// unchanged and mirror the Web `invert(1) hue-rotate(180deg)` filter in dark
/// mode. Flutter's matrix translation column uses unnormalized 0..255 values.
class GfCaptchaImage extends StatelessWidget {
  const GfCaptchaImage({
    super.key,
    required this.imageData,
    this.width = 128,
    this.height = 48,
    this.fit = BoxFit.cover,
    this.gaplessPlayback = true,
  });

  /// The captcha image as either a data URL or a raw base64 payload.
  final String imageData;
  final double width;
  final double height;
  final BoxFit fit;
  final bool gaplessPlayback;

  /// CSS `invert(1) hue-rotate(180deg)` expressed for Flutter's color matrix.
  ///
  /// The last column is deliberately 255 rather than 1: Flutter documents
  /// this column as unnormalized 0..255 color space.
  static const List<double> darkColorMatrix = <double>[
    0.574,
    -1.43,
    -0.144,
    0,
    255,
    -0.426,
    -0.43,
    -0.144,
    0,
    255,
    -0.426,
    -1.43,
    0.856,
    0,
    255,
    0,
    0,
    0,
    1,
    0,
  ];

  /// Applies [darkColorMatrix] to one representative pixel for regression
  /// tests without relying on a renderer-only [ColorFilter] introspection API.
  static Color transformDarkPixel(Color color) {
    final List<double> m = darkColorMatrix;
    int channel(double value) => value.round().clamp(0, 255);
    final int alpha = (color.a * 255).round().clamp(0, 255);
    final double red = color.r * 255;
    final double green = color.g * 255;
    final double blue = color.b * 255;
    return Color.fromARGB(
      alpha,
      channel(m[0] * red + m[1] * green + m[2] * blue + m[4]),
      channel(m[5] * red + m[6] * green + m[7] * blue + m[9]),
      channel(m[10] * red + m[11] * green + m[12] * blue + m[14]),
    );
  }

  @override
  Widget build(BuildContext context) {
    final int comma = imageData.indexOf(',');
    final String encoded = comma >= 0
        ? imageData.substring(comma + 1)
        : imageData;
    final Widget image = Image.memory(
      base64Decode(encoded),
      width: width,
      height: height,
      fit: fit,
      gaplessPlayback: gaplessPlayback,
    );
    if (Theme.of(context).brightness != Brightness.dark) return image;
    return ColorFiltered(
      colorFilter: const ColorFilter.matrix(darkColorMatrix),
      child: image,
    );
  }
}
