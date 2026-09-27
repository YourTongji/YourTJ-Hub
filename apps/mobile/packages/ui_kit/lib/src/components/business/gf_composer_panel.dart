import 'package:flutter/material.dart';

/// Inline keyboard alternative, leaving space for the editor above it.
class GfComposerPanel extends StatelessWidget {
  const GfComposerPanel({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final media = MediaQuery.of(context);
    final available =
        media.size.height - media.viewInsets.bottom - media.padding.vertical;
    return SizedBox(height: (available * .36).clamp(80.0, 300.0), child: child);
  }
}
