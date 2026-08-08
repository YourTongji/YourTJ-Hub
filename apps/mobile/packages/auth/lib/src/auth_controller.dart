// ignore_for_file: prefer_initializing_formals — 私有字段不能作为库外命名参数,
// 故用公开参数名 + 初始化列表(见 AuthController 构造函数)。

import 'package:core/core.dart';
import 'package:flutter/foundation.dart';

import 'rsa/rsa_encryptor.dart';

/// 登录状态机阶段。
enum LoginPhase {
  /// 未开始。
  idle,

  /// 需要图片验证码(登录接口返回 captchaRequired)。
  needsCaptcha,

  /// 需要 TOTP 二次验证(登录接口返回 twoFactorRequired)。
  needsTotp,

  /// 已登录。
  authenticated,

  /// 登录失败(携带错误消息)。
  failed,
}

/// 登录流程控制器:公钥加密 → 密码登录 → (验证码/TOTP)→ 会话持久化。
///
/// 对齐后端链路(已核实):
/// 1. `GET /api/login-public-key` → RSA-OAEP-256 公钥 + serverTs;
/// 2. `POST /api/login` body `{username, encryptedPassword, captchaId?, captchaCode?}`;
/// 3. 响应 `twoFactorRequired=true` → `POST /api/auth/totp/verify`(challenge token
///    通过 GfApiClient 的 New-Token 头自动注入,移动端无需 cookie 管理);
/// 4. 成功 token 写入 TokenStorage;滑动续期由 GfApiClient 回调处理。
class AuthController extends ChangeNotifier {
  AuthController({
    required AuthRepository authRepository,
    required GfApiClient apiClient,
    required TokenStorage tokenStorage,
    RsaEncryptor? rsaEncryptor,
  }) : _auth = authRepository,
       _client = apiClient,
       _tokenStorage = tokenStorage,
       _rsa = rsaEncryptor ?? RsaEncryptor();

  final AuthRepository _auth;
  final GfApiClient _client;
  final TokenStorage _tokenStorage;
  final RsaEncryptor _rsa;

  LoginPhase _phase = LoginPhase.idle;
  String _error = '';

  /// 当前验证码挑战(需要时加载)。
  CaptchaPayload? _captcha;

  /// 当前登录用户名(跨 TOTP 阶段保留)。
  String _pendingUsername = '';

  /// 登录是否进行中(防重复提交)。
  bool _busy = false;

  LoginPhase get phase => _phase;
  String get error => _error;
  CaptchaPayload? get captcha => _captcha;
  bool get busy => _busy;
  bool get isAuthenticated => _phase == LoginPhase.authenticated;

  /// 加载验证码图片(needsCaptcha 阶段)。
  Future<void> loadCaptcha() async {
    try {
      _captcha = await _auth.getCaptcha();
      notifyListeners();
    } catch (_) {
      _error = 'Failed to load captcha';
      _phase = LoginPhase.failed;
      notifyListeners();
    }
  }

  /// 密码登录。传 captchaId/captchaCode 时携带验证码。
  Future<void> login({
    required String username,
    required String password,
    String? captchaId,
    String? captchaCode,
  }) async {
    _busy = true;
    _error = '';
    notifyListeners();
    try {
      final key = await _auth.getLoginPublicKey();
      final encrypted = _rsa.encryptPassword(
        publicKeyPem: key.publicKey,
        password: password,
        serverTs: key.serverTs,
      );
      final result = await _auth.login(
        username: username,
        encryptedPassword: encrypted,
        captchaId: captchaId,
        captchaCode: captchaCode,
      );
      _pendingUsername = username;
      if (result.twoFactorRequired) {
        _phase = LoginPhase.needsTotp;
      } else {
        await _completeLogin(username: username);
      }
    } on UnauthorizedException {
      _phase = LoginPhase.failed;
      _error = 'Invalid username or password';
    } on ApiException catch (e) {
      // _mapAuthError 内部可能把阶段切换为 needsCaptcha。
      _error = _mapAuthError(e, fallback: 'Unable to sign in, please retry');
    } catch (e) {
      _phase = LoginPhase.failed;
      _error = 'Login failed: $e';
    } finally {
      _busy = false;
      notifyListeners();
    }
  }

  /// TOTP 二次验证。
  Future<void> submitTotp(String code) async {
    _busy = true;
    _error = '';
    notifyListeners();
    try {
      final ok = await _auth.totpVerify(code: code);
      if (ok) {
        await _completeLogin(username: _pendingUsername);
      } else {
        _phase = LoginPhase.failed;
        _error = 'Invalid two-factor code';
      }
    } catch (e) {
      _phase = LoginPhase.failed;
      _error = 'TOTP verification failed: $e';
    } finally {
      _busy = false;
      notifyListeners();
    }
  }

  /// 注册。
  Future<void> register({
    required String username,
    required String email,
    required String password,
    String? captchaId,
    String? captchaCode,
  }) async {
    _busy = true;
    _error = '';
    notifyListeners();
    try {
      await _auth.register(
        username: username,
        email: email,
        password: password,
        captchaId: captchaId,
        captchaCode: captchaCode,
      );
      _phase = LoginPhase.idle;
    } on ApiException catch (e) {
      _error = _mapAuthError(e, fallback: 'Unable to register, please retry');
    } catch (e) {
      _phase = LoginPhase.failed;
      _error = 'Registration failed: $e';
    } finally {
      _busy = false;
      notifyListeners();
    }
  }

  /// 找回密码:提交邮箱,服务端发送重置邮件。
  Future<void> forgotPassword({
    required String email,
    String? captchaId,
    String? captchaCode,
  }) async {
    _busy = true;
    _error = '';
    notifyListeners();
    try {
      await _auth.forgotPassword(
        email: email,
        captchaId: captchaId,
        captchaCode: captchaCode,
      );
      _phase = LoginPhase.idle;
    } on ApiException catch (e) {
      _error = _mapAuthError(
        e,
        fallback: 'Unable to request a password reset, please retry',
      );
    } catch (e) {
      _phase = LoginPhase.failed;
      _error = 'Password reset failed: $e';
    } finally {
      _busy = false;
      notifyListeners();
    }
  }

  /// 启动时从 TokenStorage 恢复会话。
  Future<bool> init() async {
    final token = await _tokenStorage.read();
    if (token != null && token.isNotEmpty) {
      _phase = LoginPhase.authenticated;
      notifyListeners();
      return true;
    }
    return false;
  }

  /// 登出:尽力通知后端吊销会话 + 清本地。
  Future<void> logout() async {
    try {
      // 尽力调用;网络失败不阻塞登出。路径必须与后端
      // `POST /api/logout`(route4api.go baseApi.POST("logout"))一致,
      // 否则服务端会话/jti 不会吊销。
      await _client.post<Object?>('/api/logout');
    } catch (_) {
      // 忽略。
    }
    await _tokenStorage.clear();
    _phase = LoginPhase.idle;
    _pendingUsername = '';
    notifyListeners();
  }

  Future<void> _completeLogin({required String username}) async {
    final token = await _tokenStorage.read();
    if (token == null || token.isEmpty) {
      // GfApiClient 会等待 New-Token 回调完成后才返回；这里仍保留
      // 协议兜底，防止服务端成功响应遗漏会话令牌。
      _phase = LoginPhase.failed;
      _error = 'Session token missing';
      return;
    }
    _phase = LoginPhase.authenticated;
    _pendingUsername = username;
  }

  String _mapAuthError(ApiException e, {required String fallback}) {
    // 用后端原始 messageCode 匹配(见 message_code.go),而非带
    // `server.` 前缀的 messageKey；认证页尚无 server.* l10n 表，
    // 所以未知错误也返回面向用户的操作级 fallback，不泄露 raw key。
    switch (e.messageCode) {
      case 'common.captchaRequired':
        _phase = LoginPhase.needsCaptcha;
        return 'Captcha required';
      case 'auth.captcha.invalid':
        _phase = LoginPhase.needsCaptcha;
        return 'Invalid or expired captcha';
      case 'auth.login.invalidRequest':
        _phase = LoginPhase.failed;
        return 'Invalid login request, please retry';
      case 'auth.password.invalidFormat':
      case 'auth.credentials.invalid':
        _phase = LoginPhase.failed;
        return 'Invalid username or password';
      case 'auth.account.frozen':
        _phase = LoginPhase.failed;
        return 'Account is frozen';
      case 'auth.email.unverified':
        _phase = LoginPhase.failed;
        return 'Please verify your email before signing in';
      case 'auth.signupDisabled':
        _phase = LoginPhase.failed;
        return 'Registration is currently disabled';
      case 'auth.username.invalid':
        _phase = LoginPhase.failed;
        return 'Username format is invalid';
      case 'auth.username.exists':
        _phase = LoginPhase.failed;
        return 'Username is already in use';
      case 'auth.email.exists':
        _phase = LoginPhase.failed;
        return 'Email is already in use';
      case 'auth.emailDomain.invalid':
        _phase = LoginPhase.failed;
        return 'Email address is invalid';
      case 'auth.emailDomain.notAllowed':
        _phase = LoginPhase.failed;
        return 'This email domain is not allowed';
      case 'auth.password.tooShort':
        _phase = LoginPhase.failed;
        return 'Password is too short';
      case 'auth.password.tooLong':
        _phase = LoginPhase.failed;
        return 'Password is too long';
      case 'auth.password.needsLetterNumber':
        _phase = LoginPhase.failed;
        return 'Password must contain both letters and numbers';
      case 'auth.passwordReset.mailFailed':
        _phase = LoginPhase.failed;
        return 'Unable to send the password reset email';
      default:
        _phase = LoginPhase.failed;
        return fallback;
    }
  }
}
