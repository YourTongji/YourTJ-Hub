import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/app_config.dart';

/// 内存 TokenStorage 测试替身。
class _MemTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

void main() {
  group('tokenStorageProvider', () {
    test('生产实现为 SecureTokenStorage(flutter_secure_storage)', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      expect(container.read(tokenStorageProvider), isA<SecureTokenStorage>());
    });
  });

  group('apiClientProvider 会话回调接线', () {
    test('onTokenRenewed 将 New-Token 写回 tokenStorage', () async {
      final container = ProviderContainer(
        overrides: [tokenStorageProvider.overrideWithValue(_MemTokenStorage())],
      );
      addTearDown(container.dispose);
      final storage = container.read(tokenStorageProvider);
      await storage.write('old-token');

      final client = container.read(apiClientProvider);
      // 模拟 GfApiClient 收到 New-Token 响应头后触发的回调。
      await client.onTokenRenewed?.call('fresh-token');

      expect(await storage.read(), 'fresh-token');
    });

    test('apiBaseUrl 为空(未注入 dart-define)时回落到平台默认', () {
      final container = ProviderContainer(
        overrides: [tokenStorageProvider.overrideWithValue(_MemTokenStorage())],
      );
      addTearDown(container.dispose);
      // AppConfig.apiBaseUrl 默认 ''(String.fromEnvironment 无注入)。
      expect(AppConfig.apiBaseUrl, isEmpty);
      final client = container.read(apiClientProvider);
      expect(client.baseUrl, GfApiClient.defaultBaseUrl);
    });

    test('onUnauthorized 清空 tokenStorage 并触发 unauthorizedEvents', () async {
      final container = ProviderContainer(
        overrides: [tokenStorageProvider.overrideWithValue(_MemTokenStorage())],
      );
      addTearDown(container.dispose);
      final storage = container.read(tokenStorageProvider);
      await storage.write('expired-token');
      expect(container.read(unauthorizedEventsProvider), 0);

      final client = container.read(apiClientProvider);
      client.onUnauthorized?.call();

      expect(await storage.read(), isNull);
      expect(container.read(unauthorizedEventsProvider), 1);
    });
  });
}
