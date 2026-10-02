import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';

@immutable
class ShareImageTheme {
  const ShareImageTheme(this.id, this.label, this.colors, {this.accentColor});

  final String id;
  final String label;
  final GfColors colors;
  final Color? accentColor;

  /// The palette's signature hue (swatches and decorative accents). Pastel
  /// themes keep the shared `primary` for links/actions, so their identity
  /// comes from the tint source instead.
  Color get accent => accentColor ?? colors.primary;

  Brightness get brightness => colors.base100.computeLuminance() < 0.5
      ? Brightness.dark
      : Brightness.light;

  /// Pastels are mobile-only GF-token variations; no Web theme parity is implied.
  /// This is called once per sheet so labels follow the active locale.
  static List<ShareImageTheme> all(AppLocalizations l10n) => [
    // 纸白的标志色取石板灰（像牛皮纸胶带），与晴蓝的蓝色分得开。
    ShareImageTheme(
      'paper',
      l10n.shareImageThemePaper,
      GfColors.light,
      accentColor: GfColors.light.iconMuted,
    ),
    ShareImageTheme(
      'sand',
      l10n.shareImageThemeSand,
      _tinted(GfColors.light.warning),
      accentColor: GfColors.light.warning,
    ),
    ShareImageTheme(
      'blue',
      l10n.shareImageThemeBlue,
      _tinted(GfColors.light.info),
      accentColor: GfColors.light.info,
    ),
    ShareImageTheme(
      'mint',
      l10n.shareImageThemeMint,
      _tinted(GfColors.light.success),
      accentColor: GfColors.light.success,
    ),
    ShareImageTheme('dark', l10n.shareImageThemeDark, GfColors.dark),
  ];

  static GfColors _tinted(Color tint) {
    final base = GfColors.light;
    Color blend(Color surface, double amount) =>
        Color.alphaBlend(tint.withValues(alpha: amount), surface);
    return GfColors(
      base100: blend(base.base100, .035),
      base200: blend(base.base200, .075),
      base300: blend(base.base300, .12),
      baseContent: base.baseContent,
      iconMuted: base.iconMuted,
      line: blend(base.line, .2),
      // Keep link and action contrast consistent across the pastel surfaces.
      primary: base.primary,
      primaryContent: base.primaryContent,
      secondary: blend(base.secondary, .075),
      secondaryContent: base.secondaryContent,
      accent: base.accent,
      accentContent: base.accentContent,
      neutral: base.neutral,
      neutralContent: base.neutralContent,
      info: base.info,
      infoContent: base.infoContent,
      success: base.success,
      successContent: base.successContent,
      warning: base.warning,
      warningContent: base.warningContent,
      error: base.error,
      errorContent: base.errorContent,
    );
  }
}
