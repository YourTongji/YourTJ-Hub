import 'package:flutter/material.dart';

import 'gf_theme.dart';

/// Typography and geometry for long-form rich content: post Markdown, the
/// server-rendered Wiki HTML and course reviews.
///
/// Every size is a logical-pixel design baseline derived from [GfTypography].
/// The system [TextScaler] is applied by Flutter at paint time and is never
/// replaced or clamped here; a reader preference multiplies the baseline
/// before that, so the effective size is `baseline × userScale` and the system
/// scaler still applies on top of it.
@immutable
class GfRichContentTypography {
  const GfRichContentTypography({
    required this.body,
    required this.h1,
    required this.h2,
    required this.h3,
    required this.h4,
    required this.h5,
    required this.h6,
    required this.inlineCode,
    required this.code,
    required this.tableHeader,
    required this.tableBody,
    required this.quote,
    required this.paragraphSpacing,
    required this.blockSpacing,
    required this.listIndent,
    required this.listSpacing,
    required this.quoteSide,
    required this.quotePadding,
    required this.codePadding,
    required this.tableCellPadding,
  });

  /// Reading body text (`GfTypography.body` at 100%).
  final TextStyle body;

  final TextStyle h1;
  final TextStyle h2;
  final TextStyle h3;
  final TextStyle h4;
  final TextStyle h5;
  final TextStyle h6;

  /// Inline `` `code` ``.
  final TextStyle inlineCode;

  /// Fenced/indented code block content (monospace, never wrapped).
  ///
  /// Intentionally colorless: highlight token styles are merged *over* this
  /// style, so a color here would flatten every token into one hue. Renderers
  /// that show code without highlighting add `GfColors.baseContent` themselves.
  final TextStyle code;

  final TextStyle tableHeader;
  final TextStyle tableBody;

  /// Blockquote text (muted body).
  final TextStyle quote;

  /// Vertical margin between consecutive paragraphs/lines.
  final double paragraphSpacing;

  /// Vertical margin around block-level rich content (code, quote, table).
  final double blockSpacing;

  /// Left margin for list contents.
  final double listIndent;

  /// Vertical margin below each list item.
  final double listSpacing;

  /// Blockquote leading rule thickness.
  final double quoteSide;

  final EdgeInsets quotePadding;
  final EdgeInsets codePadding;
  final EdgeInsets tableCellPadding;

  /// Reading baseline; the same value as [GfTypography.body].
  static const double readingBodySize = 17;

  /// Course-review baseline: the same profile, one step more compact.
  static const double compactBodySize = 15.5;

  /// Reader preference bounds (80%–140%), independent from system scaling.
  static const double minUserScale = .8;
  static const double maxUserScale = 1.4;

  /// Heading size ratios (H1..H4) applied to the body baseline.
  static const List<double> headingRatios = <double>[1.45, 1.30, 1.18, 1.08];

  /// Monospace family; falls back to platform monospace if unavailable.
  static const String codeFontFamily = 'monospace';

  /// Clamps a persisted reader preference into the supported range.
  static double clampUserScale(double value) =>
      value.clamp(minUserScale, maxUserScale);

  /// Marker box for one list level.
  ///
  /// Each nesting level subtracts this box again, so a fixed 32px indent
  /// starves nested items once system text doubles on a 320px line. The box
  /// therefore shrinks with the system scale, clamped to 16–32px.
  static double listIndentFor(double systemScale) =>
      (32 / systemScale.clamp(1, 2)).clamp(16, 32);

  /// The profile for the current theme tokens, scaled by the reader
  /// preference. [compact] selects the course-review baseline.
  static GfRichContentTypography of(
    BuildContext context, {
    double userScale = 1,
    bool compact = false,
  }) {
    // Local scale factor of the system text scaler at body size; nonlinear
    // scalers are supported because this only sizes a marker box.
    final TextScaler scaler = MediaQuery.textScalerOf(context);
    return GfRichContentTypography.standard(
      typography: GfTheme.typographyOf(context),
      colors: GfTheme.colorsOf(context),
      bodySize: compact ? compactBodySize : readingBodySize,
      userScale: userScale,
      listIndent: listIndentFor(scaler.scale(readingBodySize) / readingBodySize),
    );
  }

  /// Derives the profile from design-system tokens so no surface hand-writes
  /// font sizes. [bodySize] selects the reading or compact baseline.
  factory GfRichContentTypography.standard({
    required GfTypography typography,
    required GfColors colors,
    double bodySize = readingBodySize,
    double userScale = 1,
    double listIndent = 32,
  }) {
    final double scale = clampUserScale(userScale);
    final Color muted = colors.baseContent.withValues(alpha: .75);
    final Color faint = colors.baseContent.withValues(alpha: .6);

    TextStyle text(
      double base, {
      FontWeight weight = FontWeight.w400,
      double? height,
      Color? color,
      String? family,
    }) => TextStyle(
      fontSize: base * scale,
      fontWeight: weight,
      height: height,
      color: color ?? colors.baseContent,
      fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
      fontFamily: family,
      fontFamilyFallback: family == null
          ? null
          : const <String>['monospace', 'Menlo', 'Courier'],
    );

    double heading(int index) => bodySize * headingRatios[index];
    final double codeSize = bodySize - 1;

    return GfRichContentTypography(
      body: text(bodySize, height: typography.body.height),
      h1: text(heading(0), weight: FontWeight.w700, height: 1.25),
      h2: text(heading(1), weight: FontWeight.w700, height: 1.3),
      h3: text(heading(2), weight: FontWeight.w600, height: 1.35),
      h4: text(heading(3), weight: FontWeight.w600, height: 1.4),
      h5: text(bodySize, weight: FontWeight.w600, height: 1.4),
      h6: text(bodySize, weight: FontWeight.w600, height: 1.4, color: faint),
      inlineCode: text(codeSize, color: colors.error, family: codeFontFamily),
      // Colorless on purpose (see the field docs): `copyWith` cannot clear a
      // color, so the token base style is built without one.
      code: TextStyle(
        fontSize: codeSize * scale,
        height: 1.5,
        fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
        fontFamily: codeFontFamily,
        fontFamilyFallback: const <String>['monospace', 'Menlo', 'Courier'],
      ),
      tableHeader: text(bodySize, weight: FontWeight.w600, height: 1.4),
      tableBody: text(bodySize, height: typography.body.height),
      quote: text(bodySize, color: muted, height: typography.body.height),
      paragraphSpacing: 3,
      blockSpacing: 8,
      listIndent: listIndent,
      listSpacing: 4,
      quoteSide: 4,
      quotePadding: const EdgeInsets.fromLTRB(12, 4, 12, 4),
      codePadding: const EdgeInsets.all(12),
      tableCellPadding: const EdgeInsets.fromLTRB(8, 6, 8, 6),
    );
  }

  @override
  bool operator ==(Object other) {
    if (other is! GfRichContentTypography) return false;
    return other.body == body &&
        other.h1 == h1 &&
        other.h2 == h2 &&
        other.h3 == h3 &&
        other.h4 == h4 &&
        other.h5 == h5 &&
        other.h6 == h6 &&
        other.inlineCode == inlineCode &&
        other.code == code &&
        other.tableHeader == tableHeader &&
        other.tableBody == tableBody &&
        other.quote == quote &&
        other.paragraphSpacing == paragraphSpacing &&
        other.blockSpacing == blockSpacing &&
        other.listIndent == listIndent &&
        other.listSpacing == listSpacing &&
        other.quoteSide == quoteSide &&
        other.quotePadding == quotePadding &&
        other.codePadding == codePadding &&
        other.tableCellPadding == tableCellPadding;
  }

  @override
  int get hashCode => Object.hashAll(<Object>[
    body,
    h1,
    h2,
    h3,
    h4,
    h5,
    h6,
    inlineCode,
    code,
    tableHeader,
    tableBody,
    quote,
    paragraphSpacing,
    blockSpacing,
    listIndent,
    listSpacing,
    quoteSide,
    quotePadding,
    codePadding,
    tableCellPadding,
  ]);
}
