import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

/// Modal dialog surface mirroring web `.gf-modal` (motion.css): neutral/40
/// scrim (via `ColorScheme.scrim`) and base-100 panel with radius box.
/// [showGfModal] supplies the native dialog route and keyboard avoidance.
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
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    return td.TDialog(
      backgroundColor: colors.base100,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.box),
        side: BorderSide(color: colors.line, width: borders.width),
      ),
      width: width,
      contentPadding: padding,
      content: child,
    );
  }
}

/// Shows [builder]'s widget in a [GfModal] above the keyboard and returns its
/// dialog result. The builder receives the dialog's safe-area context.
Future<T?> showGfModal<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
}) {
  return showDialog<T>(
    context: context,
    builder: (context) => Dialog(
      backgroundColor: Colors.transparent,
      elevation: 0,
      insetPadding: const EdgeInsets.all(16),
      child: GfModal(child: Builder(builder: builder)),
    ),
    barrierDismissible: barrierDismissible,
    barrierColor: Theme.of(context).colorScheme.scrim,
  );
}
