import 'dart:convert';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/apple/apple_sign_in.dart';

String token(String jti, {int expiry = 1}) =>
    'header.${base64UrlEncode(utf8.encode(jsonEncode({'jti': jti, 'exp': expiry}))).replaceAll('=', '')}.signature';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('yourtj/apple-auth');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  late AppleSignInService service;
  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
    service = AppleSignInService();
  });
  tearDown(() {
    service.dispose();
    messenger.setMockMethodCallHandler(channel, null);
  });

  test(
    'native authorization gets a fresh nonce and no profile scopes',
    () async {
      final nonces = <String>[];
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'authorize');
        final args = call.arguments as Map;
        expect(args.keys, ['nonce']);
        nonces.add(args['nonce'] as String);
        return {
          'authorizationCode': 'one-use-code',
          'identityToken': 'signed-token',
          'userIdentifier': 'private-apple-user',
        };
      });
      final first = await service.authorize();
      final second = await service.authorize();
      expect(first!.request.nonce, nonces.first);
      expect(second!.request.nonce, nonces.last);
      expect(nonces.first, isNot(nonces.last));
      expect(
        base64Url.decode(base64Url.normalize(nonces.first)),
        hasLength(32),
      );
    },
  );

  test('user cancellation is not an authentication failure', () async {
    messenger.setMockMethodCallHandler(
      channel,
      (_) async => throw PlatformException(code: 'cancelled'),
    );
    expect(await service.authorize(), isNull);
  });

  test(
    'revocation follows session renewal but never a later account login',
    () async {
      var calls = 0;
      messenger.setMockMethodCallHandler(channel, (call) async {
        calls++;
        expect(call.method, 'credentialState');
        expect((call.arguments as Map)['userIdentifier'], 'apple-user');
        return 'revoked';
      });
      await service.rememberSession(token('apple-session'), 'apple-user');
      expect(
        await service.isRevoked(token('apple-session', expiry: 2)),
        isTrue,
      );
      expect(await service.isRevoked(token('later-google-session')), isFalse);
      expect(calls, 1);
      await service.rememberSession(token('later-password-session'), null);
      expect(await service.isRevoked(token('apple-session')), isFalse);
      expect(calls, 1);
    },
  );

  test('transient check errors are not reported as revoked', () async {
    await service.rememberSession(token('apple-session'), 'apple-user');
    messenger.setMockMethodCallHandler(
      channel,
      (_) async => throw PlatformException(code: 'unavailable'),
    );
    await expectLater(
      service.isRevoked(token('apple-session')),
      throwsA(isA<PlatformException>()),
    );
    messenger.setMockMethodCallHandler(channel, (_) async => 'authorized');
    expect(await service.isRevoked(token('apple-session')), isFalse);
  });
}
