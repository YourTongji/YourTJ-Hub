import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../schedule/schedule_grid.dart';

/// Shared planner/official timetable. Only callers own editing or persistence.
class ScheduleTimeGrid extends StatelessWidget {
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
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    // Keep grid labels and cards readable at large accessibility text sizes.
    final scale = MediaQuery.textScalerOf(context).scale(12) / 12;
    final scaled = ScheduleGridData(
      cellCourses: grid.cellCourses,
      cellSpans: grid.cellSpans,
      occupiedGrid: grid.occupiedGrid,
      rowHeights: grid.rowHeights.map((h) => (h * scale).ceil()).toList(),
      conflicts: grid.conflicts,
    );
    return Container(
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
        border: Border.all(color: colors.line),
      ),
      clipBehavior: Clip.antiAlias,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: SizedBox(
          width: (44 + 7 * 62) * scale,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _DayHeaderRow(
                timeColumnWidth: 44 * scale,
                dayColumnWidth: 62 * scale,
              ),
              SizedBox(
                height: scaled.rowHeights.fold<double>(0, (sum, h) => sum + h),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    SizedBox(
                      width: 44 * scale,
                      child: Column(
                        children: [
                          for (
                            var row = 0;
                            row < scaled.rowHeights.length;
                            row++
                          )
                            SizedBox(
                              height: scaled.rowHeights[row].toDouble(),
                              child: _TimeCell(row: row, times: times),
                            ),
                        ],
                      ),
                    ),
                    for (var day = 0; day < 7; day++)
                      SizedBox(
                        width: 62 * scale,
                        child: _DayColumn(
                          day: day,
                          grid: scaled,
                          conflicts: scaled.conflicts,
                          onTapEmptyCell: onTapEmptyCell == null
                              ? null
                              : (section) => onTapEmptyCell!(day + 1, section),
                          onTapCourse: onTapCourse,
                        ),
                      ),
                  ],
                ),
              ),
              if (emptyLabel != null)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 10),
                  child: Text(
                    emptyLabel!,
                    textAlign: TextAlign.center,
                    style: TextStyle(fontSize: 12, color: colors.iconMuted),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 天头部（周一..周日）。
class _DayHeaderRow extends StatelessWidget {
  const _DayHeaderRow({
    required this.timeColumnWidth,
    required this.dayColumnWidth,
  });

  final double timeColumnWidth;
  final double dayColumnWidth;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final List<String> days = <String>[
      l10n.scheduleDayMon,
      l10n.scheduleDayTue,
      l10n.scheduleDayWed,
      l10n.scheduleDayThu,
      l10n.scheduleDayFri,
      l10n.scheduleDaySat,
      l10n.scheduleDaySun,
    ];
    return Container(
      height: 32 * MediaQuery.textScalerOf(context).scale(11) / 11,
      decoration: BoxDecoration(
        color: colors.base200.withValues(alpha: 0.7),
        border: Border(bottom: BorderSide(color: colors.line)),
      ),
      child: Row(
        children: <Widget>[
          SizedBox(width: timeColumnWidth),
          for (final String day in days)
            SizedBox(
              width: dayColumnWidth,
              child: Center(
                child: Text(
                  day,
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: colors.baseContent.withValues(alpha: 0.7),
                  ),
                ),
              ),
            ),
        ],
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
                fontSize: 8,
                height: 1.1,
                fontWeight: FontWeight.w700,
                color: colors.primary.withValues(alpha: 0.8),
              ),
            ),
          Text(
            '${row + 1}',
            style: TextStyle(
              fontSize: 11,
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
                fontSize: 7,
                height: 1.1,
                color: colors.baseContent.withValues(alpha: 0.45),
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
            child: GestureDetector(
              onTap: () => onTapEmptyCell!(row + 1),
              behavior: HitTestBehavior.opaque,
              child: const SizedBox.expand(),
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
    required this.conflicts,
    required this.onTapCourse,
  });

  final List<PkCourseOnTable> courses;
  final Map<String, List<PkConflictItem>> conflicts;
  final void Function(PkCourseOnTable course) onTapCourse;

  @override
  Widget build(BuildContext context) {
    if (courses.length == 1) {
      return _CourseCard(
        course: courses.first,
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
    required this.conflicts,
    required this.onTap,
  });

  final PkCourseOnTable course;
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

    return Container(
      decoration: BoxDecoration(
        color: fill,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(
          color: custom ? colors.line : colors.primary.withValues(alpha: 0.22),
        ),
      ),
      clipBehavior: Clip.antiAlias,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          child: Stack(
            children: <Widget>[
              Padding(
                padding: const EdgeInsets.fromLTRB(5, 3, 5, 3),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: <Widget>[
                    Text(
                      name,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 11,
                        height: 1.15,
                        fontWeight: FontWeight.w600,
                        color: colors.baseContent,
                      ),
                    ),
                    if (!custom && (room.isNotEmpty || teacher.isNotEmpty))
                      Padding(
                        padding: const EdgeInsets.only(top: 1),
                        child: Text(
                          room.isNotEmpty ? room : teacher,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 8,
                            height: 1.1,
                            color: colors.baseContent.withValues(alpha: 0.55),
                          ),
                        ),
                      ),
                    if (weeksText.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 1),
                        child: Text(
                          weeksText,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 8,
                            height: 1.1,
                            color: colors.baseContent.withValues(alpha: 0.5),
                          ),
                        ),
                      ),
                  ],
                ),
              ),
              if (conflicted && !custom)
                Positioned(
                  right: 2,
                  top: 2,
                  child: Container(
                    width: 13,
                    height: 13,
                    decoration: BoxDecoration(
                      color: colors.error.withValues(alpha: 0.15),
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: colors.error.withValues(alpha: 0.5),
                      ),
                    ),
                    child: Icon(
                      Icons.warning_amber_rounded,
                      size: 9,
                      color: colors.error,
                    ),
                  ),
                ),
            ],
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
