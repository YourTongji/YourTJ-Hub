import 'dart:async';

import 'package:auth/auth.dart';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

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

  /// 判断回调是否仍属于当前会话。
  bool isCurrent(int epoch) => state == epoch;
}

final offlineCacheEpochProvider = NotifierProvider<OfflineCacheEpoch, int>(
  OfflineCacheEpoch.new,
);

/// Dio 实例(测试可 override 注入 mock adapter)。
final dioProvider = Provider<Dio>((ref) => Dio());

/// 登录流程专用 Dio:不复用 [dioProvider],避免主客户端安装的旧会话
/// Bearer interceptor 污染 password/TOTP/OIDC 认证请求。
final authDioProvider = Provider<Dio>((ref) => Dio());

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

/// 是否持有会话令牌。
///
/// 离线回退读取前的认证门槛:无令牌(如启动时上一会话已失效但离线缓存
/// 残留)时不得读取缓存,防止未登录态渲染上一账号的私信/话题。
Future<bool> hasSessionToken(TokenStorage storage) async {
  try {
    final String? token = await storage.read();
    return token != null && token.isNotEmpty;
  } catch (_) {
    return false;
  }
}

final apiClientProvider = Provider<GfApiClient>((ref) {
  final storage = ref.watch(tokenStorageProvider);
  // 只读捕获创建时的会话世代:该 client 固定属于此会话,边界变化后其
  // 续期/401 回调一律失效。不能 watch,否则每次边界都会重建 client。
  final int sessionEpoch = ref.read(offlineCacheEpochProvider);
  final epochNotifier = ref.read(offlineCacheEpochProvider.notifier);
  final unauthorizedNotifier = ref.read(unauthorizedEventsProvider.notifier);
  return GfApiClient(
    dio: ref.watch(dioProvider),
    tokenStorage: storage,
    // --dart-define=YOURTJ_API_BASE_URL 注入;为空时必须回落到
    // GfApiClient.defaultBaseUrl(Android 模拟器 10.0.2.2),不能把
    // Dio baseUrl 显式设成 ''(否则请求打到无效 host)。
    baseUrl: AppConfig.apiBaseUrl.isNotEmpty
        ? AppConfig.apiBaseUrl
        : GfApiClient.defaultBaseUrl,
    // 旧会话请求可能在新账号登录后才带着 New-Token 返回。client 创建时
    // 捕获会话世代,边界变化后直接丢弃续期,避免覆盖新账号 token。
    onTokenRenewed: (newToken) async {
      if (!epochNotifier.isCurrent(sessionEpoch)) return;
      await storage.write(newToken);
    },
    // 仅当前会话的首个 401 可启动 teardown。同步自增世代后,同一旧
    // client 的重复 401 与迟到 New-Token 都立即失效。离线缓存按需读取,
    // 避免 client 构造时初始化 drift 数据库。
    onUnauthorized: () {
      if (!epochNotifier.isCurrent(sessionEpoch)) return;
      epochNotifier.invalidate();
      unawaited(() async {
        try {
          await storage.clear();
        } catch (_) {
          // 清理失败仍进入登录页;新登录会在缓存清理成功后覆盖旧 token。
        }
        await clearOfflineCacheQuietly(
          ref.read(offlineTopicCacheProvider),
          ref.read(offlineChatCacheProvider),
        );
        unauthorizedNotifier.trigger();
      }());
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
