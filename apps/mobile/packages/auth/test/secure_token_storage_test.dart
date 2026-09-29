import 'package:auth/auth.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'failed token deletion cannot restore a revoked session on restart',
    () async {
      final backing = DeleteFailingStorage();
      final storage = SecureTokenStorage(storage: backing);
      await expectLater(storage.clear(), throwsStateError);
      expect(await SecureTokenStorage(storage: backing).read(), isNull);
      await storage.write('new-login');
      expect(await SecureTokenStorage(storage: backing).read(), 'new-login');
    },
  );

  test('caches the token and refreshes the cache after mutations', () async {
    final persisted = <String, String>{'yourtj.session.token': 'initial'};
    FlutterSecureStorage.setMockInitialValues(persisted);
    final storage = SecureTokenStorage();

    expect(await storage.read(), 'initial');
    persisted['yourtj.session.token'] = 'external-change';
    expect(await storage.read(), 'initial');

    await storage.write('replacement');
    expect(await storage.read(), 'replacement');

    await storage.clear();
    expect(await storage.read(), isNull);
  });
}

class DeleteFailingStorage extends FlutterSecureStorage {
  DeleteFailingStorage();
  final values = <String, String>{'yourtj.session.token': 'old-token'};
  @override
  Future<String?> read({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) async => values[key];
  @override
  Future<void> write({
    required String key,
    required String? value,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) async {
    if (value == null) {
      values.remove(key);
    } else {
      values[key] = value;
    }
  }

  @override
  Future<void> delete({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
    WindowsOptions? wOptions,
  }) async {
    if (key == 'yourtj.session.token') throw StateError('delete unavailable');
    values.remove(key);
  }
}
