import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('profile selection distinguishes explicit empty from an older server', () {
    final fixture =
        jsonDecode(
              File(
                '../../../../packages/api-contract/fixtures/user-card-success.json',
              ).readAsStringSync(),
            )
            as Map<String, dynamic>;
    final card = Map<String, dynamic>.from(fixture['result'] as Map);
    expect(UserCardPayload.fromJson(card).displayBadges, isEmpty);
    card.remove('displayBadges');
    expect(UserCardPayload.fromJson(card).displayBadges, isNull);
  });
}
