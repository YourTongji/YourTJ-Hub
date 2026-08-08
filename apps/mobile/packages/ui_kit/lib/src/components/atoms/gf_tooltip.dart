import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Tooltip styled like web `.gf-tooltip` (components.css): neutral fill,
/// radius field, px-2 py-1, 12px w500, soft shadow. Wraps the Flutter
/// [Tooltip] widget so the mobile tooltip matches the web look.
class GfTooltip extends StatelessWidget {
  const GfTooltip({super.key, required this.message, required this.child});

  final String message;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Tooltip(
      message: message,
      decoration: BoxDecoration(
        color: colors.neutral,
        borderRadius: BorderRadius.circular(radii.field),
        boxShadow: shadows.tooltip,
      ),
      textStyle: TextStyle(
        color: colors.neutralContent,
        fontSize: 12,
        fontWeight: FontWeight.w500,
      ),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: child,
    );
  }
}
