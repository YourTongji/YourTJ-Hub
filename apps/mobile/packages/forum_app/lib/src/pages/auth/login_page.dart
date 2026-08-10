import 'dart:async';
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
import '../../theme_mode.dart';

/// 登录页模式。
enum _AuthMode { login, register, forgotPassword }

/// 登录/注册/找回密码页(web auth.login 的移动端形态)。
///
/// 密码登录:公钥 → RSA-OAEP 加密 → 登录 → (验证码/TOTP 挑战)。
/// 统一身份：论坛内建 OIDC Provider 经 AppAuth + 后端 exchange 兑换。
/// 注册/找回密码:复用 AuthController 的 register/forgotPassword。
class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

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
  String _oidcError = '';

  late final AuthController _authController;

  @override
  void initState() {
    super.initState();
    _authController = AuthController(
      authRepository: AuthRepository(ref.read(apiClientProvider)),
      apiClient: ref.read(apiClientProvider),
      tokenStorage: ref.read(tokenStorageProvider),
    );
    // 进入登录页即清空上一账号的离线缓存(话题/会话/私信),防止同一设备
    // 换账号后读到上一账号的缓存数据;失败静默。
    unawaited(
      clearOfflineCacheQuietly(
        ref.read(offlineTopicCacheProvider),
        ref.read(offlineChatCacheProvider),
      ),
    );
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
      _finishAuthentication();
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
      authRepository: AuthRepository(ref.read(apiClientProvider)),
      tokenStorage: ref.read(tokenStorageProvider),
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
      _finishAuthentication();
    } finally {
      controller.dispose();
      if (mounted) setState(() => _oidcBusy = false);
    }
  }

  void _finishAuthentication() {
    if (!mounted) return;
    final NavigatorState navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop(true);
      return;
    }
    context.go('/');
  }

  void _leaveAuth() {
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

    return Scaffold(
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
            _oidcError.isNotEmpty) ...<Widget>[
          const SizedBox(height: 12),
          GfStatusMessage(
            message: _oidcError.isNotEmpty ? _oidcError : _authController.error,
          ),
        ],
        const SizedBox(height: 20),
        GfButton(
          label: _submitLabel(l10n),
          variant: GfButtonVariant.primary,
          size: GfButtonSize.extraLarge,
          expanded: true,
          loading: _authController.busy,
          onPressed: _authController.busy || _oidcBusy ? null : _submit,
        ),
        if (_mode == _AuthMode.login) ...<Widget>[
          const SizedBox(height: 12),
          GfButton(
            label: l10n.authOidcLogin,
            variant: GfButtonVariant.outline,
            size: GfButtonSize.extraLarge,
            expanded: true,
            loading: _oidcBusy,
            onPressed: _authController.busy || _oidcBusy ? null : _loginOidc,
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
        _finishAuthentication();
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
