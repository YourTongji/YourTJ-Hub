import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

import '../integration_test/harmony_system_dialog_probe.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('com.spencerccf.app_settings/methods');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  testWidgets(
    'settings probe distinguishes pending, missing bridge and retry',
    (tester) async {
      final pending = Completer<Object?>();
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.arguments, {'type': 'settings'});
        return pending.future;
      });
      await tester.pumpWidget(const MaterialApp(home: SystemDialogProbe()));
      final button = find.widgetWithText(
        TextButton,
        'Test application details settings',
      );
      await tester.tap(button);
      await tester.pump();
      expect(find.text('settings:settings:waiting'), findsOneWidget);
      expect(tester.widget<TextButton>(button).onPressed, isNull);

      pending.completeError(MissingPluginException());
      await tester.pumpAndSettle();
      expect(find.text('settings:missing_plugin'), findsOneWidget);
      expect(tester.widget<TextButton>(button).onPressed, isNotNull);

      messenger.setMockMethodCallHandler(channel, (_) async => null);
      await tester.tap(button);
      await tester.pumpAndSettle();
      expect(find.text('settings:settings:returned'), findsOneWidget);
    },
  );
}
