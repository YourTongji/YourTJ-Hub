import '../../gen/auth.dart';
import '../../gen/user.dart';
import '../gf_api_client.dart';

/// 用户资料、会话、TOTP、徽章接口。
class UserRepository {
  UserRepository(this._client);

  final GfApiClient _client;

  Future<UserCardPayload> getUserCard(int userId) {
    return _client.get<UserCardPayload>(
      '/api/user-card',
      queryParameters: {'userId': userId},
      parser: (json) => UserCardPayload.fromJson(json as Map<String, dynamic>),
    );
  }

  /// 保存用户资料。注意:后端支持 website 蜜罐字段,移动端永不发送。
  Future<bool> saveUserInfo({
    required String nickname,
    required String bio,
    required String signature,
    required String websiteName,
    String? locale,
    Map<String, ExternalLinkPayload>? externalInformation,
  }) async {
    await _client.post<Object?>(
      '/api/set-user-info',
      body: {
        'nickname': nickname,
        'bio': bio,
        'signature': signature,
        'websiteName': websiteName,
        if (locale != null && locale.isNotEmpty) 'locale': locale,
        if (externalInformation != null)
          'externalInformation': externalInformation.map(
            (key, value) => MapEntry(key, {'link': value.link}),
          ),
      },
    );
    return true;
  }

  /// 修改绑定邮箱(后端 set-user-email,改后需重新验证)。
  Future<bool> setUserEmail(String email) async {
    await _client.post<Object?>('/api/set-user-email', body: {'email': email});
    return true;
  }

  Future<bool> saveUserProfileCover(String profileCoverUrl) async {
    await _client.post<Object?>(
      '/api/set-user-profile-cover',
      body: {'profileCoverUrl': profileCoverUrl},
    );
    return true;
  }

  Future<bool> savePresetAvatar(String avatarUrl) async {
    await _client.post<Object?>(
      '/api/set-preset-avatar',
      body: {'avatarUrl': avatarUrl},
    );
    return true;
  }

  Future<bool> wearBadge(String badgeCode) async {
    await _client.post<Object?>(
      '/api/wear-badge',
      body: {'badgeCode': badgeCode},
    );
    return true;
  }

  Future<bool> changePassword({
    required String oldPassword,
    required String newPassword,
  }) async {
    await _client.post<Object?>(
      '/api/change-password',
      body: {'oldPassword': oldPassword, 'newPassword': newPassword},
    );
    return true;
  }

  Future<List<UserSessionPayload>> listSessions() {
    return _client.get<List<UserSessionPayload>>(
      '/api/user/sessions',
      parser: (json) => (json as List<dynamic>)
          .map((e) => UserSessionPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<bool> revokeSession(int id) async {
    await _client.post<Object?>('/api/user/sessions/revoke', body: {'id': id});
    return true;
  }

  Future<bool> revokeAllSessions() async {
    await _client.post<Object?>('/api/user/sessions/revoke-all');
    return true;
  }

  Future<TotpSetupPayload> getTotpSetup({required String password}) {
    return _client.post<TotpSetupPayload>(
      '/api/user/totp/setup',
      body: {'password': password},
      parser: (json) => TotpSetupPayload.fromJson(json as Map<String, dynamic>),
    );
  }

  Future<TotpEnablePayload> enableTotp({required String code}) {
    return _client.post<TotpEnablePayload>(
      '/api/user/totp/enable',
      body: {'code': code},
      parser: (json) =>
          TotpEnablePayload.fromJson(json as Map<String, dynamic>),
    );
  }

  Future<bool> disableTotp({required String code}) async {
    await _client.post<Object?>('/api/user/totp/disable', body: {'code': code});
    return true;
  }

  Future<TotpStatusPayload> getTotpStatus() {
    return _client.get<TotpStatusPayload>(
      '/api/user/totp/status',
      parser: (json) =>
          TotpStatusPayload.fromJson(json as Map<String, dynamic>),
    );
  }

  Future<Map<String, OAuthBindingPayload>> getOAuthBindings() {
    return _client.get<Map<String, OAuthBindingPayload>>(
      '/api/oauth/bindings',
      parser: (json) => (json as Map<String, dynamic>).map(
        (key, value) => MapEntry(
          key,
          OAuthBindingPayload.fromJson(value as Map<String, dynamic>),
        ),
      ),
    );
  }

  Future<bool> unbindOAuth(String provider) async {
    await _client.post<Object?>('/api/auth/$provider/unbind');
    return true;
  }
}
