import 'dart:async';
import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'current-token';
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _StreamAdapter implements HttpClientAdapter {
  _StreamAdapter(this.status, this.body, {this.headers = const {}});

  final int status;
  final String body;
  final Map<String, List<String>> headers;
  RequestOptions? request;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    request = options;
    return ResponseBody.fromString(
      body,
      status,
      headers: {
        Headers.contentTypeHeader: ['text/event-stream'],
        ...headers,
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  test('transport pins the session token and streams events', () async {
    final dio = Dio(BaseOptions(baseUrl: 'http://test'));
    final adapter = _StreamAdapter(
      200,
      'event: hello\ndata: {}\n\n',
      headers: {
        'new-token': ['renewed'],
      },
    );
    dio.httpClientAdapter = adapter;
    final renewed = <String>[];
    final client = GfApiClient(
      dio: dio,
      tokenStorage: _Tokens(),
      baseUrl: 'http://test',
      onTokenRenewed: renewed.add,
    );
    var activities = 0;
    final connection = await ForumRealtimeTransport(client).connect(
      token: 'captured-token',
      cancelToken: CancelToken(),
      onActivity: () => activities++,
    );
    final frames = await connection.events.toList();
    expect(adapter.request!.path, '/api/forum/events');
    expect(adapter.request!.headers['Authorization'], 'Bearer captured-token');
    expect(adapter.request!.headers['Accept'], 'text/event-stream');
    expect(frames.single.event, 'hello');
    expect(activities, greaterThan(0));
    expect(renewed, ['renewed']);
    connection.close();
  });

  test(
    '401 invokes session invalidation; 404 reports unsupported stream',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'http://test'));
      var unauthorized = 0;
      final client = GfApiClient(
        dio: dio,
        tokenStorage: _Tokens(),
        baseUrl: 'http://test',
        onUnauthorized: () => unauthorized++,
      );
      dio.httpClientAdapter = _StreamAdapter(401, '');
      await expectLater(
        ForumRealtimeTransport(
          client,
        ).connect(token: 'captured', cancelToken: CancelToken()),
        throwsA(
          isA<ForumRealtimeHttpException>().having(
            (e) => e.statusCode,
            'statusCode',
            401,
          ),
        ),
      );
      expect(unauthorized, 1);
      dio.httpClientAdapter = _StreamAdapter(404, '');
      await expectLater(
        ForumRealtimeTransport(
          client,
        ).connect(token: 'captured', cancelToken: CancelToken()),
        throwsA(
          isA<ForumRealtimeHttpException>().having(
            (e) => e.statusCode,
            'statusCode',
            404,
          ),
        ),
      );
    },
  );
}
