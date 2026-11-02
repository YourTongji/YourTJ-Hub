import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';

import '../components/gf_motion.dart';
import 'gf_colors.dart';
import 'gf_design.dart';
import 'gf_filled_input_border.dart';
import 'gf_shadows.dart';
import 'gf_theme_extensions.dart';
import 'gf_typography.dart';

WidgetStateProperty<Color?> _softTouchOverlay(Color color) =>
    WidgetStateProperty.resolveWith((states) {
      if (states.contains(WidgetState.focused)) {
        return color.withValues(alpha: 0.14);
      }
      if (states.contains(WidgetState.pressed)) {
        return color.withValues(alpha: 0.08);
      }
      if (states.contains(WidgetState.hovered)) {
        return color.withValues(alpha: 0.05);
      }
      return Colors.transparent;
    });

/// Builds the yourtj `ThemeData` for a [Brightness], wiring the Gf design
/// tokens into Material widgets (color scheme, shapes, typography, motion).
///
/// The Gf tokens remain reachable through `Theme.of(context).extension<T>()`
/// (see [GfTheme]).
/// [colors] 为可选的运行时覆盖色板（站点主题同步）：缺省时用内置
/// `GfColors.forBrightness`（tokens.json 镜像，唯一事实源）。
ThemeData gfThemeData(
  Brightness brightness, {
  GfColors? overrides,
  bool disableAnimations = false,
}) {
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
    surfaceContainerLow: colors.canvasGrouped,
    surfaceContainer: colors.surfaceRaised,
    surfaceContainerHigh: colors.surfaceRaised,
    surfaceContainerHighest: colors.base300,
    surfaceDim: colors.base200,
    surfaceBright: colors.base100,
    onSurfaceVariant: colors.textSecondary,
    outline: colors.outline,
    outlineVariant: colors.separator,
    shadow: colors.neutral,
    // Neutral is a light foreground in dark mode. A modal must always dim its
    // background, so its scrim cannot inherit that theme-dependent foreground.
    scrim: colors.scrim,
    inverseSurface: colors.neutral,
    onInverseSurface: colors.neutralContent,
    inversePrimary: colors.primary,
    surfaceTint: colors.primary,
  );

  const inputBorder = GfFilledInputBorder();
  final GfTypography typography = GfTypography.standard(colors.baseContent);
  final TextStyle buttonLabel = typography.bodyStrong.copyWith(
    fontSize: 15,
    height: 20 / 15,
  );
  WidgetStateProperty<Color?> disabledAware(Color enabled, Color disabled) =>
      WidgetStateProperty.resolveWith(
        (states) => states.contains(WidgetState.disabled) ? disabled : enabled,
      );

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: colorScheme,
    // Keep Material's interaction lifecycle but remove its visible ink spread.
    splashFactory: InkRipple.splashFactory,
    splashColor: Colors.transparent,
    highlightColor: colors.pressed,
    hoverColor: colors.pressed,
    scaffoldBackgroundColor: colors.canvas,
    canvasColor: colors.canvas,
    dividerColor: colors.separator,
    extensions: <ThemeExtension<dynamic>>[
      GfRadii.standard,
      GfBorders.standard,
      GfSizes.standard,
      GfShadows.standard,
      typography,
    ],
    iconTheme: IconThemeData(color: colors.textPrimary, size: GfIconSize.lg),
    textSelectionTheme: TextSelectionThemeData(
      cursorColor: colors.primary,
      selectionColor: colors.primary.withValues(alpha: 0.2),
      selectionHandleColor: colors.primary,
    ),
    // Flat navigation bar: 56px, centered 17px semibold title. The hairline
    // is drawn by GfAppBar so plain AppBars stay borderless.
    appBarTheme: AppBarTheme(
      backgroundColor: colors.canvas,
      foregroundColor: colors.textPrimary,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: true,
      toolbarHeight: 56,
      titleTextStyle: typography.title3,
    ),
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: colors.primary,
      foregroundColor: colors.primaryContent,
      shape: const CircleBorder(),
      elevation: 0,
      focusElevation: 0,
      hoverElevation: 0,
      highlightElevation: 0,
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: colors.canvas,
      surfaceTintColor: Colors.transparent,
      indicatorColor: colors.primarySoft,
      height: 64,
      overlayColor: _softTouchOverlay(colors.primary),
      labelTextStyle: WidgetStatePropertyAll(
        typography.meta.copyWith(color: colors.textSecondary),
      ),
      iconTheme: WidgetStateProperty.resolveWith((Set<WidgetState> states) {
        final bool selected = states.contains(WidgetState.selected);
        return IconThemeData(
          size: GfIconSize.lg,
          color: selected ? colors.primary : colors.textSecondary,
        );
      }),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: ButtonStyle(
        overlayColor: _softTouchOverlay(colors.baseContent),
        foregroundColor: disabledAware(colors.textPrimary, colors.textDisabled),
      ),
    ),
    tabBarTheme: TabBarThemeData(
      overlayColor: _softTouchOverlay(colors.baseContent),
    ),
    listTileTheme: ListTileThemeData(
      iconColor: colors.textSecondary,
      textColor: colors.textPrimary,
      contentPadding: const EdgeInsets.symmetric(horizontal: GfSpace.lg),
      minVerticalPadding: GfSpace.md,
      minLeadingWidth: GfIconSize.lg,
      horizontalTitleGap: GfSpace.md,
      shape: const RoundedRectangleBorder(),
      titleTextStyle: typography.body,
      subtitleTextStyle: typography.caption.copyWith(
        color: colors.textSecondary,
      ),
    ),
    dividerTheme: DividerThemeData(
      color: colors.separator,
      thickness: 0.5,
      space: 1,
    ),
    // Transient feedback uses the inverse surface, like GfToast.
    snackBarTheme: SnackBarThemeData(
      backgroundColor: colors.neutral,
      contentTextStyle: typography.subhead.copyWith(
        color: colors.neutralContent,
      ),
      actionTextColor: colors.isDark
          ? GfColors.light.primary
          : GfColors.dark.primary,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadius.md),
      ),
      behavior: SnackBarBehavior.floating,
    ),
    tooltipTheme: TooltipThemeData(
      decoration: BoxDecoration(
        color: colors.neutral,
        borderRadius: BorderRadius.circular(GfRadius.sm),
      ),
      textStyle: typography.caption.copyWith(color: colors.neutralContent),
      padding: const EdgeInsets.symmetric(
        horizontal: GfSpace.md,
        vertical: GfSpace.sm,
      ),
    ),
    switchTheme: SwitchThemeData(
      overlayColor: _softTouchOverlay(colors.primary),
      thumbColor: WidgetStatePropertyAll(Colors.white),
      trackColor: WidgetStateProperty.resolveWith((Set<WidgetState> states) {
        if (states.contains(WidgetState.selected)) {
          return states.contains(WidgetState.disabled)
              ? colors.primary.withValues(alpha: 0.4)
              : colors.primary;
        }
        return colors.fillStrong;
      }),
      trackOutlineColor: const WidgetStatePropertyAll(Colors.transparent),
      thumbIcon: const WidgetStatePropertyAll(null),
    ),
    sliderTheme: SliderThemeData(
      activeTrackColor: colors.primary,
      inactiveTrackColor: colors.fillStrong,
      thumbColor: Colors.white,
      overlayColor: colors.primary.withValues(alpha: 0.12),
      trackHeight: 4,
    ),
    progressIndicatorTheme: ProgressIndicatorThemeData(
      color: colors.primary,
      linearTrackColor: colors.fill,
      circularTrackColor: Colors.transparent,
      linearMinHeight: 3,
      borderRadius: BorderRadius.circular(GfRadius.full),
    ),
    // Buttons follow GfButton: pill shapes, one label style, visible disabled
    // fills in both themes.
    filledButtonTheme: FilledButtonThemeData(
      style: ButtonStyle(
        tapTargetSize: MaterialTapTargetSize.padded,
        minimumSize: const WidgetStatePropertyAll(Size(64, GfControl.regular)),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: 18),
        ),
        backgroundColor: disabledAware(colors.primary, colors.fill),
        foregroundColor: disabledAware(
          colors.primaryContent,
          colors.textDisabled,
        ),
        overlayColor: _softTouchOverlay(colors.primaryContent),
        shape: const WidgetStatePropertyAll(StadiumBorder()),
        textStyle: WidgetStatePropertyAll(buttonLabel),
        elevation: const WidgetStatePropertyAll(0),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: ButtonStyle(
        tapTargetSize: MaterialTapTargetSize.padded,
        minimumSize: const WidgetStatePropertyAll(Size(64, GfControl.regular)),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: 18),
        ),
        foregroundColor: disabledAware(colors.textPrimary, colors.textDisabled),
        overlayColor: _softTouchOverlay(colors.baseContent),
        side: WidgetStateProperty.resolveWith(
          (states) => BorderSide(
            color: states.contains(WidgetState.disabled)
                ? colors.fill
                : colors.outline,
          ),
        ),
        shape: const WidgetStatePropertyAll(StadiumBorder()),
        textStyle: WidgetStatePropertyAll(buttonLabel),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: ButtonStyle(
        tapTargetSize: MaterialTapTargetSize.padded,
        minimumSize: const WidgetStatePropertyAll(Size(48, GfControl.regular)),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: GfSpace.md),
        ),
        foregroundColor: disabledAware(colors.primary, colors.textDisabled),
        overlayColor: _softTouchOverlay(colors.primary),
        shape: const WidgetStatePropertyAll(StadiumBorder()),
        textStyle: WidgetStatePropertyAll(buttonLabel),
      ),
    ),
    // Form fields have a quiet filled silhouette. Every state keeps the same
    // geometry; only focus and validation add a fine outline, never a glow.
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: WidgetStateColor.resolveWith(
        (states) => states.contains(WidgetState.disabled)
            ? colors.fill.withValues(alpha: colors.fill.a * 0.6)
            : colors.fill,
      ),
      isDense: true,
      constraints: const BoxConstraints(minHeight: GfControl.field),
      contentPadding: const EdgeInsets.symmetric(
        horizontal: GfSpace.lg,
        vertical: 14,
      ),
      hintStyle: typography.body.copyWith(color: colors.textPlaceholder),
      labelStyle: typography.body.copyWith(color: colors.textSecondary),
      floatingLabelStyle: WidgetStateTextStyle.resolveWith(
        (states) => typography.caption.copyWith(
          color: states.contains(WidgetState.error)
              ? colors.error
              : states.contains(WidgetState.focused)
              ? colors.primary
              : colors.textSecondary,
        ),
      ),
      helperStyle: typography.caption.copyWith(color: colors.textSecondary),
      errorStyle: typography.caption.copyWith(color: colors.error),
      counterStyle: typography.caption.copyWith(color: colors.textSecondary),
      helperMaxLines: 3,
      errorMaxLines: 4,
      prefixIconColor: colors.textSecondary,
      suffixIconColor: colors.textSecondary,
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
      overlayColor: _softTouchOverlay(colors.primary),
      fillColor: WidgetStateProperty.resolveWith(
        (states) => states.contains(WidgetState.selected)
            ? (states.contains(WidgetState.disabled)
                  ? colors.fillStrong
                  : colors.primary)
            : Colors.transparent,
      ),
      checkColor: WidgetStatePropertyAll(colors.primaryContent),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadius.xs),
      ),
      side: WidgetStateBorderSide.resolveWith(
        (states) => states.contains(WidgetState.selected)
            ? BorderSide.none
            : BorderSide(color: colors.textPlaceholder, width: 1.5),
      ),
      visualDensity: VisualDensity.standard,
      materialTapTargetSize: MaterialTapTargetSize.padded,
    ),
    radioTheme: RadioThemeData(
      overlayColor: _softTouchOverlay(colors.primary),
      fillColor: WidgetStateProperty.resolveWith(
        (states) => states.contains(WidgetState.selected)
            ? colors.primary
            : colors.textPlaceholder,
      ),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: colors.fill,
      selectedColor: colors.primarySoft,
      labelStyle: typography.meta.copyWith(color: colors.textPrimary),
      side: BorderSide.none,
      shape: const StadiumBorder(),
      showCheckmark: false,
    ),
    cardTheme: CardThemeData(
      color: colors.surfaceGrouped,
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadius.lg),
      ),
    ),
    // Modal surfaces share the largest radius and the raised surface colour.
    dialogTheme: DialogThemeData(
      backgroundColor: colors.surfaceRaised,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadius.xl),
      ),
      titleTextStyle: typography.title2,
      contentTextStyle: typography.small.copyWith(color: colors.textSecondary),
      barrierColor: colors.scrim,
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: colors.surfaceRaised,
      surfaceTintColor: Colors.transparent,
      modalBarrierColor: colors.scrim,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(GfRadius.xl)),
      ),
    ),
    // Popup menus are actions anchored to a trigger. Material's current
    // default is `over`, which hides the trigger and makes a menu feel like a
    // replacement surface on touch devices. `under` preserves the spatial
    // relationship while Flutter still flips/clamps at viewport edges.
    popupMenuTheme: PopupMenuThemeData(
      position: PopupMenuPosition.under,
      color: colors.surfaceRaised,
      surfaceTintColor: Colors.transparent,
      elevation: colors.isDark ? 0 : 6,
      shadowColor: const Color(0x330F172A),
      menuPadding: const EdgeInsets.all(6),
      textStyle: typography.body,
      labelTextStyle: WidgetStatePropertyAll(typography.body),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadius.lg),
        side: colors.isDark
            ? BorderSide(color: colors.separator, width: 0.5)
            : BorderSide.none,
      ),
    ),
    dropdownMenuTheme: DropdownMenuThemeData(
      menuStyle: MenuStyle(
        backgroundColor: WidgetStatePropertyAll(colors.surfaceRaised),
        surfaceTintColor: const WidgetStatePropertyAll(Colors.transparent),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(GfRadius.lg),
          ),
        ),
      ),
    ),
    // Native mobile motion is owned by GfMotion (220ms in / 180ms out).
    // iOS keeps the system Cupertino transition so the edge-swipe back
    // gesture stays interactive (#642); Android keeps the Gf fade/rise.
    pageTransitionsTheme: PageTransitionsTheme(
      builders: <TargetPlatform, PageTransitionsBuilder>{
        TargetPlatform.android: GfPageTransitionsBuilder(
          disableAnimations: disableAnimations,
        ),
        TargetPlatform.iOS: const CupertinoPageTransitionsBuilder(),
      },
    ),
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
      labelLarge: buttonLabel,
      labelMedium: typography.meta,
      labelSmall: typography.label,
    ),
  );
}
