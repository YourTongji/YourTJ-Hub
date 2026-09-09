/// 排课器冲突检测：课号归一化、12×7 占用表操作、冲突判定。
///
/// 一比一移植自 web `resource/src/site/utils/pkConflict.ts`（2026-09-06 dev）。
/// 占用格为 12×7 三维数组；冲突判据 = 同一天 + 同一节次 + 周次交集非空。
/// 全部纯函数：occupied 一律返回新结构，不就地修改入参。
library;

import 'pk_arrange.dart';
import 'pk_models.dart';

/// 占用格行数（12 节）。
const int kOccupyRows = 12;

/// 占用格列数（7 天）。
const int kOccupyCols = 7;

/// 空 12×7 占用表。
List<List<List<PkOccupyCell>>> createEmptyOccupied() => List.generate(
  kOccupyRows,
  (_) => List.generate(kOccupyCols, (_) => <PkOccupyCell>[]),
);

/// 从班级课号取基础课号，兼容 "12200401" 和 "122004.01" 两种格式。
/// 上游约定：后两位是班号；带点号时取点前部分。
String getCourseBaseCode(String? code) {
  final value = (code ?? '').trim();
  final dot = value.lastIndexOf('.');
  if (dot > 0) return value.substring(0, dot);
  return value.length > 2 ? value.substring(0, value.length - 2) : value;
}

/// 判断班级课号是否属于某门课。
bool isClassOfCourse(String classCode, String? courseCode) =>
    getCourseBaseCode(classCode) == (courseCode ?? '').trim();

/// 判断两个班级课号是否属于同一门课。
bool isSameCourse(String code1, String code2) =>
    getCourseBaseCode(code1) == getCourseBaseCode(code2);

/// 从占用表删除一门课（返回新表）。
List<List<List<PkOccupyCell>>> deleteOccupied(
  List<List<List<PkOccupyCell>>> occupied,
  String code,
) => occupied
    .map(
      (row) => row
          .map(
            (cell) =>
                cell.where((item) => !isSameCourse(item.code, code)).toList(),
          )
          .toList(),
    )
    .toList();

/// 向占用表插入一门课（返回新表）。
List<List<List<PkOccupyCell>>> insertOccupied(
  List<List<List<PkOccupyCell>>> occupied,
  List<PkArrangement> arrangementInfo,
  String code,
  String courseName,
) {
  final next = occupied
      .map((row) => row.map((cell) => List<PkOccupyCell>.of(cell)).toList())
      .toList();
  for (final arr in arrangementInfo) {
    for (final time in arr.occupyTime) {
      if (time < 1 ||
          time > kOccupyRows ||
          arr.occupyDay < 1 ||
          arr.occupyDay > kOccupyCols) {
        continue;
      }
      next[time - 1][arr.occupyDay - 1].add(
        PkOccupyCell(
          code: code,
          courseName: courseName,
          occupyWeek: arr.occupyWeek,
        ),
      );
    }
  }
  return next;
}

/// [canAddCourse] 的结果。
class CanAddResult {
  const CanAddResult({required this.canAdd, this.collideCourse});

  final bool canAdd;

  /// 冲突课程描述（"课号 课程名"）。
  final String? collideCourse;
}

/// 判断一门课能否加入课表。
/// 若占用表中已存在同一基础课号的课程，先移除旧课再判定（同课换班 = 隐式替换，不报冲突）。
CanAddResult canAddCourse(
  List<PkArrangement> arrangementInfo,
  List<List<List<PkOccupyCell>>> occupied,
  String code,
) {
  String? existingCode;
  for (final row in occupied) {
    for (final cell in row) {
      for (final item in cell) {
        if (isSameCourse(item.code, code)) {
          existingCode = item.code;
          break;
        }
      }
      if (existingCode != null) break;
    }
    if (existingCode != null) break;
  }

  if (existingCode != null) {
    return canAddCourse(
      arrangementInfo,
      deleteOccupied(occupied, existingCode),
      code,
    );
  }

  for (final arr in arrangementInfo) {
    for (final time in arr.occupyTime) {
      final dayIdx = arr.occupyDay - 1;
      if (time - 1 < 0 || time - 1 >= occupied.length) continue;
      if (dayIdx < 0 || dayIdx >= kOccupyCols) continue;
      final cell = occupied[time - 1][dayIdx];
      for (final item in cell) {
        if (weeksOverlap(arr.occupyWeek, item.occupyWeek)) {
          return CanAddResult(
            canAdd: false,
            collideCourse: '${item.code} ${item.courseName}',
          );
        }
      }
    }
  }
  return const CanAddResult(canAdd: true);
}

/// 一条冲突记录。
class PkConflictItem {
  const PkConflictItem({required this.code, required this.courseName});

  /// 班级课号。
  final String code;
  final String courseName;
}

/// 自定义占位事件在占用表中的伪课号前缀（避免与真实课号碰撞）。
const String kCustomEventCodePrefix = 'custom:';

/// 冲突派生用的基础标识：custom 伪课号原样保留（getCourseBaseCode 会误裁尾部字符）。
String conflictBaseOf(String code) =>
    code.startsWith(kCustomEventCodePrefix) ? code : getCourseBaseCode(code);

/// 找出候选课程与占用表的所有冲突课程（按基础课号去重）。
/// 与 [canAddCourse] 不同：这里列举全部冲突项而非只报第一个。
List<PkConflictItem> findConflicts(
  PkCourseDetail candidate,
  List<List<List<PkOccupyCell>>> occupied,
) {
  final conflicts = <String, PkConflictItem>{};
  for (final arr in candidate.arrangementInfo) {
    for (final time in arr.occupyTime) {
      final dayIdx = arr.occupyDay - 1;
      if (time - 1 < 0 || time - 1 >= occupied.length) continue;
      if (dayIdx < 0 || dayIdx >= kOccupyCols) continue;
      final cell = occupied[time - 1][dayIdx];
      for (final item in cell) {
        if (weeksOverlap(arr.occupyWeek, item.occupyWeek)) {
          final base = conflictBaseOf(item.code);
          conflicts.putIfAbsent(
            base,
            () => PkConflictItem(code: item.code, courseName: item.courseName),
          );
        }
      }
    }
  }
  return conflicts.values.toList();
}

/// 找出候选教学班与当前占用表的冲突课程列表（排除同门课程自身）：
/// 用于在选择教学班前进行前置提示（不阻塞选择）。
List<PkConflictItem> findClassConflicts(
  PkCourseDetail candidate,
  List<List<List<PkOccupyCell>>> occupied,
) {
  final String candidateBase = conflictBaseOf(candidate.code);
  return findConflicts(candidate, occupied)
      .where((c) => conflictBaseOf(c.code) != candidateBase)
      .toList();
}

/// 判断某个排课时间段是否与占用表中的已有课程冲突（排除同门课程自身）。
bool isArrangementConflicted(
  PkArrangement arr,
  String candidateCode,
  List<List<List<PkOccupyCell>>> occupied,
) {
  final String candidateBase = conflictBaseOf(candidateCode);
  for (final int time in arr.occupyTime) {
    if (time - 1 < 0 || time - 1 >= occupied.length) continue;
    final int dayIdx = arr.occupyDay - 1;
    if (dayIdx < 0 || dayIdx >= kOccupyCols) continue;
    final List<PkOccupyCell> cell = occupied[time - 1][dayIdx];
    for (final PkOccupyCell item in cell) {
      if (conflictBaseOf(item.code) != candidateBase &&
          weeksOverlap(arr.occupyWeek, item.occupyWeek)) {
        return true;
      }
    }
  }
  return false;
}

/// 从占用表派生当前课表的全部冲突（容忍式冲突模型）：
/// 同一格子（天+节次）内周次有交集的两个不同基础课号互为冲突。
/// 返回 Map<基础课号, 冲突项[]>（含 custom: 占位事件；供课表 ⚠、列表红标、
/// 统计计数共用同一判据，与 canAddCourse/findConflicts 一致）。
Map<String, List<PkConflictItem>> deriveConflicts(
  List<List<List<PkOccupyCell>>> occupied,
) {
  final result = <String, List<PkConflictItem>>{};
  void pushConflict(String base, PkConflictItem item) {
    final list = result[base];
    if (list != null) {
      if (!list.any((existing) => existing.code == item.code)) list.add(item);
    } else {
      result[base] = [item];
    }
  }

  for (final row in occupied) {
    for (final cell in row) {
      if (cell.length < 2) continue;
      for (var i = 0; i < cell.length; i++) {
        for (var j = i + 1; j < cell.length; j++) {
          final a = cell[i];
          final b = cell[j];
          if (!weeksOverlap(a.occupyWeek, b.occupyWeek)) continue;
          final baseA = conflictBaseOf(a.code);
          final baseB = conflictBaseOf(b.code);
          if (baseA == baseB) continue; // 同课换班/同事件不标冲突
          pushConflict(
            baseA,
            PkConflictItem(code: b.code, courseName: b.courseName),
          );
          pushConflict(
            baseB,
            PkConflictItem(code: a.code, courseName: a.courseName),
          );
        }
      }
    }
  }
  return result;
}
