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

class _LiveStreamAdapter implements HttpClientAdapter {
  final body = StreamController<Uint8List>();

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async => ResponseBody(
    body.stream,
    200,
    headers: {
      Headers.contentTypeHeader: ['text/event-stream'],
    },
  );

  @override
  void close({bool force = false}) {}
}

void main() {
  test(
    'closing a live SSE while cancelling its decoder completes cleanly',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'http://test'));
      final adapter = _LiveStreamAdapter();
      dio.httpClientAdapter = adapter;
      final client = GfApiClient(
        dio: dio,
        tokenStorage: _Tokens(),
        baseUrl: 'http://test',
      );
      final connection = await ForumRealtimeTransport(
        client,
      ).connect(token: 'session', cancelToken: CancelToken());
      final subscription = connection.events.listen((_) {});
      // Let the async decoder enter its await-for with a live, idle response.
      await Future<void>.delayed(Duration.zero);
      final cancelled = subscription.cancel();
      final cleanShutdown = expectLater(cancelled, completes);
      connection.close();
      await cleanShutdown;
      await adapter.body.close();
      dio.close(force: true);
    },
  );

  test('live SSE still propagates non-cancellation transport errors', () async {
    final dio = Dio(BaseOptions(baseUrl: 'http://test'));
    final adapter = _LiveStreamAdapter();
    dio.httpClientAdapter = adapter;
    final client = GfApiClient(
      dio: dio,
      tokenStorage: _Tokens(),
      baseUrl: 'http://test',
    );
    final connection = await ForumRealtimeTransport(
      client,
    ).connect(token: 'session', cancelToken: CancelToken());
    final failure = DioException.connectionError(
      requestOptions: RequestOptions(path: '/api/forum/events'),
      reason: 'Connection lost',
    );
    final observed = expectLater(connection.events, emitsError(same(failure)));
    adapter.body.addError(failure);
    await observed;
    connection.close();
    await adapter.body.close();
    dio.close(force: true);
  });

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
