import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'topic media metadata parses and remains optional for cached payloads',
    () {
      final fixture =
          jsonDecode(
                File(
                  '../../../../packages/api-contract/fixtures/agent-search-success.json',
                ).readAsStringSync(),
              )
              as Map<String, dynamic>;
      final response = fixture['result'] as Map<String, dynamic>;
      final topics = response['topics'] as List<dynamic>;
      final topicJson = Map<String, dynamic>.from(
        topics.single as Map<dynamic, dynamic>,
      );
      final topic = TopicPayload.fromJson(topicJson);

      expect(topic.imageMetadata, hasLength(1));
      expect(topic.imageMetadata!.single.width, 1920);
      expect(topic.imageMetadata!.single.variants, hasLength(2));
      expect(topic.imageMetadata!.single.variants.last.width, 1280);
      expect(
        TopicPayload.fromJson(
          jsonDecode(jsonEncode(topic)) as Map<String, dynamic>,
        ).imageMetadata,
        topic.imageMetadata,
      );

      topicJson.remove('imageMetadata');
      expect(TopicPayload.fromJson(topicJson).imageMetadata, isNull);
    },
  );
}
