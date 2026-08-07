import 'package:flutter/material.dart';

/// Typography scale, mirroring the stable font conventions of the web
/// design system (`apps/gooseforum/resource/src/styles/*.css`).
///
/// The web has no font-size tokens, but its conventions are stable:
/// page titles 20–24/w700, list/card titles 15–16/w500-600, body 15
/// leading-relaxed, excerpt 13, meta 12/11/10. [GfTypography] turns those
/// conventions into one reusable scale; colors are applied by the caller
/// (weak text uses `baseContent` with alpha, like web `base-content/55`).
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

  /// 22 / w700 — page hero titles (web `text-xl/2xl font-bold`).
  final TextStyle display;

  /// 20 / w700 — primary page titles (web `text-2xl font-bold`).
  final TextStyle title1;

  /// 18 / w700 — section titles.
  final TextStyle title2;

  /// 17 / w700 — profile names / sub-page titles.
  final TextStyle title3;

  /// 16 / w600 — card / list titles (web `text-base font-semibold`).
  final TextStyle heading;

  /// 15 / w400, line height 1.5 — body prose (web `text-[15px] leading-relaxed`).
  final TextStyle body;

  /// 15 / w600 — emphasized body (web `font-medium/semibold` on 15px rows).
  final TextStyle bodyStrong;

  /// 13 — excerpts / secondary text (web `text-[13px]`).
  final TextStyle small;

  /// 12 — metadata (web `text-xs`).
  final TextStyle caption;

  /// 11 — compact meta / badges (web `text-[11px]`).
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
      display: style(22, FontWeight.w700, height: 1.2),
      title1: style(20, FontWeight.w700, height: 1.25),
      title2: style(18, FontWeight.w700, height: 1.3),
      title3: style(17, FontWeight.w700, height: 1.3),
      heading: style(16, FontWeight.w600, height: 1.4),
      body: style(15, FontWeight.w400, height: 1.5),
      bodyStrong: style(15, FontWeight.w600, height: 1.4),
      small: style(13, FontWeight.w400, height: 1.4),
      caption: style(12, FontWeight.w400, height: 1.4),
      meta: style(11, FontWeight.w500, height: 1.3),
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
