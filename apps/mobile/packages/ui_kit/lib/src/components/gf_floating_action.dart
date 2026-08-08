import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../theme/gf_theme.dart';

/// Floating publish action pinned to the bottom of the screen, mirroring
/// `TopicFloatingControls.vue`'s rounded floating surface.
///
/// Defaults to a circular primary button; set [label] to render the pill
/// variant with icon + text used for "join discussion". The shadow uses the
/// web `gf-shadows.floating` elevation.
class GfFloatingAction extends StatelessWidget {
  const GfFloatingAction({
    super.key,
    required this.onPressed,
    this.icon = Icons.edit,
    this.label,
    this.enabled = true,
    this.bottomInset = 16,
  });

  final VoidCallback? onPressed;
  final IconData icon;
  final String? label;
  final bool enabled;
  final double bottomInset;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    final bool pill = label != null;
    final Widget action = DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(9999),
        boxShadow: shadows.floating,
      ),
      child: SizedBox(
        width: pill ? null : 56,
        height: pill ? 44 : 56,
        child: td.TButton(
          size: td.TButtonSize.large,
          variant: td.TButtonVariant.fill,
          colorScheme: td.TButtonColorScheme.primary,
          icon: Icon(icon, size: pill ? 18 : 24),
          onPressed: enabled ? onPressed : null,
          style: ButtonStyle(
            backgroundColor: WidgetStatePropertyAll<Color>(colors.primary),
            foregroundColor: WidgetStatePropertyAll<Color>(
              colors.primaryContent,
            ),
            padding: WidgetStatePropertyAll<EdgeInsetsGeometry>(
              EdgeInsets.symmetric(horizontal: pill ? 16 : 0),
            ),
            shape: const WidgetStatePropertyAll<OutlinedBorder>(
              StadiumBorder(),
            ),
            elevation: const WidgetStatePropertyAll<double>(0),
          ),
          child: pill ? Text(label!) : null,
        ),
      ),
    );

    return Align(
      alignment: Alignment.bottomCenter,
      child: Padding(
        padding: EdgeInsets.only(bottom: bottomInset),
        child: action,
      ),
    );
  }
}
