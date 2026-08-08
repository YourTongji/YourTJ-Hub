import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

/// Toast helper mirroring web `GlobalFlash.vue` through TDesign's transient
/// feedback surface. Success and error states keep the YourTJ semantic colors.
///
/// Usage:
/// ```dart
/// showGfToast(context, 'Saved');
/// ```
void showGfToast(BuildContext context, String message, {bool error = false}) {
  final GfColors colors = GfTheme.colorsOf(context);
  if (error) {
    td.TToast.showWarning(
      message,
      context: context,
      maxLines: 3,
      iconColor: colors.error,
    );
  } else {
    td.TToast.showSuccess(
      message,
      context: context,
      maxLines: 3,
      iconColor: colors.success,
    );
  }
}
