import 'dart:async';

import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:shared_preferences/shared_preferences.dart';
import 'offline/drift_cache.dart';
import 'app_config.dart';

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

/// 离线缓存写入世代。
///
/// 会话边界(401 会话失效、登出、进入登录页)自增一次;页面在网络响应返回后
/// 校验世代,不一致说明会话已切换,必须丢弃 setState 与缓存写入,防止旧会话
/// 在途响应把上一账号数据写回刚清空的离线库(跨账号数据泄漏)。
class OfflineCacheEpoch extends Notifier<int> {
  @override
  int build() => 0;

  /// 使所有在途缓存写入失效(自增世代)。
  void invalidate() => state++;
}

final offlineCacheEpochProvider = NotifierProvider<OfflineCacheEpoch, int>(
  OfflineCacheEpoch.new,
);

/// 跨重启"待清缓存"门禁标记键(SharedPreferences,与 token/离线库独立)。
const String pendingCacheClearKey = 'yourtj.auth.pendingCacheClear';

/// 读取持久化的待清缓存标记(启动门禁用,不依赖 Riverpod 状态)。
Future<bool> readPendingCacheClearFlag() async {
  final SharedPreferences prefs = await SharedPreferences.getInstance();
  return prefs.getBool(pendingCacheClearKey) ?? false;
}

/// 跨重启的会话缓存边界门禁。
///
/// 缓存清理失败且令牌可能残留时置位;登录页缓存清理成功后清除。
/// 标记存在时 [appRouter] 强制重定向到登录页:即使进程被杀后重启,
/// 残留的令牌与未清空的旧账号离线缓存也无法进入 shell 展示
/// (跨账号数据泄漏的最后防线)。
class PendingCacheClear extends Notifier<bool> {
  @override
  bool build() => false;

  /// 置位门禁(缓存清理失败,令牌可能残留)。
  Future<void> set() async {
    final SharedPreferences prefs = await SharedPreferences.getInstance();
    await prefs.setBool(pendingCacheClearKey, true);
    state = true;
  }

  /// 清除门禁(缓存清理成功)。
  Future<void> clear() async {
    final SharedPreferences prefs = await SharedPreferences.getInstance();
    await prefs.remove(pendingCacheClearKey);
    state = false;
  }
}

final pendingCacheClearProvider = NotifierProvider<PendingCacheClear, bool>(
  PendingCacheClear.new,
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

/// 清除全部离线缓存(话题 + 会话 + 消息)。
///
/// 登出、401 会话失效、重新登录(进入登录页)时必须清空,否则同一设备上
/// 下一账号可读到上一账号缓存的私信/话题,造成跨账号数据泄漏。
Future<void> clearOfflineCache(
  OfflineTopicCache topicCache,
  OfflineChatCache chatCache,
) async {
  await topicCache.clear();
  await chatCache.clear();
}

/// 尽力清除离线缓存,失败静默(清理失败不阻塞登出/会话失效流程,
/// 下次登出或进入登录页会重试)。
Future<void> clearOfflineCacheQuietly(
  OfflineTopicCache topicCache,
  OfflineChatCache chatCache,
) async {
  try {
    await clearOfflineCache(topicCache, chatCache);
  } catch (_) {
    // 缓存不可用时忽略。
  }
}

final apiClientProvider = Provider<GfApiClient>((ref) {
  final storage = ref.watch(tokenStorageProvider);
  return GfApiClient(
    dio: ref.watch(dioProvider),
    tokenStorage: storage,
    // --dart-define=YOURTJ_API_BASE_URL 注入;为空时必须回落到
    // GfApiClient.defaultBaseUrl(Android 模拟器 10.0.2.2),不能把
    // Dio baseUrl 显式设成 ''(否则请求打到无效 host)。
    baseUrl: AppConfig.apiBaseUrl.isNotEmpty
        ? AppConfig.apiBaseUrl
        : GfApiClient.defaultBaseUrl,
    // New-Token 滑动续期:写回 tokenStorage 持久化新令牌。
    onTokenRenewed: (newToken) => storage.write(newToken),
    // 401 会话失效:清空令牌、使旧会话在途写入失效、清离线缓存并通知 UI 跳转登录页。
    onUnauthorized: () {
      storage.clear();
      ref.read(unauthorizedEventsProvider.notifier).trigger();
      // 先自增世代,让已发出的旧会话请求在返回后丢弃 setState 与缓存写入;
      // 再异步清库(清库入队晚于已在途写入的提交,不产生交错)。
      ref.read(offlineCacheEpochProvider.notifier).invalidate();
      unawaited(
        clearOfflineCacheQuietly(
          ref.read(offlineTopicCacheProvider),
          ref.read(offlineChatCacheProvider),
        ),
      );
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
