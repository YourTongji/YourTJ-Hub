/// 会话令牌的持久化抽象。
///
/// 本包不依赖任何具体存储实现;auth 包用 flutter_secure_storage 实现此接口。
abstract interface class TokenStorage {
  /// 读取当前令牌;无令牌时返回 null。
  Future<String?> read();

  /// 保存新令牌(含滑动续期得到的 New-Token)。
  Future<void> write(String token);

  /// 清除令牌(登出或 401 时)。
  Future<void> clear();
}
