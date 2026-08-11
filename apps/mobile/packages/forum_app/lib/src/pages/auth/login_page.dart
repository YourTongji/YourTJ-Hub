import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:auth/auth.dart';
import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../app_config.dart';
import '../../providers.dart';
import '../../current_user.dart';
import '../../theme_mode.dart';

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

class _LoginPageState extends ConsumerState<LoginPage> {
  final TextEditingController _username = TextEditingController();
  final TextEditingController _password = TextEditingController();
  final TextEditingController _email = TextEditingController();
  final TextEditingController _captcha = TextEditingController();
  final TextEditingController _totp = TextEditingController();

  _AuthMode _mode = _AuthMode.login;
  bool _oidcBusy = false;
  bool _finishingAuthentication = false;
  String _oidcError = '';
  // 登录放行前缓存清理失败的错误(清理成功或重试成功后清空)。
  String _cacheError = '';
  // 缓存清理失败后禁止返回旧 shell(其内存态可能含上一账号数据)。
  bool _authBlocked = false;

  late final AuthController _authController;
  late final GfApiClient _authClient;
  late final TokenStorage _authTokenStorage;

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
    });
  }

  /// 执行一次离线缓存清理;成功返回 true,失败返回 false(不抛出)。
  Future<bool> _clearOfflineCacheOnce() async {
    try {
      await clearOfflineCache(
        ref.read(offlineTopicCacheProvider),
        ref.read(offlineChatCacheProvider),
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
    _username.dispose();
    _password.dispose();
    _email.dispose();
    _captcha.dispose();
    _totp.dispose();
    _authController.dispose();
    super.dispose();
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
      await _authController.loadCaptcha();
    }
    if (mounted && _authController.phase == LoginPhase.authenticated) {
      await _finishAuthentication();
    }
  }

  Future<void> _register() async {
    final String? captchaId = _authController.captcha?.captchaId;
    final String captchaCode = _captcha.text.trim();
    await _authController.register(
      username: _username.text.trim(),
      email: _email.text.trim(),
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

  Future<void> _loginOidc() async {
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
      final bool ok = await controller.login();
      if (!mounted) return;
      if (!ok) {
        setState(() => _oidcError = controller.error);
        return;
      }
      await _finishAuthentication();
    } finally {
      controller.dispose();
      if (mounted) setState(() => _oidcBusy = false);
    }
  }

  /// 认证成功后的收尾:先确保旧账号缓存已清空,再把暂存的新会话提交到
  /// 安全存储。清理失败时新 token 从未持久化,因此重启也无法以新账号读取
  /// 旧账号离线数据。
  Future<void> _finishAuthentication() async {
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
      // 用 go('/') 替换整个导航栈,销毁 401 保留的旧 shell(及其内存态),
      // 避免新账号返回后看到上一账号的会话/消息数据。
      context.go('/');
    } finally {
      if (mounted) setState(() => _finishingAuthentication = false);
    }
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

  void _switchMode(_AuthMode mode) {
    _captcha.clear();
    setState(() {
      _mode = mode;
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
        backgroundColor: colors.base200,
        body: Stack(
          fit: StackFit.expand,
          children: <Widget>[
            const GfDotGridBackground(),
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
                    right: 8,
                    child: GfIconButton(
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
                          emphasized: true,
                          padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
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
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Align(
          alignment: Alignment.centerLeft,
          child: Image.asset(
            'assets/images/brand-default.png',
            width: 176,
            height: 44,
            fit: BoxFit.contain,
            alignment: Alignment.centerLeft,
          ),
        ),
        const SizedBox(height: 24),
        Text(
          _title(l10n),
          style: GfTheme.typographyOf(context).display.copyWith(fontSize: 27),
        ),
        const SizedBox(height: 6),
        Text(
          _subtitle(l10n),
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
          GfInput(
            controller: _username,
            labelText: _mode == _AuthMode.login
                ? l10n.authUsernameOrEmail
                : l10n.authUsername,
            prefixIcon: const Icon(Icons.person_outline, size: 20),
          ),
          const SizedBox(height: 12),
        ],
        if (_mode != _AuthMode.login) ...<Widget>[
          GfInput(
            controller: _email,
            keyboardType: TextInputType.emailAddress,
            labelText: l10n.authEmail,
            prefixIcon: const Icon(Icons.mail_outline, size: 20),
          ),
          const SizedBox(height: 12),
        ],
        if (_mode != _AuthMode.forgotPassword) ...<Widget>[
          GfInput(
            controller: _password,
            obscureText: true,
            labelText: l10n.authPassword,
            prefixIcon: const Icon(Icons.lock_outline, size: 20),
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
        if (_authController.phase == LoginPhase.needsCaptcha) ...<Widget>[
          const SizedBox(height: 4),
          if (_authController.captcha != null)
            Row(
              children: <Widget>[
                ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: Image.memory(
                    base64Decode(
                      _authController.captcha!.captchaImg.replaceFirst(
                        RegExp(r'^data:image/\w+;base64,'),
                        '',
                      ),
                    ),
                    width: 128,
                    height: 48,
                    fit: BoxFit.cover,
                    gaplessPlayback: true,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: GfInput(
                    controller: _captcha,
                    labelText: l10n.authCaptcha,
                  ),
                ),
              ],
            )
          else
            GfButton(
              label: l10n.authGetCode,
              variant: GfButtonVariant.ghost,
              onPressed: _authController.loadCaptcha,
            ),
        ],
        if (_authController.phase == LoginPhase.needsTotp) ...<Widget>[
          const SizedBox(height: 12),
          GfInput(
            controller: _totp,
            keyboardType: TextInputType.number,
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
              _authController.busy || _oidcBusy || _finishingAuthentication
              ? null
              : _submit,
        ),
        if (_mode == _AuthMode.login) ...<Widget>[
          const SizedBox(height: 12),
          GfButton(
            label: l10n.authOidcLogin,
            variant: GfButtonVariant.outline,
            size: GfButtonSize.extraLarge,
            expanded: true,
            loading: _oidcBusy || _finishingAuthentication,
            onPressed:
                _authController.busy || _oidcBusy || _finishingAuthentication
                ? null
                : _loginOidc,
          ),
        ],
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
    if (_authController.phase == LoginPhase.needsTotp) {
      await _authController.submitTotp(_totp.text.trim());
      if (mounted && _authController.phase == LoginPhase.authenticated) {
        await _finishAuthentication();
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
