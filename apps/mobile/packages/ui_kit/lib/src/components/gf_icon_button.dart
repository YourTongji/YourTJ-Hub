import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';
import 'gf_symbol.dart';

/// Icon button surface.
enum GfIconButtonVariant {
  /// Glyph only; navigation bars, rows and toolbars.
  plain,

  /// Glyph on the shared translucent fill; secondary floating actions.
  tonal,

  /// Glyph on the primary fill; a single emphasized action such as send.
  filled,
}

/// Circular action with a minimum 44px target. Native button behavior
/// keeps focus, keyboard activation and long press independent of the glyph.
/// Tonal and filled variants paint a circle 8px smaller than [size].
class GfIconButton extends StatelessWidget {
  const GfIconButton({
    super.key,
    required this.symbol,
    required this.onPressed,
    this.onLongPress,
    this.tooltip,
    this.size = 44,
    this.iconSize = GfIconSize.md,
    this.color,
    this.variant = GfIconButtonVariant.plain,
  });

  final GfIconButtonVariant variant;

  final String symbol;
  final VoidCallback? onPressed;
  final VoidCallback? onLongPress;
  final String? tooltip;
  final double size;
  final double iconSize;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final targetSize = math.max(44.0, size);
    final Color base = switch (variant) {
      GfIconButtonVariant.plain => color ?? colors.textSecondary,
      GfIconButtonVariant.tonal => color ?? colors.textPrimary,
      GfIconButtonVariant.filled => colors.primaryContent,
    };
    final Color iconColor = onPressed == null ? colors.textDisabled : base;
    final Color background = switch (variant) {
      GfIconButtonVariant.plain => Colors.transparent,
      GfIconButtonVariant.tonal => colors.fill,
      GfIconButtonVariant.filled =>
        onPressed == null ? colors.fill : (color ?? colors.primary),
    };
    final double inset = variant == GfIconButtonVariant.plain ? 0 : 4;

    final Widget iconWidget = GfSymbol(
      symbol,
      size: iconSize,
      color: iconColor,
    );

    final Widget button = SizedBox.square(
      dimension: targetSize,
      child: TextButton(
        onPressed: onPressed,
        onLongPress: onPressed == null ? null : onLongPress,
        style: ButtonStyle(
          animationDuration: GfMotion.duration(context, GfMotion.selection),
          padding: const WidgetStatePropertyAll(EdgeInsets.zero),
          backgroundBuilder: inset == 0
              ? null
              : (context, states, child) => Padding(
                  padding: EdgeInsets.all(inset),
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      color: states.contains(WidgetState.pressed)
                          ? Color.alphaBlend(colors.pressed, background)
                          : background,
                      shape: BoxShape.circle,
                    ),
                    child: child,
                  ),
                ),
          minimumSize: WidgetStatePropertyAll(Size.square(targetSize)),
          maximumSize: WidgetStatePropertyAll(Size.square(targetSize)),
          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
          visualDensity: VisualDensity.standard,
          foregroundColor: WidgetStatePropertyAll(iconColor),
          backgroundColor: const WidgetStatePropertyAll(Colors.transparent),
          overlayColor: WidgetStateProperty.resolveWith((states) {
            if (inset > 0 && !states.contains(WidgetState.focused)) {
              return Colors.transparent;
            }
            if (states.contains(WidgetState.focused)) {
              return colors.primary.withValues(alpha: 0.16);
            }
            if (states.contains(WidgetState.pressed)) {
              return colors.pressed;
            }
            if (states.contains(WidgetState.hovered)) {
              return colors.pressed;
            }
            return Colors.transparent;
          }),
          shape: const WidgetStatePropertyAll(CircleBorder()),
        ),
        child: iconWidget,
      ),
    );

    if (tooltip == null) return button;
    return MergeSemantics(
      child: Semantics(
        label: tooltip,
        button: true,
        enabled: onPressed != null,
        child: Tooltip(
          excludeFromSemantics: true,
          // Long-press-capable buttons opt out of the Tooltip's long-press
          // gesture so the native button receives it.
          triggerMode: onLongPress == null
              ? TooltipTriggerMode.longPress
              : TooltipTriggerMode.manual,
          message: tooltip!,
          child: button,
        ),
      ),
    );
  }
}
