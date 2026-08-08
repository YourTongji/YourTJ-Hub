import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../theme/gf_theme.dart';

/// Button variant, mirroring web `gf-button-*` classes.
enum GfButtonVariant {
  /// `gf-button-primary`: solid primary fill.
  primary,

  /// `gf-button-neutral`: solid neutral fill.
  neutral,

  /// `gf-button-secondary`: base-100 fill with line border.
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

/// Button size, mirroring web `gf-button-sm/md/lg/xl`.
enum GfButtonSize {
  small(32),
  medium(36),
  large(40),
  extraLarge(44);

  const GfButtonSize(this.height);

  final double height;
}

/// Button aligned with web `gf-button` semantics: rounded `gf-radius-field`,
/// 600-weight text, 60% opacity + no pointer when disabled.
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
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final TextStyle textStyle =
        (Theme.of(context).textTheme.labelLarge ?? const TextStyle()).copyWith(
          fontSize: 14,
          fontWeight: FontWeight.w600,
        );

    final (Color background, Color foreground, Color? border) = _palette(
      colors,
      borders,
    );
    final bool enabled = onPressed != null && !loading;

    final Widget? effectiveIcon = loading
        ? SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(strokeWidth: 2, color: foreground),
          )
        : icon;

    final td.TButton button = td.TButton(
      size: switch (size) {
        GfButtonSize.small => td.TButtonSize.extraSmall,
        GfButtonSize.medium => td.TButtonSize.small,
        GfButtonSize.large => td.TButtonSize.medium,
        GfButtonSize.extraLarge => td.TButtonSize.large,
      },
      variant: switch (variant) {
        GfButtonVariant.primary ||
        GfButtonVariant.neutral ||
        GfButtonVariant.danger => td.TButtonVariant.fill,
        GfButtonVariant.secondary ||
        GfButtonVariant.outline => td.TButtonVariant.outline,
        GfButtonVariant.ghost => td.TButtonVariant.ghost,
        GfButtonVariant.muted || GfButtonVariant.link => td.TButtonVariant.text,
      },
      colorScheme: switch (variant) {
        GfButtonVariant.primary ||
        GfButtonVariant.ghost ||
        GfButtonVariant.link => td.TButtonColorScheme.primary,
        GfButtonVariant.danger => td.TButtonColorScheme.danger,
        _ => td.TButtonColorScheme.defaultTheme,
      },
      icon: effectiveIcon,
      onPressed: enabled ? onPressed : null,
      style: ButtonStyle(
        backgroundColor: WidgetStatePropertyAll<Color>(background),
        foregroundColor: WidgetStatePropertyAll<Color>(foreground),
        side: WidgetStatePropertyAll<BorderSide>(
          border == null
              ? BorderSide.none
              : BorderSide(color: border, width: borders.width),
        ),
        minimumSize: WidgetStatePropertyAll<Size>(Size(0, size.height)),
        maximumSize: WidgetStatePropertyAll<Size>(
          Size(double.infinity, size.height),
        ),
        padding: WidgetStatePropertyAll<EdgeInsetsGeometry>(
          EdgeInsets.symmetric(
            horizontal: switch (size) {
              GfButtonSize.small || GfButtonSize.medium => 12,
              GfButtonSize.large || GfButtonSize.extraLarge => 16,
            },
          ),
        ),
        shape: WidgetStatePropertyAll<OutlinedBorder>(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(radii.field),
          ),
        ),
        textStyle: WidgetStatePropertyAll<TextStyle>(textStyle),
        elevation: const WidgetStatePropertyAll<double>(0),
      ),
      child: Text(label),
    );

    if (!expanded) return button;
    return SizedBox(width: double.infinity, child: button);
  }

  (Color, Color, Color?) _palette(GfColors colors, GfBorders borders) {
    switch (variant) {
      case GfButtonVariant.primary:
        return (colors.primary, colors.primaryContent, null);
      case GfButtonVariant.neutral:
        return (colors.neutral, colors.neutralContent, null);
      case GfButtonVariant.secondary:
        return (
          colors.base100,
          colors.baseContent.withValues(alpha: 0.75),
          colors.line,
        );
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
