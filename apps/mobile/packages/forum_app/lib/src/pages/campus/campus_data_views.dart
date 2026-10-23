import 'dart:math' as math;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';

export 'timetable/campus_timetable.dart'
    show CampusTimetableStatus, CampusWeekTimetable;
export 'timetable/campus_timetable_sheets.dart' show CampusCourseDetails;

class CampusMetrics extends StatelessWidget {
  const CampusMetrics({super.key, required this.data});
  final CampusDataset data;
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final type = GfTheme.typographyOf(context);
      final wide =
          constraints.maxWidth >= 360 &&
          MediaQuery.textScalerOf(context).scale(16) < 25;
      return Wrap(
        spacing: 12,
        runSpacing: 12,
        children: [
          for (final metric in data.metrics)
            SizedBox(
              width: wide
                  ? (constraints.maxWidth - 12) / 2
                  : constraints.maxWidth,
              child: GfCard(
                emphasized: true,
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfText(metric.label, style: type.caption),
                    const SizedBox(height: 8),
                    GfText(
                      metric.value.isEmpty
                          ? '—'
                          : '${metric.value}${metric.unit}',
                      style: type.title2,
                    ),
                  ],
                ),
              ),
            ),
        ],
      );
    },
  );
}

class CampusChart extends StatelessWidget {
  const CampusChart({super.key, required this.points, required this.maximum});
  final List<CampusPoint> points;
  final double maximum;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final valid = points
        .where((p) => p.value.isFinite && p.value >= 0)
        .toList();
    final max = valid.fold<double>(maximum, (m, p) => math.max(m, p.value));
    return Column(
      children: [
        for (final p in valid)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                GfText(
                  '${p.label} · ${p.value.toStringAsFixed(2)}',
                  style: GfTheme.typographyOf(context).small,
                ),
                const SizedBox(height: 8),
                Semantics(
                  label: '${p.label}: ${p.value}',
                  child: LinearProgressIndicator(
                    value: max <= 0 ? 0 : (p.value / max).clamp(0, 1),
                    minHeight: 8,
                    borderRadius: BorderRadius.circular(4),
                    color: colors.primary,
                    backgroundColor: colors.base200,
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

/// Each record grows vertically; no tiny horizontally scrolling grade table.
class CampusRecords extends StatelessWidget {
  const CampusRecords({
    super.key,
    required this.data,
    this.query = '',
    this.titleColumn = 0,
  });
  final CampusDataset data;
  final String query;
  final int titleColumn;
  @override
  Widget build(BuildContext context) {
    final rows = data.rows
        .where(
          (r) => r.join(' ').toLowerCase().contains(query.trim().toLowerCase()),
        )
        .toList();
    final type = GfTheme.typographyOf(context);
    if (rows.isEmpty) return GfText(AppLocalizations.of(context).campusNoData);
    return Column(
      children: [
        for (final row in rows)
          GfCard(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                GfText(
                  row.length > titleColumn && row[titleColumn].isNotEmpty
                      ? row[titleColumn]
                      : '—',
                  style: type.heading,
                ),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 16,
                  runSpacing: 6,
                  children: [
                    for (var i = 0; i < data.columns.length; i++)
                      if (i != titleColumn)
                        GfText(
                          '${data.columns[i]}：${i < row.length && row[i].isNotEmpty ? row[i] : '—'}',
                          style: type.small,
                        ),
                  ],
                ),
              ],
            ),
          ),
      ],
    );
  }
}
