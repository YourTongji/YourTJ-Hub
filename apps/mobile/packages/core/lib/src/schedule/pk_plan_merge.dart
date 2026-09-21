import 'dart:convert';
import 'pk_models.dart';
import 'pk_conflict.dart';

class PkPlanMergeConflict {
  const PkPlanMergeConflict(this.path, this.local, this.remote);
  final List<String> path;
  final Object? local;
  final Object? remote;
}

class PkPlanMergeResult {
  const PkPlanMergeResult(this.plan, this.conflicts);
  final PkPlan? plan;
  final List<PkPlanMergeConflict> conflicts;
}

String planValueKey(Object? value) {
  Object? normalize(Object? v) {
    if (v is num && v.isFinite && v == v.truncateToDouble()) return v.toInt();
    if (v is List) return v.map(normalize).toList();
    if (v is Map) {
      final keys = v.keys.cast<String>().where((key) => v[key] != null).toList()
        ..sort();
      return {for (final key in keys) key: normalize(v[key])};
    }
    return v;
  }

  return jsonEncode(normalize(value));
}

Map<String, dynamic> _project(PkPlan plan) => {
  'id': plan.id,
  'name': plan.name,
  'createdAt': plan.createdAt,
  'courses': {
    for (final course in plan.stagedCourses)
      course.courseCode: {
        'course': course.toJson(),
        'selected':
            plan.selectedCourses
                .where((code) => isClassOfCourse(code, course.courseCode))
                .toList()
              ..sort(),
      },
  },
  'otherSelected':
      plan.selectedCourses
          .where(
            (code) => !plan.stagedCourses.any(
              (course) => isClassOfCourse(code, course.courseCode),
            ),
          )
          .toList()
        ..sort(),
  'events': {for (final event in plan.customEvents) event.id: event.toJson()},
};
PkPlan _restore(Map value) {
  final courses = (value['courses'] as Map).values.cast<Map>().toList();
  return PkPlan.fromJson({
    'id': value['id'],
    'name': value['name'],
    'createdAt': value['createdAt'],
    'stagedCourses': courses.map((course) => course['course']).toList(),
    'selectedCourses': [
      ...courses.expand((course) => course['selected'] as List),
      ...value['otherSelected'] as List,
    ],
    'customEvents': (value['events'] as Map).values.toList(),
  });
}

String schedulePlanKey(PkPlan? plan) =>
    planValueKey(plan == null ? null : _project(plan));
PkPlanMergeResult mergeSchedulePlan(
  PkPlan? base,
  PkPlan? local,
  PkPlan? remote, [
  Map<String, String> choices = const {},
]) {
  final conflicts = <PkPlanMergeConflict>[];
  bool equal(Object? a, Object? b) => planValueKey(a) == planValueKey(b);
  Object? merge(Object? b, Object? l, Object? r, List<String> path) {
    if (equal(l, r)) return l;
    if (equal(l, b)) return r;
    if (equal(r, b)) return l;
    if (b is Map &&
        l is Map &&
        r is Map &&
        !(path.length == 2 && path.first == 'courses')) {
      final result = <String, dynamic>{};
      for (final key in {...b.keys, ...l.keys, ...r.keys}.cast<String>()) {
        final value = merge(b[key], l[key], r[key], [...path, key]);
        if (value != null) result[key] = value;
      }
      return result;
    }
    final choice = choices[jsonEncode(path)];
    if (choice != null) return choice == 'local' ? l : r;
    conflicts.add(PkPlanMergeConflict(path, l, r));
    return l;
  }

  final result = merge(
    base == null ? null : _project(base),
    local == null ? null : _project(local),
    remote == null ? null : _project(remote),
    [],
  );
  return PkPlanMergeResult(result is Map ? _restore(result) : null, conflicts);
}
