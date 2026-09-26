import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Bounded, softly rounded modal content. [showGfModal] owns the single native
/// dialog route, keyboard avoidance and focus restoration.
class GfModal extends StatelessWidget {
  const GfModal({
    super.key,
    required this.child,
    this.width,
    this.padding = const EdgeInsets.all(20),
  });

  final Widget child;
  final double? width;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: GfTheme.colorsOf(context).base100,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
      clipBehavior: Clip.antiAlias,
      child: ConstrainedBox(
        constraints: BoxConstraints(maxWidth: width ?? 560),
        child: SingleChildScrollView(padding: padding, child: child),
      ),
    );
  }
}

/// Shows a modal above shell navigation with native safe-area and IME insets.
Future<T?> showGfModal<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
}) {
  return showDialog<T>(
    context: context,
    useRootNavigator: true,
    builder: (context) => Dialog(
      backgroundColor: Colors.transparent,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      insetPadding: const EdgeInsets.all(16),
      child: GfModal(child: Builder(builder: builder)),
    ),
    barrierDismissible: barrierDismissible,
    barrierColor: Theme.of(context).colorScheme.scrim,
  );
}
