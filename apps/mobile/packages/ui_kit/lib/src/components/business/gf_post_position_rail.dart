import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Floor position rail mirroring web `PostPositionRail.vue`: a vertical
/// slider over the floor range with "earliest"/"latest" jump buttons and a
/// progress track. Drag or tap to select a floor.
class GfPostPositionRail extends StatelessWidget {
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
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    if (max <= 0) return const SizedBox.shrink();

    final double progress = (current - 1).clamp(0, max - 1) / (max - 1);

    return SizedBox(
      height: 160,
      child: Column(
        children: <Widget>[
          // Top action row.
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: <Widget>[
              TextButton(
                onPressed: onEarliest,
                style: TextButton.styleFrom(
                  foregroundColor: colors.primary,
                  padding: const EdgeInsets.symmetric(horizontal: 8),
                  minimumSize: const Size(40, 32),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                child: Text(startLabel, style: const TextStyle(fontSize: 12)),
              ),
              Text(
                currentLabel ?? '$current / $max',
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: colors.baseContent.withValues(alpha: 0.55),
                ),
              ),
              TextButton(
                onPressed: onLatest,
                style: TextButton.styleFrom(
                  foregroundColor: colors.primary,
                  padding: const EdgeInsets.symmetric(horizontal: 8),
                  minimumSize: const Size(40, 32),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                child: Text(endLabel, style: const TextStyle(fontSize: 12)),
              ),
            ],
          ),
          Expanded(
            child: LayoutBuilder(
              builder: (context, constraints) {
                return GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onTapDown: (details) => _selectFromY(
                    details.localPosition.dy,
                    constraints.maxHeight,
                  ),
                  onVerticalDragUpdate: (details) => _selectFromY(
                    details.localPosition.dy,
                    constraints.maxHeight,
                  ),
                  child: Stack(
                    alignment: Alignment.centerLeft,
                    children: <Widget>[
                      // Track.
                      Container(
                        height: 6,
                        decoration: BoxDecoration(
                          color: colors.base300,
                          borderRadius: BorderRadius.circular(3),
                        ),
                      ),
                      // Progress fill.
                      FractionallySizedBox(
                        widthFactor: progress,
                        child: Container(
                          height: 6,
                          decoration: BoxDecoration(
                            color: colors.primary,
                            borderRadius: BorderRadius.circular(3),
                          ),
                        ),
                      ),
                      // Thumb.
                      Positioned(
                        left: (constraints.maxWidth * progress).clamp(
                          0,
                          constraints.maxWidth - 16,
                        ),
                        child: Container(
                          width: 16,
                          height: 16,
                          decoration: BoxDecoration(
                            color: colors.primary,
                            shape: BoxShape.circle,
                            border: Border.all(color: colors.base100, width: 2),
                          ),
                        ),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  void _selectFromY(double y, double height) {
    if (max <= 0) return;
    final double t = (y / height).clamp(0.0, 1.0);
    final int floor = 1 + (t * (max - 1)).round();
    onSelect(floor.clamp(1, max));
  }
}
