// ignore_for_file: prefer_initializing_formals — 私有字段不能作为库外命名参数,
// 故用公开参数名 + 初始化列表(见 OidcController 构造函数)。

import 'dart:math';

import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_appauth/flutter_appauth.dart';

/// OIDC(Casdoor)登录控制器:AppAuth PKCE 授权码流程 + 后端 exchange 兑换。
///
/// 流程(对齐后端 `POST /api/auth/oidc/exchange`,已核实契约):
/// 1. 通过 Casdoor `/.well-known/openid-configuration` 发现端点;
/// 2. AppAuth `authorize()` 发起授权(redirectUri = yourtj://callback)。
///    AppAuth 自动生成 PKCE S256 challenge/verifier 与 state;
///    响应 [AuthorizationResponse] 含 authorizationCode + codeVerifier;
/// 3. 用 code + codeVerifier 调后端 `/api/auth/oidc/exchange` 兑换论坛 JWT。
///
/// 注意:移动端**不**用 `authorizeAndExchangeCode`(那是直接向 IdP 换 token);
/// 必须把 code 交给后端兑换,forum JWT 才是会话凭证。
class OidcController extends ChangeNotifier {
  OidcController({
    required AuthRepository authRepository,
    required TokenStorage tokenStorage,
    required String issuer,
    required String clientId,
    this.redirectUri = 'yourtj://callback',
    FlutterAppAuth? appAuth,
  }) : _auth = authRepository,
       _tokenStorage = tokenStorage,
       _issuer = issuer,
       _clientId = clientId,
       _appAuth = appAuth ?? const FlutterAppAuth();

  final AuthRepository _auth;
  final TokenStorage _tokenStorage;
  final String _issuer;
  final String _clientId;
  final String redirectUri;
  final FlutterAppAuth _appAuth;

  bool _busy = false;
  String _error = '';
  bool _authenticated = false;

  bool get busy => _busy;
  String get error => _error;
  bool get isAuthenticated => _authenticated;

  /// 发起 OIDC 登录:授权 → 后端兑换 → 存 token。
  Future<bool> login() async {
    _busy = true;
    _error = '';
    notifyListeners();
    try {
      // nonce 绑定:AppAuth 把 nonce 放进授权请求,后端兑换时校验
      // id_token.nonce 与本值一致(与 web HandleCallback 相同的绑定)。
      final String nonce = _randomToken(32);
      final AuthorizationResponse response = await _appAuth.authorize(
        AuthorizationRequest(
          _clientId,
          redirectUri,
          issuer: _issuer,
          scopes: ['openid', 'profile', 'email'],
          nonce: nonce,
          allowInsecureConnections: kDebugMode,
        ),
      );

      if (response.authorizationCode == null ||
          response.authorizationCode!.isEmpty) {
        _error = 'OIDC authorization cancelled';
        notifyListeners();
        return false;
      }
      if (response.codeVerifier == null || response.codeVerifier!.isEmpty) {
        _error = 'OIDC PKCE verifier missing';
        notifyListeners();
        return false;
      }

      // 用 code + codeVerifier 调后端兑换论坛 JWT。
      final String token = await _auth.oidcExchange(
        code: response.authorizationCode!,
        codeVerifier: response.codeVerifier!,
        nonce: nonce,
        redirectUri: redirectUri,
      );
      if (token.isEmpty) {
        _error = 'OIDC exchange failed';
        notifyListeners();
        return false;
      }
      await _tokenStorage.write(token);
      _authenticated = true;
      notifyListeners();
      return true;
    } catch (e) {
      _error = 'OIDC login failed: $e';
      notifyListeners();
      return false;
    } finally {
      _busy = false;
      notifyListeners();
    }
  }

  String _randomToken(int length) {
    final Random rng = Random.secure();
    const String chars =
        'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~';
    return List.generate(
      length,
      (_) => chars[rng.nextInt(chars.length)],
    ).join();
  }
}
