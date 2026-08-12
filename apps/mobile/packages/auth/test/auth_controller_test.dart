import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
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

/// 可控的异步存储，用来复现 flutter_secure_storage 写入晚于登录返回的竞态。
class GatedTokenStorage extends MemoryTokenStorage {
  final writeStarted = Completer<void>();
  final allowWrite = Completer<void>();

  @override
  Future<void> write(String token) async {
    if (!writeStarted.isCompleted) writeStarted.complete();
    await allowWrite.future;
    await super.write(token);
  }
}

/// 认证流程状态机测试(不依赖真实网络)。
///
/// 覆盖:登录成功、twoFactorRequired → TOTP 成功、验证码要求、
/// 401 错误、登出清理、启动恢复会话。
void main() {
  group('AuthController 登录状态机', () {
    test('空存储等待 New-Token 持久化后才进入 authenticated', () async {
      final storage = GatedTokenStorage();
      final dio = Dio(BaseOptions(baseUrl: 'http://fake.local'));
      dio.httpClientAdapter = MockAdapter((request) async {
        switch (request.path) {
          case '/api/login-public-key':
            return ResponseData(200, {
              'code': 0,
              'result': {
                'publicKey': 'fake-pem',
                'serverTs': 1754524800000,
                'algorithm': 'RSA-OAEP-256',
              },
            });
          case '/api/login':
            return ResponseData(
              200,
              {'code': 0, 'result': '登录成功'},
              headers: {'New-Token': 'fresh-session-token'},
            );
          default:
            fail('unexpected request path: ${request.path}');
        }
      });
      final client = GfApiClient(
        dio: dio,
        tokenStorage: storage,
        onTokenRenewed: storage.write,
      );
      final controller = AuthController(
        authRepository: AuthRepository(client),
        apiClient: client,
        tokenStorage: storage,
        rsaEncryptor: FakeRsaEncryptor(),
      );

      var loginCompleted = false;
      final loginFuture = controller
          .login(username: 'alice', password: 'secret')
          .whenComplete(() => loginCompleted = true);
      await storage.writeStarted.future;

      expect(loginCompleted, isFalse);
      expect(await storage.read(), isNull);

      storage.allowWrite.complete();
      await loginFuture;

      expect(controller.phase, LoginPhase.authenticated);
      expect(await storage.read(), 'fresh-session-token');
    });

    test('twoFactorRequired 进入 needsTotp,TOTP 成功后 authenticated', () async {
      final storage = MemoryTokenStorage()..write('challenge-token');
      final controller = _buildController(
        storage: storage,
        twoFactorRequired: true,
      );

      await controller.login(username: 'alice', password: 'secret');
      expect(controller.phase, LoginPhase.needsTotp);

      await controller.submitTotp('123456');
      expect(controller.phase, LoginPhase.authenticated);
    });

    test('TOTP invalid-code 保留 needsTotp 且可重试成功', () async {
      final storage = MemoryTokenStorage()..write('challenge-token');
      final controller = _buildController(
        storage: storage,
        twoFactorRequired: true,
        totpVerifyError: const ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'totp.code.invalid',
        ),
      );

      await controller.login(username: 'alice', password: 'secret');
      expect(controller.phase, LoginPhase.needsTotp);

      await controller.submitTotp('000000');
      // 关键:phase 保持 needsTotp,输入框不消失,可重试。
      expect(controller.phase, LoginPhase.needsTotp);
      expect(controller.error, 'Invalid two-factor code, please try again');

      // 第二次提交成功(模拟用户重试)。
      await controller.submitTotp('123456');
      expect(controller.phase, LoginPhase.authenticated);
    });

    test('TOTP rate-limited 保留 needsTotp 并展示限流文案', () async {
      final storage = MemoryTokenStorage()..write('challenge-token');
      final controller = _buildController(
        storage: storage,
        twoFactorRequired: true,
        totpVerifyError: const ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'totp.rateLimited',
        ),
      );

      await controller.login(username: 'alice', password: 'secret');
      await controller.submitTotp('111111');

      expect(controller.phase, LoginPhase.needsTotp);
      expect(controller.error, 'Too many attempts, please try again later');
    });

    test('TOTP 未知业务错误保留 needsTotp 并透出 messageKey', () async {
      final storage = MemoryTokenStorage()..write('challenge-token');
      final controller = _buildController(
        storage: storage,
        twoFactorRequired: true,
        totpVerifyError: const ApiException(
          fallbackMessage: 'Request failed',
          messageCode: 'auth.login.failed',
        ),
      );

      await controller.login(username: 'alice', password: 'secret');
      await controller.submitTotp('222222');

      expect(controller.phase, LoginPhase.needsTotp);
      expect(controller.error, 'server.auth.login.failed');
    });

    test('TOTP 网络异常保留 needsTotp 可重试', () async {
      final storage = MemoryTokenStorage()..write('challenge-token');
      final controller = _buildController(
        storage: storage,
        twoFactorRequired: true,
        totpVerifyError: Exception('connection reset'),
      );

      await controller.login(username: 'alice', password: 'secret');
      await controller.submitTotp('333333');

      expect(controller.phase, LoginPhase.needsTotp);
      expect(controller.error, contains('TOTP verification failed'));
    });

    test('TOTP 401 过期挑战进入 failed 并提示重新登录', () async {
      final storage = MemoryTokenStorage()..write('challenge-token');
      final controller = _buildController(
        storage: storage,
        twoFactorRequired: true,
        totpVerifyError: const UnauthorizedException(),
      );

      await controller.login(username: 'alice', password: 'secret');
      expect(controller.phase, LoginPhase.needsTotp);

      await controller.submitTotp('444444');

      expect(controller.phase, LoginPhase.failed);
      expect(
        controller.error,
        'Two-factor challenge expired, please sign in again',
      );
    });

    test('401 登录失败进入 failed 并带错误消息', () async {
      final controller = _buildController(
        storage: MemoryTokenStorage(),
        authFail: true,
      );

      await controller.login(username: 'alice', password: 'wrong');

      expect(controller.phase, LoginPhase.failed);
      expect(controller.error, contains('Invalid username or password'));
    });

    test('common.captchaRequired 进入 needsCaptcha', () async {
      final controller = _buildController(
        storage: MemoryTokenStorage(),
        captchaRequired: true,
      );

      await controller.login(username: 'alice', password: 'secret');

      expect(controller.phase, LoginPhase.needsCaptcha);
      // loadCaptcha 可用:拿验证码图片。
      await controller.loadCaptcha();
      expect(controller.captcha, isNotNull);
    });

    test('注册遇到 common.captchaRequired 进入 needsCaptcha', () async {
      final controller = _buildController(
        storage: MemoryTokenStorage(),
        registerCaptchaRequired: true,
      );

      await controller.register(
        username: 'alice',
        email: 'alice@example.com',
        password: 'secret123',
      );

      expect(controller.phase, LoginPhase.needsCaptcha);
      expect(controller.error, 'Captcha required');
    });

    test('找回密码遇到 common.captchaRequired 进入 needsCaptcha', () async {
      final controller = _buildController(
        storage: MemoryTokenStorage(),
        forgotCaptchaRequired: true,
      );

      await controller.forgotPassword(email: 'alice@example.com');

      expect(controller.phase, LoginPhase.needsCaptcha);
      expect(controller.error, 'Captcha required');
    });

    test('auth.credentials.invalid 映射为友好登录错误', () async {
      final controller = _buildController(
        storage: MemoryTokenStorage(),
        loginMessageCode: 'auth.credentials.invalid',
      );

      await controller.login(username: 'alice', password: 'wrong');

      expect(controller.phase, LoginPhase.failed);
      expect(controller.error, 'Invalid username or password');
      expect(controller.error, isNot(startsWith('server.')));
    });

    test('登出清理 token 并回到 idle', () async {
      final storage = MemoryTokenStorage()..write('token');
      final dio = Dio(BaseOptions(baseUrl: 'http://fake.local'));
      final adapter = MockAdapter((request) async {
        return ResponseData(200, {'code': 0, 'result': true});
      });
      dio.httpClientAdapter = adapter;
      final client = GfApiClient(dio: dio, tokenStorage: storage);
      final controller = AuthController(
        authRepository: FakeAuthRepository(),
        apiClient: client,
        tokenStorage: storage,
        rsaEncryptor: FakeRsaEncryptor(),
      );

      await controller.logout();

      expect(adapter.requests.single.path, '/api/logout');
      expect(controller.phase, LoginPhase.idle);
      expect(await storage.read(), isNull);
    });

    test('init 从存储恢复会话', () async {
      final storage = MemoryTokenStorage()..write('saved-token');
      final controller = _buildController(storage: storage);

      final restored = await controller.init();

      expect(restored, isTrue);
      expect(controller.isAuthenticated, isTrue);
    });
  });
}

/// 构建 AuthController,注入可编程 fake 依赖。
AuthController _buildController({
  required TokenStorage storage,
  bool twoFactorRequired = false,
  bool authFail = false,
  bool captchaRequired = false,
  String? loginMessageCode,
  bool registerCaptchaRequired = false,
  bool forgotCaptchaRequired = false,
  Object? totpVerifyError,
  bool totpVerifySucceeds = true,
}) {
  final storageAdapter = storage;
  final client = GfApiClient(
    dio: DioAdapter().dio,
    tokenStorage: storageAdapter,
    baseUrl: 'http://fake.local',
  );
  final auth = FakeAuthRepository(
    twoFactorRequired: twoFactorRequired,
    authFail: authFail,
    captchaRequired: captchaRequired,
    loginMessageCode: loginMessageCode,
    registerCaptchaRequired: registerCaptchaRequired,
    forgotCaptchaRequired: forgotCaptchaRequired,
    totpVerifyError: totpVerifyError,
    totpVerifySucceeds: totpVerifySucceeds,
  );
  return AuthController(
    authRepository: auth,
    apiClient: client,
    tokenStorage: storageAdapter,
    rsaEncryptor: FakeRsaEncryptor(),
  );
}

/// 极简 Dio,避免真实网络。
class DioAdapter {
  final dio = Dio();
}

/// 可编程 AuthRepository fake。
class FakeAuthRepository implements AuthRepository {
  FakeAuthRepository({
    this.twoFactorRequired = false,
    this.authFail = false,
    this.captchaRequired = false,
    this.loginMessageCode,
    this.registerCaptchaRequired = false,
    this.forgotCaptchaRequired = false,
    this.totpVerifyError,
    this.totpVerifySucceeds = true,
  });

  final bool twoFactorRequired;
  final bool authFail;
  final bool captchaRequired;
  final String? loginMessageCode;
  final bool registerCaptchaRequired;
  final bool forgotCaptchaRequired;

  /// totpVerify 注入的错误;非 null 时优先于 [totpVerifySucceeds]。
  /// 抛错后自动清空,模拟"输错一次 → 重试成功"的真实链路。
  Object? totpVerifyError;

  /// totpVerify 是否成功返回;false 时返回 false(不抛错)。
  final bool totpVerifySucceeds;

  @override
  Future<CaptchaPayload> getCaptcha() async {
    return CaptchaPayload(
      captchaId: 'cid',
      captchaImg: 'data:image/png;base64,x',
    );
  }

  @override
  Future<LoginPublicKeyPayload> getLoginPublicKey() async {
    return LoginPublicKeyPayload(
      publicKey: 'fake-pem',
      serverTs: 1754524800000,
      algorithm: 'RSA-OAEP-256',
    );
  }

  @override
  Future<LoginResult> login({
    required String username,
    required String encryptedPassword,
    String? captchaId,
    String? captchaCode,
  }) async {
    if (authFail) throw UnauthorizedException();
    if (loginMessageCode != null) {
      throw ApiException(
        fallbackMessage: 'login failed',
        messageCode: loginMessageCode,
      );
    }
    if (captchaRequired) {
      throw const ApiException(
        fallbackMessage: 'captcha required',
        messageCode: 'common.captchaRequired',
      );
    }
    return LoginResult(twoFactorRequired: twoFactorRequired);
  }

  @override
  Future<bool> totpVerify({required String code, String? recoveryCode}) async {
    final error = totpVerifyError;
    if (error != null) {
      // 只失败一次:模拟"输错一次 → 重试成功"的真实链路。
      totpVerifyError = null;
      throw error;
    }
    return totpVerifySucceeds;
  }

  @override
  Future<String> register({
    required String username,
    required String email,
    required String password,
    String? captchaId,
    String? captchaCode,
    String? locale,
  }) async {
    if (registerCaptchaRequired) {
      throw const ApiException(
        fallbackMessage: 'captcha required',
        messageCode: 'common.captchaRequired',
      );
    }
    return '注册成功';
  }

  @override
  Future<String> forgotPassword({
    required String email,
    String? captchaId,
    String? captchaCode,
  }) async {
    if (forgotCaptchaRequired) {
      throw const ApiException(
        fallbackMessage: 'captcha required',
        messageCode: 'common.captchaRequired',
      );
    }
    return '已发送';
  }

  @override
  Future<String> resetPassword({
    required String token,
    required String newPassword,
  }) async => '已重置';

  @override
  Future<String> oidcExchange({
    required String code,
    required String codeVerifier,
    required String nonce,
    required String redirectUri,
  }) async => 'oidc-token';

  @override
  Future<bool> logout() async => true;
}

/// 固定输出的 RsaEncryptor fake。
class FakeRsaEncryptor implements RsaEncryptor {
  @override
  String encryptPassword({
    required String publicKeyPem,
    required String password,
    required int serverTs,
  }) {
    return 'encrypted:$password';
  }
}

class ResponseData {
  ResponseData(this.statusCode, this.data, {this.headers = const {}});

  final int statusCode;
  final Map<String, dynamic> data;
  final Map<String, String> headers;
}

typedef MockHandler = Future<ResponseData> Function(RequestOptions request);

class MockAdapter implements HttpClientAdapter {
  MockAdapter(this._handler);

  final MockHandler _handler;
  final List<RequestOptions> requests = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    final result = await _handler(options);
    return ResponseBody.fromString(
      jsonEncode(result.data),
      result.statusCode,
      headers: {
        Headers.contentTypeHeader: ['application/json'],
        for (final entry in result.headers.entries) entry.key: [entry.value],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
