import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../gf_motion.dart';

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
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: colors.fill,
        borderRadius: BorderRadius.circular(GfRadius.full),
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

    final GfTypography typography = GfTheme.typographyOf(context);
    final Duration duration = GfMotion.duration(context, GfMotion.selection);

    return MergeSemantics(
      child: Semantics(
        selected: selected,
        button: true,
        child: AnimatedContainer(
          duration: duration,
          curve: GfMotion.enterCurve,
          decoration: BoxDecoration(
            color: selected ? gfSegmentThumbColor(colors) : Colors.transparent,
            borderRadius: BorderRadius.circular(GfRadius.full),
            boxShadow: selected
                ? colors.elevation(GfElevation.low)
                : const <BoxShadow>[],
          ),
          child: Material(
            type: MaterialType.transparency,
            child: InkWell(
              onTap: onTap,
              customBorder: const StadiumBorder(),
              child: Container(
                constraints: const BoxConstraints(minHeight: 38),
                padding: const EdgeInsets.symmetric(
                  horizontal: GfSpace.md,
                  vertical: GfSpace.sm,
                ),
                child: Align(
                  alignment: Alignment.center,
                  widthFactor: 1,
                  child: AnimatedDefaultTextStyle(
                    duration: duration,
                    style: typography.subhead.copyWith(
                      fontWeight: FontWeight.w600,
                      color: selected
                          ? colors.textPrimary
                          : colors.textSecondary,
                    ),
                    child: GfText(label, textAlign: TextAlign.center),
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

/// Selected thumb colour shared by [GfSegmented] and `GfPillSwitch`.
Color gfSegmentThumbColor(GfColors colors) => colors.isDark
    ? Color.alphaBlend(const Color(0x29FFFFFF), colors.base100)
    : colors.base100;
