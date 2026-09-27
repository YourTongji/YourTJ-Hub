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
  String? _cachedToken;
  bool _hasCachedToken = false;
  Future<String?>? _pendingRead;
  Future<void> _mutationQueue = Future<void>.value();
  int _revision = 0;

  @override
  Future<String?> read() async {
    if (_hasCachedToken) return _cachedToken;
    try {
      await _mutationQueue;
    } catch (_) {
      // A failed persistence operation still allows a fresh secure-storage read.
    }
    if (_hasCachedToken) return _cachedToken;
    final revision = _revision;
    return _pendingRead ??= _storage
        .read(key: _keyToken)
        .then((token) async {
          if (revision == _revision) {
            _cachedToken = token;
            _hasCachedToken = true;
            return token;
          }
          try {
            await _mutationQueue;
          } catch (_) {
            // A failed mutation still permits reading the persisted value.
          }
          return _hasCachedToken ? _cachedToken : null;
        })
        .whenComplete(() => _pendingRead = null);
  }

  @override
  Future<void> write(String token) {
    final revision = ++_revision;
    _cachedToken = null;
    _hasCachedToken = false;
    final operation = _mutationQueue.then((_) async {
      await _storage.write(key: _keyToken, value: token);
      if (revision == _revision) {
        _cachedToken = token;
        _hasCachedToken = true;
      }
    });
    _mutationQueue = operation.catchError((_) {});
    return operation;
  }

  @override
  Future<void> clear() {
    final revision = ++_revision;
    _cachedToken = null;
    _hasCachedToken = true;
    final operation = _mutationQueue.then((_) async {
      await _storage.delete(key: _keyToken);
      await _storage.delete(key: _keyUserId);
      await _storage.delete(key: _keyUsername);
      if (revision == _revision) {
        _cachedToken = null;
        _hasCachedToken = true;
      }
    });
    _mutationQueue = operation.catchError((_) {});
    return operation;
  }

  Future<int?> readUserId() async {
    final String? raw = await _storage.read(key: _keyUserId);
    return raw == null ? null : int.tryParse(raw);
  }

  Future<String?> readUsername() => _storage.read(key: _keyUsername);

  Future<void> writeSession({
    required int userId,
    required String username,
  }) async {
    await _storage.write(key: _keyUserId, value: '$userId');
    await _storage.write(key: _keyUsername, value: username);
  }
}
