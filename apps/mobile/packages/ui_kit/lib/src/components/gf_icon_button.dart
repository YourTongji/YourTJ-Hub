import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../theme/gf_theme.dart';

import 'gf_symbol.dart';

/// Icon-only button mirroring web `.gf-icon-button` (components.css):
/// rounded `gf-radius-field`, `icon-muted` color, hover `base-200`.
class GfIconButton extends StatelessWidget {
  const GfIconButton({
    super.key,
    this.icon,
    this.symbol,
    required this.onPressed,
    this.onLongPress,
    this.tooltip,
    this.size = 44,
    this.iconSize = 20,
    this.color,
  }) : assert(icon != null || symbol != null, 'Either icon or symbol must be provided');

  final IconData? icon;
  final String? symbol;
  final VoidCallback? onPressed;
  final VoidCallback? onLongPress;
  final String? tooltip;
  final double size;
  final double iconSize;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final Color iconColor = color ?? colors.iconMuted;

    final Widget iconWidget = symbol != null
        ? GfSymbol(symbol!, size: iconSize, color: iconColor)
        : Icon(icon, size: iconSize, color: iconColor);

    final Widget button = SizedBox.square(
      dimension: size,
      child: td.TButton(
        size: td.TButtonSize.small,
        variant: td.TButtonVariant.text,
        colorScheme: td.TButtonColorScheme.defaultTheme,
        icon: iconWidget,
        onPressed: onPressed,
        onLongPress: onLongPress,
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll<EdgeInsetsGeometry>(
            EdgeInsets.zero,
          ),
          minimumSize: WidgetStatePropertyAll<Size>(Size.square(size)),
          maximumSize: WidgetStatePropertyAll<Size>(Size.square(size)),
          foregroundColor: WidgetStatePropertyAll<Color>(iconColor),
          backgroundColor: const WidgetStatePropertyAll<Color>(
            Colors.transparent,
          ),
          shape: WidgetStatePropertyAll<OutlinedBorder>(
            RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(radii.field),
            ),
          ),
        ),
      ),
    );

    if (tooltip == null) return button;
    return Tooltip(
      // Long-press-capable buttons opt out of the Tooltip's long-press
      // gesture so the surrounding GestureDetector can receive it.
      triggerMode: onLongPress == null
          ? TooltipTriggerMode.longPress
          : TooltipTriggerMode.manual,
      message: tooltip!,
      child: button,
    );
  }
}
