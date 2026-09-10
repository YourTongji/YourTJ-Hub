import 'package:core/core.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// 基于 flutter_secure_storage 的 TokenStorage 实现。
///
/// 同时保存 token 与登录用户信息(轻量会话元数据),供论坛 UI 展示。
class SecureTokenStorage implements TokenStorage {
  SecureTokenStorage({FlutterSecureStorage? storage})
      : _storage = storage ?? const FlutterSecureStorage();

  static const _keyToken = 'yourtj.session.token';
  static const _keyUserId = 'yourtj.session.userId';
  static const _keyUsername = 'yourtj.session.username';

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read() => _storage.read(key: _keyToken);

  @override
  Future<void> write(String token) => _storage.write(key: _keyToken, value: token);

  @override
  Future<void> clear() async {
    await _storage.delete(key: _keyToken);
    await _storage.delete(key: _keyUserId);
    await _storage.delete(key: _keyUsername);
  }

  Future<int?> readUserId() async {
    final String? raw = await _storage.read(key: _keyUserId);
    return raw == null ? null : int.tryParse(raw);
  }

  Future<String?> readUsername() => _storage.read(key: _keyUsername);

  Future<void> writeSession({required int userId, required String username}) async {
    await _storage.write(key: _keyUserId, value: '$userId');
    await _storage.write(key: _keyUsername, value: username);
  }
}
