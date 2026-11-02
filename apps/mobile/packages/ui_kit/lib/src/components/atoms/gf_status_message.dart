import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import 'gf_alert.dart';

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
    final GfTone tone = switch (variant) {
      GfStatusMessageVariant.error => GfTone.error,
      GfStatusMessageVariant.success => GfTone.success,
    };

    return GfAlert(
      tone: tone,
      symbol: variant == GfStatusMessageVariant.error
          ? 'circle-alert'
          : 'circle-check',
      padding: const EdgeInsets.symmetric(horizontal: GfSpace.md, vertical: 10),
      child: GfText(
        message,
        style: TextStyle(
          color: colors.toneOnSoft(tone),
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}
