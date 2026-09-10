/// 课表网格共享几何 helper（移植自 web `resource/src/site/utils/timetableGrid.ts`）。
///
/// 交互网格与 PNG 海报必须视觉一致：行高分配算法以参数化度量
/// [RowMetrics] 保留两端差异，避免布局调整静默分叉。
library;

import 'pk_arrange.dart';
import 'pk_models.dart';
import 'pk_section_times.dart';

/// 行高度量（px）。
class RowMetrics {
  const RowMetrics({
    required this.baseH,
    required this.padV,
    required this.multiCardH,
  });

  /// 单行基准高度。
  final int baseH;

  /// 单元格上下 padding 合计。
  final int padV;

  /// 多门课叠放时单张紧凑卡片的估算高度。
  final int multiCardH;
}

/// 交互网格的行高度量（mobile）：
/// p-1(4*2)=8 + 课名12 + 教室10 + 周次10 + gap(2*4)=8 ≈ 58px。
const RowMetrics kInteractiveRowMetricsMobile = RowMetrics(
  baseH: 52,
  padV: 4,
  multiCardH: 58,
);

/// 交互网格的行高度量（桌面）。
const RowMetrics kInteractiveRowMetricsDesktop = RowMetrics(
  baseH: 58,
  padV: 8,
  multiCardH: 72,
);

/// 导出海报的行高度量：画幅固定 1140px，行高更舒展。
const RowMetrics kPosterRowMetrics = RowMetrics(
  baseH: 76,
  padV: 8,
  multiCardH: 90,
);

/// 网格几何输入：三份网格派生数据。
class GridLayout {
  const GridLayout({
    required this.cellCourses,
    required this.cellSpans,
    required this.occupiedGrid,
  });

  /// cellCourses[row][day] = 该格课程列表。
  final List<List<List<PkCourseOnTable>>> cellCourses;

  /// cellSpans[row][day] = 该格纵向跨越行数（默认 1）。
  final List<List<int>> cellSpans;

  /// occupiedGrid[row][day] = 该格是否已被上方的跨行格占用。
  final List<List<bool>> occupiedGrid;
}

int _cellCount(GridLayout grid, int row, int day) {
  final courses = grid.cellCourses[row];
  if (day >= courses.length) return 0;
  return courses[day].length;
}

int _cellSpan(GridLayout grid, int row, int day) {
  if (row >= grid.cellSpans.length || day >= grid.cellSpans[row].length) {
    return 1;
  }
  final span = grid.cellSpans[row][day];
  return span == 0 ? 1 : span;
}

bool _cellOccupied(GridLayout grid, int row, int day) {
  if (row >= grid.occupiedGrid.length || day >= grid.occupiedGrid[row].length) {
    return false;
  }
  return grid.occupiedGrid[row][day];
}

/// 动态计算每行的基准与扩展高度：
/// 统筹整网格所有单元格的最小空间需求，当同行存在单双周多门课纵向堆叠时，
/// 该节次行会自动增高；同行单门课自动均分撑满扩展后的行高，消除下半截留白。
///
/// 排序策略：span 升序（小跨度先确定基准），count 降序（同 span 内多门课先撑高行高），
/// 确保行高先被真实内容需求撑高，单门课再感知已拉升的行高。
List<int> computeRowHeights(GridLayout grid, RowMetrics metrics) {
  final rowCount = grid.cellCourses.length;
  if (rowCount == 0) return const [];

  final cells = <(int, int, int)>[]; // (span, rowIndex, dayIndex)
  for (var r = 0; r < rowCount; r++) {
    for (var d = 0; d < 7; d++) {
      if (_cellOccupied(grid, r, d)) continue;
      cells.add((_cellSpan(grid, r, d), r, d));
    }
  }

  // 主排序：span 升序，次排序：count 降序
  cells.sort((a, b) {
    final bySpan = a.$1.compareTo(b.$1);
    if (bySpan != 0) return bySpan;
    return _cellCount(grid, b.$2, b.$3).compareTo(_cellCount(grid, a.$2, a.$3));
  });

  final rowHeights = List<int>.filled(rowCount, metrics.baseH);
  for (final (span, rIndex, dayIndex) in cells) {
    final count = _cellCount(grid, rIndex, dayIndex);
    if (count == 0) continue;
    int reqInner;
    if (count == 1) {
      // 单门课只需填满当前行高，不主动拉伸（行高由多门课决定）
      final spanFill = span * metrics.baseH - metrics.padV;
      final floor = metrics.baseH - metrics.padV;
      reqInner = spanFill > floor ? spanFill : floor;
    } else {
      reqInner = count * metrics.multiCardH + (count - 1) * 4;
    }
    final reqTotal = reqInner + metrics.padV;
    var curTotal = 0;
    for (var i = 0; i < span; i++) {
      curTotal += rIndex + i < rowCount
          ? rowHeights[rIndex + i]
          : metrics.baseH;
    }
    if (reqTotal > curTotal) {
      final perRow = ((reqTotal - curTotal) / span).ceil();
      for (var i = 0; i < span; i++) {
        final idx = rIndex + i;
        if (idx < rowCount) {
          rowHeights[idx] += perRow;
        }
      }
    }
  }

  return rowHeights;
}

/// 跨 span 单元格的内部可用高度（扣除上下 padding）；offset 为锚定行下标。
int cellInnerHeightFor(
  int span,
  List<int> rowHeights,
  RowMetrics metrics, [
  int offset = 0,
]) {
  var totalH = 0;
  for (var i = 0; i < span; i++) {
    totalH += offset + i < rowHeights.length
        ? rowHeights[offset + i]
        : metrics.baseH;
  }
  final v = totalH - metrics.padV;
  final floor = metrics.baseH - metrics.padV;
  return v > floor ? v : floor;
}

/// 单元格内课程卡片的最小高度：单门课撑满，多门叠放均分（扣除卡片间 gap 4px）。
int cardMinHeightFor(
  int span,
  List<int> rowHeights,
  int courseCount,
  RowMetrics metrics, [
  int offset = 0,
]) {
  final innerH = cellInnerHeightFor(span, rowHeights, metrics, offset);
  final count = courseCount < 1 ? 1 : courseCount;
  // 单门课（含跨 2+ 节的舒展模式）：直接返回格子全高
  if (count == 1) {
    final floor = metrics.baseH - metrics.padV;
    return innerH > floor ? innerH : floor;
  }
  final v = (innerH - (count - 1) * 4) ~/ count;
  final floor = metrics.baseH - metrics.padV;
  return v > floor ? v : floor;
}

final RegExp _teacherCodeSuffixRe = RegExp(r'\([^)]*\)$');

/// 教师名（teacherAndCode "张三(T001)" → "张三"）。
String teacherNameOf(PkCourseOnTable course) =>
    (course.teacherAndCode ?? '').replaceAll(_teacherCodeSuffixRe, '').trim();

final RegExp _teacherSplitRe = RegExp(r'[,，、]');

/// 紧凑展示教师名：超过 maxVisible 位时显示「前 maxVisible-1 位 等」。
String compactTeacherName(String raw, int maxVisible) {
  if (raw.isEmpty) return '';
  final teachers = raw
      .split(_teacherSplitRe)
      .map((s) => s.trim())
      .where((s) => s.isNotEmpty)
      .toList();
  if (teachers.length <= maxVisible) return teachers.join('、');
  return '${teachers.take(maxVisible - 1).join('、')} 等';
}

/// 周次格式精简（如单周 1-15 提炼为 "1-15周(单周)"，避免粗暴罗列）。
/// 本地化标签由调用方注入：[parityLabel] 奇偶标签，[weeksTemplate] 如 "共{range}周"。
String formatDisplayWeeks(
  List<int>? weeks,
  String Function(PkWeekParity parity) parityLabel,
  String Function(String range) weeksTemplate,
) {
  if (weeks == null || weeks.isEmpty) return '';
  final parity = detectWeekParity(weeks);
  final sorted = weeks.toSet().toList()..sort();
  if (parity != null && sorted.length >= 3) {
    var isRegularStep = true;
    for (var i = 1; i < sorted.length; i++) {
      if (sorted[i] != sorted[i - 1] + 2) {
        isRegularStep = false;
        break;
      }
    }
    if (isRegularStep) {
      return '${sorted.first}-${sorted.last}周(${parityLabel(parity)})';
    }
  }
  return weeksTemplate(formatWeeksText(weeks));
}

/// 该行（1-based）是否为某时段分组首行：返回分组键（morning/afternoon/evening），
/// 非首行返回 null。UI 层把键映射为本地化标签。
String? dayPartKeyForRow(int row, List<SectionTime> sectionTimes) {
  final starts = dayPartBoundaries(sectionTimes);
  for (final entry in starts.entries) {
    if (entry.value == row) return entry.key;
  }
  return null;
}
