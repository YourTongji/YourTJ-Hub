import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'api_client_test.dart' show MockAdapter, ResponseData;

class _Storage implements TokenStorage {
  @override
  Future<String?> read() async => 'test-session';

  @override
  Future<void> write(String token) async {}

  @override
  Future<void> clear() async {}
}

void main() {
  test('reads the trusted-domain whitelist from the public endpoint', () async {
    final Dio dio = Dio();
    dio.httpClientAdapter = MockAdapter((RequestOptions request) async {
      expect(request.method, 'GET');
      expect(request.path, '/api/external-link-policy');
      return ResponseData(200, <String, dynamic>{
        'code': 0,
        'result': <String, dynamic>{
          'trustedDomains': <String>['github.com', 'bilibili.com', ''],
        },
      });
    });

    final ExternalLinkPolicyRepository repository =
        ExternalLinkPolicyRepository(
          GfApiClient(dio: dio, tokenStorage: _Storage()),
        );
    expect(await repository.fetchTrustedDomains(), <String>[
      'github.com',
      'bilibili.com',
    ]);
  });

  test(
    'an older backend without the endpoint keeps the current list',
    () async {
      final Dio dio = Dio();
      dio.httpClientAdapter = MockAdapter(
        (RequestOptions request) async => ResponseData(404, <String, dynamic>{
          'code': 1,
          'message': 'not found',
        }),
      );

      final ExternalLinkPolicyRepository repository =
          ExternalLinkPolicyRepository(
            GfApiClient(dio: dio, tokenStorage: _Storage()),
          );
      expect(await repository.fetchTrustedDomains(), isNull);
    },
  );
}
