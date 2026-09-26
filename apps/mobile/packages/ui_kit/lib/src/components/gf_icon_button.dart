import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

import 'gf_symbol.dart';

/// Quiet circular action with a minimum 44px target. Native button behavior
/// keeps focus, keyboard activation and long press independent of the glyph.
class GfIconButton extends StatelessWidget {
  const GfIconButton({
    super.key,
    required this.symbol,
    required this.onPressed,
    this.onLongPress,
    this.tooltip,
    this.size = 44,
    this.iconSize = 20,
    this.color,
  });

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
    final Color iconColor = onPressed == null
        ? (color ?? colors.iconMuted).withValues(alpha: .38)
        : color ?? colors.iconMuted;

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
          padding: const WidgetStatePropertyAll(EdgeInsets.zero),
          minimumSize: WidgetStatePropertyAll(Size.square(targetSize)),
          maximumSize: WidgetStatePropertyAll(Size.square(targetSize)),
          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
          visualDensity: VisualDensity.standard,
          foregroundColor: WidgetStatePropertyAll(iconColor),
          backgroundColor: const WidgetStatePropertyAll(Colors.transparent),
          overlayColor: WidgetStateProperty.resolveWith((states) {
            if (states.contains(WidgetState.focused)) {
              return colors.primary.withValues(alpha: 0.16);
            }
            if (states.contains(WidgetState.pressed)) {
              return colors.baseContent.withValues(alpha: 0.10);
            }
            if (states.contains(WidgetState.hovered)) {
              return colors.baseContent.withValues(alpha: 0.06);
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
