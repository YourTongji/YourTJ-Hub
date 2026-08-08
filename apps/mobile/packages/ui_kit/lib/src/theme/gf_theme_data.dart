import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart';

import '../components/gf_motion.dart';
import 'gf_colors.dart';
import 'gf_theme_extensions.dart';
import 'gf_shadows.dart';
import 'gf_typography.dart';

/// Builds the yourtj `ThemeData` for a [Brightness], wiring the Gf design
/// tokens into Material widgets (color scheme, shapes, typography, motion).
///
/// The Gf tokens remain reachable through `Theme.of(context).extension<T>()`
/// (see [GfTheme]).
ThemeData gfThemeData(Brightness brightness) {
  final GfColors colors = GfColors.forBrightness(brightness);
  final ThemeData tdesignTheme = _buildTDesignTheme(colors, brightness);

  final ColorScheme colorScheme = ColorScheme(
    brightness: brightness,
    primary: colors.primary,
    onPrimary: colors.primaryContent,
    secondary: colors.secondary,
    onSecondary: colors.secondaryContent,
    tertiary: colors.accent,
    onTertiary: colors.accentContent,
    error: colors.error,
    onError: colors.errorContent,
    surface: colors.base100,
    onSurface: colors.baseContent,
    surfaceContainerLow: colors.base200,
    surfaceContainer: colors.base200,
    surfaceContainerHigh: colors.base300,
    surfaceContainerHighest: colors.base300,
    surfaceDim: colors.base200,
    surfaceBright: colors.base100,
    onSurfaceVariant: colors.iconMuted,
    outline: colors.line,
    outlineVariant: colors.line,
    shadow: colors.neutral,
    scrim: colors.neutral.withValues(alpha: 0.4),
    inverseSurface: colors.neutral,
    onInverseSurface: colors.neutralContent,
    inversePrimary: colors.primary,
    surfaceTint: colors.primary,
  );

  const OutlineInputBorder noBorder = OutlineInputBorder(
    borderSide: BorderSide.none,
  );

  final GfTypography typography = GfTypography.standard(colors.baseContent);
  final List<ThemeExtension<dynamic>> extensions = tdesignTheme
      .extensions
      .values
      .toList(growable: true);
  extensions
    ..removeWhere((ThemeExtension<dynamic> item) => item is TInputThemeData)
    ..add(const TInputThemeData(showClearButton: false))
    ..add(GfRadii.standard)
    ..add(GfBorders.standard)
    ..add(GfSizes.standard)
    ..add(GfShadows.standard)
    ..add(typography);

  return tdesignTheme.copyWith(
    brightness: brightness,
    colorScheme: colorScheme,
    // TDesign mobile pages use a quiet page canvas and raised white/black
    // content surfaces. This also makes rows, cards and dialogs visually
    // distinct without inventing a second set of brand tokens.
    scaffoldBackgroundColor: colors.base200,
    canvasColor: colors.base100,
    dividerColor: colors.line,
    extensions: extensions,
    // Navigation bars keep the web header's flat, strong-title treatment in
    // a mobile-native 56px height.
    appBarTheme: AppBarTheme(
      backgroundColor: colors.base100,
      foregroundColor: colors.baseContent,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      toolbarHeight: 56,
      titleTextStyle: TextStyle(
        fontSize: 20,
        fontWeight: FontWeight.w700,
        color: colors.baseContent,
      ),
    ),
    // Retain Material fallbacks for third-party or legacy leaf widgets. The
    // application shell itself is rendered by TDesign-backed Gf wrappers.
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: colors.base100,
      surfaceTintColor: Colors.transparent,
      indicatorColor: colors.base300,
      height: 64,
      labelTextStyle: WidgetStatePropertyAll(
        TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          color: colors.baseContent,
        ),
      ),
      iconTheme: WidgetStateProperty.resolveWith((Set<WidgetState> states) {
        final bool selected = states.contains(WidgetState.selected);
        return IconThemeData(
          size: 24,
          color: selected ? colors.primary : colors.iconMuted,
        );
      }),
    ),
    listTileTheme: ListTileThemeData(
      iconColor: colors.iconMuted,
      textColor: colors.baseContent,
      contentPadding: const EdgeInsets.symmetric(horizontal: 16),
      minVerticalPadding: 12,
      titleTextStyle: TextStyle(fontSize: 15, color: colors.baseContent),
      subtitleTextStyle: TextStyle(
        fontSize: 13,
        color: colors.baseContent.withValues(alpha: 0.55),
      ),
    ),
    dividerTheme: DividerThemeData(
      color: colors.line,
      thickness: GfBorders.standard.width,
      space: 1,
    ),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: colors.base100,
      contentTextStyle: TextStyle(fontSize: 14, color: colors.baseContent),
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadii.standard.box),
        side: BorderSide(color: colors.line, width: GfBorders.standard.width),
      ),
      behavior: SnackBarBehavior.floating,
    ),
    switchTheme: SwitchThemeData(
      thumbColor: WidgetStateProperty.resolveWith((Set<WidgetState> states) {
        return states.contains(WidgetState.selected)
            ? colors.primaryContent
            : colors.baseContent.withValues(alpha: 0.6);
      }),
      trackColor: WidgetStateProperty.resolveWith((Set<WidgetState> states) {
        return states.contains(WidgetState.selected)
            ? colors.primary
            : colors.base300;
      }),
      trackOutlineColor: WidgetStatePropertyAll(Colors.transparent),
    ),
    // Buttons follow gf-button-* semantics (see GfButton).
    filledButtonTheme: FilledButtonThemeData(
      style: ButtonStyle(
        backgroundColor: WidgetStatePropertyAll(colors.primary),
        foregroundColor: WidgetStatePropertyAll(colors.primaryContent),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(GfRadii.standard.field),
          ),
        ),
        textStyle: WidgetStatePropertyAll(
          TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
        ),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: ButtonStyle(
        foregroundColor: WidgetStatePropertyAll(colors.baseContent),
        side: WidgetStatePropertyAll(
          BorderSide(color: colors.line, width: GfBorders.standard.width),
        ),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(GfRadii.standard.field),
          ),
        ),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: ButtonStyle(
        foregroundColor: WidgetStatePropertyAll(colors.primary),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(GfRadii.standard.field),
          ),
        ),
      ),
    ),
    // Inputs follow gf-radius-field semantics.
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: colors.base100,
      border: noBorder,
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(GfRadii.standard.field),
        borderSide: BorderSide(
          color: colors.line,
          width: GfBorders.standard.width,
        ),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(GfRadii.standard.field),
        borderSide: BorderSide(color: colors.primary, width: 1.5),
      ),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: colors.base300,
      labelStyle: TextStyle(
        color: colors.baseContent.withValues(alpha: 0.55),
        fontSize: 11,
        fontWeight: FontWeight.w500,
      ),
      side: BorderSide.none,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadii.standard.selector),
      ),
    ),
    // Cards follow gf-card semantics: mobile default has no border (the
    // hairline bottom divider is drawn by GfCard itself); `emphasized`
    // restores the desktop border+radius look.
    cardTheme: CardThemeData(
      color: colors.base100,
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadii.standard.box),
      ),
    ),
    // Dialogs / bottom sheets follow gf-floating-surface semantics.
    dialogTheme: DialogThemeData(
      backgroundColor: colors.base100,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadii.standard.box),
        side: BorderSide(color: colors.line, width: GfBorders.standard.width),
      ),
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: colors.base100,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(
          top: Radius.circular(GfRadii.standard.box),
        ),
      ),
    ),
    // Motion mirrors `resource/src/runtime/motion.ts` (see GfMotion):
    // page transitions use the standard 0.22s ease (web page-enter motion).
    pageTransitionsTheme: PageTransitionsTheme(
      builders: <TargetPlatform, PageTransitionsBuilder>{
        TargetPlatform.android: const GfPageTransitionsBuilder(),
        TargetPlatform.iOS: const GfPageTransitionsBuilder(),
      },
    ),
    splashFactory: InkSparkle.splashFactory,
    textTheme: TextTheme(
      displaySmall: typography.display,
      headlineMedium: typography.title1,
      headlineSmall: typography.title2,
      titleLarge: typography.title3,
      titleMedium: typography.heading,
      titleSmall: typography.bodyStrong,
      bodyLarge: typography.body,
      bodyMedium: typography.small,
      bodySmall: typography.caption,
      labelLarge: typography.bodyStrong.copyWith(fontSize: 14),
      labelMedium: typography.meta,
      labelSmall: typography.label,
    ),
  );
}

/// Builds TDesign's complete component-extension graph from the authoritative
/// YourTJ palette. TDesign remains an implementation detail: web-synchronised
/// Gf tokens own the colours, radii and spacing, while TDesign supplies the
/// mobile component behaviour and visual grammar.
ThemeData _buildTDesignTheme(GfColors colors, Brightness brightness) {
  Color blend(Color foreground, Color background, double opacity) {
    return Color.alphaBlend(foreground.withValues(alpha: opacity), background);
  }

  final TThemeData token = TThemeData.defaultData().copyWithTThemeData(
    'yourtj-${brightness.name}',
    colorMap: <String, Color>{
      'brandColor1': blend(colors.primary, colors.base100, 0.08),
      'brandColor2': blend(colors.primary, colors.base100, 0.14),
      'brandColor3': blend(colors.primary, colors.base100, 0.28),
      'brandColor4': blend(colors.primary, colors.base100, 0.45),
      'brandColor5': blend(colors.primary, colors.base100, 0.70),
      'brandColor6': colors.primary,
      'brandColor7': colors.primary,
      'brandColor8': blend(colors.primary, colors.neutral, 0.82),
      'brandLightColor': blend(colors.primary, colors.base100, 0.10),
      'brandFocusColor': blend(colors.primary, colors.base100, 0.16),
      'brandDisabledColor': blend(colors.primary, colors.base100, 0.30),
      'brandHoverColor': colors.primary,
      'brandNormalColor': colors.primary,
      'brandClickColor': blend(colors.primary, colors.neutral, 0.82),
      'errorLightColor': blend(colors.error, colors.base100, 0.10),
      'errorFocusColor': blend(colors.error, colors.base100, 0.16),
      'errorDisabledColor': blend(colors.error, colors.base100, 0.30),
      'errorHoverColor': colors.error,
      'errorNormalColor': colors.error,
      'errorClickColor': blend(colors.error, colors.neutral, 0.82),
      'warningLightColor': blend(colors.warning, colors.base100, 0.10),
      'warningFocusColor': blend(colors.warning, colors.base100, 0.16),
      'warningDisabledColor': blend(colors.warning, colors.base100, 0.30),
      'warningHoverColor': colors.warning,
      'warningNormalColor': colors.warning,
      'warningClickColor': blend(colors.warning, colors.neutral, 0.82),
      'successLightColor': blend(colors.success, colors.base100, 0.10),
      'successFocusColor': blend(colors.success, colors.base100, 0.16),
      'successDisabledColor': blend(colors.success, colors.base100, 0.30),
      'successHoverColor': colors.success,
      'successNormalColor': colors.success,
      'successClickColor': blend(colors.success, colors.neutral, 0.82),
      'whiteColor1': colors.base100,
      'fontGyColor1': colors.baseContent,
      'fontGyColor2': colors.baseContent.withValues(alpha: 0.72),
      'fontGyColor3': colors.baseContent.withValues(alpha: 0.48),
      'fontGyColor4': colors.baseContent.withValues(alpha: 0.32),
      'fontWhColor1': colors.primaryContent,
      'fontWhColor2': colors.primaryContent.withValues(alpha: 0.72),
      'bgColorPage': colors.base200,
      'bgColorContainer': colors.base100,
      'bgColorContainerSelect': colors.base100,
      'bgColorContainerHover': colors.base200,
      'bgColorContainerActive': colors.base300,
      'bgColorSecondaryContainer': colors.base200,
      'bgColorSecondaryContainerHover': colors.base300,
      'bgColorSecondaryContainerActive': colors.base300,
      'bgColorComponent': colors.base300,
      'bgColorComponentHover': blend(colors.baseContent, colors.base300, 0.06),
      'bgColorComponentActive': blend(colors.baseContent, colors.base300, 0.12),
      'bgColorComponentDisabled': colors.base200,
      'componentStrokeColor': colors.line,
      'componentBorderColor': colors.line,
      'textColorPrimary': colors.baseContent,
      'textColorSecondary': colors.baseContent.withValues(alpha: 0.72),
      'textColorPlaceholder': colors.baseContent.withValues(alpha: 0.48),
      'textDisabledColor': colors.baseContent.withValues(alpha: 0.32),
      'textColorAnti': colors.primaryContent,
      'textColorBrand': colors.primary,
      'textColorLink': colors.primary,
    },
    radiusMap: const <String, double>{
      'radiusSmall': 4,
      'radiusDefault': 8,
      'radiusLarge': 8,
      'radiusExtraLarge': 12,
      'radiusRound': 9999,
      'radiusCircle': 9999,
    },
    marginMap: const <String, double>{
      'spacer4': 4,
      'spacer8': 8,
      'spacer12': 12,
      'spacer16': 16,
      'spacer24': 24,
      'spacer32': 32,
      'spacer40': 40,
      'spacer48': 48,
      'spacer64': 64,
      'spacer96': 96,
      'spacer160': 160,
    },
  );

  return brightness == Brightness.dark
      ? TThemeBuilder.dark(token)
      : TThemeBuilder.light(token);
}
