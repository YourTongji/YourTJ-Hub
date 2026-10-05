import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;

  @override
  Future<void> write(String token) async {}

  @override
  Future<void> clear() async {}
}

void main() {
  test('mentionTargets uses the dedicated endpoint and parses actor type', () async {
    final dio = Dio(BaseOptions(baseUrl: 'http://test'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          expect(options.method, 'GET');
          expect(options.path, '/api/forum/mention-targets');
          expect(options.queryParameters, {'q': 'assistant', 'limit': 20});
          handler.resolve(
            Response<dynamic>(
              requestOptions: options,
              statusCode: 200,
              data: {
                'code': 0,
                'result': [
                  {
                    'userId': 34,
                    'username': 'assistant',
                    'nickname': 'Helper',
                    'avatarUrl': '/avatars/34.png',
                    'actorType': 'bot',
                  },
                ],
              },
            ),
          );
        },
      ),
    );
    final repository = TopicRepository(
      GfApiClient(dio: dio, tokenStorage: _Tokens(), baseUrl: 'http://test'),
    );

    final targets = await repository.mentionTargets(query: 'assistant');

    expect(targets.single.userId, 34);
    expect(targets.single.actorType, 'bot');
  });
}
