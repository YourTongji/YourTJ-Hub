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
  test(
    'uses the shared batch endpoint and parses the typed contract',
    () async {
      final Dio dio = Dio();
      dio.httpClientAdapter = MockAdapter((RequestOptions request) async {
        expect(request.method, 'POST');
        expect(request.path, '/api/link-previews/resolve');
        expect(request.data, <String, Object>{
          'urls': <String>['https://example.com/article'],
        });
        return ResponseData(200, <String, dynamic>{
          'code': 0,
          'result': <Map<String, Object>>[
            <String, Object>{
              'requestedUrl': 'https://example.com/article',
              'kind': 'external',
              'status': 'ready',
              'url': 'https://example.com/article',
              'displayHost': 'example.com',
              'registrableDomain': 'example.com',
              'title': 'Example article',
            },
          ],
        });
      });

      final LinkPreviewRepository repository = LinkPreviewRepository(
        GfApiClient(dio: dio, tokenStorage: _Storage()),
      );
      final List<LinkPreviewPayload> previews = await repository.resolve(
        <String>['https://example.com/article'],
      );

      expect(previews.single.isReady, isTrue);
      expect(previews.single.registrableDomain, 'example.com');
      expect(previews.single.title, 'Example article');
    },
  );
}
