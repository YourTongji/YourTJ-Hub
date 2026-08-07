import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// Floating publish action pinned to the bottom of the screen, mirroring
/// `TopicFloatingControls.vue`'s rounded floating surface.
///
/// Defaults to a circular primary button; set [label] to render the pill
/// variant with icon + text used for "join discussion". The shadow uses the
/// web `gf-shadows.floating` elevation.
class GfFloatingAction extends StatelessWidget {
  const GfFloatingAction({
    super.key,
    required this.onPressed,
    this.icon = Icons.edit,
    this.label,
    this.enabled = true,
    this.bottomInset = 16,
  });

  final VoidCallback? onPressed;
  final IconData icon;
  final String? label;
  final bool enabled;
  final double bottomInset;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    final Widget action;
    if (label != null) {
      action = Material(
        color: colors.primary,
        shape: const StadiumBorder(),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: enabled ? onPressed : null,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                Icon(icon, size: 18, color: colors.primaryContent),
                const SizedBox(width: 6),
                Text(
                  label!,
                  style: TextStyle(
                    color: colors.primaryContent,
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    } else {
      action = Material(
        color: colors.primary,
        shape: CircleBorder(
          side: BorderSide(color: colors.line, width: borders.width),
        ),
        shadowColor: Colors.transparent,
        clipBehavior: Clip.antiAlias,
        child: DecoratedBox(
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            boxShadow: shadows.floating,
          ),
          child: InkWell(
            onTap: enabled ? onPressed : null,
            customBorder: const CircleBorder(),
            child: SizedBox(
              width: 56,
              height: 56,
              child: Icon(icon, color: colors.primaryContent, size: 24),
            ),
          ),
        ),
      );
    }

    return Align(
      alignment: Alignment.bottomCenter,
      child: Padding(
        padding: EdgeInsets.only(bottom: bottomInset),
        child: action,
      ),
    );
  }
}
