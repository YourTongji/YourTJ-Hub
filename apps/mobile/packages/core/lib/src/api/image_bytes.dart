/// 图片字节嗅探：上传前按魔数把文件名扩展名校正为字节的真实格式。
///
/// 背景（issue #969）：`image_picker` 带 `maxWidth`/`imageQuality` 压缩时，
/// Android 会把无透明通道的图重编码成 JPEG，但保留原文件名；`.png` 名字 +
/// JPEG 字节会被服务端按扩展名得到 `image/png`，嗅探/解码得到 `image/jpeg`，
/// 于是稳定拒绝 `upload.image.invalidContent`。这里在唯一上传收敛点
/// [FileRepository.uploadImage] 纠正扩展名，保留压缩行为与服务端校验强度。
///
/// 与显示侧的 SVG 内容嗅探不是同一问题：显示侧只判断「是不是 SVG 文档」
/// （ui_kit 不依赖 core）；这里回答「是哪种受支持位图」。
library;

const List<int> _jpegMagic = [0xFF, 0xD8, 0xFF];
const List<int> _pngMagic = [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A];
const List<int> _bmpMagic = [0x42, 0x4D];

/// 与字节内容匹配的扩展名（含点、小写）；无法识别时返回 null。
String? imageExtensionForBytes(List<int> bytes) {
  if (_startsWith(bytes, _jpegMagic)) return '.jpg';
  if (_startsWith(bytes, _pngMagic)) return '.png';
  if (_startsWithAscii(bytes, 'GIF87a') || _startsWithAscii(bytes, 'GIF89a')) {
    return '.gif';
  }
  if (_isWebP(bytes)) return '.webp';
  if (_startsWith(bytes, _bmpMagic)) return '.bmp';
  return null;
}

/// 用嗅探结果校正文件名的扩展名；未知/空字节返回原名。
///
/// 只替换最后一个扩展名段并保留 basename（表情包显示名依赖 basename）；
/// 目录分隔符（`/`、`\`）之前的部分原样保留；无扩展名时追加。
/// 扩展名已是同一格式族（如 `.jpeg` 对 JPEG 字节）时保留原名，不做无谓重命名。
String alignImageFileNameWithBytes(String filename, List<int> bytes) {
  final extension = imageExtensionForBytes(bytes);
  if (extension == null) return filename;
  final lastSeparator = filename.lastIndexOf(RegExp(r'[/\\]'));
  final lastDot = filename.lastIndexOf('.');
  if (lastDot > lastSeparator) {
    final family = _extensionFamily[filename.substring(lastDot).toLowerCase()];
    if (family == extension) return filename;
  }
  final baseEnd = lastDot > lastSeparator ? lastDot : filename.length;
  return '${filename.substring(0, baseEnd)}$extension';
}

/// 已知图片扩展名 → 其格式族（与 [imageExtensionForBytes] 的返回值对应）。
const _extensionFamily = {
  '.jpg': '.jpg',
  '.jpeg': '.jpg',
  '.png': '.png',
  '.gif': '.gif',
  '.webp': '.webp',
  '.bmp': '.bmp',
};

bool _startsWith(List<int> bytes, List<int> prefix) {
  if (bytes.length < prefix.length) return false;
  for (var i = 0; i < prefix.length; i++) {
    if (bytes[i] != prefix[i]) return false;
  }
  return true;
}

bool _startsWithAscii(List<int> bytes, String ascii) => _startsWith(bytes, ascii.codeUnits);

/// `RIFF....WEBP`：第 0..4 字节为 RIFF、第 8..12 字节为 WEBP。
bool _isWebP(List<int> bytes) {
  if (bytes.length < 12) return false;
  return _startsWithAscii(bytes, 'RIFF') &&
      bytes[8] == 0x57 &&
      bytes[9] == 0x45 &&
      bytes[10] == 0x42 &&
      bytes[11] == 0x50;
}
