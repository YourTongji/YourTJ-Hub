import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../theme/gf_theme.dart';

/// Icon-only button mirroring web `.gf-icon-button` (components.css):
/// rounded `gf-radius-field`, `icon-muted` color, hover `base-200`.
class GfIconButton extends StatelessWidget {
  const GfIconButton({
    super.key,
    required this.icon,
    required this.onPressed,
    this.tooltip,
    this.size = 44,
    this.iconSize = 20,
  });

  final IconData icon;
  final VoidCallback? onPressed;
  final String? tooltip;
  final double size;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    final Widget button = SizedBox.square(
      dimension: size,
      child: td.TButton(
        size: td.TButtonSize.small,
        variant: td.TButtonVariant.text,
        colorScheme: td.TButtonColorScheme.defaultTheme,
        icon: Icon(icon, size: iconSize),
        onPressed: onPressed,
        style: ButtonStyle(
          padding: const WidgetStatePropertyAll<EdgeInsetsGeometry>(
            EdgeInsets.zero,
          ),
          minimumSize: WidgetStatePropertyAll<Size>(Size.square(size)),
          maximumSize: WidgetStatePropertyAll<Size>(Size.square(size)),
          foregroundColor: WidgetStatePropertyAll<Color>(colors.iconMuted),
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
    return Tooltip(message: tooltip!, child: button);
  }
}
