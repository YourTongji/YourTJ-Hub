import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'api_client_test.dart' show MockAdapter, ResponseData;

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'session';
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

void main() {
  test(
    'revision history follows the controlled fixture and version cursor',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.path, '/api/forum/posts/revisions');
        expect(request.queryParameters, {
          'postId': 9252,
          'beforeVersion': 8,
          'limit': 20,
        });
        final fixture = jsonDecode(
          File(
            '../../../../packages/api-contract/fixtures/post-revisions-success.json',
          ).readAsStringSync(),
        );
        return ResponseData(200, fixture);
      });
      final page = await PostRepository(
        GfApiClient(dio: dio, tokenStorage: _Tokens()),
      ).revisions(postId: 9252, beforeVersion: 8);
      expect(page.postId, 9252);
      expect(page.versions.single.content, 'Original reply content');
      expect(page.versions.single.editor.id, 9302);
      expect(page.hasMore, isFalse);
      dio.close();
    },
  );
  test(
    'native management submits the same ID and action semantics as Web',
    () async {
      final requests = <String, Object?>{};
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.headers['Authorization'], 'Bearer session');
        requests[request.path] = request.data;
        return ResponseData(200, {'code': 0, 'result': true});
      });
      final client = GfApiClient(dio: dio, tokenStorage: _Tokens());
      await PostRepository(client).moderate(postId: 42, ban: true);
      await TopicRepository(client).moderate(topicId: 21, ban: false);
      await TopicRepository(client).deleteTopic(topicId: 21);
      expect(requests, {
        '/api/forum/moderation/post-status': {'postId': 42, 'action': 'ban'},
        '/api/forum/moderation/topic-status': {
          'topicId': 21,
          'action': 'unban',
        },
        '/api/forum/topics/delete': {'topicId': 21},
      });
      dio.close();
    },
  );
}
