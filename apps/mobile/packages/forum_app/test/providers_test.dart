import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/src/offline/drift_cache.dart';
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
        overrides: [
          tokenStorageProvider.overrideWithValue(_MemTokenStorage()),
          offlineTopicCacheProvider.overrideWithValue(_MemOfflineCache()),
          offlineChatCacheProvider.overrideWithValue(_MemOfflineCache()),
        ],
      );
      addTearDown(container.dispose);
      final storage = container.read(tokenStorageProvider);
      await storage.write('expired-token');
      expect(container.read(unauthorizedEventsProvider), 0);

      final client = container.read(apiClientProvider);
      client.onUnauthorized?.call();

      expect(await storage.read(), isNull);
      expect(container.read(unauthorizedEventsProvider), 1);
      expect(
        container.read(offlineCacheEpochProvider),
        1,
        reason: '401 应自增缓存世代,使旧会话在途写入失效',
      );
      // 401 会话失效应清空离线缓存(防跨账号数据泄漏)。
      await Future<void>.delayed(Duration.zero);
      final topicCache = container.read(offlineTopicCacheProvider);
      final chatCache = container.read(offlineChatCacheProvider);
      expect((topicCache as _MemOfflineCache).clears, 1);
      expect((chatCache as _MemOfflineCache).clears, 1);
    });
  });
}

/// 内存离线缓存,记录 clear 调用。
class _MemOfflineCache implements OfflineTopicCache, OfflineChatCache {
  int clears = 0;

  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {}

  @override
  Future<PagePayload?> get(int topicId) async => null;

  @override
  Future<void> putConversations(List<ChatItemPayload> conversations) async {}

  @override
  Future<List<ChatItemPayload>> getConversations() async => const [];

  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {}

  @override
  Future<List<ChatMessagePayload>> getMessages(int convId) async => const [];

  @override
  Future<void> clear() async {
    clears++;
  }

  @override
  Future<void> close() async {}
}
