import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/widgets/glass_accessibility_host.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  const channel = MethodChannel('yourtj/accessibility');
  testWidgets(
    'native transparency preference stays live and stale reads cannot restore glass',
    (tester) async {
      final initial = Completer<bool>();
      final messenger = tester.binding.defaultBinaryMessenger;
      messenger.setMockMethodCallHandler(channel, (_) => initial.future);
      addTearDown(() {
        messenger.setMockMethodCallHandler(channel, null);
      });
      var opaque = false;
      await tester.pumpWidget(
        MaterialApp(
          home: GlassAccessibilityHost(
            child: Builder(
              builder: (context) {
                opaque = GfGlassSettings.opaqueOf(context);
                return const SizedBox();
              },
            ),
          ),
        ),
      );
      expect(
        opaque,
        isTrue,
        reason: 'Wait for the preference before using optics',
      );
      Future<void> emit(bool value) async {
        await messenger.handlePlatformMessage(
          channel.name,
          const StandardMethodCodec().encodeMethodCall(
            MethodCall('reduceTransparencyChanged', value),
          ),
          (_) {},
        );
        await tester.pump();
      }

      await emit(false);
      expect(opaque, isFalse);
      await emit(true);
      initial.complete(false);
      await tester.pump();
      expect(
        opaque,
        isTrue,
        reason: 'An older read must not override the native change',
      );
      await emit(false);
      expect(opaque, isFalse);
      await tester.pumpWidget(const SizedBox());
      await emit(true);
      expect(tester.takeException(), isNull);
    },
    variant: TargetPlatformVariant({TargetPlatform.iOS}),
  );
}
