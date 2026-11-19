import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:quill_native_bridge/quill_native_bridge.dart';

const harmonyClipboardChannel = MethodChannel('yourtj/clipboard');

class HarmonyQuillNativeBridge extends QuillNativeBridge {
  @override
  Future<bool> isSupported(QuillNativeBridgeFeature feature) async => const {
    QuillNativeBridgeFeature.getClipboardHtml,
    QuillNativeBridgeFeature.copyHtmlToClipboard,
    QuillNativeBridgeFeature.getClipboardImage,
    QuillNativeBridgeFeature.getClipboardGif,
    QuillNativeBridgeFeature.copyImageToClipboard,
  }.contains(feature);

  Future<T?> _read<T>(String method) async {
    try {
      return await harmonyClipboardChannel.invokeMethod<T>(method);
    } on PlatformException catch (error) {
      if (error.code == 'clipboard_permission_denied') return null;
      rethrow;
    }
  }

  @override
  Future<String?> getClipboardHtml() => _read<String>('getHtml');

  @override
  Future<Uint8List?> getClipboardImage() => _read<Uint8List>('getImage');

  @override
  Future<Uint8List?> getClipboardGif() => _read<Uint8List>('getGif');

  @override
  Future<void> copyHtmlToClipboard(String html) =>
      harmonyClipboardChannel.invokeMethod<void>('copyHtml', html);

  @override
  Future<void> copyImageToClipboard(Uint8List imageBytes) =>
      harmonyClipboardChannel.invokeMethod<void>('copyImage', imageBytes);
}
