import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_symbol.dart';

/// Floating primary action: a circular compose glyph or a labelled capsule.
/// Native button behavior retains keyboard focus and activation.
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
    final colors = GfTheme.colorsOf(context);
    final pill = label != null;
    final active = enabled && onPressed != null;
    final glyph = icon == Icons.edit
        ? GfSymbol('square-pen', size: pill ? 20 : 24)
        : Icon(icon, size: pill ? 20 : 24);
    final action = DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(999),
        boxShadow: active ? GfTheme.shadowsOf(context).floating : null,
      ),
      child: FilledButton(
        onPressed: active ? onPressed : null,
        style: ButtonStyle(
          minimumSize: WidgetStatePropertyAll(
            Size(pill ? 48 : 56, pill ? 48 : 56),
          ),
          padding: WidgetStatePropertyAll(
            EdgeInsets.symmetric(
              horizontal: pill ? 18 : 0,
              vertical: pill ? 10 : 0,
            ),
          ),
          tapTargetSize: MaterialTapTargetSize.padded,
          visualDensity: VisualDensity.standard,
          backgroundColor: WidgetStatePropertyAll(
            active ? colors.primary : colors.base300,
          ),
          foregroundColor: WidgetStatePropertyAll(
            active ? colors.primaryContent : colors.iconMuted,
          ),
          shape: const WidgetStatePropertyAll(StadiumBorder()),
          elevation: const WidgetStatePropertyAll(0),
          textStyle: const WidgetStatePropertyAll(
            TextStyle(fontSize: 15, height: 1.3, fontWeight: FontWeight.w600),
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            glyph,
            if (pill) ...[
              const SizedBox(width: 8),
              Flexible(child: Text(label!, textAlign: TextAlign.center)),
            ],
          ],
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
