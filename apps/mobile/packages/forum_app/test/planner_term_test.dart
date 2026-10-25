import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/schedule/planner_term.dart';

void main() {
  test('catalog and school term names match without guessing short terms', () {
    expect(canonicalPlannerTerm('2026-2027学年第1学期'), '2026-2027-1');
    expect(canonicalPlannerTerm('2025-2026 第二学期'), '2025-2026-2');
    expect(canonicalPlannerTerm('2025-2026-1'), '2025-2026-1');
    expect(canonicalPlannerTerm('2025-2026学年短学期'), isNull);
    expect(canonicalPlannerTerm(''), isNull);
  });
}
