// ignore_for_file: prefer_initializing_formals — 私有字段不能作为库外命名参数,
// 故用公开参数名 + 初始化列表(见 OidcController 构造函数)。

import 'dart:math';
import 'dart:convert';

import 'package:crypto/crypto.dart';
import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_appauth/flutter_appauth.dart';

/// One manual native-app authorization attempt.
///
/// The values are intentionally kept in memory only. The caller must retain
/// this object until the matching callback arrives and then pass the same
/// object to [OidcController.completeManualAuthorization].
class OidcAuthorizationSession {
  const OidcAuthorizationSession({
    required this.authorizationUri,
    required this.state,
    required this.nonce,
    required this.codeVerifier,
    required this.redirectUri,
    required this.provider,
  });

  final Uri authorizationUri;
  final String state;
  final String nonce;
  final String codeVerifier;
  final String redirectUri;
  final String provider;
}

/// OIDC 登录控制器:AppAuth PKCE 授权码流程 + 后端 exchange 兑换。
///
/// 流程(对齐后端 `POST /api/auth/oidc/exchange`,已核实契约):
/// 1. 通过论坛内建 OIDC Provider `/.well-known/openid-configuration` 发现端点;
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

  /// Builds the authorization request used by Android's external-browser
  /// transport. No browser or AppAuth call is made here.
  OidcAuthorizationSession beginManualAuthorization({
    required String provider,
  }) {
    _validateProvider(provider);
    final Uri issuerUri = Uri.parse(_issuer);
    if (!issuerUri.hasScheme || issuerUri.host.isEmpty) {
      throw ArgumentError.value(_issuer, 'issuer', 'must be an absolute URI');
    }

    final String codeVerifier = _randomToken(64);
    final String codeChallenge = base64Url
        .encode(sha256.convert(ascii.encode(codeVerifier)).bytes)
        .replaceAll('=', '');
    final String state = _randomToken(32);
    final String nonce = _randomToken(32);
    final String issuerPath = issuerUri.path.endsWith('/')
        ? issuerUri.path
        : '${issuerUri.path}/';
    final Uri authorizationUri = issuerUri.replace(
      path: '${issuerPath}authorize',
      queryParameters: <String, String>{
        'client_id': _clientId,
        'redirect_uri': redirectUri,
        'response_type': 'code',
        'scope': 'openid profile email',
        'state': state,
        'nonce': nonce,
        'code_challenge': codeChallenge,
        'code_challenge_method': 'S256',
        'login_hint': provider,
      },
    );

    return OidcAuthorizationSession(
      authorizationUri: authorizationUri,
      state: state,
      nonce: nonce,
      codeVerifier: codeVerifier,
      redirectUri: redirectUri,
      provider: provider,
    );
  }

  /// Completes one manual browser callback through the existing forum-session
  /// exchange. The callback is fully revalidated even if Android already
  /// filtered its intent.
  Future<bool> completeManualAuthorization(
    OidcAuthorizationSession session,
    Uri callbackUri,
  ) async {
    _busy = true;
    _error = '';
    notifyListeners();
    try {
      if (!_matchesRedirectUri(callbackUri, session.redirectUri)) {
        return _fail('OIDC callback rejected');
      }

      final Map<String, List<String>> query = callbackUri.queryParametersAll;
      final String? state = _singleQueryValue(query, 'state');
      if (state != session.state) {
        return _fail('OIDC state mismatch');
      }

      final String? callbackError = _singleQueryValue(query, 'error');
      if (callbackError != null && callbackError.isNotEmpty) {
        return _fail(
          callbackError == 'access_denied'
              ? 'OIDC authorization cancelled'
              : 'OIDC authorization error',
        );
      }

      final String? code = _singleQueryValue(query, 'code');
      if (code == null || code.isEmpty) {
        return _fail('OIDC authorization code missing');
      }

      final String token = await _auth.oidcExchange(
        code: code,
        codeVerifier: session.codeVerifier,
        nonce: session.nonce,
        redirectUri: session.redirectUri,
      );
      if (token.isEmpty) {
        return _fail('OIDC exchange failed');
      }
      await _tokenStorage.write(token);
      _authenticated = true;
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      _error = e.messageKey;
      notifyListeners();
      return false;
    } catch (_) {
      return _fail('OIDC exchange failed');
    } finally {
      _busy = false;
      notifyListeners();
    }
  }

  /// Records a transport failure without exposing callback or token data.
  void setExternalError(String message) {
    _error = message;
    notifyListeners();
  }

  /// 发起 OIDC 登录:授权 → 后端兑换 → 存 token。
  Future<bool> login({String? provider}) async {
    if (provider != null) _validateProvider(provider);
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
          additionalParameters: provider == null
              ? null
              : {'login_hint': provider},
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
    } on ApiException catch (e) {
      // 后端已应答的业务失败(HTTP 2xx envelope 或 4xx/5xx ApiFailure):
      // 展示稳定的 messageCode 便于上层本地化,不再吞成笼统的"OIDC login failed"。
      _error = e.messageKey;
      notifyListeners();
      return false;
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

  static void _validateProvider(String provider) {
    if (provider != 'google' && provider != 'github' && provider != 'tongji') {
      throw ArgumentError.value(provider, 'provider');
    }
  }

  static String? _singleQueryValue(
    Map<String, List<String>> query,
    String key,
  ) {
    final List<String>? values = query[key];
    return values?.length == 1 ? values!.single : null;
  }

  static bool _matchesRedirectUri(Uri callbackUri, String redirectUri) {
    final Uri expected = Uri.parse(redirectUri);
    return callbackUri.scheme == expected.scheme &&
        callbackUri.host == expected.host &&
        callbackUri.port == expected.port &&
        callbackUri.path == expected.path &&
        callbackUri.userInfo.isEmpty &&
        callbackUri.fragment.isEmpty;
  }

  bool _fail(String message) {
    _error = message;
    notifyListeners();
    return false;
  }
}
