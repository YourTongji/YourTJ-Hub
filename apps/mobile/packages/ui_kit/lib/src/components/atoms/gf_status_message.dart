import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Inline status message, mirroring web `.gf-status-message` (components.css):
/// radius 6, px-3 py-2, 14px w500. Tint fills use the 10% alpha semantic color.
enum GfStatusMessageVariant {
  /// `gf-status-message-error`: error/10 fill, error text.
  error,

  /// `gf-status-message-success`: success/10 fill, success text.
  success,
}

/// Form-level status message aligned with web `gf-status-message`.
class GfStatusMessage extends StatelessWidget {
  const GfStatusMessage({
    super.key,
    required this.message,
    this.variant = GfStatusMessageVariant.error,
  });

  final String message;
  final GfStatusMessageVariant variant;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    final (Color background, Color foreground) = switch (variant) {
      GfStatusMessageVariant.error => (
        colors.error.withValues(alpha: 0.10),
        colors.error,
      ),
      GfStatusMessageVariant.success => (
        colors.success.withValues(alpha: 0.10),
        colors.success,
      ),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        message,
        style: TextStyle(
          color: foreground,
          fontSize: 14,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}
