import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Quiet inline feedback, distinct from modal confirmation surfaces.
class GfAlert extends StatelessWidget {
  const GfAlert({
    super.key,
    required this.child,
    this.icon,
    this.padding = const EdgeInsets.all(16),
  });

  final Widget child;
  final Widget? icon;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: colors.base300,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          if (icon != null) ...<Widget>[icon!, const SizedBox(width: 12)],
          Expanded(child: child),
        ],
      ),
    );
  }
}
