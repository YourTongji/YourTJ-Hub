import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Accessible horizontal floor selection. A drag previews locally and commits
/// once on release, so selecting a distant floor does not issue many requests.
class GfPostPositionRail extends StatefulWidget {
  const GfPostPositionRail({
    super.key,
    required this.current,
    required this.max,
    required this.onSelect,
    required this.onEarliest,
    required this.onLatest,
    this.startLabel = '最早',
    this.endLabel = '最新',
    this.currentLabel,
  });
  final int current;
  final int max;
  final ValueChanged<int> onSelect;
  final VoidCallback onEarliest;
  final VoidCallback onLatest;
  final String startLabel;
  final String endLabel;
  final String? currentLabel;
  @override
  State<GfPostPositionRail> createState() => _GfPostPositionRailState();
}

class _GfPostPositionRailState extends State<GfPostPositionRail> {
  late double _value;
  double get _maximum => widget.max < 1 ? 1 : widget.max.toDouble();
  @override
  void initState() {
    super.initState();
    _value = widget.current.toDouble().clamp(1, _maximum);
  }

  @override
  void didUpdateWidget(GfPostPositionRail oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.current != oldWidget.current || widget.max != oldWidget.max) {
      _value = widget.current.toDouble().clamp(1, _maximum);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.max <= 0) return const SizedBox.shrink();
    final colors = GfTheme.colorsOf(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            TextButton(
              onPressed: widget.onEarliest,
              child: Text(widget.startLabel),
            ),
            Text(
              widget.currentLabel ?? '${_value.round()} / ${widget.max}',
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                color: colors.baseContent,
              ),
            ),
            TextButton(
              onPressed: widget.onLatest,
              child: Text(widget.endLabel),
            ),
          ],
        ),
        Slider(
          value: _value,
          min: 1,
          max: _maximum,
          semanticFormatterCallback: (value) =>
              '${value.round()} / ${widget.max}',
          label: '${_value.round()}',
          onChanged: widget.max <= 1
              ? null
              : (value) => setState(() => _value = value),
          onChangeEnd: widget.max <= 1
              ? null
              : (value) => widget.onSelect(value.round()),
        ),
      ],
    );
  }
}
