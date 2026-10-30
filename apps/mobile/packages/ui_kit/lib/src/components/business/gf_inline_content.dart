import 'dart:math' as math;

import 'package:flutter/widgets.dart';

/// A text controller whose lines can hold inline content taller than its
/// text, such as sticker images drawn at a multiple of the font size.
///
/// [EditableText] pins every line to the text's strut by default, so taller
/// inline content would spill over the lines above and below. Composers that
/// receive such a controller let its lines grow instead, and size their
/// one-line compact state to show a whole line.
abstract interface class GfInlineContentController {
  /// Height of the tallest inline element, in multiples of the font size;
  /// null while the text holds none.
  double? get inlineExtentScale;

  /// Lines grow to fit inline content; plain lines keep the text height.
  static const StrutStyle strut = StrutStyle(forceStrutHeight: false);

  /// The strut a field showing [controller] needs; null keeps the default.
  static StrutStyle? strutFor(TextEditingController? controller) =>
      controller is GfInlineContentController ? strut : null;

  /// Line height, in multiples of the font size, that shows one whole line of
  /// [controller]'s text when its lines are [textLineHeight] tall.
  static double lineScale(
    TextEditingController? controller,
    double textLineHeight,
  ) {
    final inline = switch (controller) {
      GfInlineContentController(:final inlineExtentScale) => inlineExtentScale,
      _ => null,
    };
    return inline == null ? textLineHeight : math.max(textLineHeight, inline);
  }
}
