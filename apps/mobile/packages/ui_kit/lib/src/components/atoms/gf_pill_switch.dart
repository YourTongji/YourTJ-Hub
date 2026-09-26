import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';

class GfPillOption<T> {
  const GfPillOption({
    required this.label,
    required this.value,
    required this.symbol,
  });

  final String label;
  final T value;
  final String symbol;
}

/// Calm segmented choice with a grey track and native focusable options.
/// Each option keeps a separate 44px target and a scalable label.
class GfPillSwitch<T> extends StatelessWidget {
  const GfPillSwitch({
    super.key,
    required this.options,
    required this.selected,
    required this.onSelected,
  });

  final List<GfPillOption<T>> options;
  final T selected;
  final ValueChanged<T> onSelected;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Container(
      padding: const EdgeInsets.all(2),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          for (final GfPillOption<T> option in options)
            Flexible(
              child: _GfPillItem<T>(
                option: option,
                selected: option.value == selected,
                onPressed: () => onSelected(option.value),
              ),
            ),
        ],
      ),
    );
  }
}

class _GfPillItem<T> extends StatelessWidget {
  const _GfPillItem({
    required this.option,
    required this.selected,
    required this.onPressed,
  });

  final GfPillOption<T> option;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final foreground = selected ? colors.baseContent : colors.iconMuted;
    return MergeSemantics(
      child: Semantics(
        selected: selected,
        child: TextButton(
          onPressed: onPressed,
          style: ButtonStyle(
            minimumSize: const WidgetStatePropertyAll(Size(44, 44)),
            padding: const WidgetStatePropertyAll(
              EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            ),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            visualDensity: VisualDensity.standard,
            backgroundColor: WidgetStatePropertyAll(
              selected ? colors.base100 : Colors.transparent,
            ),
            foregroundColor: WidgetStatePropertyAll(foreground),
            elevation: const WidgetStatePropertyAll(0),
            shape: const WidgetStatePropertyAll(StadiumBorder()),
            textStyle: const WidgetStatePropertyAll(
              TextStyle(fontSize: 13, height: 1.3, fontWeight: FontWeight.w600),
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              GfSymbol(option.symbol, size: 16, color: foreground),
              const SizedBox(width: 6),
              Flexible(child: Text(option.label, textAlign: TextAlign.center)),
            ],
          ),
        ),
      ),
    );
  }
}
