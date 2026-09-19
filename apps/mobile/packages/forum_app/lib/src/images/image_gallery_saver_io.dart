import 'package:flutter/foundation.dart';
import 'package:gal/gal.dart';

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
  await Gal.putImageBytes(bytes, name: _galleryName(fileName));
  return true;
}

String _galleryName(String fileName) {
  final int extensionStart = fileName.lastIndexOf('.');
  return extensionStart > 0 ? fileName.substring(0, extensionStart) : fileName;
}
