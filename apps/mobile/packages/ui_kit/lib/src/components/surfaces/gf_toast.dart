import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';

/// Toast helper mirroring web `GlobalFlash.vue` on mobile: a bottom,
/// full-width floating snackbar with base-100 background, 1px line border,
/// radius box and the `gf-shadows.floating` elevation. The web flash motion
/// (0.28s, -14px + scale 0.98) is approximated by the Material snackbar
/// entrance; content/colors match the design tokens.
///
/// Usage:
/// ```dart
/// showGfToast(context, 'Saved');
/// ```
void showGfToast(BuildContext context, String message, {bool error = false}) {
  final GfColors colors = GfTheme.colorsOf(context);
  final ScaffoldMessengerState messenger = ScaffoldMessenger.of(context);

  messenger
    ..hideCurrentSnackBar()
    ..showSnackBar(
      SnackBar(
        content: Text(
          message,
          style: TextStyle(
            fontSize: 14,
            color: error ? colors.error : colors.baseContent,
          ),
        ),
        duration: GfMotion.comfortable,
        behavior: SnackBarBehavior.floating,
        margin: const EdgeInsets.fromLTRB(16, 0, 16, 16),
      ),
    );
}
