import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/auth/android_oidc_callback_source.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('test/yourtj/oidc');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test(
    'rejects unsafe authorization origins before calling the native browser',
    () async {
      var calls = 0;
      messenger.setMockMethodCallHandler(channel, (_) async {
        calls++;
        return true;
      });
      final source = MainActivityOidcCallbackSource(methodChannel: channel);
      for (final url in [
        'http://example.com/auth',
        'https://user@example.com/auth',
        'yourtj://callback',
      ]) {
        await expectLater(
          source.openAuthorization(Uri.parse(url)),
          throwsArgumentError,
        );
      }
      expect(calls, 0);
    },
  );

  test(
    'passes the complete HTTPS authorization URI and propagates browser refusal',
    () async {
      final uri = Uri.https('example.com', '/auth', {
        'state': 'sample-state',
        'code_challenge': 'sample-challenge',
      });
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'openAuthorization');
        expect(call.arguments, {'url': uri.toString()});
        return false;
      });
      expect(
        await MainActivityOidcCallbackSource(
          methodChannel: channel,
        ).openAuthorization(uri),
        isFalse,
      );
    },
  );
}
