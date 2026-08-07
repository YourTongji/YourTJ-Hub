import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'offline/drift_cache.dart';

/// 会话令牌存储:生产用 auth 包 SecureTokenStorage(flutter_secure_storage)。
/// 测试可通过 override 注入内存实现。
final tokenStorageProvider = Provider<TokenStorage>(
  (ref) => SecureTokenStorage(),
);

/// 会话失效事件计数(401):每次未授权响应 +1。
/// GfShell 监听后清会话并跳转登录页。
class UnauthorizedNotifier extends Notifier<int> {
  @override
  int build() => 0;

  void trigger() => state++;
}

final unauthorizedEventsProvider = NotifierProvider<UnauthorizedNotifier, int>(
  UnauthorizedNotifier.new,
);

/// Dio 实例(测试可 override 注入 mock adapter)。
final dioProvider = Provider<Dio>((ref) => Dio());

/// drift 数据库单例(话题 + IM 会话缓存共用)。
final offlineDatabaseProvider = Provider<AppDatabase>((ref) {
  final db = openDatabase();
  ref.onDispose(db.close);
  return db;
});

/// 已浏览话题离线缓存(生产为 drift;测试 override 为 no-op)。
final offlineTopicCacheProvider = Provider<OfflineTopicCache>((ref) {
  return DriftOfflineCache(ref.watch(offlineDatabaseProvider));
});

/// IM 会话离线缓存(与话题缓存共用同一 drift 数据库)。
final offlineChatCacheProvider = Provider<OfflineChatCache>((ref) {
  return DriftOfflineCache(ref.watch(offlineDatabaseProvider));
});

final apiClientProvider = Provider<GfApiClient>((ref) {
  final storage = ref.watch(tokenStorageProvider);
  return GfApiClient(
    dio: ref.watch(dioProvider),
    tokenStorage: storage,
    // New-Token 滑动续期:写回 tokenStorage 持久化新令牌。
    onTokenRenewed: (newToken) => storage.write(newToken),
    // 401 会话失效:清空令牌并通知 UI 跳转登录页。
    onUnauthorized: () {
      storage.clear();
      ref.read(unauthorizedEventsProvider.notifier).trigger();
    },
  );
});

final pageRepositoryProvider = Provider<PageRepository>((ref) {
  return PageRepository(ref.watch(apiClientProvider));
});

final topicRepositoryProvider = Provider<TopicRepository>((ref) {
  return TopicRepository(ref.watch(apiClientProvider));
});

final postRepositoryProvider = Provider<PostRepository>((ref) {
  return PostRepository(ref.watch(apiClientProvider));
});

final userRepositoryProvider = Provider<UserRepository>((ref) {
  return UserRepository(ref.watch(apiClientProvider));
});

final notificationRepositoryProvider = Provider<NotificationRepository>((ref) {
  return NotificationRepository(ref.watch(apiClientProvider));
});

final chatRepositoryProvider = Provider<ChatRepository>((ref) {
  return ChatRepository(ref.watch(apiClientProvider));
});

final fileRepositoryProvider = Provider<FileRepository>((ref) {
  return FileRepository(ref.watch(apiClientProvider));
});

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository(ref.watch(apiClientProvider));
});
