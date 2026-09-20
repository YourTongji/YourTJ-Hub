import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('campus fixtures match the Dart projection', () {
    Map<String, dynamic> result(String name) {
      final fixture = File('../../../../packages/api-contract/fixtures/$name');
      return (jsonDecode(fixture.readAsStringSync())
              as Map<String, dynamic>)['result']
          as Map<String, dynamic>;
    }

    final rules = CampusCalendarSettings.fromJson(
      result('campus-calendar-rules-success.json'),
    );
    expect(rules.rules.holidays.single.startDate, '2026-10-01');
    expect(rules.rules.moves.single.fromDate, '2026-10-06');
    expect(rules.rules.moves.single.toDate, '2026-09-20');
    expect(rules.toJson(), result('campus-calendar-rules-success.json'));
    final draft = CampusCalendarDraft.fromJson(
      result('campus-calendar-draft-success.json'),
    );
    expect(draft.warnings, isEmpty);
    expect(draft.rules.moves.single.toDate, '2026-09-20');
    final status = CampusStatus.fromJson(result('campus-status-success.json'));
    expect(status.binding!.maskedId, '23••••01');
    expect(status.candidate, isNull);
    final dataset = CampusDataset.fromJson(
      result('campus-dataset-success.json'),
    );
    expect(dataset.key, 'cet');
    expect(dataset.series.single.value, 500);
    expect(dataset.messages, isEmpty);
    final message = CampusMessageDetail.fromJson(
      result('campus-message-success.json'),
    );
    expect(message.id, '123');
    expect(message.content, isNotEmpty);
    expect(message.links.single.url, startsWith('https://'));
    final messages = CampusDataset.fromJson({
      ...result('campus-dataset-success.json'),
      'key': 'messages',
      'messages': [result('campus-message-success.json')],
    });
    expect(messages.messages.single.id, '123');
  });
}
