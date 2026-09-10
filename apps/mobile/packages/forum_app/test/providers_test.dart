import 'dart:async';

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

class _GatedClearTokenStorage extends _MemTokenStorage {
  final clearStarted = Completer<void>();
  final allowClear = Completer<void>();

  @override
  Future<void> clear() async {
    if (!clearStarted.isCompleted) clearStarted.complete();
    await allowClear.future;
    await super.clear();
  }
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

    test('会话边界后旧 client 的 New-Token 续期被丢弃', () async {
      final storage = _MemTokenStorage();
      final container = ProviderContainer(
        overrides: [tokenStorageProvider.overrideWithValue(storage)],
      );
      addTearDown(container.dispose);
      await storage.write('old-token');

      final client = container.read(apiClientProvider);
      // 会话边界(进入登录页/登出)使旧 client 捕获的世代失效。
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      // 旧会话在途请求此时才带着 New-Token 返回,必须丢弃,否则会覆盖
      // 新登录写入的令牌。
      await client.onTokenRenewed?.call('stale-renewal');

      expect(
        await storage.read(),
        'old-token',
        reason: '过期 client 的续期不得覆盖当前会话令牌',
      );
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

    test('onUnauthorized 完成 token/cache 清理后才触发事件', () async {
      final storage = _GatedClearTokenStorage();
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          offlineTopicCacheProvider.overrideWithValue(_MemOfflineCache()),
          offlineChatCacheProvider.overrideWithValue(_MemOfflineCache()),
        ],
      );
      addTearDown(container.dispose);
      await storage.write('expired-token');
      expect(container.read(unauthorizedEventsProvider), 0);

      final client = container.read(apiClientProvider);
      client.onUnauthorized?.call();
      await storage.clearStarted.future;

      expect(await storage.read(), 'expired-token');
      expect(
        container.read(unauthorizedEventsProvider),
        0,
        reason: 'token clear 未完成前不得放行 UI 跳转并开始新登录',
      );
      expect(container.read(offlineCacheEpochProvider), 1);

      storage.allowClear.complete();
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);

      expect(await storage.read(), isNull);
      expect(container.read(unauthorizedEventsProvider), 1);
      final topicCache = container.read(offlineTopicCacheProvider);
      final chatCache = container.read(offlineChatCacheProvider);
      expect((topicCache as _MemOfflineCache).clears, 1);
      expect((chatCache as _MemOfflineCache).clears, 1);
    });

    test('重复 401 只清理并通知一次', () async {
      final storage = _GatedClearTokenStorage();
      final topicCache = _MemOfflineCache();
      final chatCache = _MemOfflineCache();
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          offlineTopicCacheProvider.overrideWithValue(topicCache),
          offlineChatCacheProvider.overrideWithValue(chatCache),
        ],
      );
      addTearDown(container.dispose);
      await storage.write('expired-token');

      final client = container.read(apiClientProvider);
      client.onUnauthorized?.call();
      await storage.clearStarted.future;
      storage.allowClear.complete();
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);

      expect(container.read(unauthorizedEventsProvider), 1);
      expect(topicCache.clears, 1);
      expect(chatCache.clears, 1);

      // 同一旧 client 的第二个 401:世代已变化,必须被忽略。
      client.onUnauthorized?.call();
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);
      expect(container.read(unauthorizedEventsProvider), 1);
      expect(topicCache.clears, 1);
      expect(chatCache.clears, 1);
    });

    test('会话边界后重建的 client 恢复续期与 401 处理', () async {
      final storage = _GatedClearTokenStorage();
      final topicCache = _MemOfflineCache();
      final chatCache = _MemOfflineCache();
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(storage),
          offlineTopicCacheProvider.overrideWithValue(topicCache),
          offlineChatCacheProvider.overrideWithValue(chatCache),
        ],
      );
      addTearDown(container.dispose);
      await storage.write('old-token');

      final stale = container.read(apiClientProvider);
      // 会话边界(进入登录页/登出/401)使旧 client 捕获的世代失效。
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await stale.onTokenRenewed?.call('stale-renewal');
      stale.onUnauthorized?.call();
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);
      expect(
        await storage.read(),
        'old-token',
        reason: '旧 client 在边界后必须忽略续期与 401',
      );
      expect(container.read(unauthorizedEventsProvider), 0);

      // 登录提交成功后重建 provider(生产在 _finishAuthentication 中 invalidate)。
      container.invalidate(apiClientProvider);
      final fresh = container.read(apiClientProvider);
      expect(fresh, isNot(same(stale)), reason: '登录后必须重建主 API client');

      // 新 client 恢复滑动续期:New-Token 写回 tokenStorage。
      await fresh.onTokenRenewed?.call('fresh-token');
      expect(await storage.read(), 'fresh-token');

      // 新 client 恢复 401 处理:清除 token/cache 并通知 UI 跳转登录页。
      fresh.onUnauthorized?.call();
      await storage.clearStarted.future;
      storage.allowClear.complete();
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);
      expect(await storage.read(), isNull);
      expect(container.read(unauthorizedEventsProvider), 1);
      expect(topicCache.clears, 1);
      expect(chatCache.clears, 1);
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
