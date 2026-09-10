import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/src/asset_url.dart';

void main() {
  test('相对上传资源路径解析到 API host', () {
    expect(
      resolveApiAssetUrl('/file/img/2026/08/example.webp'),
      '${GfApiClient.defaultBaseUrl}/file/img/2026/08/example.webp',
    );
  });

  test('绝对资源 URL 保持不变', () {
    const url = 'https://cdn.example.com/example.webp';
    expect(resolveApiAssetUrl(url), url);
  });
}
