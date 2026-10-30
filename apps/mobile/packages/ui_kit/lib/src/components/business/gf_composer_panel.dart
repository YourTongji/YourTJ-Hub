import 'dart:math' as math;

import 'package:flutter/material.dart';

/// Inline keyboard alternative, leaving space for the editor above it.
class GfComposerPanel extends StatelessWidget {
  const GfComposerPanel({super.key, required this.child, this.minHeight = 0});
  final Widget child;

  /// Height the content needs to be useful; honoured up to 60% of the space
  /// left for the composer.
  final double minHeight;

  @override
  Widget build(BuildContext context) {
    final media = MediaQuery.of(context);
    final available =
        media.size.height - media.viewInsets.bottom - media.padding.vertical;
    final height = math.max(
      (available * .36).clamp(80.0, 300.0),
      math.min(minHeight, available * .6),
    );
    return SizedBox(height: height, child: child);
  }
}
