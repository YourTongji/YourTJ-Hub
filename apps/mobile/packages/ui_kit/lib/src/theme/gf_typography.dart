import 'package:flutter/material.dart';

/// Mobile type ramp. Twelve roles on fixed line heights replace ad-hoc sizes:
/// every UI string picks a role, and only weight or colour may vary on top.
///
/// | role       | size/line | weight | use                                   |
/// |------------|-----------|--------|---------------------------------------|
/// | display    | 28/36     | 700    | profile names, hero titles            |
/// | title1     | 22/30     | 700    | page titles in content                |
/// | title2     | 20/28     | 600    | dialog, sheet and empty-state titles  |
/// | title3     | 17/24     | 600    | navigation titles, card titles        |
/// | heading    | 16/24     | 600    | row titles, section headings          |
/// | body       | 16/24     | 400    | fields, rows, messages                |
/// | bodyStrong | 16/24     | 600    | emphasized body, author names         |
/// | small      | 15/22     | 400    | excerpts, descriptions                |
/// | subhead    | 14/20     | 400    | row subtitles, compact buttons        |
/// | caption    | 13/18     | 400    | metadata, timestamps, helper text     |
/// | meta       | 12/16     | 500    | badges, chips, tags                   |
/// | label      | 11/14     | 600    | counters, overlines                   |
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
    required this.subhead,
    required this.caption,
    required this.meta,
    required this.label,
  });

  final TextStyle display;

  final TextStyle title1;

  final TextStyle title2;

  final TextStyle title3;

  final TextStyle heading;

  final TextStyle body;

  final TextStyle bodyStrong;

  final TextStyle small;

  final TextStyle subhead;

  final TextStyle caption;

  final TextStyle meta;

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
      display: style(28, FontWeight.w700, height: 36 / 28),
      title1: style(22, FontWeight.w700, height: 30 / 22),
      title2: style(20, FontWeight.w600, height: 28 / 20),
      title3: style(17, FontWeight.w600, height: 24 / 17),
      heading: style(16, FontWeight.w600, height: 24 / 16),
      body: style(16, FontWeight.w400, height: 24 / 16),
      bodyStrong: style(16, FontWeight.w600, height: 24 / 16),
      small: style(15, FontWeight.w400, height: 22 / 15),
      subhead: style(14, FontWeight.w400, height: 20 / 14),
      caption: style(13, FontWeight.w400, height: 18 / 13),
      meta: style(12, FontWeight.w500, height: 16 / 12),
      label: style(11, FontWeight.w600, height: 14 / 11, letterSpacing: 0.2),
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
    TextStyle? subhead,
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
      subhead: subhead ?? this.subhead,
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
      subhead: TextStyle.lerp(subhead, other.subhead, t)!,
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
        other.subhead == subhead &&
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
    subhead,
    caption,
    meta,
    label,
  );
}
