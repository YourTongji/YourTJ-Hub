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
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Tooltip(
      message: message,
      decoration: BoxDecoration(
        color: colors.neutral,
        borderRadius: BorderRadius.circular(GfRadius.sm),
        boxShadow: shadows.tooltip,
      ),
      textStyle: GfTheme.typographyOf(
        context,
      ).caption.copyWith(color: colors.neutralContent),
      padding: const EdgeInsets.symmetric(
        horizontal: GfSpace.md,
        vertical: GfSpace.sm - 2,
      ),
      child: child,
    );
  }
}
