import '../../gen/push_device.dart';
import '../gf_api_client.dart';

/// 原生推送设备注册（push/config、push/device/register|unregister）。
///
/// 均为登录态操作（Bearer）；服务端未配置通道时由 push/config.native 门控。
class PushRepository {
  PushRepository(this._client) : _sessionToken = null;
  PushRepository._session(this._client, this._sessionToken);

  final String? _sessionToken;
  Map<String, dynamic>? get _headers =>
      _sessionToken == null ? null : {'Authorization': 'Bearer $_sessionToken'};

  /// Freeze ownership for registration and its eventual logout cleanup.
  Future<PushRepository> forSession() async {
    final token = _sessionToken ?? await _client.tokenStorage.read();
    if (token == null || token.isEmpty) throw StateError('No session');
    return PushRepository._session(_client, token);
  }

  final GfApiClient _client;

  /// 通道状态（web push + native 门控）。
  Future<PushConfigPayload> config() => _client.get<PushConfigPayload>(
    '/api/forum/push/config',
    headers: _headers,
    parser: (json) =>
        PushConfigPayload.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// 注册/更新当前设备（按 user_id + token upsert；token 轮换时重调）。
  Future<bool> registerDevice({
    required String platform,
    required String token,
    String? provider,
  }) async {
    await _client.post<Object?>(
      '/api/forum/push/device/register',
      headers: _headers,
      body: RegisterPushDeviceInput(
        platform: platform,
        token: token,
        provider: provider,
      ).toJson(),
    );
    return true;
  }

  /// 注销当前用户的指定设备（幂等；登出时调用）。
  Future<bool> unregisterDevice({required String token}) async {
    await _client.post<Object?>(
      '/api/forum/push/device/unregister',
      headers: _headers,
      body: UnregisterPushDeviceInput(token: token).toJson(),
    );
    return true;
  }
}
