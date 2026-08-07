import 'package:flutter/material.dart';

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

    final (Color background, Color foreground, Color? border) = _palette(
      colors,
      borders,
    );
    final bool enabled = onPressed != null && !loading;

    final Widget content = Row(
      mainAxisSize: expanded ? MainAxisSize.max : MainAxisSize.min,
      mainAxisAlignment: MainAxisAlignment.center,
      children: <Widget>[
        if (loading)
          SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(strokeWidth: 2, color: foreground),
          )
        else
          ?icon,
        if (loading || icon != null) const SizedBox(width: 6),
        Text(
          label,
          style: TextStyle(
            color: foreground,
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );

    final Widget button = Opacity(
      opacity: enabled ? 1 : 0.6,
      child: Material(
        color: background,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radii.field),
          side: border == null
              ? BorderSide.none
              : BorderSide(color: border, width: borders.width),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: enabled ? onPressed : null,
          child: Container(
            height: size.height,
            padding: const EdgeInsets.symmetric(horizontal: 16),
            alignment: Alignment.center,
            child: content,
          ),
        ),
      ),
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
