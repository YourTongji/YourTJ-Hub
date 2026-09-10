// Runs on a simulator/device. Accept the actual OS authorization dialog when shown.
// This checks the native permission bridge only, not provider delivery.
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/src/push/push_service.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('system notification permission is requestable and readable', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: Text('YourTJ notification permission test')),
      ),
    );
    final driver = PushDriver();
    final allowed = await driver.permission(request: true);
    expect(
      allowed,
      isTrue,
      reason: 'Accept the OS notification permission dialog',
    );
    expect(await driver.permission(request: false), isTrue);
    // No registration: no token leaves the device and no provider SDK is started.
  });
}
