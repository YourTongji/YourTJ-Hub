import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:auth/auth.dart';
import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../app_config.dart';
import '../../providers.dart';

/// 登录页模式。
enum _AuthMode { login, register, forgotPassword }

/// 登录/注册/找回密码页(web auth.login 的移动端形态)。
///
/// 密码登录:公钥 → RSA-OAEP 加密 → 登录 → (验证码/TOTP 挑战)。
/// 统一身份:OIDC(Casdoor)经 AppAuth + 后端 exchange 兑换。
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

  late final AuthController _authController;

  @override
  void initState() {
    super.initState();
    _authController = AuthController(
      authRepository: AuthRepository(ref.read(apiClientProvider)),
      apiClient: ref.read(apiClientProvider),
      tokenStorage: ref.read(tokenStorageProvider),
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
      Navigator.of(context).pop(true);
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
    final controller = OidcController(
      authRepository: AuthRepository(ref.read(apiClientProvider)),
      tokenStorage: ref.read(tokenStorageProvider),
      issuer: AppConfig.oidcIssuer,
      clientId: AppConfig.oidcClientId,
    );
    final ok = await controller.login();
    if (mounted && ok) {
      Navigator.of(context).pop(true);
    }
  }

  String _title(AppLocalizations l10n) => switch (_mode) {
    _AuthMode.login => l10n.authLoginTitle,
    _AuthMode.register => l10n.authRegisterTitle,
    _AuthMode.forgotPassword => l10n.authForgotTitle,
  };

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: GfAppBar(title: Text(_title(l10n))),
      body: ListenableBuilder(
        listenable: _authController,
        builder: (context, _) {
          return SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  l10n.loginWelcome,
                  style: GfTheme.typographyOf(context).display,
                ),
                const SizedBox(height: 24),
                // 模式切换(web gf-segmented)。
                GfSegmented<_AuthMode>(
                  segments: [
                    (l10n.loginModeLogin, _AuthMode.login),
                    (l10n.loginModeRegister, _AuthMode.register),
                    (l10n.loginModeForgot, _AuthMode.forgotPassword),
                  ],
                  selected: _mode,
                  onSelected: (value) => setState(() => _mode = value),
                ),
                const SizedBox(height: 20),
                // 用户名(登录/注册)。
                if (_mode != _AuthMode.forgotPassword) ...[
                  GfInput(
                    controller: _username,
                    labelText: l10n.authUsernameOrEmail,
                  ),
                  const SizedBox(height: 12),
                ],
                // 邮箱(注册/找回)。
                if (_mode != _AuthMode.login) ...[
                  GfInput(
                    controller: _email,
                    keyboardType: TextInputType.emailAddress,
                    labelText: l10n.authEmail,
                  ),
                  const SizedBox(height: 12),
                ],
                // 密码(登录/注册)。
                if (_mode != _AuthMode.forgotPassword) ...[
                  GfInput(
                    controller: _password,
                    obscureText: true,
                    labelText: l10n.authPassword,
                  ),
                  const SizedBox(height: 12),
                ],
                // 验证码挑战。
                if (_authController.phase == LoginPhase.needsCaptcha) ...[
                  const SizedBox(height: 12),
                  if (_authController.captcha != null)
                    Row(
                      children: [
                        Image.memory(
                          base64Decode(
                            _authController.captcha!.captchaImg.replaceFirst(
                              RegExp(r'^data:image/\w+;base64,'),
                              '',
                            ),
                          ),
                          width: 140,
                          height: 48,
                          fit: BoxFit.cover,
                          gaplessPlayback: true,
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
                // TOTP 挑战。
                if (_authController.phase == LoginPhase.needsTotp) ...[
                  const SizedBox(height: 12),
                  GfInput(
                    controller: _totp,
                    keyboardType: TextInputType.number,
                    labelText: l10n.authTwoFactorCode,
                    onSubmitted: (_) =>
                        _authController.submitTotp(_totp.text.trim()),
                  ),
                ],
                if (_authController.error.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Text(
                    _authController.error,
                    style: GfTheme.typographyOf(
                      context,
                    ).small.copyWith(color: colors.error),
                  ),
                ],
                const SizedBox(height: 20),
                GfButton(
                  label: _submitLabel(l10n),
                  variant: GfButtonVariant.primary,
                  expanded: true,
                  loading: _authController.busy,
                  onPressed: _authController.busy ? null : _submit,
                ),
                if (_mode == _AuthMode.login) ...[
                  const SizedBox(height: 12),
                  GfButton(
                    label: l10n.authCasdoorLogin,
                    variant: GfButtonVariant.outline,
                    expanded: true,
                    loading: _authController.busy,
                    onPressed: _authController.busy ? null : _loginOidc,
                  ),
                ],
              ],
            ),
          );
        },
      ),
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

  Future<void> _submit() {
    if (_authController.phase == LoginPhase.needsTotp) {
      return _authController.submitTotp(_totp.text.trim());
    }
    return switch (_mode) {
      _AuthMode.login => _login(),
      _AuthMode.register => _register(),
      _AuthMode.forgotPassword => _forgotPassword(),
    };
  }
}
