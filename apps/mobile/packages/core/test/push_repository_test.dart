import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'api_client_test.dart' show MockAdapter, ResponseData;

class Tokens implements TokenStorage {
  String value = 'session';
  @override
  Future<String?> read() async => value;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

void main() {
  test('push uses authenticated forum routes and explicit transport', () async {
    final paths = <String>[];
    final dio = Dio();
    addTearDown(dio.close);
    dio.httpClientAdapter = MockAdapter((request) async {
      paths.add(request.path);
      expect(request.headers['Authorization'], 'Bearer session');
      if (request.path.endsWith('/register')) {
        expect(request.data, {
          'platform': 'android',
          'provider': 'jpush',
          'token': 'rid',
        });
      }
      return ResponseData(200, {
        'code': 0,
        'msg': 'success',
        'result': request.method == 'GET'
            ? {
                'configured': false,
                'native': {
                  'apnsEnabled': true,
                  'fcmEnabled': false,
                  'jpushEnabled': true,
                },
              }
            : true,
      });
    });
    final tokens = Tokens();
    final repo = await PushRepository(
      GfApiClient(dio: dio, tokenStorage: tokens),
    ).forSession();
    tokens.value = 'different-account';
    expect((await repo.config()).native!.jpushEnabled, true);
    await repo.registerDevice(
      platform: 'android',
      provider: 'jpush',
      token: 'rid',
    );
    await repo.unregisterDevice(token: 'rid');
    expect(paths, [
      '/api/forum/push/config',
      '/api/forum/push/device/register',
      '/api/forum/push/device/unregister',
    ]);
  });
}
