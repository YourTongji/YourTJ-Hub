import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:auth/auth.dart';
import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../app_config.dart';
import '../../navigation/auth_navigation.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../current_user.dart';
import '../../theme_mode.dart';
import '../../widgets/language_picker.dart';
import 'auth_ime_stabilizer.dart';
import 'android_oidc_callback_source.dart';
import 'android_oidc_coordinator.dart';
import 'login_captcha_handoff.dart';

/// 登录页模式。
enum _AuthMode { login, register, forgotPassword }

/// 登录流程专用的内存令牌暂存区。
///
/// Password/TOTP 的 `New-Token` 与 OIDC exchange 返回的论坛 JWT 在旧账号
/// 离线缓存清理成功前只写入这里，绝不提前进入 Keychain/Keystore。这样即使
/// 清库失败或进程被杀，也不存在“新账号持久化令牌 + 旧账号离线缓存”的组合。
class _StagedTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

class _AuthImePostFrameTimer implements AuthImeTimer, LoginCaptchaForwardTimer {
  _AuthImePostFrameTimer(this._cancelCallback);

  final VoidCallback _cancelCallback;

  @override
  void cancel() => _cancelCallback();
}

class _AuthImeDartTimer implements AuthImeTimer {
  _AuthImeDartTimer(this._timer);

  final Timer _timer;

  @override
  void cancel() => _timer.cancel();
}

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key, this.authController, this.authTokenStorage})
    : assert(authController == null || authTokenStorage != null);

  /// 测试注入:默认 null 时页面内部构造 [AuthController]。
  final AuthController? authController;

  /// 测试注入:认证成功令牌的内存暂存区,必须与 [authController] 使用同一实例。
  final TokenStorage? authTokenStorage;
  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage>
    with WidgetsBindingObserver {
  final TextEditingController _username = TextEditingController();
  final TextEditingController _password = TextEditingController();
  final FocusNode _usernameFocusNode = FocusNode();
  final FocusNode _passwordFocusNode = FocusNode();
  final FocusNode _captchaFocusNode = FocusNode();
  final TextEditingController _email = TextEditingController();
  final TextEditingController _captcha = TextEditingController();
  final TextEditingController _totp = TextEditingController();

  final _confirmPassword = TextEditingController();
  LoginPageProps? _registration;
  bool _registrationLoading = false;
  String? _registrationError;
  String? _emailDomain;
  bool _agreed = false;
  _AuthMode _mode = _AuthMode.login;
  bool _oidcBusy = false;
  bool _finishingAuthentication = false;
  bool _passwordInteractionStarted = false;
  bool _loginCaptchaRevealed = false;
  bool _captchaRevealFrameScheduled = false;
  bool _captchaFocusEligible = false;
  bool _suppressPasswordTapOutside = false;
  Future<void>? _captchaLoadFuture;
  final Stopwatch _authImeClock = Stopwatch()..start();
  late final AuthImeStabilizer<FocusNode> _authIme;
  late final LoginCaptchaForwardHandoff _captchaHandoff;
  String _oidcError = '';
  // 登录放行前缓存清理失败的错误(清理成功或重试成功后清空)。
  String _cacheError = '';
  // 缓存清理失败后禁止返回旧 shell(其内存态可能含上一账号数据)。
  bool _authBlocked = false;

  late final AuthController _authController;
  late final GfApiClient _authClient;
  late final TokenStorage _authTokenStorage;
  AndroidOidcCoordinator? _androidOidcCoordinator;

  // 进入登录页时启动的缓存清理;登录放行前必须成功。
  Future<bool>? _cacheClearFuture;

  @override
  void initState() {
    super.initState();
    _authTokenStorage = widget.authTokenStorage ?? _StagedTokenStorage();
    _authClient = GfApiClient(
      dio: ref.read(authDioProvider),
      tokenStorage: _authTokenStorage,
      baseUrl: AppConfig.apiBaseUrl.isNotEmpty
          ? AppConfig.apiBaseUrl
          : GfApiClient.defaultBaseUrl,
      onTokenRenewed: _authTokenStorage.write,
    );
    _authController =
        widget.authController ??
        AuthController(
          authRepository: AuthRepository(_authClient),
          apiClient: _authClient,
          tokenStorage: _authTokenStorage,
        );
    _authIme = AuthImeStabilizer<FocusNode>(
      enabled: !kIsWeb && defaultTargetPlatform == TargetPlatform.android,
      isFocused: (node) => node.hasFocus,
      isImeVisible: _isImeVisible,
      showIme: _showAuthIme,
      releaseFocus: (node) => node.unfocus(),
      requestFocus: (node) => node.requestFocus(),
      now: () => _authImeClock.elapsed,
      schedule: _scheduleAuthIme,
      recoverySource: _passwordFocusNode,
      onRecoverySourceDismissed: (_) => _completePasswordStage(),
      targetName: (node) => node == _usernameFocusNode
          ? 'username'
          : node == _captchaFocusNode
          ? 'captcha'
          : 'password',
      onLog: (message) => debugPrint(message),
    );
    _captchaHandoff = LoginCaptchaForwardHandoff(
      isMounted: () => mounted,
      isCaptchaEligible: () =>
          _mode == _AuthMode.login &&
          _loginCaptchaRevealed &&
          _authController.captcha != null,
      requestCaptchaFocus: _captchaFocusNode.requestFocus,
      schedule: _scheduleCaptchaForward,
    );
    WidgetsBinding.instance.addObserver(this);
    _usernameFocusNode.addListener(_onUsernameFocusChanged);
    _passwordFocusNode.addListener(_onPasswordFocusChanged);
    _captchaFocusNode.addListener(_onCaptchaFocusChanged);

    // 进入登录页即进入新会话边界:先使旧会话在途写入失效,再清空缓存。
    // 两者都延迟到首帧后执行(避免在 widget 构建期修改 provider),且
    // 世代失效必须先于清库,保证不变量:
    //   - 失效前提交的旧会话写入会被随后的清库清掉;
    //   - 失效后提交的写入会被世代守卫拦截。
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      ref.read(offlineCacheEpochProvider.notifier).invalidate();
      // 进入登录页即会话边界:使缓存的当前用户身份失效(旧账号 id 不再
      // 被后续新 shell 读取)。
      ref.invalidate(currentUserProvider);
      _cacheClearFuture = _clearOfflineCacheOnce();
      _loadRegistration();
      _authIme.metricsChanged();
    });
  }

  /// 执行一次离线缓存清理;成功返回 true,失败返回 false(不抛出)。
  Future<bool> _clearOfflineCacheOnce() async {
    try {
      await clearOfflineCache(
        ref.read(offlineTopicCacheProvider),
        ref.read(offlineChatCacheProvider),
        ref.read(scheduleWidgetBridgeProvider),
      );
      return true;
    } catch (_) {
      return false;
    }
  }

  /// 登录放行前确保上一账号缓存已清空:首次清理失败(SQLite 慢/锁)时
  /// 重试一次;仍失败返回 false,由调用方留在登录页提示重试。
  Future<bool> _ensureCacheCleared() async {
    Future<bool> attempt = _cacheClearFuture ??= _clearOfflineCacheOnce();
    if (await attempt) return true;
    attempt = _cacheClearFuture = _clearOfflineCacheOnce();
    return attempt;
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _androidOidcCoordinator?.dispose();
    _androidOidcCoordinator = null;
    _captchaHandoff.dispose();
    _authIme.dispose();
    _usernameFocusNode.removeListener(_onUsernameFocusChanged);
    _passwordFocusNode.removeListener(_onPasswordFocusChanged);
    _captchaFocusNode.removeListener(_onCaptchaFocusChanged);
    _usernameFocusNode.dispose();
    _passwordFocusNode.dispose();
    _captchaFocusNode.dispose();
    _confirmPassword.dispose();
    _username.dispose();
    _password.dispose();
    _email.dispose();
    _captcha.dispose();
    _totp.dispose();
    _authController.dispose();
    super.dispose();
  }

  @override
  void didChangeMetrics() {
    _authIme.metricsChanged();
  }

  bool _isImeVisible() => View.of(context).viewInsets.bottom > 0;

  void _showAuthIme() {
    unawaited(_requestAuthImeShow());
  }

  Future<void> _requestAuthImeShow() async {
    try {
      await SystemChannels.textInput.invokeMethod<void>('TextInput.show');
    } catch (error) {
      if (kDebugMode) {
        debugPrint('[AuthIme] TextInput.show failed=${error.runtimeType}');
      }
    }
  }

  AuthImeTimer _scheduleAuthIme(Duration delay, VoidCallback callback) {
    if (delay == Duration.zero) {
      bool cancelled = false;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!cancelled) callback();
      });
      return _AuthImePostFrameTimer(() => cancelled = true);
    }

    final Timer timer = Timer(delay, callback);
    return _AuthImeDartTimer(timer);
  }

  LoginCaptchaForwardTimer _scheduleCaptchaForward(VoidCallback callback) {
    bool cancelled = false;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!cancelled) callback();
    });
    return _AuthImePostFrameTimer(() => cancelled = true);
  }

  void _onUsernameFocusChanged() {
    _authIme.focusChanged(_usernameFocusNode, _usernameFocusNode.hasFocus);
  }

  void _onPasswordFocusChanged() {
    _authIme.focusChanged(_passwordFocusNode, _passwordFocusNode.hasFocus);
    if (_passwordFocusNode.hasFocus) {
      _passwordInteractionStarted = true;
      _prefetchLoginCaptcha();
    }
  }

  void _onCaptchaFocusChanged() {
    if (!_captchaFocusEligible) return;
    _authIme.focusChanged(_captchaFocusNode, _captchaFocusNode.hasFocus);
  }

  void _onPasswordChanged(String _) {
    _passwordInteractionStarted = true;
    _prefetchLoginCaptcha();
  }

  void _prefetchLoginCaptcha() {
    if (_mode != _AuthMode.login) return;
    unawaited(
      _loadCaptchaIfNeeded(preservePhaseOnError: true, silentOnError: true),
    );
  }

  void _completePasswordStage() {
    if (!_passwordInteractionStarted ||
        _mode != _AuthMode.login ||
        _loginCaptchaRevealed) {
      return;
    }

    // Invalidate every password recovery/watchdog generation before moving
    // focus. A stale AuthIme callback must never restore the password field.
    _authIme.cancel();
    _captchaHandoff.begin();
    _loginCaptchaRevealed = true;
    _passwordFocusNode.unfocus();
    // A failed prefetch has no payload, so this explicit handoff retries it;
    // an existing payload is reused and can focus on the next frame.
    unawaited(
      _loadCaptchaIfNeeded(
        preservePhaseOnError: true,
        silentOnError: false,
      ).then((_) => _captchaHandoff.captchaEligibilityChanged()),
    );
    if (!mounted || _captchaRevealFrameScheduled) return;
    _captchaRevealFrameScheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _captchaRevealFrameScheduled = false;
      if (mounted) setState(() {});
    });
  }

  Future<void> _loadCaptchaIfNeeded({
    required bool preservePhaseOnError,
    required bool silentOnError,
    bool force = false,
  }) {
    if (!force && _authController.captcha != null) {
      return Future<void>.value();
    }
    final Future<void>? inFlight = _captchaLoadFuture;
    if (inFlight != null) return inFlight;

    final Future<void> future = _loadCaptchaRequest(
      preservePhaseOnError: preservePhaseOnError,
      silentOnError: silentOnError,
    );
    _captchaLoadFuture = future;
    unawaited(
      future.whenComplete(() {
        if (identical(_captchaLoadFuture, future)) {
          _captchaLoadFuture = null;
        }
      }),
    );
    return future;
  }

  Future<void> _loadCaptchaRequest({
    required bool preservePhaseOnError,
    required bool silentOnError,
  }) async {
    try {
      await _authController.loadCaptcha(
        preservePhaseOnError: preservePhaseOnError,
        silentOnError: silentOnError,
      );
    } catch (_) {
      // Prefetch and an explicit refresh are recoverable UI operations.
      // Keep provider errors from escaping an unawaited pointer path.
    }
  }

  void _captureAuthFocusIntent(FocusNode target) {
    // An explicit field target wins over any automatic captcha focus waiting
    // for the next frame.
    _suppressPasswordTapOutside =
        target == _usernameFocusNode || target == _captchaFocusNode;
    _captchaHandoff.cancel();
    _authIme.pointerDown(target);
  }

  void _onPasswordTapOutside() {
    final bool explicitTarget = _suppressPasswordTapOutside;
    _suppressPasswordTapOutside = false;
    // A username/captcha pointer-down cancels an already pending automatic
    // captcha focus. Before the handoff has begun, however, the same pointer
    // is the ordinary password-stage completion gesture.
    if (explicitTarget && _loginCaptchaRevealed) {
      return;
    }
    _completePasswordStage();
  }

  Widget _withAuthFocusIntent(FocusNode target, Widget child) {
    return Listener(
      behavior: HitTestBehavior.translucent,
      onPointerDown: (_) => _captureAuthFocusIntent(target),
      child: child,
    );
  }

  Widget _buildCaptchaInput(AppLocalizations l10n) {
    final Widget input = GfInput(
      controller: _captcha,
      focusNode: _mode == _AuthMode.login ? _captchaFocusNode : null,
      labelText: l10n.authCaptcha,
      autofillHints: const [],
      autocorrect: false,
      enableSuggestions: false,
      textInputAction: TextInputAction.done,
      onSubmitted: (_) => _submit(),
    );
    return _mode == _AuthMode.login
        ? _withAuthFocusIntent(_captchaFocusNode, input)
        : input;
  }

  void _loadVisibleCaptcha() {
    unawaited(
      _loadCaptchaIfNeeded(
        preservePhaseOnError: _mode == _AuthMode.login,
        silentOnError: false,
        force: true,
      ).then((_) => _captchaHandoff.captchaEligibilityChanged()),
    );
  }

  Future<void> _login() async {
    final String? captchaId = _authController.captcha?.captchaId;
    final String captchaCode = _captcha.text.trim();
    await _authController.login(
      username: _username.text.trim(),
      password: _password.text,
      captchaId: (captchaId == null || captchaId.isEmpty) ? null : captchaId,
      captchaCode: captchaCode.isEmpty ? null : captchaCode,
    );
    if (mounted && _authController.phase == LoginPhase.needsCaptcha) {
      await _loadCaptchaIfNeeded(
        preservePhaseOnError: _mode == _AuthMode.login,
        silentOnError: false,
        force: true,
      );
    }
    if (mounted && _authController.phase == LoginPhase.authenticated) {
      await _finishAuthentication(saveAutofill: true);
    }
  }

  Future<void> _register() async {
    final options = _registration;
    if (options == null || _registrationLoading) return;
    final l10n = AppLocalizations.of(context);
    if (_password.text != _confirmPassword.text) {
      setState(() => _registrationError = l10n.authPasswordMismatch);
      return;
    }
    if ((options.termsOfServiceEnabled || options.privacyPolicyEnabled) &&
        !_agreed) {
      return;
    }
    setState(() => _registrationError = null);
    final email = options.allowedDomains.isEmpty
        ? _email.text.trim()
        : '${_email.text.trim()}@$_emailDomain';
    final String? captchaId = _authController.captcha?.captchaId;
    final String captchaCode = _captcha.text.trim();
    await _authController.register(
      username: _username.text.trim(),
      email: email,
      password: _password.text,
      captchaId: (captchaId == null || captchaId.isEmpty) ? null : captchaId,
      captchaCode: captchaCode.isEmpty ? null : captchaCode,
    );
    if (mounted && _authController.phase == LoginPhase.needsCaptcha) {
      await _authController.loadCaptcha();
    }
    if (mounted && _authController.error.isEmpty) {
      showGfToast(context, AppLocalizations.of(context).authRegisterSuccess);
      setState(() => _mode = _AuthMode.login);
    }
  }

  Future<void> _forgotPassword() async {
    final String? captchaId = _authController.captcha?.captchaId;
    final String captchaCode = _captcha.text.trim();
    await _authController.forgotPassword(
      email: _email.text.trim(),
      captchaId: (captchaId == null || captchaId.isEmpty) ? null : captchaId,
      captchaCode: captchaCode.isEmpty ? null : captchaCode,
    );
    if (mounted && _authController.phase == LoginPhase.needsCaptcha) {
      await _authController.loadCaptcha();
    }
    if (mounted && _authController.error.isEmpty) {
      showGfToast(context, AppLocalizations.of(context).authResetEmailSent);
      setState(() => _mode = _AuthMode.login);
    }
  }

  Future<void> _loginOidc(String provider) async {
    if (_oidcBusy) return;
    setState(() {
      _oidcBusy = true;
      _oidcError = '';
    });
    final controller = OidcController(
      authRepository: AuthRepository(_authClient),
      tokenStorage: _authTokenStorage,
      issuer: AppConfig.oidcIssuer,
      clientId: AppConfig.oidcClientId,
    );
    try {
      final bool ok;
      if (!kIsWeb && defaultTargetPlatform == TargetPlatform.android) {
        final AndroidOidcCoordinator coordinator = AndroidOidcCoordinator(
          begin: (value) =>
              controller.beginManualAuthorization(provider: value),
          complete: controller.completeManualAuthorization,
          reportError: controller.setExternalError,
          callbackSource: MainActivityOidcCallbackSource(),
          launchExternal: (uri) =>
              launchUrl(uri, mode: LaunchMode.externalApplication),
        );
        _androidOidcCoordinator = coordinator;
        ok = await coordinator.login(provider: provider);
      } else {
        ok = await controller.login(provider: provider);
      }
      if (!mounted) return;
      if (!ok) {
        setState(() => _oidcError = controller.error);
        return;
      }
      await _finishAuthentication();
    } finally {
      _androidOidcCoordinator?.dispose();
      _androidOidcCoordinator = null;
      controller.dispose();
      if (mounted) setState(() => _oidcBusy = false);
    }
  }

  /// 认证成功后的收尾:先确保旧账号缓存已清空,再把暂存的新会话提交到
  /// 安全存储。清理失败时新 token 从未持久化,因此重启也无法以新账号读取
  /// 旧账号离线数据。
  Future<void> _finishAuthentication({bool saveAutofill = false}) async {
    if (!mounted || _finishingAuthentication) return;
    setState(() => _finishingAuthentication = true);
    try {
      if (!await _ensureCacheCleared()) {
        await _authTokenStorage.clear();
        if (!mounted) return;
        setState(() {
          _authBlocked = true;
          _cacheError = AppLocalizations.of(context).authCacheClearFailed;
        });
        return;
      }

      final String? token = await _authTokenStorage.read();
      if (token == null || token.isEmpty) {
        if (!mounted) return;
        setState(() {
          _authBlocked = true;
          _cacheError = AppLocalizations.of(context).authSessionSaveFailed;
        });
        return;
      }

      try {
        await ref.read(tokenStorageProvider).write(token);
      } catch (_) {
        // 安全存储写入可能在落盘后抛错,结果不确定。缓存已经清空,所以重启
        // 不会泄漏旧数据;当前进程仍禁止返回旧 shell,避免其内存态被新令牌复用。
        await _authTokenStorage.clear();
        if (!mounted) return;
        setState(() {
          _authBlocked = true;
          _cacheError = AppLocalizations.of(context).authSessionSaveFailed;
        });
        return;
      }
      await _authTokenStorage.clear();
      if (!mounted) return;
      _authBlocked = false;
      _cacheError = '';
      // 新 token 已接受:使缓存的当前用户身份失效,新 shell 重新从
      // 新令牌解析账号 id,避免 ProfilePage 仍用上一账号的 id 请求数据。
      ref.invalidate(currentUserProvider);
      // 会话边界:重建主 API client。旧 client 捕获的是上一会话的 epoch,
      // 其 New-Token 续期与 401 回调已永久失效;新 shell 首次读取时重建
      // 并捕获当前 epoch,恢复新账号的滑动续期与 401 清理。
      ref.invalidate(apiClientProvider);
      // Replace the old stack after accepting the new session, then restore
      // only a validated native location. No pending write action is replayed.
      if (saveAutofill) TextInput.finishAutofillContext(shouldSave: true);
      final epoch = ref.read(offlineCacheEpochProvider);
      final session = ref.read(offlineCacheEpochProvider.notifier);
      restoreAuthContext(
        GoRouter.of(context),
        _returnTo,
        isCurrent: () => session.isCurrent(epoch),
      );
    } finally {
      if (mounted) setState(() => _finishingAuthentication = false);
    }
  }

  String? get _returnTo {
    if (GoRouter.maybeOf(context) == null) return null;
    return safeAuthReturnTo(
      GoRouterState.of(context).uri.queryParameters['returnTo'],
    );
  }

  void _leaveAuth() {
    // 缓存清理失败后会话已丢弃:禁止返回旧 shell(内存态可能含上一账号
    // 数据),只能重试登录。
    if (_authBlocked || _finishingAuthentication) return;
    final NavigatorState navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    context.go('/');
  }

  String _title(AppLocalizations l10n) => switch (_mode) {
    _AuthMode.login => l10n.authLoginTitle,
    _AuthMode.register => l10n.authRegisterTitle,
    _AuthMode.forgotPassword => l10n.authForgotTitle,
  };

  String _subtitle(AppLocalizations l10n) => switch (_mode) {
    _AuthMode.login => l10n.authLoginSubtitle,
    _AuthMode.register => l10n.authRegisterSubtitle,
    _AuthMode.forgotPassword => l10n.authForgotSubtitle,
  };

  Future<void> _loadRegistration() async {
    if (_registrationLoading) return;
    setState(() {
      _registrationLoading = true;
      _registrationError = null;
    });
    try {
      // The shared page client still owns the previous account credential.
      final payload = await PageRepository(_authClient).fetch('/login');
      final options = parsePageProps<LoginPageProps>(payload);
      if (options == null) throw StateError('Invalid login page');
      if (!mounted) return;
      setState(() {
        _registration = options;
        _emailDomain = options.allowedDomains.firstOrNull;
      });
    } catch (error) {
      if (mounted) {
        setState(
          () => _registrationError = resolveErrorMessage(
            AppLocalizations.of(context),
            error,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _registrationLoading = false);
    }
  }

  void _switchMode(_AuthMode mode) {
    if (_authController.busy || _finishingAuthentication) return;
    _authIme.cancel();
    _captchaHandoff.cancel();
    if (mode == _AuthMode.register && _registration == null) {
      _loadRegistration();
    }
    _captcha.clear();
    setState(() {
      _mode = mode;
      _loginCaptchaRevealed = false;
      _passwordInteractionStarted = false;
      _suppressPasswordTapOutside = false;
      _oidcError = '';
    });
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final Brightness brightness = Theme.of(context).brightness;

    // 缓存清理失败后禁止任何返回(含系统返回手势),只能重试登录。
    return PopScope(
      canPop: !_authBlocked && !_finishingAuthentication,
      child: Scaffold(
        backgroundColor: colors.base100,
        body: Stack(
          fit: StackFit.expand,
          children: <Widget>[
            SafeArea(
              child: Stack(
                children: <Widget>[
                  Positioned(
                    top: 4,
                    left: 8,
                    child: GfIconButton(
                      icon: Icons.arrow_back,
                      tooltip: l10n.commonBack,
                      size: 44,
                      onPressed: _leaveAuth,
                    ),
                  ),
                  Positioned(
                    top: 4,
                    right: 56,
                    child: IconButton(
                      icon: const GfSymbol('languages'),
                      tooltip: l10n.settingsAppLanguage,
                      onPressed: () => showAppLanguagePicker(context),
                    ),
                  ),
                  Positioned(
                    top: 4,
                    right: 8,
                    child: GfIconButton(
                      tooltip: l10n.settingsAppearance,
                      icon: brightness == Brightness.dark
                          ? Icons.light_mode_outlined
                          : Icons.dark_mode_outlined,
                      onPressed: () => ref
                          .read(themeModeProvider.notifier)
                          .toggleDark(brightness != Brightness.dark),
                    ),
                  ),
                  Center(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.fromLTRB(20, 64, 20, 32),
                      child: ConstrainedBox(
                        constraints: const BoxConstraints(maxWidth: 520),
                        child: GfCard(
                          showDivider: false,
                          padding: const EdgeInsets.fromLTRB(4, 16, 4, 24),
                          child: ListenableBuilder(
                            listenable: _authController,
                            builder: (BuildContext context, Widget? child) {
                              return _buildCardContent(context, l10n, colors);
                            },
                          ),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildCardContent(
    BuildContext context,
    AppLocalizations l10n,
    GfColors colors,
  ) {
    final bool showCaptcha =
        _authController.phase == LoginPhase.needsCaptcha ||
        (_mode == _AuthMode.login && _loginCaptchaRevealed);
    final CaptchaPayload? captcha = _authController.captcha;
    final bool captchaFocusEligible =
        _mode == _AuthMode.login && showCaptcha && captcha != null;
    if (_captchaFocusEligible != captchaFocusEligible) {
      _captchaFocusEligible = captchaFocusEligible;
      if (!captchaFocusEligible) {
        _authIme.cancel();
      } else {
        _captchaHandoff.captchaEligibilityChanged();
      }
    }

    return AutofillGroup(
      onDisposeAction: AutofillContextAction.cancel,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          Align(
            alignment: Alignment.centerLeft,
            child: Text('YourTJ', style: GfTheme.typographyOf(context).display),
          ),
          const SizedBox(height: 24),
          Text(
            _title(l10n),
            style: GfTheme.typographyOf(context).display.copyWith(fontSize: 27),
          ),
          const SizedBox(height: 6),
          Text(
            _mode == _AuthMode.login && _returnTo != null && _returnTo != '/'
                ? l10n.authContinueAfterLogin
                : _subtitle(l10n),
            style: GfTheme.typographyOf(
              context,
            ).small.copyWith(color: colors.baseContent.withValues(alpha: 0.55)),
          ),
          const SizedBox(height: 22),
          if (_mode != _AuthMode.forgotPassword) ...<Widget>[
            GfSegmented<_AuthMode>(
              segments: <(String, _AuthMode)>[
                (l10n.loginModeLogin, _AuthMode.login),
                (l10n.loginModeRegister, _AuthMode.register),
              ],
              selected: _mode,
              onSelected: _switchMode,
            ),
            const SizedBox(height: 20),
          ],
          if (_mode != _AuthMode.forgotPassword) ...<Widget>[
            _withAuthFocusIntent(
              _usernameFocusNode,
              GfInput(
                controller: _username,
                focusNode: _usernameFocusNode,
                autofillHints: [
                  _mode == _AuthMode.login
                      ? AutofillHints.username
                      : AutofillHints.newUsername,
                ],
                autocorrect: false,
                enableSuggestions: false,
                textInputAction: TextInputAction.next,
                onSubmitted: (_) => _mode == _AuthMode.login
                    ? _passwordFocusNode.requestFocus()
                    : FocusScope.of(context).nextFocus(),
                labelText: _mode == _AuthMode.login
                    ? l10n.authUsernameOrEmail
                    : l10n.authUsername,
                prefixIcon: const Icon(Icons.person_outline, size: 20),
              ),
            ),
            const SizedBox(height: 12),
          ],
          if (_mode != _AuthMode.login) ...<Widget>[
            GfInput(
              controller: _email,
              keyboardType: TextInputType.emailAddress,
              autofillHints:
                  _mode == _AuthMode.register &&
                      _registration?.allowedDomains.isNotEmpty == true
                  ? const []
                  : const [AutofillHints.email],
              autocorrect: false,
              enableSuggestions: false,
              textInputAction: _mode == _AuthMode.forgotPassword
                  ? TextInputAction.done
                  : TextInputAction.next,
              onSubmitted: (_) => _mode == _AuthMode.forgotPassword
                  ? _submit()
                  : FocusScope.of(context).nextFocus(),
              labelText:
                  _mode == _AuthMode.register &&
                      _registration?.allowedDomains.isNotEmpty == true
                  ? l10n.authEmailPrefix
                  : l10n.authEmail,
              prefixIcon: const Icon(Icons.mail_outline, size: 20),
            ),
            if (_mode == _AuthMode.register &&
                _registration?.allowedDomains.isNotEmpty == true) ...[
              const SizedBox(height: 8),
              DropdownButtonFormField<String>(
                key: const Key('register-email-domain'),
                initialValue: _emailDomain,
                isExpanded: true,
                decoration: InputDecoration(labelText: l10n.authEmailDomain),
                items: _registration!.allowedDomains
                    .map(
                      (domain) => DropdownMenuItem(
                        value: domain,
                        child: Text('@$domain'),
                      ),
                    )
                    .toList(),
                onChanged: (value) => setState(() => _emailDomain = value),
              ),
            ],
            const SizedBox(height: 12),
          ],
          if (_mode != _AuthMode.forgotPassword) ...<Widget>[
            TapRegion(
              key: const Key('login-password-region'),
              onTapOutside: (_) => _onPasswordTapOutside(),
              child: _withAuthFocusIntent(
                _passwordFocusNode,
                GfInput(
                  controller: _password,
                  focusNode: _passwordFocusNode,
                  obscureText: true,
                  autofillHints: [
                    _mode == _AuthMode.login
                        ? AutofillHints.password
                        : AutofillHints.newPassword,
                  ],
                  autocorrect: false,
                  enableSuggestions: false,
                  textInputAction: TextInputAction.next,
                  onEditingComplete: _mode == _AuthMode.login
                      ? _completePasswordStage
                      : () => FocusScope.of(context).nextFocus(),
                  labelText: l10n.authPassword,
                  prefixIcon: const Icon(Icons.lock_outline, size: 20),
                  onChanged: _onPasswordChanged,
                ),
              ),
            ),
            if (_mode == _AuthMode.login) ...<Widget>[
              const SizedBox(height: 4),
              Align(
                alignment: Alignment.centerRight,
                child: GfButton(
                  label: l10n.authForgotPassword,
                  variant: GfButtonVariant.link,
                  size: GfButtonSize.small,
                  onPressed: () => _switchMode(_AuthMode.forgotPassword),
                ),
              ),
            ] else
              const SizedBox(height: 12),
          ],
          if (_mode == _AuthMode.register) ...[
            GfInput(
              controller: _confirmPassword,
              labelText: l10n.authConfirmPassword,
              obscureText: true,
              autofillHints: const [AutofillHints.newPassword],
              autocorrect: false,
              enableSuggestions: false,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _submit(),
            ),
            if (_registrationLoading) const LinearProgressIndicator(),
            if (_registrationError != null) ...[
              GfStatusMessage(message: _registrationError!),
              if (_registration == null)
                TextButton(
                  onPressed: _loadRegistration,
                  child: Text(l10n.commonRetry),
                ),
            ],
            if (_registration?.termsOfServiceEnabled == true ||
                _registration?.privacyPolicyEnabled == true) ...[
              Material(
                color: Colors.transparent,
                child: CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                  value: _agreed,
                  onChanged: (value) => setState(() => _agreed = value == true),
                  title: Text(l10n.authAgreePolicies),
                ),
              ),
              Wrap(
                children: [
                  if (_registration!.termsOfServiceEnabled)
                    TextButton(
                      onPressed: () => context.push('/terms'),
                      child: Text(l10n.siteInfoTerms),
                    ),
                  if (_registration!.privacyPolicyEnabled)
                    TextButton(
                      onPressed: () => context.push('/privacy'),
                      child: Text(l10n.siteInfoPrivacy),
                    ),
                ],
              ),
            ],
          ],
          if (showCaptcha) ...<Widget>[
            const SizedBox(height: 4),
            KeyedSubtree(
              key: const Key('login-captcha'),
              child: captcha != null
                  ? LayoutBuilder(
                      builder: (context, constraints) {
                        final image = ClipRRect(
                          borderRadius: BorderRadius.circular(8),
                          child: GfCaptchaImage(imageData: captcha.captchaImg),
                        );
                        // Leave room for a complete code at the user's text size.
                        final inputWidth = MediaQuery.textScalerOf(
                          context,
                        ).scale(140);
                        if (constraints.maxWidth < 140 + inputWidth) {
                          return Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              image,
                              const SizedBox(height: 12),
                              _buildCaptchaInput(l10n),
                            ],
                          );
                        }
                        return Row(
                          children: [
                            image,
                            const SizedBox(width: 12),
                            Expanded(child: _buildCaptchaInput(l10n)),
                          ],
                        );
                      },
                    )
                  : GfButton(
                      label: l10n.authGetCode,
                      variant: GfButtonVariant.ghost,
                      onPressed: _loadVisibleCaptcha,
                    ),
            ),
          ],
          if (_authController.phase == LoginPhase.needsTotp) ...<Widget>[
            const SizedBox(height: 12),
            GfInput(
              controller: _totp,
              keyboardType: TextInputType.number,
              autofillHints: const [AutofillHints.oneTimeCode],
              autocorrect: false,
              enableSuggestions: false,
              textInputAction: TextInputAction.done,
              labelText: l10n.authTwoFactorCode,
              prefixIcon: const Icon(Icons.shield_outlined, size: 20),
              onSubmitted: (_) => _submit(),
            ),
          ],
          if (_authController.error.isNotEmpty ||
              _oidcError.isNotEmpty ||
              _cacheError.isNotEmpty) ...<Widget>[
            const SizedBox(height: 12),
            GfStatusMessage(
              message: _cacheError.isNotEmpty
                  ? _cacheError
                  : _oidcError.isNotEmpty
                  ? _oidcError
                  : _authController.error,
            ),
          ],
          const SizedBox(height: 20),
          GfButton(
            label: _submitLabel(l10n),
            variant: GfButtonVariant.primary,
            size: GfButtonSize.extraLarge,
            expanded: true,
            loading: _authController.busy || _finishingAuthentication,
            onPressed:
                _authController.busy ||
                    _oidcBusy ||
                    _finishingAuthentication ||
                    (_mode == _AuthMode.register &&
                        (_registration == null ||
                            _registrationLoading ||
                            ((_registration!.termsOfServiceEnabled ||
                                    _registration!.privacyPolicyEnabled) &&
                                !_agreed)))
                ? null
                : _submit,
          ),
          if (_mode != _AuthMode.forgotPassword) _buildSsoOptions(l10n),
          if (_mode == _AuthMode.login && _registrationError != null)
            TextButton(
              onPressed: _loadRegistration,
              child: Text(l10n.commonRetry),
            ),
          if (_mode == _AuthMode.forgotPassword) ...<Widget>[
            const SizedBox(height: 8),
            GfButton(
              label: l10n.authBackToLogin,
              variant: GfButtonVariant.link,
              expanded: true,
              onPressed: () => _switchMode(_AuthMode.login),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildSsoOptions(AppLocalizations l10n) {
    final options = _registration;
    if (options == null) return const SizedBox.shrink();
    final providers = [
      if (options.tongjiReady) 'tongji',
      if (_mode == _AuthMode.login && options.googleReady) 'google',
      if (_mode == _AuthMode.login && options.githubUrl.isNotEmpty) 'github',
    ];
    if (providers.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const SizedBox(height: 20),
        Text(
          l10n.authSignInMethods,
          style: Theme.of(context).textTheme.labelLarge,
        ),
        const SizedBox(height: 8),
        for (final provider in providers) ...[
          OutlinedButton.icon(
            icon: GfSymbol(
              provider == 'tongji' ? 'graduation-cap' : provider,
              size: 22,
            ),
            style: OutlinedButton.styleFrom(
              minimumSize: const Size.fromHeight(48),
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
            ),
            label: Text(switch (provider) {
              'tongji' => l10n.loginTongji,
              'google' => l10n.loginGoogle,
              _ => l10n.loginGithub,
            }, textAlign: TextAlign.center),
            onPressed:
                _authController.busy || _oidcBusy || _finishingAuthentication
                ? null
                : () => _loginOidc(provider),
          ),
          const SizedBox(height: 8),
        ],
        if (options.tongjiReady) ...[
          Text(
            l10n.loginTongjiHint,
            style: Theme.of(context).textTheme.bodySmall,
          ),
          if (options.termsOfServiceEnabled ||
              options.privacyPolicyEnabled) ...[
            const SizedBox(height: 8),
            Text(
              l10n.loginTongjiPolicies,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            Wrap(
              children: [
                if (options.termsOfServiceEnabled)
                  TextButton(
                    onPressed: () => context.push('/terms'),
                    child: Text(l10n.siteInfoTerms),
                  ),
                if (options.privacyPolicyEnabled)
                  TextButton(
                    onPressed: () => context.push('/privacy'),
                    child: Text(l10n.siteInfoPrivacy),
                  ),
              ],
            ),
          ],
        ],
      ],
    );
  }

  String _submitLabel(AppLocalizations l10n) => switch (_mode) {
    _AuthMode.login =>
      _authController.phase == LoginPhase.needsTotp
          ? l10n.authVerify
          : l10n.authLoginTitle,
    _AuthMode.register => l10n.authRegisterTitle,
    _AuthMode.forgotPassword => l10n.authSendResetEmail,
  };

  Future<void> _submit() async {
    if (_authController.busy || _oidcBusy || _finishingAuthentication) return;
    if (_authController.phase == LoginPhase.needsTotp) {
      await _authController.submitTotp(_totp.text.trim());
      if (mounted && _authController.phase == LoginPhase.authenticated) {
        await _finishAuthentication(saveAutofill: true);
      }
      return;
    }
    await switch (_mode) {
      _AuthMode.login => _login(),
      _AuthMode.register => _register(),
      _AuthMode.forgotPassword => _forgotPassword(),
    };
  }
}
