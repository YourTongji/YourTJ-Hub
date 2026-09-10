import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'api_client_test.dart' show MockAdapter, ResponseData;

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

void main() {
  test(
    'Wiki search uses the controlled page-grouped fixture and anchors',
    () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
      addTearDown(dio.close);
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.path, '/api/wiki/search');
        expect(request.queryParameters, {'q': '保研'});
        return ResponseData(
          200,
          jsonDecode(
            File(
              '../../../../packages/api-contract/fixtures/wiki-search-success.json',
            ).readAsStringSync(),
          ),
        );
      });
      final result = await WikiRepository(
        GfApiClient(dio: dio, tokenStorage: _Tokens()),
      ).search('保研');
      expect(result.searchUnavailable, isFalse);
      expect(result.query, '保研');
      expect(result.items.first.titleHit, isTrue);
      expect(result.items.first.score, 0.92);
      expect(result.items.first.hitType, 'title');
      expect(result.items.last.titleHit, isFalse);
      expect(result.items.last.hitType, 'body');
      expect(result.items.first.path, '同济新手教程/academics/保研');
      expect(result.items.first.anchors, ['s-1', 's-2']);
      expect(result.items.first.snippet, contains('<mark>保研</mark>'));
      expect(
        WikiSearchResult.fromJson({
          'query': '保研',
          'total': 0,
          'items': [],
          'searchUnavailable': true,
        }).searchUnavailable,
        isTrue,
      );
    },
  );
}
