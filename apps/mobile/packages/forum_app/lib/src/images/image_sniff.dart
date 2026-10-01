/// Magic-byte sniffing that keeps uploaded image filenames honest.
///
/// The platform image picker may re-encode a selected image while keeping
/// its original name: on Android, a PNG without an alpha channel is resized
/// and written back as JPEG bytes under a `.png` name (issue #969). The
/// server derives the content type authoritatively from the file extension
/// and rejects a mismatch, so the client rewrites the extension to the
/// sniffed format before uploading.
library;

/// Sniffs the image format from the magic-number header of [bytes] and
/// returns the canonical format name (`jpeg`, `png`, `gif`, `webp`, `bmp`),
/// or `null` when the header is missing or not a recognized image.
String? sniffImageFormat(List<int> bytes) {
  bool startsWith(List<int> signature) {
    if (bytes.length < signature.length) return false;
    for (var i = 0; i < signature.length; i++) {
      if (bytes[i] != signature[i]) return false;
    }
    return true;
  }

  if (startsWith(const [0xFF, 0xD8, 0xFF])) return 'jpeg';
  if (startsWith(const [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A])) {
    return 'png';
  }
  if (startsWith(const [0x47, 0x49, 0x46, 0x38])) return 'gif';
  if (startsWith(const [0x42, 0x4D])) return 'bmp';
  // WebP: "RIFF" <u32 size> "WEBP".
  if (bytes.length >= 12 &&
      bytes[0] == 0x52 &&
      bytes[1] == 0x49 &&
      bytes[2] == 0x46 &&
      bytes[3] == 0x46 &&
      bytes[8] == 0x57 &&
      bytes[9] == 0x45 &&
      bytes[10] == 0x42 &&
      bytes[11] == 0x50) {
    return 'webp';
  }
  return null;
}

/// Extension → sniffed format family. Mirrors the server upload allowlist.
const _formatByExtension = {
  '.jpg': 'jpeg',
  '.jpeg': 'jpeg',
  '.png': 'png',
  '.gif': 'gif',
  '.webp': 'webp',
  '.bmp': 'bmp',
};

/// Canonical extension used when renaming to [format].
const _extensionByFormat = {
  'jpeg': '.jpg',
  'png': '.png',
  'gif': '.gif',
  'webp': '.webp',
  'bmp': '.bmp',
};

/// Returns [filename] with its extension rewritten to the sniffed image
/// format when the two disagree. Unrecognized bytes and filenames without
/// an extension are returned unchanged so the server keeps rejecting them
/// rather than the client guessing.
String normalizeImageFilename(String filename, List<int> bytes) {
  final format = sniffImageFormat(bytes);
  if (format == null) return filename;
  final dot = filename.lastIndexOf('.');
  if (dot <= 0) return filename;
  final currentExtension = filename.substring(dot).toLowerCase();
  if (_formatByExtension[currentExtension] == format) return filename;
  return '${filename.substring(0, dot)}${_extensionByFormat[format]}';
}
