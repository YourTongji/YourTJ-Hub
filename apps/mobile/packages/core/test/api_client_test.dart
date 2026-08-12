import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

/// 内存版 TokenStorage 测试替身。
class _MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

void main() {
  group('GfApiClient', () {
    late _MemoryTokenStorage storage;
    late Dio dio;
    late GfApiClient client;
    final renewedTokens = <String>[];
    var unauthorizedCalls = 0;

    void setupClient({String? initialToken}) {
      storage = _MemoryTokenStorage();
      if (initialToken != null) storage.write(initialToken);
      dio = Dio(BaseOptions(baseUrl: 'http://test'));
      renewedTokens.clear();
      unauthorizedCalls = 0;
      client = GfApiClient(
        dio: dio,
        tokenStorage: storage,
        onTokenRenewed: renewedTokens.add,
        onUnauthorized: () => unauthorizedCalls++,
      );
    }

    test('Bearer 令牌注入', () async {
      setupClient(initialToken: 'tok-1');
      final adapter = MockAdapter((request) async {
        expect(request.headers['Authorization'], 'Bearer tok-1');
        return ResponseData(200, {'code': 0, 'result': true});
      });
      dio.httpClientAdapter = adapter;

      final result = await client.post<bool>(
        '/api/forum/topics/status',
        body: {'topicId': 1, 'topicStatus': 1},
      );
      expect(result, isTrue);
      expect(adapter.requests, hasLength(1));
      expect(adapter.requests.single.path, '/api/forum/topics/status');
    });

    test('无令牌时不带 Authorization 头', () async {
      setupClient();
      final adapter = MockAdapter((request) async {
        expect(request.headers.containsKey('Authorization'), isFalse);
        return ResponseData(200, {'code': 0, 'result': true});
      });
      dio.httpClientAdapter = adapter;

      await client.post<bool>('/api/ping');
      expect(adapter.requests, hasLength(1));
    });

    test('New-Token 响应头触发 onTokenRenewed 回调', () async {
      setupClient(initialToken: 'old-token');
      final adapter = MockAdapter((request) async {
        return ResponseData(
          200,
          {'code': 0, 'result': true},
          headers: {'New-Token': 'fresh-token'},
        );
      });
      dio.httpClientAdapter = adapter;

      await client.get<bool>('/api/ping', parser: (json) => json as bool);
      expect(renewedTokens, ['fresh-token']);
    });

    test('请求会等待异步 New-Token 持久化完成', () async {
      storage = _MemoryTokenStorage();
      dio = Dio(BaseOptions(baseUrl: 'http://test'));
      final writeStarted = Completer<void>();
      final allowWrite = Completer<void>();
      client = GfApiClient(
        dio: dio,
        tokenStorage: storage,
        onTokenRenewed: (token) async {
          writeStarted.complete();
          await allowWrite.future;
          await storage.write(token);
        },
      );
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(
          200,
          {'code': 0, 'result': true},
          headers: {'New-Token': 'persisted-token'},
        );
      });

      var requestCompleted = false;
      final request = client
          .get<bool>('/api/ping', parser: (json) => json as bool)
          .whenComplete(() => requestCompleted = true);
      await writeStarted.future;

      expect(requestCompleted, isFalse);
      expect(await storage.read(), isNull);

      allowWrite.complete();
      expect(await request, isTrue);
      expect(await storage.read(), 'persisted-token');
    });

    test('401 触发 onUnauthorized 回调并抛 UnauthorizedException', () async {
      setupClient(initialToken: 'expired');
      final adapter = MockAdapter((request) async {
        return ResponseData(401, {'code': 1, 'messageCode': 'auth.required'});
      });
      dio.httpClientAdapter = adapter;

      await expectLater(
        client.get<bool>('/api/forum/notifications'),
        throwsA(isA<UnauthorizedException>()),
      );
      expect(unauthorizedCalls, 1);
    });

    test('429 读 Retry-After 抛 RateLimitException', () async {
      setupClient(initialToken: 'tok');
      final adapter = MockAdapter((request) async {
        return ResponseData(
          429,
          {
            'code': 1,
            'messageCode': 'rate.limited',
            'params': {'retryAfterSeconds': 30},
          },
          headers: {'Retry-After': '30'},
        );
      });
      dio.httpClientAdapter = adapter;

      await expectLater(
        client.get<bool>('/api/ping'),
        throwsA(
          isA<RateLimitException>()
              .having((e) => e.retryAfterSeconds, 'retryAfterSeconds', 30)
              .having((e) => e.messageCode, 'messageCode', 'rate.limited'),
        ),
      );
    });

    test('429 无 Retry-After 头时 retryAfterSeconds 为 null', () async {
      setupClient();
      final adapter = MockAdapter((request) async {
        return ResponseData(429, {'code': 1, 'messageCode': 'rate.limited'});
      });
      dio.httpClientAdapter = adapter;

      await expectLater(
        client.get<bool>('/api/ping'),
        throwsA(
          isA<RateLimitException>().having(
            (e) => e.retryAfterSeconds,
            'retryAfterSeconds',
            isNull,
          ),
        ),
      );
    });

    test('code != 0 业务失败抛 ApiException 并保留 messageCode', () async {
      setupClient();
      final adapter = MockAdapter((request) async {
        return ResponseData(200, {
          'code': 1,
          'messageCode': 'auth.login.invalidCredentials',
          'params': {'username': 'alice'},
        });
      });
      dio.httpClientAdapter = adapter;

      await expectLater(
        client.post<bool>('/api/login', body: {}),
        throwsA(
          isA<ApiException>()
              .having(
                (e) => e.messageCode,
                'messageCode',
                'auth.login.invalidCredentials',
              )
              .having(
                (e) => e.messageKey,
                'messageKey',
                'server.auth.login.invalidCredentials',
              )
              .having((e) => e.params?['username'], 'params', 'alice'),
        ),
      );
    });

    test('网络错误(DioException)映射为 NetworkException', () async {
      setupClient();
      dio.httpClientAdapter = MockAdapter((request) async {
        throw DioException(
          requestOptions: request,
          type: DioExceptionType.connectionError,
          error: 'connection refused',
        );
      });

      await expectLater(
        client.get<bool>('/api/ping'),
        throwsA(isA<NetworkException>()),
      );
    });

    test('DioException 无 status 时 NetworkException 不携带 statusCode', () async {
      setupClient();
      dio.httpClientAdapter = MockAdapter((request) async {
        throw DioException(
          requestOptions: request,
          type: DioExceptionType.connectionError,
        );
      });

      await expectLater(
        client.get<bool>('/api/ping'),
        throwsA(
          isA<NetworkException>().having(
            (e) => e.statusCode,
            'statusCode',
            isNull,
          ),
        ),
      );
    });

    test(
      '400 envelope 抛 ApiFailureException 并保留 messageCode/params/status',
      () async {
        setupClient();
        final adapter = MockAdapter((request) async {
          return ResponseData(400, {
            'code': 1,
            'messageCode': 'common.request.invalidFormat',
            'params': {'field': 'code'},
          });
        });
        dio.httpClientAdapter = adapter;

        await expectLater(
          client.post<bool>('/api/auth/oidc/exchange', body: {}),
          throwsA(
            isA<ApiFailureException>()
                .having((e) => e.statusCode, 'statusCode', 400)
                .having(
                  (e) => e.messageCode,
                  'messageCode',
                  'common.request.invalidFormat',
                )
                .having(
                  (e) => e.messageKey,
                  'messageKey',
                  'server.common.request.invalidFormat',
                )
                .having((e) => e.params?['field'], 'params', 'code'),
          ),
        );
      },
    );

    test('403 冻结 envelope 抛 ApiFailureException 并保留 messageCode', () async {
      setupClient();
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(403, {
          'code': 1,
          'messageCode': 'oauth.account.frozen',
          'params': {'action': 'login'},
        });
      });

      await expectLater(
        client.post<bool>('/api/auth/oidc/exchange', body: {}),
        throwsA(
          isA<ApiFailureException>()
              .having((e) => e.statusCode, 'statusCode', 403)
              .having(
                (e) => e.messageCode,
                'messageCode',
                'oauth.account.frozen',
              )
              .having((e) => e.params?['action'], 'params', 'login'),
        ),
      );
    });

    test('404 envelope 抛 ApiFailureException 并保留 messageCode', () async {
      setupClient();
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(404, {'code': 1, 'messageCode': 'route.notFound'});
      });

      await expectLater(
        client.get<bool>('/api/forum/does-not-exist'),
        throwsA(
          isA<ApiFailureException>()
              .having((e) => e.statusCode, 'statusCode', 404)
              .having((e) => e.messageCode, 'messageCode', 'route.notFound'),
        ),
      );
    });

    test('500 envelope 抛 ApiFailureException 并保留 messageCode', () async {
      setupClient();
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(500, {
          'code': 1,
          'messageCode': 'oauth.token.failed',
        });
      });

      await expectLater(
        client.post<bool>('/api/auth/oidc/exchange', body: {}),
        throwsA(
          isA<ApiFailureException>()
              .having((e) => e.statusCode, 'statusCode', 500)
              .having(
                (e) => e.messageCode,
                'messageCode',
                'oauth.token.failed',
              ),
        ),
      );
    });

    test(
      '4xx 无 envelope body 时降级为无 messageCode 的 ApiFailureException',
      () async {
        setupClient();
        dio.httpClientAdapter = MockAdapter((request) async {
          return ResponseData(403, {'error': 'forbidden'});
        });

        await expectLater(
          client.get<bool>('/api/ping'),
          throwsA(
            isA<ApiFailureException>()
                .having((e) => e.statusCode, 'statusCode', 403)
                .having((e) => e.messageCode, 'messageCode', isNull)
                .having(
                  (e) => e.fallbackMessage,
                  'fallbackMessage',
                  'Request failed with status 403',
                ),
          ),
        );
      },
    );

    test('5xx 空 body 时降级为无 messageCode 的 ApiFailureException', () async {
      setupClient();
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(502, <String, dynamic>{});
      });

      await expectLater(
        client.get<bool>('/api/ping'),
        throwsA(
          isA<ApiFailureException>()
              .having((e) => e.statusCode, 'statusCode', 502)
              .having((e) => e.messageCode, 'messageCode', isNull),
        ),
      );
    });

    test('成功响应经 parser 解析 result', () async {
      setupClient();
      final adapter = MockAdapter((request) async {
        return ResponseData(200, {
          'code': 0,
          'result': {
            'captchaId': 'abc',
            'captchaImg': 'data:image/png;base64,xxx',
          },
        });
      });
      dio.httpClientAdapter = adapter;

      final captcha = await client.get<CaptchaPayload>(
        '/api/get-captcha',
        parser: (json) => CaptchaPayload.fromJson(json as Map<String, dynamic>),
      );
      expect(captcha.captchaId, 'abc');
      expect(captcha.captchaImg, startsWith('data:image/png'));
    });
  });

  group('repositories 请求体', () {
    test('AuthRepository.login 请求体含加密密码且无蜜罐字段', () async {
      final storage = _MemoryTokenStorage();
      final dio = Dio(BaseOptions(baseUrl: 'http://test'));
      final client = GfApiClient(dio: dio, tokenStorage: storage);
      late Map<String, dynamic> capturedBody;
      dio.httpClientAdapter = MockAdapter((request) async {
        capturedBody = request.data as Map<String, dynamic>;
        return ResponseData(200, {'code': 0, 'result': true});
      });
      final repo = AuthRepository(client);

      await repo.login(
        username: 'alice',
        encryptedPassword: 'ciphertext',
        captchaId: 'cid',
        captchaCode: 'cval',
      );
      expect(capturedBody['username'], 'alice');
      expect(capturedBody['encryptedPassword'], 'ciphertext');
      expect(capturedBody.containsKey('website'), isFalse);
    });

    test('AuthRepository.oidcExchange 使用受控请求与响应镜像', () async {
      final storage = _MemoryTokenStorage();
      final dio = Dio(BaseOptions(baseUrl: 'http://test'));
      final client = GfApiClient(dio: dio, tokenStorage: storage);
      late Map<String, dynamic> capturedBody;
      dio.httpClientAdapter = MockAdapter((request) async {
        capturedBody = request.data as Map<String, dynamic>;
        return ResponseData(200, {
          'code': 0,
          'result': {'token': 'forum-session-token'},
        });
      });

      final token = await AuthRepository(client).oidcExchange(
        code: 'authorization-code',
        codeVerifier: 'pkce-verifier',
        nonce: 'request-nonce',
        redirectUri: 'yourtj://callback',
      );

      expect(capturedBody, {
        'code': 'authorization-code',
        'codeVerifier': 'pkce-verifier',
        'nonce': 'request-nonce',
        'redirectUri': 'yourtj://callback',
      });
      expect(token, 'forum-session-token');
    });

    test('TopicRepository.writeTopic 请求体含蜜罐? 绝不包含', () async {
      final storage = _MemoryTokenStorage();
      final dio = Dio(BaseOptions(baseUrl: 'http://test'));
      final client = GfApiClient(dio: dio, tokenStorage: storage);
      late Map<String, dynamic> capturedBody;
      dio.httpClientAdapter = MockAdapter((request) async {
        capturedBody = request.data as Map<String, dynamic>;
        return ResponseData(200, {'code': 0, 'result': 42});
      });
      final repo = TopicRepository(client);

      final id = await repo.writeTopic(
        topicId: 0,
        title: '标题',
        content: '内容',
        categoryIds: [1],
        topicStatus: 1,
      );
      expect(id, 42);
      expect(capturedBody.containsKey('website'), isFalse);
      expect(capturedBody['categoryId'], [1]);
    });

    test('NotificationRepository.fetchNotifications 游标参数', () async {
      final storage = _MemoryTokenStorage();
      final dio = Dio(BaseOptions(baseUrl: 'http://test'));
      final client = GfApiClient(dio: dio, tokenStorage: storage);
      late Uri capturedUri;
      dio.httpClientAdapter = MockAdapter((request) async {
        capturedUri = request.uri;
        return ResponseData(200, {
          'code': 0,
          'result': {
            'items': <dynamic>[],
            'nextCursor': 0,
            'hasNext': false,
            'unreadCount': 0,
          },
        });
      });
      final repo = NotificationRepository(client);

      final result = await repo.fetchNotifications(
        filter: 'unread',
        cursor: 20,
        limit: 10,
      );
      expect(result.unreadCount, 0);
      expect(capturedUri.path, '/api/forum/notifications');
      expect(capturedUri.queryParameters['filter'], 'unread');
      expect(capturedUri.queryParameters['cursor'], '20');
      expect(capturedUri.queryParameters['limit'], '10');
    });

    test('PageRepository 发送 X-Goose-Page 请求头', () async {
      final storage = _MemoryTokenStorage();
      final dio = Dio(BaseOptions(baseUrl: 'http://test'));
      final client = GfApiClient(dio: dio, tokenStorage: storage);
      late Map<String, dynamic> capturedHeaders;
      dio.httpClientAdapter = MockAdapter((request) async {
        capturedHeaders = request.headers;
        return ResponseData(200, {
          'component': 'home.index',
          'props': {
            'sort': 'hot',
            'tabs': <dynamic>[],
            'topics': <dynamic>[],
            'pagination': {
              'page': 1,
              'nextPage': 1,
              'hasNext': false,
              'nextUrl': '',
            },
            'announcement': {'enabled': false, 'html': ''},
          },
          'meta': {'title': 'yourtj'},
          'layout': {
            'site': {
              'name': 'yourtj',
              'description': '',
              'logo': '',
              'favicon': '',
              'brandType': '',
              'brandText': '',
              'brandImage': '',
            },
            'viewer': {
              'id': 0,
              'username': '',
              'email': '',
              'avatarUrl': '',
              'isAuthenticated': false,
              'canAccessAdmin': false,
              'isModerator': false,
              'requiresEmailVerification': false,
              'adminPermissions': <dynamic>[],
            },
            'sidebar': {'categories': <dynamic>[], 'activeKey': ''},
            'footer': {'links': <dynamic>[], 'primary': <dynamic>[]},
            'unread': {'notifications': false, 'messages': false},
            'theme': {
              'enabled': false,
              'current': 'gf-light',
              'themeColor': '',
            },
          },
          'url': '/',
          'version': '1.0',
        });
      });
      final repo = PageRepository(client);

      final page = await repo.home();
      expect(page.component, 'home.index');
      expect(capturedHeaders[GfApiClient.pageRequestHeader], 'true');
      final props = parsePageProps<HomeProps>(page);
      expect(props, isNotNull);
      expect(props!.sort, 'hot');
    });
  });
}

/// 最小 ResponseData 容器,避免依赖 dio 的 Response 内部构造。
class ResponseData {
  ResponseData(this.statusCode, this.data, {this.headers = const {}});

  final int statusCode;
  final Map<String, dynamic> data;
  final Map<String, String> headers;
}

typedef MockHandler = Future<ResponseData> Function(RequestOptions request);

class MockAdapter implements HttpClientAdapter {
  MockAdapter(this._handler);

  final MockHandler _handler;
  final List<RequestOptions> requests = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    final result = await _handler(options);
    return ResponseBody.fromString(
      jsonEncode(result.data),
      result.statusCode,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
        for (final entry in result.headers.entries) entry.key: [entry.value],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
