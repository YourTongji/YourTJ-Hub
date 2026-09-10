import 'dart:convert';
import 'dart:io';

import 'package:core/src/gen/pk.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('materialization request preserves the selected audience', () {
    expect(const PkMaterializeRequest(term: '121', audience: 'graduate').toJson(), {'term': '121', 'audience': 'graduate'});
    expect(const PkMaterializeRequest(term: '121').toJson()['audience'], 'undergraduate');
  });
  test('materialization result matches the shared admin fixture', () {
    final file = File(
      '../../../../packages/api-contract/fixtures/pk-materialize-success.json',
    );
    final envelope =
        jsonDecode(file.readAsStringSync()) as Map<String, dynamic>;
    final result = PkMaterializeResult.fromJson(
      envelope['result'] as Map<String, dynamic>,
    );
    expect(result.calendarId, 121);
    expect(result.coursesInserted, 1);
    expect(result.offeringsInserted, 1);
    expect(result.aliasesInserted, 2);
    expect(result.coursesUpdated, 0);
    expect(result.offeringsUpdated, 0);
    expect(result.instructorsInserted, 1);
    expect(result.aliasesSkipped, 0);
  });
}
