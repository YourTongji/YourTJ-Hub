import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';

import '../components/gf_motion.dart';
import 'gf_colors.dart';
import 'gf_filled_input_border.dart';
import 'gf_theme_extensions.dart';
import 'gf_shadows.dart';
import 'gf_typography.dart';

/// Builds the yourtj `ThemeData` for a [Brightness], wiring the Gf design
/// tokens into Material widgets (color scheme, shapes, typography, motion).
///
/// The Gf tokens remain reachable through `Theme.of(context).extension<T>()`
/// (see [GfTheme]).
/// [colors] 为可选的运行时覆盖色板（站点主题同步）：缺省时用内置
/// `GfColors.forBrightness`（tokens.json 镜像，唯一事实源）。
ThemeData gfThemeData(Brightness brightness, {GfColors? overrides}) {
  // 运行时覆盖色板（站点主题同步）；缺省用内置 tokens.json 镜像（唯一事实源）。
  final GfColors colors = overrides ?? GfColors.forBrightness(brightness);

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
    // Neutral is a light foreground in dark mode. A modal must always dim its
    // background, so its scrim cannot inherit that theme-dependent foreground.
    scrim: Colors.black.withValues(alpha: 0.4),
    inverseSurface: colors.neutral,
    onInverseSurface: colors.neutralContent,
    inversePrimary: colors.primary,
    surfaceTint: colors.primary,
  );

  const inputBorder = GfFilledInputBorder();
  final GfTypography typography = GfTypography.standard(colors.baseContent);

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: colorScheme,
    scaffoldBackgroundColor: colors.base100,
    canvasColor: colors.base100,
    dividerColor: colors.line,
    extensions: <ThemeExtension<dynamic>>[
      GfRadii.standard,
      GfBorders.standard,
      GfSizes.standard,
      GfShadows.standard,
      typography,
    ],
    iconTheme: IconThemeData(color: colors.baseContent, size: 24),
    textSelectionTheme: TextSelectionThemeData(
      cursorColor: colors.primary,
      selectionColor: colors.primary.withValues(alpha: 0.2),
      selectionHandleColor: colors.primary,
    ),
    // Navigation bars keep the web header's flat, strong-title treatment in
    // a mobile-native 56px height.
    appBarTheme: AppBarTheme(
      backgroundColor: colors.base100,
      foregroundColor: colors.baseContent,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: true,
      toolbarHeight: 56,
      titleTextStyle: TextStyle(
        fontSize: 18,
        fontWeight: FontWeight.w600,
        color: colors.baseContent,
      ),
    ),
    // Native controls and Gf wrappers share the same application-owned styles.
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: colors.primary,
      foregroundColor: colors.primaryContent,
      shape: const CircleBorder(),
      elevation: 2,
      focusElevation: 3,
      hoverElevation: 3,
      highlightElevation: 4,
    ),
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
        tapTargetSize: MaterialTapTargetSize.padded,
        backgroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled)
              ? colors.base300
              : colors.primary,
        ),
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled)
              ? colors.baseContent.withValues(alpha: 0.38)
              : colors.primaryContent,
        ),
        shape: WidgetStatePropertyAll(StadiumBorder()),
        textStyle: WidgetStatePropertyAll(
          TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
        ),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: ButtonStyle(
        tapTargetSize: MaterialTapTargetSize.padded,
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled)
              ? colors.baseContent.withValues(alpha: 0.38)
              : colors.baseContent,
        ),
        side: WidgetStateProperty.resolveWith(
          (states) => BorderSide(
            color: states.contains(WidgetState.disabled)
                ? colors.line.withValues(alpha: 0.5)
                : colors.line,
            width: GfBorders.standard.width,
          ),
        ),
        shape: WidgetStatePropertyAll(StadiumBorder()),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: ButtonStyle(
        tapTargetSize: MaterialTapTargetSize.padded,
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled)
              ? colors.baseContent.withValues(alpha: 0.38)
              : colors.primary,
        ),
        shape: WidgetStatePropertyAll(StadiumBorder()),
      ),
    ),
    // Form fields have a quiet filled silhouette. Every state keeps the same
    // geometry; only focus and validation add a fine outline, never a glow.
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: WidgetStateColor.resolveWith(
        (states) => states.contains(WidgetState.disabled)
            ? colors.base300.withValues(alpha: 0.55)
            : colors.base300,
      ),
      isDense: true,
      constraints: const BoxConstraints(minHeight: 52),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      hintStyle: TextStyle(fontSize: 16, color: colors.iconMuted),
      labelStyle: TextStyle(fontSize: 16, color: colors.iconMuted),
      floatingLabelStyle: WidgetStateTextStyle.resolveWith(
        (states) => TextStyle(
          fontSize: 14,
          color: states.contains(WidgetState.error)
              ? colors.error
              : states.contains(WidgetState.focused)
              ? colors.primary
              : colors.iconMuted,
        ),
      ),
      helperStyle: TextStyle(fontSize: 13, color: colors.iconMuted),
      errorStyle: TextStyle(fontSize: 13, color: colors.error),
      helperMaxLines: 3,
      errorMaxLines: 4,
      prefixIconColor: colors.iconMuted,
      suffixIconColor: colors.iconMuted,
      prefixIconConstraints: const BoxConstraints(minWidth: 48, minHeight: 48),
      suffixIconConstraints: const BoxConstraints(minWidth: 48, minHeight: 48),
      border: inputBorder,
      enabledBorder: inputBorder,
      disabledBorder: inputBorder,
      focusedBorder: inputBorder.copyWith(
        borderSide: BorderSide(color: colors.primary, width: 1.5),
      ),
      errorBorder: inputBorder.copyWith(
        borderSide: BorderSide(color: colors.error),
      ),
      focusedErrorBorder: inputBorder.copyWith(
        borderSide: BorderSide(color: colors.error, width: 1.5),
      ),
    ),
    checkboxTheme: CheckboxThemeData(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(5)),
      side: BorderSide(color: colors.iconMuted, width: 1.5),
      visualDensity: VisualDensity.standard,
      materialTapTargetSize: MaterialTapTargetSize.padded,
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
    // Floating surfaces have a softer silhouette than compact inline controls.
    dialogTheme: DialogThemeData(
      backgroundColor: colors.base100,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: colors.base100,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
    ),
    // Popup menus are actions anchored to a trigger. Material's current
    // default is `over`, which hides the trigger and makes a menu feel like a
    // replacement surface on touch devices. `under` preserves the spatial
    // relationship while Flutter still flips/clamps at viewport edges.
    popupMenuTheme: PopupMenuThemeData(
      position: PopupMenuPosition.under,
      color: colors.base100,
      surfaceTintColor: Colors.transparent,
      textStyle: TextStyle(color: colors.baseContent, fontSize: 15),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
    ),
    // Motion mirrors `resource/src/runtime/motion.ts` (see GfMotion):
    // page transitions use the standard 0.22s ease (web page-enter motion).
    // iOS keeps the system Cupertino transition so the edge-swipe back
    // gesture stays interactive (#642); Android keeps the Gf fade/rise.
    pageTransitionsTheme: PageTransitionsTheme(
      builders: <TargetPlatform, PageTransitionsBuilder>{
        TargetPlatform.android: const GfPageTransitionsBuilder(),
        TargetPlatform.iOS: const CupertinoPageTransitionsBuilder(),
      },
    ),
    splashFactory: InkRipple.splashFactory,
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
