import 'package:flutter/cupertino.dart';

import 'package:flutter/material.dart';

import 'gf_colors.dart';
import 'gf_theme_extensions.dart';
import 'gf_typography.dart';

/// Builds the yourtj `ThemeData` for a [Brightness], wiring the Gf design
/// tokens into Material widgets (color scheme, shapes, typography, motion).
///
/// The Gf tokens remain reachable through `Theme.of(context).extension<T>()`
/// (see [GfTheme]).
ThemeData gfThemeData(Brightness brightness) {
  final GfColors colors = GfColors.forBrightness(brightness);

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
    surfaceContainerHighest: colors.base300,
    onSurfaceVariant: colors.iconMuted,
    outline: colors.line,
    outlineVariant: colors.line,
    shadow: colors.neutral,
    inverseSurface: colors.neutral,
    onInverseSurface: colors.neutralContent,
    inversePrimary: colors.primary,
    surfaceTint: colors.primary,
  );

  const OutlineInputBorder noBorder = OutlineInputBorder(
    borderSide: BorderSide.none,
  );

  final GfTypography typography = GfTypography.standard(colors.baseContent);

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: colorScheme,
    scaffoldBackgroundColor: colors.base100,
    dividerColor: colors.line,
    extensions: <ThemeExtension<dynamic>>[
      GfRadii.standard,
      GfBorders.standard,
      GfSizes.standard,
      typography,
    ],
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
    // Cards follow gf-card semantics (see GfCard).
    cardTheme: CardThemeData(
      color: colors.base100,
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadii.standard.box),
        side: BorderSide(color: colors.line, width: GfBorders.standard.width),
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
    // Motion mirrors `resource/src/runtime/motion.ts` (see GfMotion).
    pageTransitionsTheme: PageTransitionsTheme(
      builders: <TargetPlatform, PageTransitionsBuilder>{
        TargetPlatform.android: const FadeUpwardsPageTransitionsBuilder(),
        TargetPlatform.iOS: const CupertinoPageTransitionsBuilder(),
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
