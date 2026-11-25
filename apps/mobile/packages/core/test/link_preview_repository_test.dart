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

  test(
    'caches resolved previews in LRU memory, fetches only missing URLs, and respects TTL',
    () async {
      var calls = 0;
      final requestedBatches = <List<String>>[];
      var now = DateTime.utc(2026, 11, 26, 10, 0);
      final Dio dio = Dio();
      dio.httpClientAdapter = MockAdapter((RequestOptions request) async {
        calls++;
        final urls = List<String>.from(
          (request.data as Map<String, Object>)['urls'] as List,
        );
        requestedBatches.add(urls);
        return ResponseData(200, <String, dynamic>{
          'code': 0,
          'result': [
            for (final u in urls)
              <String, Object>{
                'requestedUrl': u,
                'kind': 'external',
                'status': 'ready',
                'url': u,
                'displayHost': 'example.com',
                'registrableDomain': 'example.com',
                'title': 'Title for $u',
              },
          ],
        });
      });

      final repo = LinkPreviewRepository(
        GfApiClient(dio: dio, tokenStorage: _Storage()),
        cacheTtl: const Duration(minutes: 10),
        now: () => now,
      );

      final first = await repo.resolve(['https://example.com/a']);
      expect(first.single.title, 'Title for https://example.com/a');
      expect(calls, 1);

      // Repeat hit makes zero network calls.
      final second = await repo.resolve(['https://example.com/a']);
      expect(second.single.title, 'Title for https://example.com/a');
      expect(calls, 1);

      // Mixed hit + miss fetches only the missing URL.
      final mixed = await repo.resolve([
        'https://example.com/a',
        'https://example.com/b',
      ]);
      expect(mixed.map((e) => e.url).toList(), [
        'https://example.com/a',
        'https://example.com/b',
      ]);
      expect(calls, 2);
      expect(requestedBatches.last, ['https://example.com/b']);

      // Expiry triggers re-fetch; clearCache also evicts.
      now = now.add(const Duration(minutes: 11));
      await repo.resolve(['https://example.com/a']);
      expect(calls, 3);
      repo.clearCache();
      await repo.resolve(['https://example.com/a']);
      expect(calls, 4);
    },
  );
}
