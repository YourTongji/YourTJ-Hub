import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/auth/harmony_passkeys.dart';
import 'package:passkeys/exceptions.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('yourtj/passkeys');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test(
    'preserves signed client data while translating Huawei response names',
    () async {
      final clientData = base64Url.encode(
        utf8.encode('{"origin":"https://example.com"}'),
      );
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'authenticate');
        expect(jsonDecode(call.arguments as String), {
          'challenge': 'challenge',
        });
        return jsonEncode({
          'id': 'AQI=',
          'rawId': 'AQI=',
          'type': 'public-key',
          'response': {
            'clientDataJson': clientData,
            'authenticatorData': 'AwQ=',
            'signature': 'BQY=',
            'userHandle': null,
          },
          'clientExtensionResults': null,
        });
      });
      final result = await HarmonyPasskeys.authenticate({
        'challenge': 'challenge',
      });
      expect(result['id'], 'AQI');
      final response = result['response'] as Map<String, dynamic>;
      expect(
        base64Url.decode(
          base64Url.normalize(response['clientDataJSON'] as String),
        ),
        utf8.encode('{"origin":"https://example.com"}'),
      );
      expect(response.containsKey('clientDataJson'), isFalse);
      expect(response['userHandle'], isNull);
      expect(result['clientExtensionResults'], isEmpty);
    },
  );

  test(
    'maps cancellation and unsupported devices to existing safe errors',
    () async {
      messenger.setMockMethodCallHandler(channel, (_) async {
        throw PlatformException(code: 'cancelled');
      });
      await expectLater(
        HarmonyPasskeys.register({}),
        throwsA(isA<PasskeyAuthCancelledException>()),
      );
      messenger.setMockMethodCallHandler(channel, (_) async {
        throw PlatformException(code: 'deviceNotSupported');
      });
      expect(await HarmonyPasskeys.isAvailable(), isFalse);
    },
  );

  test('rejects malformed native responses', () async {
    messenger.setMockMethodCallHandler(channel, (_) async => '{}');
    await expectLater(HarmonyPasskeys.authenticate({}), throwsFormatException);
  });

  test(
    'registration keeps attestation bytes and normalizes NULL extensions',
    () async {
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'register');
        return jsonEncode({
          'id': 'AQI=',
          'rawId': 'AQI=',
          'type': 'public-key',
          'response': {'clientDataJson': 'AwQ=', 'attestationObject': '+/8='},
          'clientExtensionResults': 'NULL',
        });
      });
      final credential = await HarmonyPasskeys.register({});
      final response = credential['response'] as Map<String, dynamic>;
      expect(response['attestationObject'], '-_8');
      expect(response['clientDataJSON'], 'AwQ');
      expect(credential['clientExtensionResults'], isEmpty);
    },
  );
}
