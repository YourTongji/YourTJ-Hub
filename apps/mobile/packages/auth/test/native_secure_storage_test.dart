import 'package:auth/auth.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

class _HarmonyStorage extends NativeSecureStorage {
  @override
  bool get isHarmonyPlatform => true;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  final storage = _HarmonyStorage();
  final values = <String, String>{};
  final calls = <MethodCall>[];

  setUp(() {
    values.clear();
    calls.clear();
    messenger.setMockMethodCallHandler(NativeSecureStorage.channel, (
      call,
    ) async {
      calls.add(call);
      final args = call.arguments as Map<Object?, Object?>?;
      final key = args?['key'] as String?;
      return switch (call.method) {
        'read' => values[key],
        'containsKey' => values.containsKey(key),
        'write' => values[key!] = args!['value'] as String,
        'delete' => values.remove(key),
        'readAll' => Map<String, String>.of(values),
        'deleteAll' => values.clear(),
        _ => throw MissingPluginException(),
      };
    });
  });
  tearDown(
    () => messenger.setMockMethodCallHandler(NativeSecureStorage.channel, null),
  );

  test(
    'HarmonyOS storage preserves read, delete and enumeration contracts',
    () async {
      await storage.write(key: 'draft', value: 'sample');
      expect(await storage.read(key: 'draft'), 'sample');
      expect(await storage.containsKey(key: 'draft'), isTrue);
      expect(await storage.readAll(), {'draft': 'sample'});
      await storage.write(key: 'draft', value: null);
      expect(await storage.read(key: 'draft'), isNull);
      expect(calls.where((call) => call.method == 'delete').length, 1);
      await storage.write(key: 'other', value: 'sample');
      await storage.deleteAll();
      expect(await storage.readAll(), isEmpty);
    },
  );

  test('native encryption errors propagate without a fallback', () async {
    messenger.setMockMethodCallHandler(NativeSecureStorage.channel, (_) async {
      throw PlatformException(code: 'secure_storage_failed');
    });
    await expectLater(
      storage.read(key: 'draft'),
      throwsA(isA<PlatformException>()),
    );
    await expectLater(
      storage.write(key: 'draft', value: 'sample'),
      throwsA(isA<PlatformException>()),
    );
  });
}
