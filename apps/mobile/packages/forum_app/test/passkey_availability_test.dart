import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/auth/passkey_availability.dart';

void main() {
  test('only probes the current supported platform', () async {
    var webCalls = 0;
    var androidCalls = 0;
    var iosCalls = 0;
    Future<bool> web() async {
      webCalls++;
      return true;
    }

    Future<bool> android() async {
      androidCalls++;
      return true;
    }

    Future<bool> ios() async {
      iosCalls++;
      return false;
    }

    expect(
      await resolvePasskeyAvailability(
        isWeb: false,
        platform: TargetPlatform.android,
        web: web,
        android: android,
        ios: ios,
      ),
      isTrue,
    );
    expect(androidCalls, 1);
    expect(webCalls, 0);
    expect(iosCalls, 0);

    expect(
      await resolvePasskeyAvailability(
        isWeb: false,
        platform: TargetPlatform.iOS,
        web: web,
        android: android,
        ios: ios,
      ),
      isFalse,
    );
    expect(iosCalls, 1);

    expect(
      await resolvePasskeyAvailability(
        isWeb: false,
        platform: TargetPlatform.linux,
        web: web,
        android: android,
        ios: ios,
      ),
      isFalse,
    );
    expect(
      await resolvePasskeyAvailability(
        isWeb: true,
        platform: TargetPlatform.linux,
        web: web,
        android: android,
        ios: ios,
      ),
      isTrue,
    );
    expect(webCalls, 1);
  });
}
