import '../../gen/auth.dart';
import '../gf_api_client.dart';

/// 认证相关接口:登录公钥、登录、注册、验证码、找回/重置密码、OIDC 交换、TOTP 校验。
///
/// 注意:登录密码的 RSA-OAEP-256 加密由 auth 包负责,
/// 本层只透传加密后的密文;蜜罐字段(website)在此代码路径中绝不出现。
class AuthRepository {
  AuthRepository(this._client);

  final GfApiClient _client;

  Future<CaptchaPayload> getCaptcha() {
    return _client.get<CaptchaPayload>(
      '/api/get-captcha',
      parser: (json) => CaptchaPayload.fromJson(json as Map<String, dynamic>),
    );
  }

  Future<LoginPublicKeyPayload> getLoginPublicKey() {
    return _client.get<LoginPublicKeyPayload>(
      '/api/login-public-key',
      parser: (json) => LoginPublicKeyPayload.fromJson(json as Map<String, dynamic>),
    );
  }

  /// 登录。成功返回 twoFactorRequired;result 可能是字符串(普通成功)或
  /// `{twoFactorRequired, message}` 映射(TOTP 挑战)。
  Future<LoginResult> login({
    required String username,
    required String encryptedPassword,
    String? captchaId,
    String? captchaCode,
  }) {
    return _client.post<LoginResult>(
      '/api/login',
      body: {
        'username': username,
        'encryptedPassword': encryptedPassword,
        if (captchaId != null && captchaId.isNotEmpty) 'captchaId': captchaId,
        if (captchaCode != null && captchaCode.isNotEmpty) 'captchaCode': captchaCode,
      },
      parser: (json) => json is Map<String, dynamic>
          ? LoginResult.fromJson(json)
          : const LoginResult(twoFactorRequired: false),
    );
  }

  /// 注册,成功返回服务端消息文案。
  Future<String> register({
    required String username,
    required String email,
    required String password,
    String? captchaId,
    String? captchaCode,
    String? locale,
  }) {
    return _client.post<String>(
      '/api/register',
      body: {
        'userName': username,
        'email': email,
        'passWord': password,
        if (captchaId != null && captchaId.isNotEmpty) 'captchaId': captchaId,
        if (captchaCode != null && captchaCode.isNotEmpty) 'captchaCode': captchaCode,
        if (locale != null && locale.isNotEmpty) 'locale': locale,
      },
    );
  }

  /// 找回密码,成功返回服务端消息文案。
  Future<String> forgotPassword({
    required String email,
    String? captchaId,
    String? captchaCode,
  }) {
    return _client.post<String>(
      '/api/forgot-password',
      body: {
        'email': email,
        if (captchaId != null && captchaId.isNotEmpty) 'captchaId': captchaId,
        if (captchaCode != null && captchaCode.isNotEmpty) 'captchaCode': captchaCode,
      },
    );
  }

  /// 重置密码,成功返回服务端消息文案。
  Future<String> resetPassword({
    required String token,
    required String newPassword,
  }) {
    return _client.post<String>(
      '/api/reset-password',
      body: {'token': token, 'newPassword': newPassword},
    );
  }

  /// 移动端 OIDC 登录:用 Casdoor 授权码 + PKCE verifier 交换会话令牌。
  Future<String> oidcExchange({
    required String code,
    required String codeVerifier,
    required String redirectUri,
  }) {
    return _client.post<String>(
      '/api/auth/oidc/exchange',
      body: {'code': code, 'codeVerifier': codeVerifier, 'redirectUri': redirectUri},
      parser: (json) {
        if (json is String) return json;
        return (json as Map<String, dynamic>)['token'] as String? ?? '';
      },
    );
  }

  /// 两步验证:登录返回 twoFactorRequired 后,用 challenge token 校验验证码。
  Future<bool> totpVerify({required String code, String? recoveryCode}) async {
    await _client.post<Object?>(
      '/api/auth/totp/verify',
      body: {
        'code': code,
        if (recoveryCode != null && recoveryCode.isNotEmpty) 'recoveryCode': recoveryCode,
      },
    );
    return true;
  }
}
