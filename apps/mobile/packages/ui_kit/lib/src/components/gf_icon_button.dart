import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';

/// Icon-only button mirroring web `.gf-icon-button` (components.css):
/// rounded `gf-radius-field`, `icon-muted` color, hover `base-200`.
class GfIconButton extends StatelessWidget {
  const GfIconButton({
    super.key,
    required this.icon,
    required this.onPressed,
    this.tooltip,
    this.size = 36,
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
    final bool enabled = onPressed != null;

    final Widget button = Opacity(
      opacity: enabled ? 1 : 0.6,
      child: Material(
        color: Colors.transparent,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radii.field),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onPressed,
          child: AnimatedContainer(
            duration: GfMotion.instant,
            curve: GfMotion.standardEase,
            width: size,
            height: size,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: Colors.transparent,
              borderRadius: BorderRadius.circular(radii.field),
            ),
            child: Icon(icon, size: iconSize, color: colors.iconMuted),
          ),
        ),
      ),
    );

    if (tooltip == null) return button;
    return Tooltip(message: tooltip!, child: button);
  }
}
