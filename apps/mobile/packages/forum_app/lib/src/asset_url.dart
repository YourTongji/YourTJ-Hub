import 'package:core/core.dart';

import 'app_config.dart';

/// 把后端返回的相对资源路径解析为完整 URL(头像/封面等静态资源)。
///
/// 后端 avatarUrl 形如 `/static/pic/9.webp`(相对路径),Flutter 的
/// `Image.network` 需要绝对 URL。baseUrl 与 GfApiClient 一致:
/// 优先 `AppConfig.apiBaseUrl`(--dart-define 注入),为空时回落到
/// Android 模拟器默认 `10.0.2.2`。
String resolveApiAssetUrl(String path) {
  if (path.isEmpty) return path;
  if (path.startsWith('http://') || path.startsWith('https://')) return path;
  final String base = AppConfig.apiBaseUrl.isNotEmpty
      ? AppConfig.apiBaseUrl
      : GfApiClient.defaultBaseUrl;
  return path.startsWith('/') ? '$base$path' : '$base/$path';
}
