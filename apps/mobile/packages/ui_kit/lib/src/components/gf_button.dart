import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Button variant, mirroring web `gf-button-*` classes.
enum GfButtonVariant {
  /// `gf-button-primary`: solid primary fill.
  primary,

  /// `gf-button-neutral`: solid neutral fill.
  neutral,

  /// Quiet secondary action on a low-contrast grey fill.
  secondary,

  /// `gf-button-danger`: solid error fill.
  danger,

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
/// Scaled or wrapped labels can grow beyond these minimums.
enum GfButtonSize {
  small(32),
  medium(40),
  large(44),
  extraLarge(48);

  const GfButtonSize(this.height);

  final double height;
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
    final TextStyle textStyle =
        (Theme.of(context).textTheme.labelLarge ?? const TextStyle()).copyWith(
          fontSize: size == GfButtonSize.small ? 14 : 16,
          height: 1.25,
          fontWeight: FontWeight.w600,
        );

    final (Color background, Color foreground, Color? border) = _palette(
      colors,
    );
    final bool enabled = onPressed != null && !loading;

    final Widget? effectiveIcon = loading
        ? SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(strokeWidth: 2, color: foreground),
          )
        : icon;

    final button = FilledButton(
      onPressed: enabled ? onPressed : null,
      style: ButtonStyle(
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
            _ => colors.base300,
          };
        }),
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.disabled) && !loading
              ? colors.baseContent.withValues(alpha: 0.38)
              : foreground,
        ),
        overlayColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.focused)) {
            return foreground.withValues(alpha: 0.16);
          }
          if (states.contains(WidgetState.pressed)) {
            return foreground.withValues(alpha: 0.12);
          }
          if (states.contains(WidgetState.hovered)) {
            return foreground.withValues(alpha: 0.08);
          }
          return Colors.transparent;
        }),
        side: WidgetStateProperty.resolveWith(
          (states) => border == null
              ? BorderSide.none
              : BorderSide(
                  color: states.contains(WidgetState.disabled) && !loading
                      ? border.withValues(alpha: 0.5)
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
            horizontal: switch (size) {
              GfButtonSize.small || GfButtonSize.medium => 12,
              GfButtonSize.large || GfButtonSize.extraLarge => 16,
            },
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
            IconTheme.merge(
              data: const IconThemeData(size: 18),
              child: effectiveIcon,
            ),
            const SizedBox(width: 8),
          ],
          Flexible(child: Text(label, textAlign: TextAlign.center)),
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
        return (colors.base200, colors.baseContent, null);
      case GfButtonVariant.danger:
        return (colors.error, colors.errorContent, null);
      case GfButtonVariant.ghost:
        return (Colors.transparent, colors.primary, null);
      case GfButtonVariant.outline:
        return (Colors.transparent, colors.baseContent, colors.line);
      case GfButtonVariant.muted:
        return (
          Colors.transparent,
          colors.baseContent.withValues(alpha: 0.55),
          null,
        );
      case GfButtonVariant.link:
        return (Colors.transparent, colors.primary, null);
    }
  }
}
