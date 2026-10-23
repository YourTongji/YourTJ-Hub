import 'package:flutter/material.dart';
import 'dart:math' as math;

import 'gf_colors.dart';

/// Mobile component geometry and derived colour roles.
///
/// The colour tokens in `tokens.json` stay a 1:1 mirror of the Web palette.
/// This file owns the mobile-only layer built on top of them: one spacing
/// grid, one radius scale, one set of control heights, text emphasis levels,
/// surface roles and elevations. Components read these instead of literal
/// numbers so that every surface, field and row shares the same rhythm.

/// 4pt spacing grid. Page gutters use [gutter]; groups are separated by
/// [section].
abstract final class GfSpace {
  static const double xxs = 2;
  static const double xs = 4;
  static const double sm = 8;
  static const double md = 12;
  static const double lg = 16;
  static const double xl = 20;
  static const double xxl = 24;
  static const double xxxl = 32;

  /// Horizontal page inset and inset-group margin.
  static const double gutter = 16;

  /// Vertical gap between grouped sections.
  static const double section = 24;
}

/// Radius scale. Nested shapes stay concentric: an inner radius equals the
/// outer radius minus the padding between them.
///
/// - [xs] static tags and metadata labels
/// - [sm] icon tiles, tooltips, small thumbnails
/// - [md] form fields, menu items, images inside cards, callouts
/// - [lg] cards, inset groups, menus, toasts, message bubbles
/// - [xl] dialogs and bottom sheets
/// - [full] buttons, chips, search, segmented controls, avatars
abstract final class GfRadius {
  static const double xs = 6;
  static const double sm = 10;
  static const double md = 14;
  static const double lg = 20;
  static const double xl = 28;
  static const double full = 999;
}

/// Painted control heights. Touch targets stay at least [touch] regardless
/// of the painted height.
abstract final class GfControl {
  static const double compact = 32;
  static const double regular = 40;
  static const double large = 48;

  /// Text fields and the full-width actions aligned with them.
  static const double field = 52;

  /// Minimum interactive target.
  static const double touch = 44;

  /// Single-line list row.
  static const double row = 52;
}

/// Glyph sizes on the shared 24px ReIcon grid.
abstract final class GfIconSize {
  static const double sm = 16;
  static const double md = 20;
  static const double lg = 24;
}

/// Shadow levels. Dark surfaces rely on their lighter fill and a hairline;
/// callers use [GfColorRoles.elevation] to pick the right treatment.
abstract final class GfElevation {
  static const Color _ink = Color(0xFF0F172A);

  /// Resting raised surface: segmented thumbs, emphasized cards.
  static final List<BoxShadow> low = <BoxShadow>[
    BoxShadow(
      color: _ink.withValues(alpha: 0.06),
      offset: const Offset(0, 1),
      blurRadius: 2,
    ),
    BoxShadow(
      color: _ink.withValues(alpha: 0.04),
      offset: const Offset(0, 2),
      blurRadius: 8,
    ),
  ];

  /// Floating UI: menus, toasts, floating bars and actions.
  static final List<BoxShadow> mid = <BoxShadow>[
    BoxShadow(
      color: _ink.withValues(alpha: 0.14),
      offset: const Offset(0, 10),
      blurRadius: 28,
      spreadRadius: -8,
    ),
    BoxShadow(
      color: _ink.withValues(alpha: 0.06),
      offset: const Offset(0, 2),
      blurRadius: 6,
    ),
  ];

  /// Modal surfaces above a scrim.
  static final List<BoxShadow> high = <BoxShadow>[
    BoxShadow(
      color: _ink.withValues(alpha: 0.24),
      offset: const Offset(0, 24),
      blurRadius: 56,
      spreadRadius: -16,
    ),
  ];
}

/// Semantic tone for badges, callouts, icon tiles and toasts.
enum GfTone { neutral, primary, info, success, warning, error }

/// Colour roles derived from the mirrored palette.
///
/// Text uses exactly four emphasis levels, fills are translucent so they read
/// the same on plain and grouped surfaces, and tones expose a soft background
/// plus a foreground that keeps text contrast on it.
extension GfColorRoles on GfColors {
  bool get isDark => base100.computeLuminance() < 0.5;

  // Text emphasis.
  Color get textPrimary => baseContent;
  Color get textSecondary => iconMuted;
  Color get textPlaceholder => iconMuted.withValues(alpha: isDark ? 0.7 : 0.78);
  Color get textDisabled => baseContent.withValues(alpha: 0.32);

  // Surfaces.
  /// Default page canvas (feeds, detail pages, chats).
  Color get canvas => base100;

  /// Canvas behind inset groups (settings, forms, profile editing).
  Color get canvasGrouped => isDark ? base200 : base300;

  /// Card and inset-group surface on [canvasGrouped].
  Color get surfaceGrouped => isDark ? base300 : base100;

  /// Raised surface for menus, sheets, dialogs and toasts.
  Color get surfaceRaised =>
      isDark ? Color.alphaBlend(const Color(0x14FFFFFF), base100) : base100;

  /// Quiet fill for fields, chips, tracks, incoming bubbles and secondary
  /// buttons. Translucent so it adapts to the surface beneath.
  Color get fill => baseContent.withValues(alpha: isDark ? 0.08 : 0.05);

  /// Pressed or emphasized fill.
  Color get fillStrong => baseContent.withValues(alpha: isDark ? 0.14 : 0.09);

  /// Hairline separators and outlines.
  Color get separator => line;

  /// Visible outline for outline buttons and selected cards.
  Color get outline => baseContent.withValues(alpha: isDark ? 0.2 : 0.14);

  /// Selected chip and soft primary backgrounds.
  Color get primarySoft => primary.withValues(alpha: isDark ? 0.18 : 0.1);

  /// Press overlay for content-coloured controls.
  Color get pressed => baseContent.withValues(alpha: isDark ? 0.1 : 0.06);

  /// Modal barrier.
  Color get scrim => Colors.black.withValues(alpha: isDark ? 0.6 : 0.36);

  /// Tone base colour.
  Color tone(GfTone tone) => switch (tone) {
    GfTone.neutral => iconMuted,
    GfTone.primary => primary,
    GfTone.info => info,
    GfTone.success => success,
    GfTone.warning => warning,
    GfTone.error => error,
  };

  /// Soft background for a tone.
  Color toneSoft(GfTone value) => value == GfTone.neutral
      ? fill
      : tone(value).withValues(alpha: isDark ? 0.18 : 0.1);

  /// Small status labels keep 4.5:1 contrast on the neutral surfaces that
  /// host them. Derive the foreground without altering shared palette tokens.
  Color toneOnSoft(GfTone value) {
    if (value == GfTone.neutral) return textSecondary;
    final base = tone(value);
    final surfaces = [base100, base200, base300, surfaceRaised];
    for (var step = 0; step <= 20; step++) {
      final foreground = Color.lerp(base, baseContent, step / 20)!;
      final readable = surfaces.every((surface) {
        final background = Color.alphaBlend(toneSoft(value), surface);
        final a = foreground.computeLuminance(),
            b = background.computeLuminance();
        return (math.max(a, b) + .05) / (math.min(a, b) + .05) >= 4.55;
      });
      if (readable) return foreground;
    }
    return baseContent;
  }

  /// Shadow for a raised surface; dark mode keeps surfaces flat.
  List<BoxShadow> elevation(List<BoxShadow> level) =>
      isDark ? const <BoxShadow>[] : level;
}

/// Hairline thickness: two device pixels, so separators stay crisp on every
/// density without reading as heavy rules.
double gfHairline(BuildContext context) {
  final double ratio = MediaQuery.maybeDevicePixelRatioOf(context) ?? 2;
  return (2 / ratio).clamp(0.5, 1.0);
}
