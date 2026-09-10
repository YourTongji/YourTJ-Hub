import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

class GfPillOption<T> {
  const GfPillOption({
    required this.label,
    required this.value,
    required this.icon,
  });

  final String label;
  final T value;
  final IconData icon;
}

/// Compact capsule switch backed by TDesign buttons.
///
/// Mirrors the web home feed-mode control: a bordered base-100 track and a
/// solid primary active option. It intentionally stays content-sized so it
/// can sit beside page actions on narrow mobile toolbars.
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
    final GfBorders borders = GfTheme.bordersOf(context);

    return Container(
      padding: const EdgeInsets.all(1),
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: colors.line, width: borders.width),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          for (final GfPillOption<T> option in options)
            _GfPillItem<T>(
              option: option,
              selected: option.value == selected,
              onPressed: () => onSelected(option.value),
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
    final Color foreground = selected
        ? colors.primaryContent
        : colors.baseContent.withValues(alpha: 0.55);

    return td.TButton(
      size: td.TButtonSize.extraSmall,
      variant: selected ? td.TButtonVariant.fill : td.TButtonVariant.text,
      colorScheme: selected
          ? td.TButtonColorScheme.primary
          : td.TButtonColorScheme.defaultTheme,
      icon: Icon(option.icon, size: 15, color: foreground),
      onPressed: onPressed,
      style: ButtonStyle(
        minimumSize: const WidgetStatePropertyAll<Size>(Size(0, 28)),
        maximumSize: const WidgetStatePropertyAll<Size>(
          Size(double.infinity, 28),
        ),
        padding: const WidgetStatePropertyAll<EdgeInsetsGeometry>(
          EdgeInsets.symmetric(horizontal: 8),
        ),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
        backgroundColor: WidgetStatePropertyAll<Color>(
          selected ? colors.primary : Colors.transparent,
        ),
        foregroundColor: WidgetStatePropertyAll<Color>(foreground),
        elevation: const WidgetStatePropertyAll<double>(0),
        shape: const WidgetStatePropertyAll<OutlinedBorder>(StadiumBorder()),
        textStyle: const WidgetStatePropertyAll<TextStyle>(
          TextStyle(fontSize: 11, fontWeight: FontWeight.w600),
        ),
      ),
      child: Text(option.label),
    );
  }
}
