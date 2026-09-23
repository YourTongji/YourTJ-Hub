import 'dart:convert';

import 'package:flutter_appauth/flutter_appauth.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pointycastle/digests/sha256.dart';

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
  String? lastNonce;
  String? lastRedirectUri;

  @override
  Future<String> oidcExchange({
    required String code,
    required String codeVerifier,
    required String nonce,
    required String redirectUri,
  }) async {
    exchangeCalls++;
    lastCode = code;
    lastVerifier = codeVerifier;
    lastNonce = nonce;
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

  @override
  Future<bool> logout() async => true;
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
    // exchange 收到 code + verifier + 同一 nonce + redirectUri。
    expect(auth.exchangeCalls, 1);
    expect(auth.lastCode, 'auth-code-1');
    expect(auth.lastVerifier, 'pkce-verifier-1');
    expect(auth.lastNonce, appAuth.lastRequest!.nonce);
    expect(auth.lastRedirectUri, redirectUri);
    // token 已持久化。
    expect(await storage.read(), 'forum-jwt-1');
  });

  for (final provider in ['google', 'github', 'tongji']) {
    test('$provider selection preserves the PKCE exchange and nonce', () async {
      final storage = MemoryTokenStorage();
      final appAuth = FakeAppAuth(
        response: const AuthorizationResponse(
          authorizationCode: 'social-code',
          codeVerifier: 'social-verifier',
        ),
      );
      final auth = FakeAuthRepository();
      final controller = buildController(
        storage: storage,
        appAuth: appAuth,
        auth: auth,
      );
      expect(await controller.login(provider: provider), isTrue);
      expect(appAuth.lastRequest!.additionalParameters, {
        'login_hint': provider,
      });
      expect(auth.lastVerifier, 'social-verifier');
      expect(auth.lastNonce, appAuth.lastRequest!.nonce);
      expect(auth.lastRedirectUri, redirectUri);
      expect(await storage.read(), 'forum-jwt');
    });
  }
  test('unsupported social provider never opens the browser', () async {
    final appAuth = FakeAppAuth();
    final controller = buildController(
      storage: MemoryTokenStorage(),
      appAuth: appAuth,
      auth: FakeAuthRepository(),
    );
    await expectLater(
      controller.login(provider: 'untrusted'),
      throwsArgumentError,
    );
    expect(appAuth.authorizeCalls, 0);
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

  test('exchange 收到 403 冻结 ApiFailure → 保留稳定 messageCode', () async {
    final storage = MemoryTokenStorage();
    final appAuth = FakeAppAuth(
      response: const AuthorizationResponse(
        authorizationCode: 'auth-code-frozen',
        codeVerifier: 'pkce-verifier-frozen',
      ),
    );
    final auth = FakeAuthRepository()
      ..exchangeError = const ApiFailureException(
        fallbackMessage: 'Request failed with status 403',
        messageCode: 'oauth.account.frozen',
        params: {'action': 'login'},
        statusCode: 403,
      );
    final controller = buildController(
      storage: storage,
      appAuth: appAuth,
      auth: auth,
    );

    final ok = await controller.login();

    expect(ok, isFalse);
    expect(controller.isAuthenticated, isFalse);
    // 暴露稳定 messageCode 供上层本地化,而不是笼统的 "OIDC login failed"。
    expect(controller.error, 'server.oauth.account.frozen');
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

  group('manual Android authorization', () {
    test(
      'all providers build RFC 8252 authorization requests without AppAuth',
      () {
        for (final provider in ['google', 'github', 'tongji']) {
          final appAuth = FakeAppAuth();
          final controller = buildController(
            storage: MemoryTokenStorage(),
            appAuth: appAuth,
            auth: FakeAuthRepository(),
          );

          final session = controller.beginManualAuthorization(
            provider: provider,
          );
          final query = session.authorizationUri.queryParameters;
          expect(session.provider, provider);
          expect(
            session.codeVerifier,
            matches(RegExp(r'^[A-Za-z0-9._~-]{43,128}$')),
          );
          final challenge = base64Url
              .encode(
                SHA256Digest().process(ascii.encode(session.codeVerifier)),
              )
              .replaceAll('=', '');
          expect(query['client_id'], clientId);
          expect(query['redirect_uri'], redirectUri);
          expect(query['response_type'], 'code');
          expect(query['scope'], 'openid profile email');
          expect(query['state'], session.state);
          expect(query['nonce'], session.nonce);
          expect(query['code_challenge'], challenge);
          expect(query['code_challenge_method'], 'S256');
          expect(query['login_hint'], provider);
          expect(appAuth.authorizeCalls, 0);
        }
      },
    );

    test('state and nonce are independently generated for each session', () {
      final controller = buildController(
        storage: MemoryTokenStorage(),
        appAuth: FakeAppAuth(),
        auth: FakeAuthRepository(),
      );

      final first = controller.beginManualAuthorization(provider: 'google');
      final second = controller.beginManualAuthorization(provider: 'google');

      expect(first.state, isNot(second.state));
      expect(first.nonce, isNot(second.nonce));
      expect(first.state, isNot(first.nonce));
    });

    test(
      'matching callback exchanges exact session values and stores the forum JWT',
      () async {
        final storage = MemoryTokenStorage();
        final auth = FakeAuthRepository(exchangeToken: 'manual-forum-jwt');
        final controller = buildController(
          storage: storage,
          appAuth: FakeAppAuth(),
          auth: auth,
        );
        final session = controller.beginManualAuthorization(provider: 'github');

        final ok = await controller.completeManualAuthorization(
          session,
          Uri.parse(
            'yourtj://callback?code=manual-code&state=${session.state}',
          ),
        );

        expect(ok, isTrue);
        expect(auth.lastCode, 'manual-code');
        expect(auth.lastVerifier, session.codeVerifier);
        expect(auth.lastNonce, session.nonce);
        expect(auth.lastRedirectUri, session.redirectUri);
        expect(await storage.read(), 'manual-forum-jwt');
        expect(controller.isAuthenticated, isTrue);
      },
    );

    for (final invalidCallback in <String>[
      'yourtj://callback?code=code&state=wrong',
      'other://callback?code=code&state=state',
      'yourtj://other?code=code&state=state',
      'yourtj://callback/path?code=code&state=state',
      'yourtj://user@callback?code=code&state=state',
      'yourtj://callback?code=code&state=state#fragment',
    ]) {
      test(
        'rejects callback lookalike without exchanging: $invalidCallback',
        () async {
          final auth = FakeAuthRepository();
          final controller = buildController(
            storage: MemoryTokenStorage(),
            appAuth: FakeAppAuth(),
            auth: auth,
          );
          final session = controller.beginManualAuthorization(
            provider: 'tongji',
          );
          final callback = invalidCallback.replaceFirst(
            'state=state',
            'state=${session.state}',
          );

          expect(
            await controller.completeManualAuthorization(
              session,
              Uri.parse(callback),
            ),
            isFalse,
          );
          expect(auth.exchangeCalls, 0);
        },
      );
    }

    test('callback error and missing code do not exchange', () async {
      final auth = FakeAuthRepository();
      final controller = buildController(
        storage: MemoryTokenStorage(),
        appAuth: FakeAppAuth(),
        auth: auth,
      );
      final denied = controller.beginManualAuthorization(provider: 'google');

      expect(
        await controller.completeManualAuthorization(
          denied,
          Uri.parse(
            'yourtj://callback?error=access_denied&state=${denied.state}',
          ),
        ),
        isFalse,
      );
      expect(controller.error, 'OIDC authorization cancelled');
      expect(auth.exchangeCalls, 0);

      final missingCode = controller.beginManualAuthorization(
        provider: 'google',
      );
      expect(
        await controller.completeManualAuthorization(
          missingCode,
          Uri.parse('yourtj://callback?state=${missingCode.state}'),
        ),
        isFalse,
      );
      expect(controller.error, 'OIDC authorization code missing');
      expect(auth.exchangeCalls, 0);
    });

    test('empty exchanged token is a normal failure', () async {
      final auth = FakeAuthRepository(exchangeToken: '');
      final controller = buildController(
        storage: MemoryTokenStorage(),
        appAuth: FakeAppAuth(),
        auth: auth,
      );
      final session = controller.beginManualAuthorization(provider: 'google');

      expect(
        await controller.completeManualAuthorization(
          session,
          Uri.parse('yourtj://callback?code=code&state=${session.state}'),
        ),
        isFalse,
      );
      expect(controller.error, 'OIDC exchange failed');
      expect(controller.isAuthenticated, isFalse);
    });
  });
}
