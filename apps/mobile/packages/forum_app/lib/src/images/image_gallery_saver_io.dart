import 'package:flutter/foundation.dart';
import 'package:gal/gal.dart';
import 'package:image/image.dart' as image;

Future<bool> saveImageToGallery(Uint8List bytes, String fileName) async {
  // gal's Android implementation considers API 29's non-album access
  // granted even though it writes through the legacy external-storage path.
  // Request the album-level permission there so the subsequent write cannot
  // fail with a missing WRITE_EXTERNAL_STORAGE grant.
  final bool toAlbum = defaultTargetPlatform == TargetPlatform.android;
  if (!await Gal.hasAccess(toAlbum: toAlbum) &&
      !await Gal.requestAccess(toAlbum: toAlbum)) {
    return false;
  }
  final String name = _galleryName(fileName);
  try {
    await Gal.putImageBytes(bytes, name: name);
    return true;
  } on GalException catch (error) {
    // Permission and storage failures are not fixed by changing the image
    // format. Do not hide those states behind a second native write attempt.
    if (error.type != GalExceptionType.notSupportedFormat &&
        error.type != GalExceptionType.unexpected) {
      return false;
    }
    final image.Image? decoded = image.decodeImage(bytes);
    if (decoded == null) return false;
    final Uint8List pngBytes = Uint8List.fromList(image.encodePng(decoded));
    try {
      await Gal.putImageBytes(pngBytes, name: '$name.png');
      return true;
    } on GalException {
      return false;
    }
  }
}

String _galleryName(String fileName) {
  final int extensionStart = fileName.lastIndexOf('.');
  return extensionStart > 0 ? fileName.substring(0, extensionStart) : fileName;
}
