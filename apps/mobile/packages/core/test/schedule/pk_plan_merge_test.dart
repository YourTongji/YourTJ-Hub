import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final fixtures =
      jsonDecode(
            File(
              '../../../../packages/api-contract/fixtures/pk-plan-merge-cases.json',
            ).readAsStringSync(),
          )
          as List;
  PkPlan? plan(Object? value) => value == null
      ? null
      : PkPlan.fromJson(Map<String, dynamic>.from(value as Map));
  for (final fixture in fixtures) {
    test('shared merge fixture: ${fixture['name']}', () {
      final result = mergeSchedulePlan(
        plan(fixture['base']),
        plan(fixture['local']),
        plan(fixture['remote']),
        Map<String, String>.from(fixture['choices'] as Map),
      );
      expect(
        schedulePlanKey(result.plan),
        schedulePlanKey(plan(fixture['expected'])),
      );
      expect(
        result.conflicts.map((c) => c.path).toList(),
        fixture['conflicts'],
      );
    });
  }
}
