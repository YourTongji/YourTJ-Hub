import 'dart:math' as math;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../schedule/schedule_grid.dart';

/// Shared planner/official timetable. Only callers own editing or persistence.
class ScheduleTimeGrid extends StatefulWidget {
  const ScheduleTimeGrid({
    super.key,
    required this.grid,
    required this.times,
    required this.onTapCourse,
    this.onTapEmptyCell,
    this.emptyLabel,
  });
  final String? emptyLabel;
  final ScheduleGridData grid;
  final List<SectionTime> times;
  final void Function(int day, int section)? onTapEmptyCell;
  final ValueChanged<PkCourseOnTable> onTapCourse;

  @override
  State<ScheduleTimeGrid> createState() => _ScheduleTimeGridState();
}

class _ScheduleTimeGridState extends State<ScheduleTimeGrid> {
  final _horizontal = ScrollController();

  @override
  void dispose() {
    _horizontal.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final l = AppLocalizations.of(context);
    final scaler = MediaQuery.textScalerOf(context);
    // Small text can scale more than titles under nonlinear accessibility scaling.
    final scale = [
      9.5,
      10.0,
      11.0,
      12.0,
    ].map((size) => scaler.scale(size) / size).fold(1.0, math.max);
    final minimumHeights = computeRowHeights(
      GridLayout(
        cellCourses: widget.grid.cellCourses,
        cellSpans: widget.grid.cellSpans,
        occupiedGrid: widget.grid.occupiedGrid,
      ),
      const RowMetrics(baseH: 64, padV: 4, multiCardH: 64),
    );
    final scaled = ScheduleGridData(
      cellCourses: widget.grid.cellCourses,
      cellSpans: widget.grid.cellSpans,
      occupiedGrid: widget.grid.occupiedGrid,
      rowHeights: [
        for (var row = 0; row < minimumHeights.length; row++)
          (math.max(minimumHeights[row], widget.grid.rowHeights[row]) * scale)
              .ceil(),
      ],
      conflicts: widget.grid.conflicts,
    );
    final bodyHeight = scaled.rowHeights.fold<double>(0, (sum, h) => sum + h);
    final headerHeight = 36 * scale;
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.hasBoundedWidth
            ? constraints.maxWidth
            : 588 * scale;
        final timeWidth = math.min(56 * scale, width * .4);
        final available = width - timeWidth - 2;
        final dayWidth = math.max(76 * scale, available / 7);
        final scrolls = dayWidth * 7 > available + 1;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Container(
              decoration: BoxDecoration(
                color: colors.base100,
                borderRadius: BorderRadius.circular(
                  GfTheme.radiiOf(context).box,
                ),
                border: Border.all(color: colors.line),
              ),
              clipBehavior: Clip.antiAlias,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(
                    width: timeWidth,
                    child: Column(
                      children: [
                        _HeaderCell(
                          label: l.scheduleTimeAxis,
                          height: headerHeight,
                        ),
                        for (var row = 0; row < scaled.rowHeights.length; row++)
                          SizedBox(
                            height: scaled.rowHeights[row].toDouble(),
                            child: _TimeCell(row: row, times: widget.times),
                          ),
                      ],
                    ),
                  ),
                  Expanded(
                    child: Scrollbar(
                      controller: _horizontal,
                      child: SingleChildScrollView(
                        key: const PageStorageKey('schedule-grid-horizontal'),
                        controller: _horizontal,
                        scrollDirection: Axis.horizontal,
                        child: SizedBox(
                          width: 7 * dayWidth,
                          child: Column(
                            children: [
                              Row(
                                children: [
                                  for (final day in _weekdayLabels(l))
                                    SizedBox(
                                      width: dayWidth,
                                      child: _HeaderCell(
                                        label: day,
                                        height: headerHeight,
                                      ),
                                    ),
                                ],
                              ),
                              SizedBox(
                                height: bodyHeight,
                                child: Row(
                                  crossAxisAlignment:
                                      CrossAxisAlignment.stretch,
                                  children: [
                                    for (var day = 0; day < 7; day++)
                                      SizedBox(
                                        width: dayWidth,
                                        child: _DayColumn(
                                          day: day,
                                          grid: scaled,
                                          conflicts: scaled.conflicts,
                                          onTapEmptyCell:
                                              widget.onTapEmptyCell == null
                                              ? null
                                              : (section) =>
                                                    widget.onTapEmptyCell!(
                                                      day + 1,
                                                      section,
                                                    ),
                                          onTapCourse: widget.onTapCourse,
                                        ),
                                      ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            if (scrolls)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  l.scheduleGridScrollHint,
                  style: GfTheme.typographyOf(context).caption,
                ),
              ),
            if (widget.emptyLabel != null)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Text(widget.emptyLabel!, textAlign: TextAlign.center),
              ),
          ],
        );
      },
    );
  }
}

List<String> _weekdayLabels(AppLocalizations l) => [
  l.scheduleDayMon,
  l.scheduleDayTue,
  l.scheduleDayWed,
  l.scheduleDayThu,
  l.scheduleDayFri,
  l.scheduleDaySat,
  l.scheduleDaySun,
];

class _HeaderCell extends StatelessWidget {
  const _HeaderCell({required this.label, required this.height});
  final String label;
  final double height;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Container(
      height: height,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: colors.base200.withValues(alpha: .7),
        border: Border(
          bottom: BorderSide(color: colors.line),
          right: BorderSide(color: colors.line),
        ),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w600,
          color: colors.baseContent,
        ),
      ),
    );
  }
}

class _TimeCell extends StatelessWidget {
  const _TimeCell({required this.row, required this.times});

  final int row;
  final List<SectionTime> times;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final SectionTime? time = row < times.length ? times[row] : null;
    final String? partKey = dayPartKeyForRow(row + 1, times);
    final String? partLabel = switch (partKey) {
      'morning' => l10n.scheduleMorning,
      'afternoon' => l10n.scheduleAfternoon,
      'evening' => l10n.scheduleEvening,
      _ => null,
    };
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 1, vertical: 3),
      decoration: BoxDecoration(
        border: Border(
          right: BorderSide(color: colors.line),
          bottom: BorderSide(color: colors.line),
        ),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: <Widget>[
          if (partLabel != null)
            Text(
              partLabel,
              style: TextStyle(
                fontSize: 9.5,
                height: 1.1,
                fontWeight: FontWeight.w700,
                color: colors.primary.withValues(alpha: 0.8),
              ),
            ),
          Text(
            '${row + 1}',
            style: TextStyle(
              fontSize: 12,
              height: 1.1,
              fontWeight: FontWeight.w700,
              color: colors.baseContent.withValues(alpha: 0.7),
            ),
          ),
          if (time != null)
            Text(
              '${time.start}\n${time.end}',
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 9.5,
                height: 1.1,
                color: colors.baseContent.withValues(alpha: 0.75),
              ),
            ),
        ],
      ),
    );
  }
}

/// 单天列：Stack 分层 —— 行发丝线底 + 空格点击层 + 课程卡（行高累计定位）。
class _DayColumn extends StatelessWidget {
  const _DayColumn({
    required this.day,
    required this.grid,
    required this.conflicts,
    required this.onTapEmptyCell,
    required this.onTapCourse,
  });

  final int day;
  final ScheduleGridData grid;
  final Map<String, List<PkConflictItem>> conflicts;
  final void Function(int section)? onTapEmptyCell;
  final void Function(PkCourseOnTable course) onTapCourse;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final int rows = grid.rowHeights.length;
    final List<Widget> children = <Widget>[];

    double yOf(int row) {
      double total = 0;
      for (int i = 0; i < row; i++) {
        total += grid.rowHeights[i].toDouble();
      }
      return total;
    }

    // 行背景（发丝线）。
    for (int row = 0; row < rows; row++) {
      children.add(
        Positioned(
          top: yOf(row),
          left: 0,
          right: 0,
          height: grid.rowHeights[row].toDouble(),
          child: IgnorePointer(
            child: Container(
              decoration: BoxDecoration(
                border: Border(
                  right: BorderSide(color: colors.line),
                  bottom: BorderSide(color: colors.line),
                ),
              ),
            ),
          ),
        ),
      );
    }

    // 单元格：课程卡 or 空格点击层（跨行簇只渲染锚点格）。
    for (int row = 0; row < rows; row++) {
      final List<PkCourseOnTable> courses = grid.cellCourses[row][day];
      final bool covered = grid.occupiedGrid[row][day];
      if (covered) continue;
      if (courses.isNotEmpty) {
        final int span = grid.cellSpans[row][day] < 1
            ? 1
            : grid.cellSpans[row][day];
        double height = 0;
        for (int i = 0; i < span && row + i < rows; i++) {
          height += grid.rowHeights[row + i].toDouble();
        }
        children.add(
          Positioned(
            top: yOf(row),
            left: 1,
            right: 1,
            height: height,
            child: _CourseCell(
              courses: courses,
              compact: courses.length > 1 || span == 1,
              conflicts: conflicts,
              onTapCourse: onTapCourse,
            ),
          ),
        );
      } else if (onTapEmptyCell != null) {
        children.add(
          Positioned(
            top: yOf(row),
            left: 0,
            right: 0,
            height: grid.rowHeights[row].toDouble(),
            child: Semantics(
              button: true,
              excludeSemantics: true,
              label: AppLocalizations.of(context).scheduleEmptyCell(
                _weekdayLabels(AppLocalizations.of(context))[day],
                row + 1,
              ),
              onTap: () => onTapEmptyCell!(row + 1),
              child: Material(
                color: Colors.transparent,
                child: InkWell(
                  excludeFromSemantics: true,
                  onTap: () => onTapEmptyCell!(row + 1),
                  child: const SizedBox.expand(),
                ),
              ),
            ),
          ),
        );
      }
    }

    return Stack(children: children);
  }
}

class _CourseCell extends StatelessWidget {
  const _CourseCell({
    required this.courses,
    required this.compact,
    required this.conflicts,
    required this.onTapCourse,
  });

  final List<PkCourseOnTable> courses;
  final bool compact;
  final Map<String, List<PkConflictItem>> conflicts;
  final void Function(PkCourseOnTable course) onTapCourse;

  @override
  Widget build(BuildContext context) {
    if (courses.length == 1) {
      return _CourseCard(
        course: courses.first,
        compact: compact,
        conflicts: conflicts,
        onTap: () => onTapCourse(courses.first),
      );
    }
    return Column(
      children: <Widget>[
        for (int i = 0; i < courses.length; i++)
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(top: 1, bottom: 1),
              child: _CourseCard(
                course: courses[i],
                compact: compact,
                conflicts: conflicts,
                onTap: () => onTapCourse(courses[i]),
              ),
            ),
          ),
      ],
    );
  }
}

/// 课程卡（课表格内）。
class _CourseCard extends StatelessWidget {
  const _CourseCard({
    required this.course,
    required this.compact,
    required this.conflicts,
    required this.onTap,
  });

  final PkCourseOnTable course;
  final bool compact;
  final Map<String, List<PkConflictItem>> conflicts;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool custom = course.code.startsWith(kCustomEventCodePrefix);
    final bool conflicted = _isConflicted(course);
    final Color fill = custom ? colors.base200 : _slotColor(context, course);
    final String name = course.courseName.isNotEmpty
        ? course.courseName
        : course.code;
    final String teacher = compactTeacherName(teacherNameOf(course), 2);
    final String weeksText = formatDisplayWeeks(
      course.occupyWeek,
      (PkWeekParity parity) => parity == PkWeekParity.odd
          ? l10n.scheduleParityOdd
          : l10n.scheduleParityEven,
      (String range) => l10n.scheduleWeeksN(range),
    );
    final String room = course.occupyRoom ?? '';

    final accent = HSLColor.fromColor(fill)
        .withLightness(
          Theme.of(context).brightness == Brightness.dark ? .72 : .38,
        )
        .toColor();
    final parity = detectWeekParity(course.occupyWeek);
    final compactWeeks = switch (parity) {
      PkWeekParity.odd => l10n.scheduleParityOdd,
      PkWeekParity.even => l10n.scheduleParityEven,
      _ => weeksText,
    };
    final action = custom ? null : onTap;
    final label = [
      name,
      _weekdayLabels(l10n)[course.occupyDay - 1],
      l10n.scheduleSectionsN(formatWeeksText(course.occupyTime)),
      room,
      teacherNameOf(course),
      weeksText,
      if (conflicted) l10n.scheduleConflictBadge,
    ].where((part) => part.isNotEmpty).join(' · ');
    return Semantics(
      label: label,
      button: action != null,
      onTap: action,
      excludeSemantics: true,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 1),
        decoration: BoxDecoration(
          color: fill,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
            color: custom ? colors.line : accent.withValues(alpha: .22),
          ),
        ),
        clipBehavior: Clip.antiAlias,
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            onTap: action,
            excludeFromSemantics: true,
            focusColor: accent.withValues(alpha: .22),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 4),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  if (!custom)
                    Center(
                      child: Container(
                        width: 22,
                        height: 2,
                        decoration: BoxDecoration(
                          color: accent,
                          borderRadius: BorderRadius.circular(2),
                        ),
                      ),
                    ),
                  Text(
                    name,
                    maxLines: compact ? 1 : 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 12,
                      height: 1.2,
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                  if (room.isNotEmpty)
                    Text(
                      room,
                      maxLines: compact ? 1 : 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 11,
                        height: 1.15,
                        fontWeight: FontWeight.w500,
                        color: colors.baseContent,
                      ),
                    ),
                  if (!custom && teacher.isNotEmpty && !compact)
                    Text(
                      teacher,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 10,
                        height: 1.15,
                        color: colors.baseContent.withValues(alpha: .8),
                      ),
                    ),
                  if (weeksText.isNotEmpty || conflicted)
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            compact ? compactWeeks : weeksText,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              fontSize: 9.5,
                              height: 1.15,
                              color: colors.baseContent.withValues(alpha: .8),
                            ),
                          ),
                        ),
                        if (conflicted)
                          GfSymbol(
                            'circle-alert',
                            size: 14,
                            color: colors.error,
                          ),
                      ],
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  /// 冲突判定：deriveConflicts 以基础课号/custom 伪课号为键。
  bool _isConflicted(PkCourseOnTable course) {
    final String base = course.code.startsWith(kCustomEventCodePrefix)
        ? course.code
        : getCourseBaseCode(course.code);
    return (conflicts[base] ?? const <PkConflictItem>[]).isNotEmpty;
  }

  Color _slotColor(BuildContext context, PkCourseOnTable course) {
    final int slot = courseColorSlotFor(
      course.code.startsWith('campus:') || course.code.isEmpty
          ? course.courseName
          : course.code,
    );
    final bool dark = Theme.of(context).brightness == Brightness.dark;
    const List<Color> light = <Color>[
      Color(0xFFE8EEFF),
      Color(0xFFDCF2FE),
      Color(0xFFDDF3E7),
      Color(0xFFFDF0D5),
      Color(0xFFFFE9E7),
      Color(0xFFF1E8FB),
      Color(0xFFE0F2FE),
      Color(0xFFFFECF1),
    ];
    const List<Color> darkColors = <Color>[
      Color(0xFF1E2A4A),
      Color(0xFF12303F),
      Color(0xFF123527),
      Color(0xFF3A2F10),
      Color(0xFF482029),
      Color(0xFF2E1D40),
      Color(0xFF12303F),
      Color(0xFF402030),
    ];
    return (dark ? darkColors : light)[(slot - 1) % 8];
  }
}
