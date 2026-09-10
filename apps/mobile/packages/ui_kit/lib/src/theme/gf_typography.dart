import 'package:flutter/material.dart';

/// Mobile typography using the Web design system's hierarchy and weights.
/// Body and secondary text are enlarged for handheld reading; tighter leading
/// keeps feeds compact without shrinking the text or overriding system scaling.
@immutable
class GfTypography extends ThemeExtension<GfTypography> {
  const GfTypography({
    required this.display,
    required this.title1,
    required this.title2,
    required this.title3,
    required this.heading,
    required this.body,
    required this.bodyStrong,
    required this.small,
    required this.caption,
    required this.meta,
    required this.label,
  });

  /// 24 / 32 / w700 — page hero titles (web `text-xl/2xl font-bold`).
  final TextStyle display;

  /// 24 / 32 / w700 — primary page titles (web `text-2xl font-bold`).
  final TextStyle title1;

  /// 18 / w700 — section titles.
  final TextStyle title2;

  /// 17 / w700 — profile names / sub-page titles.
  final TextStyle title3;

  /// 16 / 24 / w700 — card / list titles (web `text-base font-semibold`).
  final TextStyle heading;

  /// 17 / w400, line height 1.4 — readable mobile prose in compact rows.
  final TextStyle body;

  /// 16 / w600 — emphasized body and author names.
  final TextStyle bodyStrong;

  /// 15 / 21 — excerpts / secondary text.
  final TextStyle small;

  /// 13 — metadata.
  final TextStyle caption;

  /// 12 — compact meta / badges.
  final TextStyle meta;

  /// 10 / w700 / tracking-wide — uppercase group labels (web
  /// `text-[10px] font-bold uppercase tracking-wide`). Callers apply
  /// `.toUpperCase()` to the text.
  final TextStyle label;

  /// Builds the scale for a surface text color (typically `baseContent`).
  factory GfTypography.standard(Color baseContent) {
    TextStyle style(
      double size,
      FontWeight weight, {
      double? height,
      double letterSpacing = 0,
    }) {
      return TextStyle(
        fontSize: size,
        fontWeight: weight,
        height: height,
        letterSpacing: letterSpacing,
        color: baseContent,
        fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
      );
    }

    return GfTypography(
      display: style(24, FontWeight.w700, height: 32 / 24),
      title1: style(24, FontWeight.w700, height: 32 / 24),
      title2: style(18, FontWeight.w700, height: 26 / 18),
      title3: style(17, FontWeight.w700, height: 1.3),
      heading: style(16, FontWeight.w700, height: 1.5),
      body: style(17, FontWeight.w400, height: 1.4),
      bodyStrong: style(16, FontWeight.w600, height: 1.4),
      small: style(15, FontWeight.w400, height: 1.4),
      caption: style(13, FontWeight.w400, height: 1.5),
      meta: style(12, FontWeight.w500, height: 1.3),
      label: style(10, FontWeight.w700, height: 1.2, letterSpacing: 0.5),
    );
  }

  @override
  GfTypography copyWith({
    TextStyle? display,
    TextStyle? title1,
    TextStyle? title2,
    TextStyle? title3,
    TextStyle? heading,
    TextStyle? body,
    TextStyle? bodyStrong,
    TextStyle? small,
    TextStyle? caption,
    TextStyle? meta,
    TextStyle? label,
  }) {
    return GfTypography(
      display: display ?? this.display,
      title1: title1 ?? this.title1,
      title2: title2 ?? this.title2,
      title3: title3 ?? this.title3,
      heading: heading ?? this.heading,
      body: body ?? this.body,
      bodyStrong: bodyStrong ?? this.bodyStrong,
      small: small ?? this.small,
      caption: caption ?? this.caption,
      meta: meta ?? this.meta,
      label: label ?? this.label,
    );
  }

  @override
  GfTypography lerp(GfTypography? other, double t) {
    if (other == null) return this;
    return GfTypography(
      display: TextStyle.lerp(display, other.display, t)!,
      title1: TextStyle.lerp(title1, other.title1, t)!,
      title2: TextStyle.lerp(title2, other.title2, t)!,
      title3: TextStyle.lerp(title3, other.title3, t)!,
      heading: TextStyle.lerp(heading, other.heading, t)!,
      body: TextStyle.lerp(body, other.body, t)!,
      bodyStrong: TextStyle.lerp(bodyStrong, other.bodyStrong, t)!,
      small: TextStyle.lerp(small, other.small, t)!,
      caption: TextStyle.lerp(caption, other.caption, t)!,
      meta: TextStyle.lerp(meta, other.meta, t)!,
      label: TextStyle.lerp(label, other.label, t)!,
    );
  }

  @override
  bool operator ==(Object other) {
    if (other is! GfTypography) return false;
    return other.display == display &&
        other.title1 == title1 &&
        other.title2 == title2 &&
        other.title3 == title3 &&
        other.heading == heading &&
        other.body == body &&
        other.bodyStrong == bodyStrong &&
        other.small == small &&
        other.caption == caption &&
        other.meta == meta &&
        other.label == label;
  }

  @override
  int get hashCode => Object.hash(
    display,
    title1,
    title2,
    title3,
    heading,
    body,
    bodyStrong,
    small,
    caption,
    meta,
    label,
  );
}
