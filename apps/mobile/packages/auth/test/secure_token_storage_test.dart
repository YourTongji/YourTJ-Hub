import 'package:auth/auth.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

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
