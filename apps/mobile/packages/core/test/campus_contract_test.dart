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
