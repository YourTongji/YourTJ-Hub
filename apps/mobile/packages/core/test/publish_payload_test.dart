import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('new topic payload normalizes Go nil categoryIds to an empty list', () {
    final PublishTopicPayload payload = PublishTopicPayload.fromJson(
      <String, dynamic>{
        'title': '',
        'content': '',
        'categoryIds': null,
        'topicStatus': 0,
      },
    );

    expect(payload.categoryIds, isEmpty);
  });

  test('edit topic payload preserves selected category ids', () {
    final PublishTopicPayload payload = PublishTopicPayload.fromJson(
      <String, dynamic>{
        'title': 'Title',
        'content': 'Body',
        'categoryIds': <int>[1, 3],
        'topicStatus': 1,
      },
    );

    expect(payload.categoryIds, <int>[1, 3]);
  });
}
