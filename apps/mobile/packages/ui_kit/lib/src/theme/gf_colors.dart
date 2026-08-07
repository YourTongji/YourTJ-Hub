import 'package:flutter/material.dart';

/// Design-token color palette, mirrored 1:1 from
/// `apps/gooseforum/resource/src/styles/tokens.css` (oklch converted to sRGB
/// hex, fixed in `tokens.json`).
@immutable
class GfColors {
  const GfColors({
    required this.base100,
    required this.base200,
    required this.base300,
    required this.baseContent,
    required this.iconMuted,
    required this.line,
    required this.primary,
    required this.primaryContent,
    required this.secondary,
    required this.secondaryContent,
    required this.accent,
    required this.accentContent,
    required this.neutral,
    required this.neutralContent,
    required this.info,
    required this.infoContent,
    required this.success,
    required this.successContent,
    required this.warning,
    required this.warningContent,
    required this.error,
    required this.errorContent,
  });

  /// Surface base. `--gf-color-base-100`.
  final Color base100;

  /// Slightly raised surface / hover fill. `--gf-color-base-200`.
  final Color base200;

  /// Chip / pressed fill. `--gf-color-base-300`.
  final Color base300;

  /// Primary text on base surfaces. `--gf-color-base-content`.
  final Color baseContent;

  /// Muted icons. `--gf-color-icon-muted`.
  final Color iconMuted;

  /// Hairline borders / dividers. `--gf-color-line`.
  final Color line;

  /// Brand / interactive emphasis. `--gf-color-primary`.
  final Color primary;

  /// Text / icons on primary fills. `--gf-color-primary-content`.
  final Color primaryContent;

  /// Secondary surface (tabs, subtle fills). `--gf-color-secondary`.
  final Color secondary;

  /// Text on secondary fills. `--gf-color-secondary-content`.
  final Color secondaryContent;

  /// Accent highlight. `--gf-color-accent`.
  final Color accent;

  /// Text / icons on accent fills. `--gf-color-accent-content`.
  final Color accentContent;

  /// Inverted / active tab fill. `--gf-color-neutral`.
  final Color neutral;

  /// Text on neutral fills. `--gf-color-neutral-content`.
  final Color neutralContent;

  /// Informational status. `--gf-color-info`.
  final Color info;

  /// Text on info fills. `--gf-color-info-content`.
  final Color infoContent;

  /// Success status. `--gf-color-success`.
  final Color success;

  /// Text on success fills. `--gf-color-success-content`.
  final Color successContent;

  /// Warning status. `--gf-color-warning`.
  final Color warning;

  /// Text on warning fills. `--gf-color-warning-content`.
  final Color warningContent;

  /// Error / destructive status. `--gf-color-error`.
  final Color error;

  /// Text on error fills. `--gf-color-error-content`.
  final Color errorContent;

  /// All 22 color tokens keyed by their `tokens.css` names (without the
  /// `color-` prefix). Used by tests to keep `GfColors` in sync with
  /// `tokens.json`.
  Map<String, Color> get asMap => {
        'base-100': base100,
        'base-200': base200,
        'base-300': base300,
        'base-content': baseContent,
        'icon-muted': iconMuted,
        'line': line,
        'primary': primary,
        'primary-content': primaryContent,
        'secondary': secondary,
        'secondary-content': secondaryContent,
        'accent': accent,
        'accent-content': accentContent,
        'neutral': neutral,
        'neutral-content': neutralContent,
        'info': info,
        'info-content': infoContent,
        'success': success,
        'success-content': successContent,
        'warning': warning,
        'warning-content': warningContent,
        'error': error,
        'error-content': errorContent,
      };

  /// Light palette (`:root` in `tokens.css`).
  static const GfColors light = GfColors(
    base100: Color(0xFFFFFFFF),
    base200: Color(0xFFF8FAFC),
    base300: Color(0xFFF3F4F6),
    baseContent: Color(0xFF020618),
    iconMuted: Color(0xFF62748E),
    line: Color(0xFFE2E8F0),
    primary: Color(0xFF2563EB),
    primaryContent: Color(0xFFF8FAFC),
    secondary: Color(0xFFF1F5F9),
    secondaryContent: Color(0xFF0F172B),
    accent: Color(0xFF00BC7D),
    accentContent: Color(0xFF03130C),
    neutral: Color(0xFF0F172B),
    neutralContent: Color(0xFFF8FAFC),
    info: Color(0xFF3B82F6),
    infoContent: Color(0xFFEFF6FF),
    success: Color(0xFF00A63E),
    successContent: Color(0xFFF0FDF4),
    warning: Color(0xFFFE9A00),
    warningContent: Color(0xFFFFFBEB),
    error: Color(0xFFE7000B),
    errorContent: Color(0xFFFEF2F2),
  );

  /// Dark palette (`[data-theme="gf-dark"]` in `tokens.css`).
  static const GfColors dark = GfColors(
    base100: Color(0xFF101010),
    base200: Color(0xFF070707),
    base300: Color(0xFF1F1F1F),
    baseContent: Color(0xFFE6E6E6),
    iconMuted: Color(0xFFA8A8A8),
    line: Color(0xFF303030),
    primary: Color(0xFF439BF7),
    primaryContent: Color(0xFF070707),
    secondary: Color(0xFF1F1F1F),
    secondaryContent: Color(0xFFE6E6E6),
    accent: Color(0xFF00C950),
    accentContent: Color(0xFF070707),
    neutral: Color(0xFFE6E6E6),
    neutralContent: Color(0xFF070707),
    info: Color(0xFF58B1EB),
    infoContent: Color(0xFF070707),
    success: Color(0xFF00D593),
    successContent: Color(0xFF070707),
    warning: Color(0xFFFFB900),
    warningContent: Color(0xFF070707),
    error: Color(0xFFFF637E),
    errorContent: Color(0xFF070707),
  );

  /// Picks the palette for a [Brightness].
  static GfColors forBrightness(Brightness brightness) =>
      brightness == Brightness.dark ? dark : light;
}
