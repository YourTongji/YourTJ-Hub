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
import '../../apple/apple_sign_in.dart';
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
import 'sign_in_methods_sheet.dart';

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
  final Object _authInputGroup = Object();
  final FocusNode _usernameFocusNode = FocusNode();
  final FocusNode _passwordFocusNode = FocusNode();
  final FocusNode _confirmPasswordFocusNode = FocusNode();
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
  // Restores focus to the "more sign-in methods" control after its sheet
  // closes without a choice.
  final FocusNode _moreMethodsFocusNode = FocusNode();
  bool _finishingAuthentication = false;
  bool _passwordInteractionStarted = false;
  // 明文显示是逐字段的显式用户选择,默认保持遮蔽。
  bool _passwordVisible = false;
  bool _confirmPasswordVisible = false;
  bool _loginCaptchaRevealed = false;
  bool _captchaRevealFrameScheduled = false;
  bool _captchaFocusEligible = false;
  bool _suppressPasswordTapOutside = false;
  bool _captchaRefreshing = false;
  bool _captchaCodeMissing = false;
  Future<void>? _captchaLoadFuture;
  final Stopwatch _authImeClock = Stopwatch()..start();
  late final AuthImeStabilizer<FocusNode> _authIme;
  late final LoginCaptchaForwardHandoff _captchaHandoff;
  String _oidcError = '';
  // 登录放行前缓存清理失败的错误(清理成功或重试成功后清空)。
  String _cacheError = '';
  // 缓存清理失败后禁止返回旧 shell(其内存态可能含上一账号数据)。
  bool _authBlocked = false;
  // 本次登录流程是否已推进会话世代(入口存在旧会话,或认证成功)。
  bool _sessionBoundaryAdvanced = false;

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
          registrationErrorMessage: (error) => mounted
              ? resolveErrorMessage(AppLocalizations.of(context), error)
              : null,
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
      requestCaptchaFocus: () {
        _authIme.focusRequested(_captchaFocusNode);
        _captchaFocusNode.requestFocus();
      },
      schedule: _scheduleCaptchaForward,
    );
    WidgetsBinding.instance.addObserver(this);
    _usernameFocusNode.addListener(_onUsernameFocusChanged);
    _passwordFocusNode.addListener(_onPasswordFocusChanged);
    _captchaFocusNode.addListener(_onCaptchaFocusChanged);

    // 进入登录页时,若仍持有旧会话(登出/401 竞态),先使旧会话在途写入
    // 失效,再清空缓存。两者都延迟到首帧后执行(避免在 widget 构建期修改
    // provider),且世代失效必须先于清库,保证不变量:
    //   - 失效前提交的旧会话写入会被随后的清库清掉;
    //   - 失效后提交的写入会被世代守卫拦截。
    // 游客打开登录页不是账号切换:取消后必须回到原页面继续浏览,因此不能
    // 推进世代——下方仍存活的页面(如话题详情)会因世代失配而永久置空。
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      // 使缓存的当前用户身份失效(旧账号 id 不再被后续新 shell 读取)。
      ref.invalidate(currentUserProvider);
      unawaited(_beginLoginSessionBoundary());
      _loadRegistration();
      _authIme.metricsChanged();
    });
  }

  /// 进入登录页的会话边界:只在旧会话仍存在时推进世代(生成新的账号边界)。
  /// 游客入口不推进;若之后认证成功,[_finishAuthentication] 会补上这次账号
  /// 切换的边界。
  ///
  /// 缓存清理与原语义一致:进入登录页即启动(上次登出/401 清理失败时每次
  /// 进入都会重试),且不因 token 读取期间页面被返回而跳过;清理句柄在
  /// await 前捕获,避免对已卸载页面的 ref 访问。
  Future<void> _beginLoginSessionBoundary() async {
    final storage = ref.read(tokenStorageProvider);
    final epochNotifier = ref.read(offlineCacheEpochProvider.notifier);
    final topicCache = ref.read(offlineTopicCacheProvider);
    final chatCache = ref.read(offlineChatCacheProvider);
    final widgetBridge = ref.read(scheduleWidgetBridgeProvider);
    final bool hasSession = await hasSessionToken(storage);
    if (hasSession && !_sessionBoundaryAdvanced) {
      _sessionBoundaryAdvanced = true;
      epochNotifier.invalidate();
    }
    final Future<bool> future = _runCacheClear(
      () => clearOfflineCache(topicCache, chatCache, widgetBridge),
    );
    // 认证提交若已抢先启动清库,复用它而不是替换。
    if (mounted) _cacheClearFuture ??= future;
  }

  /// 执行一次离线缓存清理;成功返回 true,失败返回 false(不抛出)。
  Future<bool> _clearOfflineCacheOnce() => _runCacheClear(
    () => clearOfflineCache(
      ref.read(offlineTopicCacheProvider),
      ref.read(offlineChatCacheProvider),
      ref.read(scheduleWidgetBridgeProvider),
    ),
  );

  Future<bool> _runCacheClear(Future<void> Function() clear) async {
    try {
      await clear();
      return true;
    } catch (error, stack) {
      // Keep the login failure diagnosable without printing storage payloads,
      // credentials or exception messages that may embed private values.
      debugPrint('Session cache cleanup failed: ${error.runtimeType}');
      debugPrintStack(stackTrace: stack);
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
    _confirmPasswordFocusNode.dispose();
    _captchaFocusNode.dispose();
    _moreMethodsFocusNode.dispose();
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
    if (_usernameFocusNode.hasFocus) _captchaHandoff.cancel();
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

  void _completePasswordStage({bool focusCaptcha = false}) {
    if (!_passwordInteractionStarted || _mode != _AuthMode.login) return;

    // Revealing the next field is separate from entering it. Only IME Next
    // authorizes an automatic handoff; a dismissal or outside tap must not
    // reopen the keyboard or override the user's explicitly selected control.
    if (focusCaptcha) {
      _authIme.cancel();
      _captchaHandoff.begin();
    } else {
      _captchaHandoff.cancel();
      if (_loginCaptchaRevealed) return;
    }
    _loginCaptchaRevealed = true;
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
    _captchaHandoff.cancel();
    if (!explicitTarget) {
      _authIme.cancel();
      _passwordFocusNode.unfocus();
    }
    _completePasswordStage();
  }

  // One group lets a direct field-to-field tap keep its target, while a tap
  // on the surrounding form dismisses every auth keyboard on iOS and Android.
  Widget _withAuthInputRegion(Widget child) => TapRegion(
    groupId: _authInputGroup,
    onTapOutside: (_) {
      _authIme.cancel();
      _captchaHandoff.cancel();
      FocusScope.of(context).unfocus();
    },
    child: child,
  );

  Widget _withAuthFocusIntent(FocusNode target, Widget child) {
    return _withAuthInputRegion(
      Listener(
        behavior: HitTestBehavior.translucent,
        onPointerDown: (_) => _captureAuthFocusIntent(target),
        child: child,
      ),
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
      onChanged: (_) {
        if (_captchaCodeMissing) setState(() => _captchaCodeMissing = false);
      },
    );
    return _mode == _AuthMode.login
        ? _withAuthFocusIntent(_captchaFocusNode, input)
        : _withAuthInputRegion(input);
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

  /// 手动刷新验证码:请求新的一张,并在新挑战到达后清空旧输入。
  ///
  /// 单飞由 [_buildCaptchaChallenge] 保证:刷新在途时验证码图不可点击
  /// (onTap 为 null),因此这里不会重入。刷新失败时保留旧图与仍有效的
  /// 旧输入,由错误条提示,用户可再次点击重试。
  void _refreshCaptcha() {
    final CaptchaPayload? previous = _authController.captcha;
    setState(() => _captchaRefreshing = true);
    unawaited(
      _loadCaptchaIfNeeded(
            preservePhaseOnError: true,
            silentOnError: false,
            force: true,
          )
          .then((_) {
            if (!mounted) return;
            // 只有新挑战真正到达才作废旧输入:失败时旧图与旧码在服务端
            // 仍然有效,用户可以直接重试提交或再次点击刷新。
            if (!identical(_authController.captcha, previous)) {
              _captcha.clear();
              if (_captchaCodeMissing) {
                setState(() => _captchaCodeMissing = false);
              }
            }
            _captchaHandoff.captchaEligibilityChanged();
          })
          .whenComplete(() {
            if (mounted) setState(() => _captchaRefreshing = false);
          }),
    );
  }

  /// 可点击刷新的验证码图。
  ///
  /// 点击请求新一张并清空旧输入;在途时显示进度并忽略再次点击。图片属于
  /// 输入组,点刷新不会触发 TapRegion 的 onTapOutside 收起键盘。
  Widget _buildCaptchaChallenge(CaptchaPayload captcha, AppLocalizations l10n) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Widget image = ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: GfCaptchaImage(imageData: captcha.captchaImg),
    );
    return TapRegion(
      groupId: _authInputGroup,
      child: MergeSemantics(
        child: Semantics(
          label: l10n.authRefreshCaptcha,
          button: true,
          enabled: !_captchaRefreshing,
          child: InkWell(
            key: const Key('login-captcha-refresh'),
            onTap: _captchaRefreshing ? null : _refreshCaptcha,
            borderRadius: BorderRadius.circular(8),
            child: Stack(
              alignment: Alignment.center,
              children: <Widget>[
                image,
                if (_captchaRefreshing)
                  Positioned.fill(
                    child: ColoredBox(
                      color: colors.base100.withValues(alpha: 0.55),
                      child: const Center(
                        child: GfProgressIndicator(strokeWidth: 2),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
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
    final email = _registrationEmail(options, l10n);
    if (email == null) return;
    final String? captchaId = _authController.captcha?.captchaId;
    final String captchaCode = _captcha.text.trim();
    final messageCode = await _authController.register(
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
      final l10n = AppLocalizations.of(context);
      showGfToast(
        context,
        messageCode == 'auth.register.emailVerify'
            ? l10n.authRegisterEmailVerify
            : l10n.authRegisterSuccess,
      );
      setState(() => _mode = _AuthMode.login);
    }
  }

  String? _registrationEmail(LoginPageProps options, AppLocalizations l10n) {
    final entered = _email.text.trim();
    if (options.allowedDomains.isEmpty) return entered;

    // Pasting a full address must not append a second domain. Only a published
    // domain can be selected, and a foreign address is never silently rewritten.
    final parts = entered.split('@');
    final prefix = parts.first;
    final malformed =
        parts.length > 2 || prefix.isEmpty || RegExp(r'\s').hasMatch(entered);
    final domain = parts.length == 1
        ? _emailDomain
        : options.allowedDomains
              .where(
                (domain) => domain.toLowerCase() == parts.last.toLowerCase(),
              )
              .firstOrNull;
    if (malformed || domain == null) {
      setState(() {
        _registrationError = resolveErrorMessage(
          l10n,
          ApiException(
            fallbackMessage: 'Invalid email address',
            messageCode: malformed || parts.last.isEmpty
                ? 'auth.emailDomain.invalid'
                : 'auth.emailDomain.notAllowed',
          ),
        );
      });
      return null;
    }
    if (parts.length == 2) {
      setState(() {
        _email.text = prefix;
        _emailDomain = domain;
      });
    }
    return '$prefix@$domain';
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

  Future<void> _loginApple() async {
    if (_oidcBusy || _authController.busy || _finishingAuthentication) return;
    setState(() {
      _oidcBusy = true;
      _oidcError = '';
    });
    try {
      final proof = await ref.read(appleSignInProvider).authorize();
      if (!mounted || proof == null) return;
      final token = await AuthRepository(
        _authClient,
      ).appleExchange(proof.request);
      if (!mounted) return;
      await _authTokenStorage.write(token);
      await _finishAuthentication(appleUserIdentifier: proof.userIdentifier);
    } catch (error) {
      await _authTokenStorage.clear();
      if (mounted) {
        setState(
          () => _oidcError = resolveErrorMessage(
            AppLocalizations.of(context),
            error,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _oidcBusy = false);
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
  Future<void> _finishAuthentication({
    bool saveAutofill = false,
    String? appleUserIdentifier,
  }) async {
    if (!mounted || _finishingAuthentication) return;
    setState(() => _finishingAuthentication = true);
    try {
      // 游客进入登录页没有推进会话世代,认证成功才是账号切换:先补上
      // 边界,再确保缓存已清空,最后才提交新令牌。游客会话只写公开数据,
      // 入口清库之后没有再落盘账号私有数据,因此这里无需重复清库。
      if (!_sessionBoundaryAdvanced) {
        _sessionBoundaryAdvanced = true;
        ref.read(offlineCacheEpochProvider.notifier).invalidate();
      }
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
        if (supportsNativeAppleSignIn) {
          await ref
              .read(appleSignInProvider)
              .rememberSession(token, appleUserIdentifier);
        }
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
    _authIme.cancel();
    _captchaHandoff.cancel();
    FocusScope.of(context).unfocus();
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
    FocusScope.of(context).unfocus();
    if (mode == _AuthMode.register && _registration == null) {
      _loadRegistration();
    }
    _captcha.clear();
    setState(() {
      _mode = mode;
      _loginCaptchaRevealed = false;
      _passwordInteractionStarted = false;
      _suppressPasswordTapOutside = false;
      _captchaCodeMissing = false;
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
        body: SafeArea(
          child: Column(
            children: <Widget>[
              // Keep navigation outside the form's scroll/hit-test area, even
              // when an error, large text or the keyboard makes the form tall.
              SizedBox(
                height: 48,
                child: Stack(
                  children: <Widget>[
                    Positioned(
                      top: 4,
                      left: 8,
                      child: GfIconButton(
                        symbol: 'chevron-left',
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
                        symbol: brightness == Brightness.dark ? 'sun' : 'moon',
                        tooltip: brightness == Brightness.dark
                            ? l10n.commonUseLightTheme
                            : l10n.commonUseDarkTheme,
                        onPressed: () => ref
                            .read(themeModeProvider.notifier)
                            .toggleDark(brightness != Brightness.dark),
                      ),
                    ),
                  ],
                ),
              ),
              Expanded(
                child: LayoutBuilder(
                  builder: (BuildContext context, BoxConstraints constraints) {
                    // A short form area - a small phone, or the keyboard
                    // covering it - drops the decorative header so the form
                    // and its sign-in methods stay reachable (issue #888).
                    final bool compactHeader =
                        constraints.maxHeight < _compactHeaderHeight;
                    return Center(
                      child: SingleChildScrollView(
                        padding: const EdgeInsets.fromLTRB(20, 4, 20, 12),
                        child: ConstrainedBox(
                          constraints: const BoxConstraints(maxWidth: 520),
                          child: GfCard(
                            showDivider: false,
                            padding: const EdgeInsets.fromLTRB(4, 8, 4, 12),
                            child: ListenableBuilder(
                              listenable: _authController,
                              builder: (BuildContext context, Widget? child) {
                                return _buildCardContent(
                                  context,
                                  l10n,
                                  colors,
                                  compactHeader: compactHeader,
                                );
                              },
                            ),
                          ),
                        ),
                      ),
                    );
                  },
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// Form areas shorter than this drop the brand lockup and subtitle.
  static const double _compactHeaderHeight = 700;

  Widget _buildCardContent(
    BuildContext context,
    AppLocalizations l10n,
    GfColors colors, {
    required bool compactHeader,
  }) {
    final bool registrationHeader = _mode == _AuthMode.register;
    final bool showBrand = !compactHeader;
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
          // Keep the fields reachable in compact form areas by hiding the
          // decorative brand lockup.
          if (showBrand) ...[
            Align(
              alignment: Alignment.center,
              child: Transform.translate(
                // The wordmark asset's visible alpha bounds sit about 2 px
                // right of its canvas center; correct that optical offset.
                offset: const Offset(-2, 0),
                child: Image.asset(
                  Theme.of(context).brightness == Brightness.dark
                      ? 'assets/images/brand-default-dark.webp'
                      : 'assets/images/brand-default.webp',
                  width: 192,
                  height: 44,
                  fit: BoxFit.contain,
                  semanticLabel: 'YourTJ',
                ),
              ),
            ),
            const SizedBox(height: 16),
          ],
          Text(_title(l10n), style: GfTheme.typographyOf(context).display),
          if (!compactHeader) ...[
            SizedBox(height: registrationHeader ? 8 : 6),
            Text(
              _mode == _AuthMode.login && _returnTo != null && _returnTo != '/'
                  ? l10n.authContinueAfterLogin
                  : _subtitle(l10n),
              style: GfTheme.typographyOf(context).small.copyWith(
                color: colors.baseContent.withValues(alpha: 0.64),
              ),
            ),
          ],
          SizedBox(height: registrationHeader ? 16 : 12),
          if (_mode != _AuthMode.forgotPassword) ...<Widget>[
            GfSegmented<_AuthMode>(
              segments: <(String, _AuthMode)>[
                (l10n.loginModeLogin, _AuthMode.login),
                (l10n.loginModeRegister, _AuthMode.register),
              ],
              selected: _mode,
              onSelected: _switchMode,
            ),
            const SizedBox(height: 12),
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
                onEditingComplete: () {
                  if (_mode == _AuthMode.login) {
                    _captchaHandoff.cancel();
                    _authIme.focusRequested(_passwordFocusNode);
                    _passwordFocusNode.requestFocus();
                  } else {
                    FocusScope.of(context).nextFocus();
                  }
                },
                labelText: _mode == _AuthMode.login
                    ? l10n.authUsernameOrEmail
                    : l10n.authUsername,
                prefixIcon: const GfSymbol('user-round', size: 20),
              ),
            ),
            const SizedBox(height: 8),
          ],
          if (_mode != _AuthMode.login) ...<Widget>[
            _withAuthInputRegion(
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
                onEditingComplete: _mode == _AuthMode.register
                    ? () => FocusScope.of(context).nextFocus()
                    : null,
                onSubmitted: _mode == _AuthMode.forgotPassword
                    ? (_) => _submit()
                    : null,
                labelText:
                    _mode == _AuthMode.register &&
                        _registration?.allowedDomains.isNotEmpty == true
                    ? l10n.authEmailPrefix
                    : l10n.authEmail,
                prefixIcon: const GfSymbol('mail', size: 20),
              ),
            ),
            if (_mode == _AuthMode.register &&
                _registration?.allowedDomains.isNotEmpty == true) ...[
              const SizedBox(height: 8),
              DropdownButtonFormField<String>(
                key: const Key('register-email-domain'),
                initialValue: _emailDomain,
                isExpanded: true,
                itemHeight: null,
                borderRadius: BorderRadius.circular(16),
                icon: const GfSymbol('chevron-down', size: 20),
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
                  obscureText: !_passwordVisible,
                  autofillHints: [
                    _mode == _AuthMode.login
                        ? AutofillHints.password
                        : AutofillHints.newPassword,
                  ],
                  autocorrect: false,
                  enableSuggestions: false,
                  textInputAction: TextInputAction.next,
                  onEditingComplete: _mode == _AuthMode.login
                      ? () => _completePasswordStage(focusCaptcha: true)
                      // 显示开关也是可聚焦后缀控件,nextFocus 会先停在它上面;
                      // 注册流程显式前进到确认密码。
                      : _confirmPasswordFocusNode.requestFocus,
                  labelText: l10n.authPassword,
                  prefixIcon: const GfSymbol('key-round', size: 20),
                  suffixIcon: _PasswordVisibilityToggle(
                    key: const Key('login-password-visibility'),
                    visible: _passwordVisible,
                    showLabel: l10n.authShowPassword,
                    hideLabel: l10n.authHidePassword,
                    onPressed: () =>
                        setState(() => _passwordVisible = !_passwordVisible),
                  ),
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
            _withAuthInputRegion(
              GfInput(
                controller: _confirmPassword,
                focusNode: _confirmPasswordFocusNode,
                labelText: l10n.authConfirmPassword,
                obscureText: !_confirmPasswordVisible,
                autofillHints: const [AutofillHints.newPassword],
                autocorrect: false,
                enableSuggestions: false,
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => _submit(),
                suffixIcon: _PasswordVisibilityToggle(
                  key: const Key('register-confirm-password-visibility'),
                  visible: _confirmPasswordVisible,
                  showLabel: l10n.authShowPassword,
                  hideLabel: l10n.authHidePassword,
                  onPressed: () => setState(
                    () => _confirmPasswordVisible = !_confirmPasswordVisible,
                  ),
                ),
              ),
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
                        final image = _buildCaptchaChallenge(captcha, l10n);
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
            _withAuthInputRegion(
              GfInput(
                controller: _totp,
                keyboardType: TextInputType.number,
                autofillHints: const [AutofillHints.oneTimeCode],
                autocorrect: false,
                enableSuggestions: false,
                textInputAction: TextInputAction.done,
                labelText: l10n.authTwoFactorCode,
                prefixIcon: const GfSymbol('shield-check', size: 20),
                onSubmitted: (_) => _submit(),
              ),
            ),
          ],
          if (_captchaCodeMissing ||
              _authController.error.isNotEmpty ||
              _oidcError.isNotEmpty ||
              _cacheError.isNotEmpty) ...<Widget>[
            const SizedBox(height: 12),
            GfStatusMessage(
              message: _captchaCodeMissing
                  ? l10n.authCaptchaRequired
                  : _cacheError.isNotEmpty
                  ? _cacheError
                  : _oidcError.isNotEmpty
                  ? _oidcError
                  : _authController.error,
            ),
          ],
          const SizedBox(height: 12),
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
                    _captchaRefreshing ||
                    (_mode == _AuthMode.register &&
                        (_registration == null ||
                            _registrationLoading ||
                            ((_registration!.termsOfServiceEnabled ||
                                    _registration!.privacyPolicyEnabled) &&
                                !_agreed)))
                ? null
                : _submit,
          ),
          if (_mode != _AuthMode.forgotPassword) _buildSignInMethods(l10n),
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

  /// The secondary providers stay folded so the common phone viewport shows
  /// the whole form without scrolling (issue #888).
  Widget _buildSignInMethods(AppLocalizations l10n) {
    final LoginPageProps? options = _registration;
    if (options == null) return const SizedBox.shrink();
    final List<SignInMethod> methods = availableSignInMethods(
      options,
      loginMode: _mode == _AuthMode.login,
    );
    if (methods.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        const SizedBox(height: 12),
        OutlinedButton.icon(
          key: const Key('login-more-methods'),
          focusNode: _moreMethodsFocusNode,
          icon: const GfSymbol('chevron-down', size: 22),
          label: Text(l10n.authMoreSignInMethods, textAlign: TextAlign.center),
          onPressed:
              _authController.busy || _oidcBusy || _finishingAuthentication
              ? null
              : () => _showSignInMethods(methods, options),
        ),
      ],
    );
  }

  /// Dismisses the form keyboard, then owns the chosen flow here so busy state
  /// and provider errors stay on the page rather than in a closed sheet.
  Future<void> _showSignInMethods(
    List<SignInMethod> methods,
    LoginPageProps options,
  ) async {
    _authIme.cancel();
    _captchaHandoff.cancel();
    FocusScope.of(context).unfocus();
    final SignInMethod? method = await showGfBottomSheet<SignInMethod>(
      context,
      builder: (_) => SignInMethodsSheet(
        methods: methods,
        termsOfServiceEnabled: options.termsOfServiceEnabled,
        privacyPolicyEnabled: options.privacyPolicyEnabled,
      ),
    );
    if (!mounted) return;
    if (method == null) {
      _restoreMoreMethodsFocus();
    } else if (method == SignInMethod.apple) {
      await _loginApple();
    } else {
      await _loginOidc(method.name);
    }
  }

  /// A dismissed sheet hands focus back to its control. A chosen provider
  /// starts its own flow instead, so the control must not take focus back.
  void _restoreMoreMethodsFocus() {
    final Duration exit = GfMotion.duration(context, GfMotion.layout);
    unawaited(
      Future<void>.delayed(exit, () {
        if (mounted) _moreMethodsFocusNode.requestFocus();
      }),
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

  /// 服务端已要求验证码,但输入框为空:此时提交必然被 `captchaRequired`
  /// 拒绝并消耗一次登录限流额度,Web 端同样先做本地校验。
  bool get _requiresCaptchaCode =>
      _authController.phase == LoginPhase.needsCaptcha &&
      _captcha.text.trim().isEmpty;

  Future<void> _submit() async {
    if (_authController.busy || _oidcBusy || _finishingAuthentication) return;
    // 刷新在途:挑战与已输入内容都在更换中(按钮同样是禁用态,这里兜底
    // 键盘提交路径),提交只会浪费一次登录尝试。
    if (_captchaRefreshing) return;
    if (_requiresCaptchaCode) {
      setState(() => _captchaCodeMissing = true);
      return;
    }
    if (_captchaCodeMissing) setState(() => _captchaCodeMissing = false);
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

/// 密码字段的明文/遮蔽切换。
///
/// 无状态:显示状态由所属页面持有,标签随状态变化以便读屏播报当前动作。
class _PasswordVisibilityToggle extends StatelessWidget {
  const _PasswordVisibilityToggle({
    super.key,
    required this.visible,
    required this.showLabel,
    required this.hideLabel,
    required this.onPressed,
  });

  final bool visible;
  final String showLabel;
  final String hideLabel;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final String label = visible ? hideLabel : showLabel;
    // 与发布页工具栏同一模式:合并语义让读屏把标签、按钮身份和开关状态
    // 读成一个控件;按钮保持可聚焦,键盘/D-pad 也能到达显示开关。
    return MergeSemantics(
      child: Semantics(
        toggled: visible,
        child: GfIconButton(
          symbol: visible ? 'eye-off' : 'eye',
          tooltip: label,
          size: 44,
          iconSize: 20,
          onPressed: onPressed,
        ),
      ),
    );
  }
}
