import 'package:flutter_appauth/flutter_appauth.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:auth/auth.dart';
import 'package:core/core.dart';

/// 内存 TokenStorage,便于测试。
class MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

/// 可编程 FlutterAppAuth fake:拦截 authorize 调用,返回预设响应或抛错。
class FakeAppAuth extends FlutterAppAuth {
  FakeAppAuth({this.response, this.throwError});

  final AuthorizationResponse? response;
  final Object? throwError;
  AuthorizationRequest? lastRequest;
  int authorizeCalls = 0;

  @override
  Future<AuthorizationResponse> authorize(AuthorizationRequest request) async {
    authorizeCalls++;
    lastRequest = request;
    if (throwError != null) throw throwError!;
    return response ?? const AuthorizationResponse();
  }
}

/// 可编程 AuthRepository fake:记录 oidcExchange 参数,返回预设 token 或抛错。
class FakeAuthRepository implements AuthRepository {
  FakeAuthRepository({this.exchangeToken = 'forum-jwt'});

  String exchangeToken;
  Object? exchangeError;
  int exchangeCalls = 0;
  String? lastCode;
  String? lastVerifier;
  String? lastRedirectUri;

  @override
  Future<String> oidcExchange({
    required String code,
    required String codeVerifier,
    required String redirectUri,
  }) async {
    exchangeCalls++;
    lastCode = code;
    lastVerifier = codeVerifier;
    lastRedirectUri = redirectUri;
    if (exchangeError != null) throw exchangeError!;
    return exchangeToken;
  }

  @override
  Future<CaptchaPayload> getCaptcha() => throw UnimplementedError();

  @override
  Future<LoginPublicKeyPayload> getLoginPublicKey() =>
      throw UnimplementedError();

  @override
  Future<LoginResult> login({
    required String username,
    required String encryptedPassword,
    String? captchaId,
    String? captchaCode,
  }) => throw UnimplementedError();

  @override
  Future<String> register({
    required String username,
    required String email,
    required String password,
    String? captchaId,
    String? captchaCode,
    String? locale,
  }) => throw UnimplementedError();

  @override
  Future<String> forgotPassword({
    required String email,
    String? captchaId,
    String? captchaCode,
  }) => throw UnimplementedError();

  @override
  Future<String> resetPassword({
    required String token,
    required String newPassword,
  }) => throw UnimplementedError();

  @override
  Future<bool> totpVerify({required String code, String? recoveryCode}) =>
      throw UnimplementedError();
}

/// OidcController authorize→exchange 调用链测试(不依赖真实 AppAuth/网络)。
///
/// 覆盖后端 `POST /api/auth/oidc/exchange` 契约:
/// AppAuth 授权码 + PKCE verifier → 后端兑换 forum JWT → secure storage 持久化。
void main() {
  const issuer = 'http://127.0.0.1:8001';
  const clientId = 'f29f6177fac30dc47d14';
  const redirectUri = 'yourtj://callback';

  OidcController buildController({
    required TokenStorage storage,
    required FakeAppAuth appAuth,
    required FakeAuthRepository auth,
  }) {
    return OidcController(
      authRepository: auth,
      tokenStorage: storage,
      issuer: issuer,
      clientId: clientId,
      redirectUri: redirectUri,
      appAuth: appAuth,
    );
  }

  test('登录成功:authorize 携带 PKCE/issuer,exchange 后 token 持久化', () async {
    final storage = MemoryTokenStorage();
    final appAuth = FakeAppAuth(
      response: const AuthorizationResponse(
        authorizationCode: 'auth-code-1',
        codeVerifier: 'pkce-verifier-1',
      ),
    );
    final auth = FakeAuthRepository(exchangeToken: 'forum-jwt-1');
    final controller = buildController(
      storage: storage,
      appAuth: appAuth,
      auth: auth,
    );

    final ok = await controller.login();

    expect(ok, isTrue);
    expect(controller.isAuthenticated, isTrue);
    expect(controller.busy, isFalse);
    // AppAuth 请求参数正确。
    expect(appAuth.lastRequest!.clientId, clientId);
    expect(appAuth.lastRequest!.issuer, issuer);
    expect(appAuth.lastRequest!.redirectUrl, redirectUri);
    expect(appAuth.lastRequest!.scopes, ['openid', 'profile', 'email']);
    expect(appAuth.lastRequest!.nonce, isNotEmpty);
    // exchange 收到 code + verifier + redirectUri。
    expect(auth.exchangeCalls, 1);
    expect(auth.lastCode, 'auth-code-1');
    expect(auth.lastVerifier, 'pkce-verifier-1');
    expect(auth.lastRedirectUri, redirectUri);
    // token 已持久化。
    expect(await storage.read(), 'forum-jwt-1');
  });

  test('授权取消(无 authorizationCode)→ 不调 exchange,不持久化', () async {
    final storage = MemoryTokenStorage();
    final appAuth = FakeAppAuth(response: const AuthorizationResponse());
    final auth = FakeAuthRepository();
    final controller = buildController(
      storage: storage,
      appAuth: appAuth,
      auth: auth,
    );

    final ok = await controller.login();

    expect(ok, isFalse);
    expect(controller.isAuthenticated, isFalse);
    expect(controller.error, contains('cancelled'));
    expect(auth.exchangeCalls, 0);
    expect(await storage.read(), isNull);
  });

  test('PKCE verifier 缺失 → 报错且不调 exchange', () async {
    final storage = MemoryTokenStorage();
    final appAuth = FakeAppAuth(
      response: const AuthorizationResponse(authorizationCode: 'code-only'),
    );
    final auth = FakeAuthRepository();
    final controller = buildController(
      storage: storage,
      appAuth: appAuth,
      auth: auth,
    );

    final ok = await controller.login();

    expect(ok, isFalse);
    expect(controller.error, contains('verifier'));
    expect(auth.exchangeCalls, 0);
    expect(await storage.read(), isNull);
  });

  test('exchange 抛错 → 登录失败且不持久化', () async {
    final storage = MemoryTokenStorage();
    final appAuth = FakeAppAuth(
      response: const AuthorizationResponse(
        authorizationCode: 'auth-code-2',
        codeVerifier: 'pkce-verifier-2',
      ),
    );
    final auth = FakeAuthRepository()..exchangeError = Exception('backend 500');
    final controller = buildController(
      storage: storage,
      appAuth: appAuth,
      auth: auth,
    );

    final ok = await controller.login();

    expect(ok, isFalse);
    expect(controller.isAuthenticated, isFalse);
    expect(controller.error, contains('failed'));
    expect(auth.exchangeCalls, 1);
    expect(await storage.read(), isNull);
  });

  test('exchange 返回空 token → 登录失败', () async {
    final storage = MemoryTokenStorage();
    final appAuth = FakeAppAuth(
      response: const AuthorizationResponse(
        authorizationCode: 'auth-code-3',
        codeVerifier: 'pkce-verifier-3',
      ),
    );
    final auth = FakeAuthRepository(exchangeToken: '');
    final controller = buildController(
      storage: storage,
      appAuth: appAuth,
      auth: auth,
    );

    final ok = await controller.login();

    expect(ok, isFalse);
    expect(controller.isAuthenticated, isFalse);
    expect(controller.error, contains('exchange failed'));
    expect(await storage.read(), isNull);
  });

  test('authorize 抛错(取消/无浏览器)→ 登录失败', () async {
    final storage = MemoryTokenStorage();
    final appAuth = FakeAppAuth(throwError: Exception('user cancelled'));
    final auth = FakeAuthRepository();
    final controller = buildController(
      storage: storage,
      appAuth: appAuth,
      auth: auth,
    );

    final ok = await controller.login();

    expect(ok, isFalse);
    expect(controller.isAuthenticated, isFalse);
    expect(controller.error, contains('OIDC login failed'));
    expect(auth.exchangeCalls, 0);
    expect(await storage.read(), isNull);
  });
}
