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
  Map<String, dynamic> fixture(String name) =>
      jsonDecode(
            File(
              '../../../../packages/api-contract/fixtures/$name.json',
            ).readAsStringSync(),
          )
          as Map<String, dynamic>;

  test('deleted content preserves server restore/permanent permissions', () {
    final page = UserContentPage.fromJson(
      fixture('deleted-content-success')['result'] as Map<String, dynamic>,
    );
    expect(page.items.single.canRestore, isFalse);
    expect(page.items.single.canPermanent, isTrue);
    expect(page.hasMore, isFalse);
    expect(page.nextCursorId, page.items.single.id);
  });

  test(
    'pagination and content kind are sent with authenticated requests',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
      final adapter = MockAdapter((request) async {
        expect(request.path, '/api/forum/user/deleted-content');
        expect(request.queryParameters['contentType'], 'post');
        expect(request.queryParameters['cursorId'], 100);
        expect(request.headers['Authorization'], 'Bearer session');
        return ResponseData(200, fixture('deleted-content-success'));
      });
      dio.httpClientAdapter = adapter;
      final repo = ContentRepository(
        GfApiClient(dio: dio, tokenStorage: _Tokens()),
      );
      await repo.list(contentType: 'post', deleted: true, cursor: 100);
      dio.close();
    },
  );

  test(
    'batch result retains individual failures and only explicit force',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.data, {
          'contentType': 'topic',
          'contentIds': [1, 2],
        });
        return ResponseData(200, {
          'code': 0,
          'result': {
            'succeeded': 1,
            'failed': 1,
            'results': [
              {'contentId': 1, 'success': true},
              {'contentId': 2, 'success': false, 'message': 'Not recoverable'},
            ],
          },
        });
      });
      final repo = ContentRepository(
        GfApiClient(dio: dio, tokenStorage: _Tokens()),
      );
      final results = await repo.delete(contentType: 'topic', ids: [1, 2]);
      expect(results.last.success, isFalse);
      expect(results.last.message, 'Not recoverable');
      dio.close();
    },
  );
}
