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

/// 捕获 multipart 请求里 `file` 部件的 filename。
String? uploadedFilename(RequestOptions request, {String field = 'file'}) {
  final form = request.data as FormData;
  return form.files.singleWhere((entry) => entry.key == field).value.filename;
}

GfApiClient _clientFor(Dio dio) => GfApiClient(dio: dio, tokenStorage: _Tokens());

void main() {
  test('uploadImage renames a .PNG file whose bytes are JPEG', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
    final adapter = MockAdapter((request) async {
      expect(request.method, 'POST');
      expect(request.path, '/file/img-upload');
      expect(uploadedFilename(request), 'scaled_IMG.jpg');
      return ResponseData(200, {
        'code': 0,
        'result': 'https://cdn.example.test/scaled_IMG.jpg',
      });
    });
    dio.httpClientAdapter = adapter;
    final repository = FileRepository(_clientFor(dio));

    final url = await repository.uploadImage(
      bytes: [0xFF, 0xD8, 0xFF, 0xE0],
      filename: 'scaled_IMG.PNG',
    );

    expect(adapter.requests, hasLength(1));
    expect(url, 'https://cdn.example.test/scaled_IMG.jpg');
    dio.close();
  });

  test('uploadImage keeps a matching fixed name (cover.webp)', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
    dio.httpClientAdapter = MockAdapter((request) async {
      expect(uploadedFilename(request), 'cover.webp');
      return ResponseData(200, {
        'code': 0,
        'result': 'https://cdn.example.test/cover.webp',
      });
    });
    final repository = FileRepository(_clientFor(dio));

    await repository.uploadImage(
      bytes: [0x52, 0x49, 0x46, 0x46, 0, 0, 0, 0, 0x57, 0x45, 0x42, 0x50],
      filename: 'cover.webp',
    );

    dio.close();
  });

  test('uploadImage leaves unknown bytes and filenames untouched', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
    dio.httpClientAdapter = MockAdapter((request) async {
      expect(uploadedFilename(request), 'sticker-name.png');
      return ResponseData(200, {
        'code': 0,
        'result': 'https://cdn.example.test/sticker-name.png',
      });
    });
    final repository = FileRepository(_clientFor(dio));

    await repository.uploadImage(
      bytes: [0x00, 0x01, 0x02, 0x03],
      filename: 'sticker-name.png',
    );

    dio.close();
  });

  test('uploadAvatar keeps its own filename path unchanged', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://example.test'));
    dio.httpClientAdapter = MockAdapter((request) async {
      expect(request.path, '/api/upload-avatar');
      expect(uploadedFilename(request, field: 'avatar'), 'avatar.webp');
      return ResponseData(200, {
        'code': 0,
        'result': {'avatarUrl': 'https://cdn.example.test/avatar.webp'},
      });
    });
    final repository = FileRepository(_clientFor(dio));

    final avatarUrl = await repository.uploadAvatar(
      bytes: [0x52, 0x49, 0x46, 0x46, 0, 0, 0, 0, 0x57, 0x45, 0x42, 0x50],
      filename: 'avatar.webp',
    );

    expect(avatarUrl, 'https://cdn.example.test/avatar.webp');
    dio.close();
  });
}
