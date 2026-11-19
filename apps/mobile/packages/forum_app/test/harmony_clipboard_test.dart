// The local editor clipboard API is intentionally tested while experimental.
// ignore_for_file: experimental_member_use

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_quill/flutter_quill.dart';
// This test verifies the internal contract of the local editor adaptation.
// ignore: implementation_imports
import 'package:flutter_quill/src/common/utils/quill_native_provider.dart';
// ignore: implementation_imports
import 'package:flutter_quill/src/editor_toolbar_controller_shared/clipboard/default_clipboard_service.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  final harmony = TargetPlatform.values
      .where((p) => p.name == 'ohos')
      .firstOrNull;
  const channel = MethodChannel('yourtj/clipboard');
  final calls = <MethodCall>[];
  group(
    'HarmonyOS editor clipboard',
    () {
      setUp(() {
        debugDefaultTargetPlatformOverride = harmony;
        QuillNativeProvider.instance = null;
        calls.clear();
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(channel, (call) async {
              calls.add(call);
              return switch (call.method) {
                'hasData' => true,
                'getHtml' => '<p>synthetic <b>bold</b></p>',
                'getImage' => Uint8List.fromList([1, 2, 3]),
                'getGif' => Uint8List.fromList([71, 73, 70]),
                _ => null,
              };
            });
      });
      tearDown(() {
        debugDefaultTargetPlatformOverride = null;
        QuillNativeProvider.instance = null;
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(channel, null);
      });
      test('capability checks never access clipboard contents', () async {
        expect(
          await QuillNativeProvider.instance.isSupported(
            QuillNativeBridgeFeature.getClipboardHtml,
          ),
          isTrue,
        );
        expect(
          await QuillNativeProvider.instance.isSupported(
            QuillNativeBridgeFeature.getClipboardImage,
          ),
          isTrue,
        );
        expect(
          await QuillNativeProvider.instance.isSupported(
            QuillNativeBridgeFeature.getClipboardFiles,
          ),
          isFalse,
        );
        expect(calls, isEmpty);
      });
      test(
        'toolbar queries presence while explicit paste uses native data',
        () async {
          final service = DefaultClipboardService();
          expect(await service.hasClipboardContent, isTrue);
          expect(calls.map((call) => call.method), ['hasData']);
          expect(await service.getHtmlText(), '<p>synthetic <b>bold</b></p>');
          expect(await service.getImageFile(), [1, 2, 3]);
          expect(await service.getGifFile(), [71, 73, 70]);
          await QuillNativeProvider.instance.copyHtmlToClipboard('<p>copy</p>');
          expect(calls.last.method, 'copyHtml');
          expect(calls.last.arguments, '<p>copy</p>');
          await service.copyImage(Uint8List.fromList([4, 5]));
          expect(calls.last.method, 'copyImage');
          expect(calls.last.arguments, [4, 5]);
        },
      );
      test('native HTML paste preserves bold content in the editor', () async {
        final controller = QuillController.basic();
        addTearDown(controller.dispose);
        expect(await controller.clipboardPaste(), isTrue);
        expect(controller.document.toPlainText().trimRight(), 'synthetic bold');
        expect(
          controller.document.toDelta().toJson(),
          contains(equals({'insert': 'bold', 'attributes': {'bold': true}})),
        );
        expect(calls.map((call) => call.method), ['getHtml']);
      });
      test(
        'permission rejection leaves rich paste empty without hiding other errors',
        () async {
          TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
              .setMockMethodCallHandler(channel, (_) async {
                throw PlatformException(code: 'clipboard_permission_denied');
              });
          expect(await DefaultClipboardService().getHtmlText(), isNull);
          expect(await DefaultClipboardService().getImageFile(), isNull);
          TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
              .setMockMethodCallHandler(channel, (_) async {
                throw PlatformException(code: 'clipboard_failed');
              });
          await expectLater(
            DefaultClipboardService().getHtmlText(),
            throwsA(isA<PlatformException>()),
          );
        },
      );
    },
    skip: harmony == null ? 'Requires Flutter OH' : false,
  );
}
