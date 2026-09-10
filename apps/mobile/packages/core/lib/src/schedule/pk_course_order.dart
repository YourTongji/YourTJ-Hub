/// 排课列表排序：专业计划内课程置顶（移植自 web `pkCourseOrder.ts`）。
library;

/// 将专业计划内课程置顶，同时保留两组内的接口/备选池顺序（稳定分区）。
List<T> sortPlannedCoursesFirst<T>(
  List<T> courses,
  List<T> compulsoryCourses,
  String Function(T) courseCodeOf,
) {
  final plannedCodes = compulsoryCourses.map(courseCodeOf).toSet();
  final planned = <T>[];
  final other = <T>[];
  for (final course in courses) {
    (plannedCodes.contains(courseCodeOf(course)) ? planned : other).add(course);
  }
  return [...planned, ...other];
}
