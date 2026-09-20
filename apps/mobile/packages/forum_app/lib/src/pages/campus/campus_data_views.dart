import 'dart:math' as math;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'campus_private_surface.dart';
import 'campus_state.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../widgets/schedule_time_grid.dart';
import 'campus_helpers.dart';
import 'campus_calendar_export.dart';

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
                    Text(metric.label, style: type.caption),
                    const SizedBox(height: 8),
                    Text(
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
                Text(
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
    if (rows.isEmpty) return Text(AppLocalizations.of(context).campusNoData);
    return Column(
      children: [
        for (final row in rows)
          GfCard(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
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
                        Text(
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

class CampusWeekTimetable extends StatefulWidget {
  const CampusWeekTimetable({
    super.key,
    required this.data,
    required this.week,
    required this.maxWeek,
    required this.times,
  });
  final CampusDataset data;
  final int week;
  final int maxWeek;
  final List<SectionTime> times;
  @override
  State<CampusWeekTimetable> createState() => _CampusWeekTimetableState();
}

class _CampusWeekTimetableState extends State<CampusWeekTimetable> {
  late int week = widget.week;
  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final grid = campusGrid(widget.data.events, week);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        CampusCalendarExportButton(enabled: widget.data.events.isNotEmpty),
        const SizedBox(height: 12),
        Wrap(
          alignment: WrapAlignment.center,
          crossAxisAlignment: WrapCrossAlignment.center,
          spacing: 8,
          children: [
            IconButton(
              tooltip: MaterialLocalizations.of(context).previousPageTooltip,
              onPressed: week > 1 ? () => setState(() => week--) : null,
              icon: const Icon(Icons.chevron_left),
            ),
            Text(
              l.scheduleWeekN(week),
              style: GfTheme.typographyOf(context).heading,
            ),
            IconButton(
              tooltip: MaterialLocalizations.of(context).nextPageTooltip,
              onPressed: week < widget.maxWeek
                  ? () => setState(() => week++)
                  : null,
              icon: const Icon(Icons.chevron_right),
            ),
            TextButton(
              onPressed: () => setState(() => week = widget.week),
              child: Text(l.scheduleCurrentWeek),
            ),
          ],
        ),
        const SizedBox(height: 12),
        ScheduleTimeGrid(
          grid: grid,
          times: sectionTimesFor(
            grid.rowHeights.length,
            widget.times.isEmpty ? null : widget.times,
          ),
          onTapCourse: (course) {
            final index = int.tryParse(course.code.split(':')[1]);
            final selectedWeek = week;
            if (index == null) return;
            showGfBottomSheet<void>(
              context,
              builder: (_) => CampusPrivateSurface(
                builder: (_) =>
                    CampusCourseDetails(week: selectedWeek, index: index),
              ),
            );
          },
        ),
        if (campusCoursesForWeek(widget.data.events, week).isEmpty)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(l.campusNoData),
          ),
      ],
    );
  }
}

/// The modal retains only a numeric selection, never a captured school record.
/// Its content follows the same session/background boundary as the campus page.
class CampusCourseDetails extends ConsumerWidget {
  const CampusCourseDetails({
    super.key,
    required this.week,
    required this.index,
  });
  final int week;
  final int index;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(campusControllerProvider);
    final events = campusCoursesForWeek(
      state.data['timetable']?.events ?? [],
      week,
    );
    final event = index >= 0 && index < events.length ? events[index] : null;
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (event == null)
              Text(l.campusNoData)
            else ...[
              Text(event.name, style: GfTheme.typographyOf(context).title2),
              const SizedBox(height: 12),
              Text(
                [
                  event.teacher,
                  event.room,
                  event.campus,
                  l.schedulePeriodRange('${event.start}–${event.end}'),
                  l.scheduleWeeksN(formatWeeksText(event.weeks)),
                ].where((s) => s.isNotEmpty).join('\n'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
