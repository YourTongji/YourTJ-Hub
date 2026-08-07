import 'package:flutter/material.dart';

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
      GfShadows.standard,
      typography,
    ],
    // App bars follow the web header (h-16 = 64px, 20px w700 title, flat).
    appBarTheme: AppBarTheme(
      backgroundColor: colors.base100,
      foregroundColor: colors.baseContent,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      toolbarHeight: 64,
      titleTextStyle: TextStyle(
        fontSize: 20,
        fontWeight: FontWeight.w700,
        color: colors.baseContent,
      ),
    ),
    // Bottom navigation follows the web visual language (base-100 surface,
    // 12px labels, primary selected / icon-muted unselected). The 1px top
    // divider is drawn by GfShell's container.
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
