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

/// 认证流程状态机测试(不依赖真实网络)。
///
/// 覆盖:登录成功、twoFactorRequired → TOTP 成功、验证码要求、
/// 401 错误、登出清理、启动恢复会话。
void main() {
  group('AuthController 登录状态机', () {
    test('登录成功进入 authenticated 且 token 持久化', () async {
      final storage = MemoryTokenStorage()..write('stored-token');
      final controller = _buildController(storage: storage);

      await controller.login(username: 'alice', password: 'secret');

      expect(controller.phase, LoginPhase.authenticated);
      expect(controller.isAuthenticated, isTrue);
      expect(await storage.read(), 'stored-token');
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

    test('401 登录失败进入 failed 并带错误消息', () async {
      final controller = _buildController(
        storage: MemoryTokenStorage(),
        authFail: true,
      );

      await controller.login(username: 'alice', password: 'wrong');

      expect(controller.phase, LoginPhase.failed);
      expect(controller.error, contains('Invalid username or password'));
    });

    test('登出清理 token 并回到 idle', () async {
      final storage = MemoryTokenStorage()..write('token');
      final controller = _buildController(storage: storage);
      await controller.login(username: 'alice', password: 'secret');
      expect(controller.isAuthenticated, isTrue);

      await controller.logout();

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
  FakeAuthRepository({this.twoFactorRequired = false, this.authFail = false});

  final bool twoFactorRequired;
  final bool authFail;

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
    return LoginResult(twoFactorRequired: twoFactorRequired);
  }

  @override
  Future<bool> totpVerify({required String code, String? recoveryCode}) async =>
      true;

  @override
  Future<String> register({
    required String username,
    required String email,
    required String password,
    String? captchaId,
    String? captchaCode,
    String? locale,
  }) async => '注册成功';

  @override
  Future<String> forgotPassword({
    required String email,
    String? captchaId,
    String? captchaCode,
  }) async => '已发送';

  @override
  Future<String> resetPassword({
    required String token,
    required String newPassword,
  }) async => '已重置';

  @override
  Future<String> oidcExchange({
    required String code,
    required String codeVerifier,
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
