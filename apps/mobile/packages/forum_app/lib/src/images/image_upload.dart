import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../providers.dart';

/// Picks a compressed image from [source], uploads it through the forum file
/// endpoint, and returns the URL that can be embedded in Markdown.
///
/// A cancelled picker returns `null`; upload failures are intentionally allowed
/// to propagate so the page that owns the interaction can present localised
/// feedback.
Future<String?> pickAndUploadImage({
  required WidgetRef ref,
  ImagePicker? picker,
  ImageSource source = ImageSource.gallery,
}) async {
  final XFile? picked = await (picker ?? ImagePicker()).pickImage(
    source: source,
    maxWidth: 2048,
    imageQuality: 85,
  );
  if (picked == null) return null;

  final List<int> bytes = await picked.readAsBytes();
  return ref
      .read(fileRepositoryProvider)
      .uploadImage(bytes: bytes, filename: picked.name);
}
