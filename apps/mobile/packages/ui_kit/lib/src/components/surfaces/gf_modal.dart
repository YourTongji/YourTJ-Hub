import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Modal dialog surface mirroring web `.gf-modal` (motion.css): neutral/40
/// scrim (via `ColorScheme.scrim`), base-100 panel with radius box and
/// `gf-shadows.menu`, entering with a 0.16s fade + child scale(0.98)/6px
/// rise. Use [showGfModal] instead of bare `showDialog` for a consistent
/// dialog look across the app.
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
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Dialog(
      backgroundColor: colors.base100,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.box),
        side: BorderSide(color: colors.line, width: borders.width),
      ),
      child: Container(
        width: width,
        padding: padding,
        decoration: BoxDecoration(boxShadow: shadows.menu),
        child: child,
      ),
    );
  }
}

/// Shows [builder]'s widget in a [GfModal] with the web modal motion
/// (0.16s fade + scale 0.98 + 6px rise). Returns the dialog result.
Future<T?> showGfModal<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
}) {
  return showDialog<T>(
    context: context,
    barrierDismissible: barrierDismissible,
    barrierColor: Theme.of(context).colorScheme.scrim,
    builder: (BuildContext context) => GfModal(child: builder(context)),
  );
}
