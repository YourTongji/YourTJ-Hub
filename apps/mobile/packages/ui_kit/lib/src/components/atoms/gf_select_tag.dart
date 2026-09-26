import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';

/// Controlled capsule selection with compact paint and a padded touch target.
class GfSelectTag extends StatelessWidget {
  const GfSelectTag({
    super.key,
    required this.label,
    required this.selected,
    required this.onChanged,
    this.icon,
  });

  final String label;
  final bool selected;
  final ValueChanged<bool>? onChanged;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final foreground = selected ? colors.base100 : colors.baseContent;
    return MergeSemantics(
      child: Semantics(
        selected: selected,
        child: Opacity(
          opacity: onChanged == null ? 0.5 : 1,
          child: TextButton(
            onPressed: onChanged == null ? null : () => onChanged!(!selected),
            style: ButtonStyle(
              backgroundColor: WidgetStatePropertyAll(
                selected ? colors.baseContent : colors.base200,
              ),
              foregroundColor: WidgetStatePropertyAll(foreground),
              minimumSize: const WidgetStatePropertyAll(Size(44, 32)),
              padding: const WidgetStatePropertyAll(
                EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              ),
              tapTargetSize: MaterialTapTargetSize.padded,
              visualDensity: VisualDensity.standard,
              shape: const WidgetStatePropertyAll(StadiumBorder()),
              textStyle: const WidgetStatePropertyAll(
                TextStyle(
                  fontSize: 14,
                  height: 1.25,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (icon != null || selected) ...[
                  if (icon != null)
                    Icon(icon, size: 16)
                  else
                    GfSymbol('check', size: 16, color: foreground),
                  const SizedBox(width: 6),
                ],
                Flexible(child: Text(label, textAlign: TextAlign.center)),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
