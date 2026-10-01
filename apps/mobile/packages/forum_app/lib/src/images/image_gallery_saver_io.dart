import 'package:flutter/foundation.dart';
import 'package:gal/gal.dart';
import 'package:image/image.dart' as image;

import 'image_gallery_saver.dart';

export 'image_gallery_saver.dart' show ImageGallerySaveResult;

Future<ImageGallerySaveResult> saveImageToGallery(
  Uint8List bytes,
  String fileName,
) async {
  // Android 10 can report general access while denying the legacy album write.
  final bool toAlbum = defaultTargetPlatform == TargetPlatform.android;
  try {
    if (!await Gal.hasAccess(toAlbum: toAlbum) &&
        !await Gal.requestAccess(toAlbum: toAlbum)) {
      debugPrint('Gallery save denied (permission request was declined).');
      return ImageGallerySaveResult.permissionDenied;
    }
    final String name = _galleryName(fileName);
    try {
      await Gal.putImageBytes(bytes, name: name);
      return ImageGallerySaveResult.saved;
    } on GalException catch (error) {
      debugPrint('Gallery image save failed (${error.type}).');
      if (error.type == GalExceptionType.accessDenied) {
        return ImageGallerySaveResult.permissionDenied;
      }
      // A PNG retry only helps when the original encoding is unsupported.
      if (error.type != GalExceptionType.notSupportedFormat &&
          error.type != GalExceptionType.unexpected) {
        return ImageGallerySaveResult.failed;
      }
      final decoded = image.decodeImage(bytes);
      if (decoded == null) return ImageGallerySaveResult.failed;
      final pngBytes = Uint8List.fromList(image.encodePng(decoded));
      try {
        await Gal.putImageBytes(pngBytes, name: '$name.png');
        return ImageGallerySaveResult.saved;
      } on GalException catch (retryError) {
        debugPrint('Gallery PNG retry failed (${retryError.type}).');
        return retryError.type == GalExceptionType.accessDenied
            ? ImageGallerySaveResult.permissionDenied
            : ImageGallerySaveResult.failed;
      }
    }
  } on GalException catch (error) {
    debugPrint('Gallery save failed (${error.type}).');
    return error.type == GalExceptionType.accessDenied
        ? ImageGallerySaveResult.permissionDenied
        : ImageGallerySaveResult.failed;
  } catch (_) {
    debugPrint('Gallery save failed (unexpected exception).');
    return ImageGallerySaveResult.failed;
  }
}

String _galleryName(String fileName) {
  final int extensionStart = fileName.lastIndexOf('.');
  return extensionStart > 0 ? fileName.substring(0, extensionStart) : fileName;
}
