import 'dart:typed_data';

/// Web has no native gallery. The web implementation uses the browser file
/// download path in [image_save.dart] instead.
Future<bool> saveImageToGallery(Uint8List bytes, String fileName) async =>
    false;
