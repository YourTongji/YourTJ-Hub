import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../gf_symbol.dart';

/// Controlled capsule selection with compact paint and a padded touch target.
class GfSelectTag extends StatelessWidget {
  const GfSelectTag({
    super.key,
    required this.label,
    required this.selected,
    required this.onChanged,
    this.symbol,
  });

  final String label;
  final bool selected;
  final ValueChanged<bool>? onChanged;
  final String? symbol;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final foreground = selected ? colors.primary : colors.textPrimary;
    return MergeSemantics(
      child: Semantics(
        selected: selected,
        child: Opacity(
          opacity: onChanged == null ? 0.5 : 1,
          child: TextButton(
            onPressed: onChanged == null ? null : () => onChanged!(!selected),
            style: ButtonStyle(
              backgroundColor: WidgetStatePropertyAll(
                selected ? colors.primarySoft : colors.fill,
              ),
              foregroundColor: WidgetStatePropertyAll(foreground),
              overlayColor: WidgetStatePropertyAll(colors.pressed),
              minimumSize: const WidgetStatePropertyAll(
                Size(44, GfControl.compact),
              ),
              padding: const WidgetStatePropertyAll(
                EdgeInsets.symmetric(horizontal: 14, vertical: 6),
              ),
              tapTargetSize: MaterialTapTargetSize.padded,
              visualDensity: VisualDensity.standard,
              shape: const WidgetStatePropertyAll(StadiumBorder()),
              textStyle: const WidgetStatePropertyAll(
                TextStyle(
                  fontSize: 14,
                  height: 20 / 14,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (symbol != null || selected) ...[
                  if (symbol != null)
                    GfSymbol(symbol!, size: GfIconSize.sm, color: foreground)
                  else
                    GfSymbol('check', size: GfIconSize.sm, color: foreground),
                  const SizedBox(width: GfSpace.xs + 2),
                ],
                Flexible(child: GfText(label, textAlign: TextAlign.center)),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
