import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import '../twemoji/twemoji_text.dart';
import 'atoms/gf_loading_indicator.dart';
import 'gf_motion.dart';

/// Button variant, mirroring web `gf-button-*` classes.
enum GfButtonVariant {
  /// `gf-button-primary`: solid primary fill.
  primary,

  /// `gf-button-neutral`: solid neutral fill.
  neutral,

  /// Quiet secondary action on the shared translucent fill.
  secondary,

  /// Soft primary fill with primary text (follow, join, accept).
  tonal,

  /// `gf-button-danger`: solid error fill.
  danger,

  /// Soft error fill with error text: sign out, leave, secondary destructive
  /// actions that should not shout.
  dangerSoft,

  /// `gf-button-ghost`: transparent with primary text.
  ghost,

  /// `gf-button-outline`: transparent with line border (web secondary-lite).
  outline,

  /// `gf-button-muted`: muted text that brightens on hover.
  muted,

  /// Text-only primary link.
  link,
}

/// Painted button heights; Flutter adds a separate 48px minimum touch target.
/// Scaled or wrapped labels can grow beyond these minimums. [extraLarge]
/// matches form field height for full-width form actions.
enum GfButtonSize {
  small(GfControl.compact, 14, 14, GfIconSize.sm),
  medium(GfControl.regular, 15, 18, 18),
  large(GfControl.large, 16, 22, GfIconSize.md),
  extraLarge(GfControl.field, 16, 24, GfIconSize.md);

  const GfButtonSize(this.height, this.fontSize, this.padding, this.iconSize);

  final double height;
  final double fontSize;
  final double padding;
  final double iconSize;
}

/// Flat capsule action with an independent touch target and scalable label.
/// Flutter owns focus, keyboard activation and disabled semantics.
class GfButton extends StatelessWidget {
  const GfButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.variant = GfButtonVariant.primary,
    this.size = GfButtonSize.medium,
    this.icon,
    this.loading = false,
    this.expanded = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final GfButtonVariant variant;
  final GfButtonSize size;
  final Widget? icon;

  /// Shows a small spinner instead of [icon] and blocks taps.
  final bool loading;

  /// Whether the button stretches to fill available width.
  final bool expanded;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final TextStyle textStyle = GfTheme.typographyOf(
      context,
    ).bodyStrong.copyWith(fontSize: size.fontSize, height: 20 / size.fontSize);

    final (Color background, Color foreground, Color? border) = _palette(
      colors,
    );
    final bool enabled = onPressed != null && !loading;

    final Widget? effectiveIcon = loading
        ? SizedBox(
            width: size.iconSize,
            height: size.iconSize,
            child: GfProgressIndicator(strokeWidth: 2, color: foreground),
          )
        : icon;

    final button = FilledButton(
      onPressed: enabled ? onPressed : null,
      style: ButtonStyle(
        animationDuration: GfMotion.duration(context, GfMotion.press),
        // Pending is a distinct state: it blocks activation while retaining
        // enough contrast for the progress indicator and its action label.
        backgroundColor: WidgetStateProperty.resolveWith((states) {
          if (!states.contains(WidgetState.disabled) || loading) {
            return background;
          }
          return switch (variant) {
            GfButtonVariant.ghost ||
            GfButtonVariant.outline ||
            GfButtonVariant.muted ||
            GfButtonVariant.link => Colors.transparent,
            _ => colors.fill,
          };
        }),
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled) && !loading
              ? colors.textDisabled
              : foreground,
        ),
        overlayColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.focused)) {
            return foreground.withValues(alpha: 0.16);
          }
          if (states.contains(WidgetState.pressed)) {
            return foreground.withValues(alpha: 0.1);
          }
          if (states.contains(WidgetState.hovered)) {
            return foreground.withValues(alpha: 0.05);
          }
          return Colors.transparent;
        }),
        side: WidgetStateProperty.resolveWith(
          (states) => border == null
              ? BorderSide.none
              : BorderSide(
                  color: states.contains(WidgetState.disabled) && !loading
                      ? colors.fill
                      : border,
                  width: borders.width,
                ),
        ),
        minimumSize: WidgetStatePropertyAll<Size>(Size(44, size.height)),
        maximumSize: const WidgetStatePropertyAll<Size>(Size.infinite),
        // Keep the compact fill independent from the hit target, including on
        // desktop/tablet where the ambient theme may use compact density.
        tapTargetSize: MaterialTapTargetSize.padded,
        visualDensity: VisualDensity.standard,
        padding: WidgetStatePropertyAll<EdgeInsetsGeometry>(
          EdgeInsets.symmetric(
            vertical: size == GfButtonSize.small ? 6 : 8,
            horizontal: variant == GfButtonVariant.link ? 4 : size.padding,
          ),
        ),
        shape: WidgetStatePropertyAll<OutlinedBorder>(const StadiumBorder()),
        textStyle: WidgetStatePropertyAll<TextStyle>(textStyle),
        elevation: const WidgetStatePropertyAll<double>(0),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          if (effectiveIcon != null) ...[
            // Glyphs default to 24px; the slot scales them to the size step.
            SizedBox.square(
              dimension: size.iconSize,
              child: FittedBox(
                child: IconTheme.merge(
                  data: IconThemeData(size: size.iconSize, color: foreground),
                  child: effectiveIcon,
                ),
              ),
            ),
            SizedBox(width: size == GfButtonSize.small ? 6 : 8),
          ],
          Flexible(child: GfText(label, textAlign: TextAlign.center)),
        ],
      ),
    );

    if (!expanded) return button;
    return SizedBox(width: double.infinity, child: button);
  }

  (Color, Color, Color?) _palette(GfColors colors) {
    switch (variant) {
      case GfButtonVariant.primary:
        return (colors.primary, colors.primaryContent, null);
      case GfButtonVariant.neutral:
        return (colors.neutral, colors.neutralContent, null);
      case GfButtonVariant.secondary:
        return (colors.fill, colors.textPrimary, null);
      case GfButtonVariant.tonal:
        return (colors.primarySoft, colors.primary, null);
      case GfButtonVariant.danger:
        return (colors.error, colors.errorContent, null);
      case GfButtonVariant.dangerSoft:
        return (colors.toneSoft(GfTone.error), colors.error, null);
      case GfButtonVariant.ghost:
        return (Colors.transparent, colors.primary, null);
      case GfButtonVariant.outline:
        return (Colors.transparent, colors.textPrimary, colors.outline);
      case GfButtonVariant.muted:
        return (Colors.transparent, colors.textSecondary, null);
      case GfButtonVariant.link:
        return (Colors.transparent, colors.primary, null);
    }
  }
}
