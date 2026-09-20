import 'package:core/core.dart';

/// 课程卡片配色槽位数（web courseColorSlots 8 槽；种子 hash 取模）。
const int kCourseColorSlotCount = 8;

/// 依据课程稳定标识（课号/课名）hash 出 1-based 配色槽位，同课同色。
int courseColorSlotFor(String seed) {
  var h = 0;
  for (var i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.codeUnitAt(i)) & 0xFFFFFFFF;
  }
  return (h % kCourseColorSlotCount) + 1;
}

/// 课表网格派生数据（不持久化，the planner 变更时重建）。
class ScheduleGridData {
  const ScheduleGridData({
    required this.cellCourses,
    required this.cellSpans,
    required this.occupiedGrid,
    required this.rowHeights,
    required this.conflicts,
  });

  /// cellCourses[row][day]：该格课程（同节次簇合并后；被上方 rowspan 覆盖的
  /// 槽位为空数组）。
  final List<List<List<PkCourseOnTable>>> cellCourses;

  /// cellSpans[row][day]：该格纵向跨越行数（默认 1）。
  final List<List<int>> cellSpans;

  /// occupiedGrid[row][day]：该格是否已被上方跨行格占用。
  final List<List<bool>> occupiedGrid;

  /// 每行渲染高度（px；kInteractiveRowMetricsMobile 计算）。
  final List<int> rowHeights;

  /// deriveConflicts 全量冲突（key 为基础课号或 custom: 伪课号）。
  final Map<String, List<PkConflictItem>> conflicts;
}

/// 从（周次过滤后的）平铺课表行构建网格：
/// 同天按节次区间聚类（相交/包含同格）→ consolidate 同班多段 → spans/覆盖表
/// → computeRowHeights 行高。occup 仅用于 deriveConflicts（全量冲突标注）。
ScheduleGridData buildScheduleGridData(
  List<PkCourseOnTable> table,
  List<List<List<PkOccupyCell>>> occupied, {
  required int maxRows,
}) {
  final List<List<PkCourseOnTable>> byDay = List.generate(
    7,
    (_) => <PkCourseOnTable>[],
  );
  for (final course in table) {
    if (course.occupyTime.isEmpty) continue;
    if (course.occupyDay < 1 || course.occupyDay > 7) continue;
    if (course.occupyTime.any((s) => s < 1 || s > maxRows)) continue;
    byDay[course.occupyDay - 1].add(course);
  }

  final List<List<List<PkCourseOnTable>>> cellCourses = List.generate(
    maxRows,
    (_) => List.generate(7, (_) => <PkCourseOnTable>[]),
  );
  final List<List<int>> cellSpans = List.generate(
    maxRows,
    (_) => List.filled(7, 1),
  );
  final List<List<bool>> covered = List.generate(
    maxRows,
    (_) => List.filled(7, false),
  );

  for (int day = 0; day < 7; day++) {
    final List<PkDayCluster<PkCourseOnTable>> clusters = clusterBySections(
      byDay[day],
      (course) => course.occupyTime,
    );
    for (final cluster in clusters) {
      final List<PkConsolidatableCourse> consolidated =
          consolidateSameClassArrangements(
            cluster.items
                .map(
                  (course) => PkConsolidatableCourse(
                    code: course.code,
                    courseName: course.courseName,
                    occupyDay: course.occupyDay,
                    occupyTime: List<int>.from(course.occupyTime),
                    occupyWeek: course.occupyWeek == null
                        ? null
                        : List<int>.from(course.occupyWeek!),
                    teacherAndCode: course.teacherAndCode,
                    arrangementText: course.arrangementText,
                    occupyRoom: course.occupyRoom,
                    showText: course.showText,
                  ),
                )
                .toList(),
          );
      final int row = cluster.start - 1;
      final int span = cluster.end - row;
      cellCourses[row][day] = consolidated
          .map(
            (c) => PkCourseOnTable(
              showText: c.showText ?? '',
              courseName: c.courseName,
              code: c.code,
              occupyTime: List<int>.from(c.occupyTime),
              occupyDay: c.occupyDay,
              occupyWeek: c.occupyWeek == null
                  ? null
                  : List<int>.from(c.occupyWeek!),
              teacherAndCode: c.teacherAndCode,
              arrangementText: c.arrangementText,
              occupyRoom: c.occupyRoom,
            ),
          )
          .toList();
      cellSpans[row][day] = span;
      for (int r = row + 1; r < row + span && r < maxRows; r++) {
        covered[r][day] = true;
      }
    }
  }

  final GridLayout layout = GridLayout(
    cellCourses: cellCourses,
    cellSpans: cellSpans,
    occupiedGrid: covered,
  );
  final List<int> rowHeights = computeRowHeights(
    layout,
    kInteractiveRowMetricsMobile,
  );
  final Map<String, List<PkConflictItem>> conflicts = deriveConflicts(occupied);

  return ScheduleGridData(
    cellCourses: cellCourses,
    cellSpans: cellSpans,
    occupiedGrid: covered,
    rowHeights: rowHeights,
    conflicts: conflicts,
  );
}
