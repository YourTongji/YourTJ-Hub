import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'api_client_test.dart' show MockAdapter, ResponseData;
import 'push_repository_test.dart' show Tokens;

void main() {
  test(
    'new topics include reply policy, edits omit it, and explicit updates send false',
    () async {
      final requests = <Map<String, dynamic>>[];
      final paths = <String>[];
      final dio = Dio();
      addTearDown(dio.close);
      dio.httpClientAdapter = MockAdapter((request) async {
        paths.add(request.path);
        requests.add(Map<String, dynamic>.from(request.data as Map));
        return ResponseData(200, {
          'code': 0,
          'result': request.path.endsWith('/write') ? 42 : true,
        });
      });
      final repo = TopicRepository(
        GfApiClient(dio: dio, tokenStorage: Tokens()),
      );
      for (final topicId in [0, 42]) {
        await repo.writeTopicResult(
          topicId: topicId,
          title: 'Title',
          content: 'Body',
          categoryIds: [1],
          topicStatus: 1,
          agentRepliesDisabled: true,
        );
      }
      await repo.updateAgentReplies(topicId: 42, disabled: false);
      expect(requests[0]['agentRepliesDisabled'], isTrue);
      expect(requests[1].containsKey('agentRepliesDisabled'), isFalse);
      expect(paths.last, '/api/forum/topics/agent-replies');
      expect(requests.last, {'topicId': 42, 'agentRepliesDisabled': false});
    },
  );
}
