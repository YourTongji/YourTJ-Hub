import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('private notes use the shared authenticated contract fixture', () {
    final fixture =
        jsonDecode(
              File(
                '../../../../packages/api-contract/fixtures/private-notes-success.json',
              ).readAsStringSync(),
            )
            as Map<String, dynamic>;
    final result = PrivateNotesPayload.fromJson(
      fixture['result'] as Map<String, dynamic>,
    );
    expect(result.ownerId, 1);
    expect(result.notes.single.targetUserId, 2);
    expect(result.notes.single.note, '实验搭档');
  });
}
