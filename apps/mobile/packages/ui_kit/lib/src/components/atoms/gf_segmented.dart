import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Segmented control, mirroring web `.gf-segmented` / `.gf-segmented-item`
/// (components.css): 1px line border, radius field, base-200 track with 2px
/// padding; items are h-8 (32px), 14px w600, radius field-2 (6px); the active
/// item is base-100 with primary text, a 1px line ring and `shadow-sm`.
class GfSegmented<T> extends StatelessWidget {
  const GfSegmented({
    super.key,
    required this.segments,
    required this.selected,
    required this.onSelected,
    this.expanded = true,
  });

  /// (label, value) pairs rendered left to right.
  final List<(String, T)> segments;

  final T selected;
  final ValueChanged<T> onSelected;

  /// Whether the control stretches to fill available width (web
  /// `inline-grid`; mobile keeps it full-width by default).
  final bool expanded;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    return Container(
      padding: const EdgeInsets.all(2),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(radii.field),
        border: Border.all(color: colors.line, width: borders.width),
      ),
      child: Row(
        mainAxisSize: expanded ? MainAxisSize.max : MainAxisSize.min,
        children: <Widget>[
          for (int i = 0; i < segments.length; i++)
            Expanded(
              child: _SegmentedItem<T>(
                label: segments[i].$1,
                value: segments[i].$2,
                selected: segments[i].$2 == selected,
                onTap: () => onSelected(segments[i].$2),
              ),
            ),
        ],
      ),
    );
  }
}

class _SegmentedItem<T> extends StatelessWidget {
  const _SegmentedItem({
    required this.label,
    required this.value,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final T value;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);

    return Material(
      color: selected ? colors.base100 : Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.field - 2),
        side: selected
            ? BorderSide(color: colors.line, width: 1)
            : BorderSide.none,
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(radii.field - 2),
        child: Container(
          height: 32,
          padding: const EdgeInsets.symmetric(horizontal: 8),
          alignment: Alignment.center,
          child: Text(
            label,
            style: TextStyle(
              color: selected
                  ? colors.primary
                  : colors.baseContent.withValues(alpha: 0.55),
              fontSize: 14,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}
