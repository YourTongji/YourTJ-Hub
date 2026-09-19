import 'dart:typed_data';

import 'package:gal/gal.dart';

Future<bool> saveImageToGallery(Uint8List bytes, String fileName) async {
  if (!await Gal.hasAccess() && !await Gal.requestAccess()) return false;
  await Gal.putImageBytes(bytes, name: fileName);
  return true;
}
