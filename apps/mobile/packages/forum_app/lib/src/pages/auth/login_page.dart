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
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(AppLocalizations.of(context).authRegisterSuccess),
        ),
      );
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
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(AppLocalizations.of(context).authResetEmailSent),
        ),
      );
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
      appBar: AppBar(title: Text(_title(l10n))),
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
                  style: TextStyle(
                    fontSize: 22,
                    fontWeight: FontWeight.w700,
                    color: colors.baseContent,
                  ),
                ),
                const SizedBox(height: 24),
                // 模式切换。
                Align(
                  alignment: Alignment.centerLeft,
                  child: GfTabBar(
                    tabs: <GfTab>[
                      GfTab(label: l10n.loginModeLogin, value: _AuthMode.login),
                      GfTab(
                        label: l10n.loginModeRegister,
                        value: _AuthMode.register,
                      ),
                      GfTab(
                        label: l10n.loginModeForgot,
                        value: _AuthMode.forgotPassword,
                      ),
                    ],
                    selected: _mode,
                    onSelected: (Object value) =>
                        setState(() => _mode = value as _AuthMode),
                  ),
                ),
                const SizedBox(height: 20),
                // 用户名(登录/注册)。
                if (_mode != _AuthMode.forgotPassword) ...[
                  TextField(
                    controller: _username,
                    decoration: InputDecoration(
                      labelText: l10n.authUsernameOrEmail,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
                // 邮箱(注册/找回)。
                if (_mode != _AuthMode.login) ...[
                  TextField(
                    controller: _email,
                    keyboardType: TextInputType.emailAddress,
                    decoration: InputDecoration(
                      labelText: l10n.authEmail,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
                // 密码(登录/注册)。
                if (_mode != _AuthMode.forgotPassword) ...[
                  TextField(
                    controller: _password,
                    obscureText: true,
                    decoration: InputDecoration(
                      labelText: l10n.authPassword,
                      border: const OutlineInputBorder(),
                    ),
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
                          child: TextField(
                            controller: _captcha,
                            decoration: InputDecoration(
                              labelText: l10n.authCaptcha,
                              border: const OutlineInputBorder(),
                            ),
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
                  TextField(
                    controller: _totp,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                      labelText: l10n.authTwoFactorCode,
                      border: const OutlineInputBorder(),
                    ),
                    onSubmitted: (_) =>
                        _authController.submitTotp(_totp.text.trim()),
                  ),
                ],
                if (_authController.error.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Text(
                    _authController.error,
                    style: TextStyle(color: colors.error, fontSize: 13),
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
