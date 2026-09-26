import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Quiet segmented choice with native focus, selected semantics and a minimum
/// 48-pixel target. Labels may wrap as the system text size grows.
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

    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: colors.base300,
        borderRadius: BorderRadius.circular(18),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final stretch = expanded && constraints.hasBoundedWidth;
          return Row(
            mainAxisSize: stretch ? MainAxisSize.max : MainAxisSize.min,
            children: [
              for (final segment in segments)
                Flexible(
                  fit: stretch ? FlexFit.tight : FlexFit.loose,
                  child: _SegmentedItem<T>(
                    label: segment.$1,
                    selected: segment.$2 == selected,
                    onTap: () => onSelected(segment.$2),
                  ),
                ),
            ],
          );
        },
      ),
    );
  }
}

class _SegmentedItem<T> extends StatelessWidget {
  const _SegmentedItem({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return MergeSemantics(
      child: Semantics(
        selected: selected,
        button: true,
        child: Material(
          color: selected ? colors.base100 : Colors.transparent,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(14),
          ),
          child: InkWell(
            onTap: onTap,
            borderRadius: BorderRadius.circular(14),
            child: Container(
              constraints: const BoxConstraints(minHeight: 48),
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              child: Align(
                alignment: Alignment.center,
                widthFactor: 1,
                child: Text(
                  label,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: selected
                        ? colors.baseContent
                        : colors.baseContent.withValues(alpha: 0.55),
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
