import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image/image.dart' as img;
import 'package:image_picker/image_picker.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../format.dart';
import '../../server_messages.dart';
import '../../theme_mode.dart';
import '../../widgets/status_views.dart';
import '../../current_user.dart';
import '../../widgets/skeletons.dart';

enum _SettingsTab {
  profile,
  account,
  privacy,
  binding,
  security;

  String label(AppLocalizations l10n) => switch (this) {
    _SettingsTab.profile => l10n.settingsTabProfile,
    _SettingsTab.account => l10n.settingsTabAccount,
    _SettingsTab.privacy => l10n.settingsTabPrivacy,
    _SettingsTab.binding => l10n.settingsTabBinding,
    _SettingsTab.security => l10n.settingsTabSecurity,
  };
}

/// 设置页(web settings.index 的移动端形态)。
///
/// 5 tab 对齐 web:资料 / 账户 / 隐私 / 绑定 / 安全。
class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({super.key});

  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends ConsumerState<SettingsPage> {
  _SettingsTab _tab = _SettingsTab.profile;
  AsyncValue<List<UserSessionPayload>> _sessions = const AsyncValue.loading();
  AsyncValue<SettingsUserPayload> _user = const AsyncValue.loading();
  bool _uploadingAvatar = false;
  final ImagePicker _imagePicker = ImagePicker();

  @override
  void initState() {
    super.initState();
    _loadSessions();
    _loadUser();
  }

  /// 加载设置页账户数据(settings.index 数据通道 → 徽章等)。
  Future<void> _loadUser({bool silent = false}) async {
    final SettingsUserPayload? previous = _user.value;
    if (!silent || previous == null) {
      setState(() => _user = const AsyncValue.loading());
    }
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/settings');
      final SettingsPageProps? props = parsePageProps<SettingsPageProps>(
        payload,
      );
      if (!mounted) return;
      setState(() {
        _user = props == null
            ? AsyncValue.error(
                AppLocalizations.of(context).commonParseFailed,
                StackTrace.empty,
              )
            : AsyncValue.data(props.user);
      });
    } catch (e, st) {
      if (!mounted) return;
      if (silent && previous != null) {
        showGfToast(context, '$e', error: true);
        return;
      }
      setState(() => _user = AsyncValue.error(e, st));
    }
  }

  Future<void> _refresh() async {
    await Future.wait<void>(<Future<void>>[
      _loadUser(silent: true),
      if (_tab == _SettingsTab.security) _loadSessions(silent: true),
    ]);
  }

  void _leaveSettings() {
    final NavigatorState navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    context.go('/profile');
  }

  /// 徽章佩戴选择:底部弹出可佩戴徽章列表,点选调 wear-badge。
  Future<void> _pickBadge(SettingsUserPayload user) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final wearable = user.wearableBadges;
    if (wearable.isEmpty) {
      _snack(l10n.settingsBadgeNoOptions);
      return;
    }
    final String? selected = await showGfBottomSheet<String>(
      context,
      builder: (ctx) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                l10n.settingsBadgePick,
                style: GfTheme.typographyOf(
                  context,
                ).heading.copyWith(fontWeight: FontWeight.w700),
              ),
            ),
            for (final b in wearable)
              GfSettingRow(
                leading: Icon(
                  b.iconType == 'image'
                      ? Icons.image_outlined
                      : Icons.workspace_premium_outlined,
                  color: colorFromHex(
                    b.color,
                    fallback: GfTheme.colorsOf(context).warning,
                  ),
                ),
                title: b.name,
                subtitleWidget: Text(
                  b.description,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: b.code == user.wornBadgeCode
                    ? Icon(
                        Icons.check,
                        color: GfTheme.colorsOf(context).success,
                      )
                    : null,
                onTap: () => Navigator.pop(ctx, b.code),
              ),
          ],
        ),
      ),
    );
    if (selected == null) return;
    try {
      await ref.read(userRepositoryProvider).wearBadge(selected);
      if (mounted) {
        showGfToast(context, l10n.settingsBadgeUpdated);
      }
      _loadUser(silent: true);
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.settingsBadgeFailed('$e'), error: true);
      }
    }
  }

  /// 修改密码:对话框输入旧/新密码,调 change-password。
  Future<void> _changePassword() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final oldCtrl = TextEditingController();
    final newCtrl = TextEditingController();
    final confirmed = await showGfAlertDialog<bool>(
      context,
      builder: (ctx) => GfAlertDialog(
        title: Text(l10n.settingsChangePassword),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            GfInput(
              controller: oldCtrl,
              obscureText: true,
              decoration: InputDecoration(
                labelText: l10n.settingsCurrentPassword,
              ),
            ),
            const SizedBox(height: 8),
            GfInput(
              controller: newCtrl,
              obscureText: true,
              decoration: InputDecoration(labelText: l10n.authNewPassword),
            ),
          ],
        ),
        actions: [
          GfButton(
            label: l10n.commonCancel,
            variant: GfButtonVariant.ghost,
            onPressed: () => Navigator.pop(ctx, false),
          ),
          GfButton(
            label: l10n.commonSave,
            onPressed: () => Navigator.pop(ctx, true),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    final oldPwd = oldCtrl.text.trim();
    final newPwd = newCtrl.text.trim();
    if (oldPwd.isEmpty || newPwd.isEmpty) {
      _snack(l10n.settingsFillComplete);
      return;
    }
    try {
      await ref
          .read(userRepositoryProvider)
          .changePassword(oldPassword: oldPwd, newPassword: newPwd);
      if (mounted) {
        showGfToast(context, l10n.settingsPasswordUpdated);
      }
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.settingsPasswordFailed('$e'), error: true);
      }
    }
  }

  /// 资料编辑:昵称/简介/签名 → set-user-info。
  Future<void> _editProfile(SettingsUserPayload user) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final nickCtrl = TextEditingController(text: user.nickname);
    final bioCtrl = TextEditingController(text: user.bio);
    final sigCtrl = TextEditingController(text: user.signature);
    final ok = await showGfAlertDialog<bool>(
      context,
      builder: (ctx) => GfAlertDialog(
        title: Text(l10n.settingsEditProfile),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            GfInput(
              controller: nickCtrl,
              maxLength: 30,
              decoration: InputDecoration(labelText: l10n.settingsNickname),
            ),
            GfInput(
              controller: bioCtrl,
              maxLines: 3,
              decoration: InputDecoration(labelText: l10n.settingsBio),
            ),
            GfInput(
              controller: sigCtrl,
              maxLines: 2,
              decoration: InputDecoration(labelText: l10n.settingsSignature),
            ),
          ],
        ),
        actions: [
          GfButton(
            label: l10n.commonCancel,
            variant: GfButtonVariant.ghost,
            onPressed: () => Navigator.pop(ctx, false),
          ),
          GfButton(
            label: l10n.commonSave,
            onPressed: () => Navigator.pop(ctx, true),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ref
          .read(userRepositoryProvider)
          .saveUserInfo(
            nickname: nickCtrl.text.trim(),
            bio: bioCtrl.text.trim(),
            signature: sigCtrl.text.trim(),
            websiteName: '',
          );
      if (mounted) {
        showGfToast(context, l10n.settingsInfoSaved);
      }
      _loadUser(silent: true);
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.settingsInfoFailed('$e'), error: true);
      }
    }
  }

  /// 邮箱修改 → set-user-email(需登录密码 re-auth,校验通过才提交)。
  Future<void> _changeEmail() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final emailCtrl = TextEditingController();
    final pwdCtrl = TextEditingController();
    final ok = await showGfAlertDialog<bool>(
      context,
      builder: (ctx) => GfAlertDialog(
        title: Text(l10n.settingsEmail),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            GfInput(
              controller: emailCtrl,
              keyboardType: TextInputType.emailAddress,
              decoration: InputDecoration(labelText: l10n.settingsNewEmail),
            ),
            const SizedBox(height: 8),
            GfInput(
              controller: pwdCtrl,
              obscureText: true,
              decoration: InputDecoration(
                labelText: l10n.settingsCurrentPassword,
              ),
            ),
          ],
        ),
        actions: [
          GfButton(
            label: l10n.commonCancel,
            variant: GfButtonVariant.ghost,
            onPressed: () => Navigator.pop(ctx, false),
          ),
          GfButton(
            label: l10n.commonSave,
            onPressed: () => Navigator.pop(ctx, true),
          ),
        ],
      ),
    );
    if (ok != true) return;
    final email = emailCtrl.text.trim();
    final password = pwdCtrl.text.trim();
    if (email.isEmpty || password.isEmpty) {
      _snack(l10n.settingsFillComplete);
      return;
    }
    try {
      await ref.read(userRepositoryProvider).setUserEmail(email, password);
      if (mounted) {
        showGfToast(context, l10n.settingsEmailUpdated);
      }
    } on ApiException catch (e) {
      if (mounted && e.messageCode == 'auth.password.oauthRequired') {
        showGfToast(
          context,
          l10n.settingsEmailOAuthReauthRequired,
          error: true,
        );
      } else if (mounted) {
        showGfToast(context, l10n.settingsEmailFailed('$e'), error: true);
      }
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.settingsEmailFailed('$e'), error: true);
      }
    }
  }

  /// OAuth 绑定管理:列出绑定状态,可解绑。
  Future<void> _manageOAuth() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    try {
      final bindings = await ref
          .read(userRepositoryProvider)
          .getOAuthBindings();
      if (!mounted) return;
      await showGfBottomSheet<void>(
        context,
        builder: (ctx) => SafeArea(
          child: ListView(
            shrinkWrap: true,
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  l10n.settingsOAuthBindings,
                  style: GfTheme.typographyOf(
                    context,
                  ).heading.copyWith(fontWeight: FontWeight.w700),
                ),
              ),
              for (final entry in bindings.entries)
                GfSettingRow(
                  icon: Icons.link,
                  title: entry.key,
                  description: entry.value.bound
                      ? l10n.settingsBound
                      : l10n.settingsUnbound,
                  trailing: entry.value.bound
                      ? GfButton(
                          label: l10n.settingsUnbind,
                          variant: GfButtonVariant.ghost,
                          size: GfButtonSize.small,
                          onPressed: () async {
                            Navigator.pop(ctx);
                            try {
                              await ref
                                  .read(userRepositoryProvider)
                                  .unbindOAuth(entry.key);
                              if (mounted) {
                                showGfToast(context, l10n.settingsUnboundDone);
                              }
                            } catch (e) {
                              if (mounted) {
                                showGfToast(
                                  context,
                                  l10n.settingsUnbindFailed('$e'),
                                  error: true,
                                );
                              }
                            }
                          },
                        )
                      : null,
                ),
            ],
          ),
        ),
      );
    } catch (e) {
      if (mounted) _snack(l10n.settingsLoadBindingsFailed('$e'));
    }
  }

  /// TOTP 管理:状态 → 启用(密码+密钥+恢复码)/禁用。
  Future<void> _manageTotp() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    try {
      final status = await ref.read(userRepositoryProvider).getTotpStatus();
      if (!mounted) return;
      if (status.enabled) {
        // 禁用。
        final codeCtrl = TextEditingController();
        final ok = await showGfAlertDialog<bool>(
          context,
          builder: (ctx) => GfAlertDialog(
            title: Text(l10n.settingsTotpDisableTitle),
            content: GfInput(
              controller: codeCtrl,
              decoration: InputDecoration(labelText: l10n.settingsTotpCode),
            ),
            actions: [
              GfButton(
                label: l10n.commonCancel,
                variant: GfButtonVariant.ghost,
                onPressed: () => Navigator.pop(ctx, false),
              ),
              GfButton(
                label: l10n.settingsTotpDisable,
                variant: GfButtonVariant.danger,
                onPressed: () => Navigator.pop(ctx, true),
              ),
            ],
          ),
        );
        if (ok != true) return;
        try {
          await ref
              .read(userRepositoryProvider)
              .disableTotp(code: codeCtrl.text.trim());
          if (mounted) {
            showGfToast(context, l10n.settingsTotpDisabled);
          }
        } catch (e) {
          if (mounted) {
            showGfToast(context, l10n.settingsTotpFailed('$e'), error: true);
          }
        }
        return;
      }
      // 启用:先要密码 → setup → enable → 展示恢复码。
      final pwdCtrl = TextEditingController();
      final okPwd = await showGfAlertDialog<bool>(
        context,
        builder: (ctx) => GfAlertDialog(
          title: Text(l10n.settingsTotpEnableTitle),
          content: GfInput(
            controller: pwdCtrl,
            obscureText: true,
            decoration: InputDecoration(labelText: l10n.settingsTotpPassword),
          ),
          actions: [
            GfButton(
              label: l10n.commonCancel,
              variant: GfButtonVariant.ghost,
              onPressed: () => Navigator.pop(ctx, false),
            ),
            GfButton(
              label: l10n.settingsTotpNext,
              onPressed: () => Navigator.pop(ctx, true),
            ),
          ],
        ),
      );
      if (okPwd != true) return;
      final setup = await ref
          .read(userRepositoryProvider)
          .getTotpSetup(password: pwdCtrl.text.trim());
      if (!mounted) return;
      final codeCtrl = TextEditingController();
      final okCode = await showGfAlertDialog<bool>(
        context,
        builder: (ctx) => GfAlertDialog(
          title: Text(l10n.settingsTotpScanSecret),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SelectableText(setup.secret),
              const SizedBox(height: 8),
              GfInput(
                controller: codeCtrl,
                decoration: InputDecoration(labelText: l10n.settingsTotpCode),
              ),
            ],
          ),
          actions: [
            GfButton(
              label: l10n.commonCancel,
              variant: GfButtonVariant.ghost,
              onPressed: () => Navigator.pop(ctx, false),
            ),
            GfButton(
              label: l10n.settingsTotpEnable,
              onPressed: () => Navigator.pop(ctx, true),
            ),
          ],
        ),
      );
      if (okCode != true) return;
      final enabled = await ref
          .read(userRepositoryProvider)
          .enableTotp(code: codeCtrl.text.trim());
      if (!mounted) return;
      await showGfAlertDialog<void>(
        context,
        builder: (ctx) => GfAlertDialog(
          title: Text(l10n.settingsTotpEnabled),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(l10n.settingsTotpRecoveryCodes),
              const SizedBox(height: 8),
              for (final c in enabled.recoveryCodes)
                SelectableText(
                  c,
                  style: const TextStyle(fontFamily: 'monospace'),
                ),
            ],
          ),
          actions: [
            GfButton(
              label: l10n.settingsTotpDone,
              onPressed: () => Navigator.pop(ctx),
            ),
          ],
        ),
      );
    } catch (e) {
      if (mounted) _snack(l10n.settingsTotpFailed('$e'));
    }
  }

  Future<void> _loadSessions({bool silent = false}) async {
    final List<UserSessionPayload>? previous = _sessions.value;
    if (!silent || previous == null) {
      setState(() => _sessions = const AsyncValue.loading());
    }
    try {
      final sessions = await ref.read(userRepositoryProvider).listSessions();
      if (mounted) {
        setState(() => _sessions = AsyncValue.data(sessions));
      }
    } catch (e, st) {
      if (!mounted) return;
      if (silent && previous != null) {
        showGfToast(context, '$e', error: true);
        return;
      }
      setState(() => _sessions = AsyncValue.error(e, st));
    }
  }

  Future<void> _revokeSession(int id) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    try {
      await ref.read(userRepositoryProvider).revokeSession(id);
      _loadSessions(silent: true);
      if (mounted) {
        showGfToast(context, l10n.settingsRevoked);
      }
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.settingsRevokeFailed('$e'), error: true);
      }
    }
  }

  /// 吊销全部会话:后端原子删除全部会话并递增 tokenVersion,当前 JWT
  /// 立即失效,必须走统一登出清理(token/认证状态/离线缓存)并跳转登录页,
  /// 否则本地继续持有已被后端撤销的 JWT。
  Future<void> _revokeAll() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    try {
      await ref.read(userRepositoryProvider).revokeAllSessions();
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.settingsOpFailed('$e'), error: true);
      }
      return;
    }
    await _signOutLocally(successMessage: l10n.settingsRevokeAllDone);
  }

  void _toggleDarkMode(bool value) {
    ref.read(themeModeProvider.notifier).toggleDark(value);
  }

  /// 选择头像图片,前端转码为 webp 后上传(与验收标准"头像上传(前端转 webp)"一致)。
  Future<void> _pickAvatar() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    if (_uploadingAvatar) return;
    final XFile? picked = await _imagePicker.pickImage(
      source: ImageSource.gallery,
      maxWidth: 512,
      maxHeight: 512,
      imageQuality: 90,
    );
    if (picked == null) return;

    setState(() => _uploadingAvatar = true);
    try {
      final img.Image? decoded = img.decodeImage(await picked.readAsBytes());
      if (decoded == null) {
        throw StateError(l10n.settingsImageDecodeFailed);
      }
      final Uint8List webp = img.encodeWebP(decoded);
      final String url = await ref
          .read(fileRepositoryProvider)
          .uploadAvatar(bytes: webp, filename: 'avatar.webp');
      if (mounted) {
        showGfToast(context, l10n.settingsAvatarUploaded(url));
      }
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          l10n.settingsAvatarUploadFailed('$e'),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _uploadingAvatar = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool isDark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      appBar: GfAppBar(
        leading: GfIconButton(
          icon: Icons.arrow_back,
          tooltip: l10n.commonBack,
          size: 44,
          onPressed: _leaveSettings,
        ),
        title: Text(l10n.settingsTitle),
      ),
      body: Column(
        children: [
          // Tab 栏(对齐 web settingsTabLabel: profile/account/privacy/binding/security)。
          Container(
            height: 44,
            alignment: Alignment.centerLeft,
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: GfTabBar(
              tabs: <GfTab>[
                for (final tab in _SettingsTab.values)
                  GfTab(label: tab.label(l10n), value: tab),
              ],
              selected: _tab,
              onSelected: (Object value) =>
                  setState(() => _tab = value as _SettingsTab),
            ),
          ),
          const GfDivider(),
          Expanded(child: _buildTabBody(l10n, isDark: isDark)),
        ],
      ),
    );
  }

  Widget _buildTabBody(AppLocalizations l10n, {required bool isDark}) {
    final bool needsUser =
        _tab == _SettingsTab.profile || _tab == _SettingsTab.account;
    if (needsUser && _user.isLoading && !_user.hasValue) {
      return const GfSettingsSkeleton();
    }
    if (needsUser && _user.hasError && !_user.hasValue) {
      return GfErrorRetry(
        message: resolveErrorMessage(l10n, _user.error!),
        onRetry: _loadUser,
      );
    }

    return GfScrollToTop(
      semanticLabel: l10n.commonBackToTop,
      key: ValueKey<_SettingsTab>(_tab),
      builder: (_, ScrollController controller) => RefreshIndicator(
        onRefresh: _refresh,
        child: switch (_tab) {
          _SettingsTab.profile => _buildProfileTab(l10n, controller),
          _SettingsTab.account => _buildAccountTab(l10n, controller),
          _SettingsTab.privacy => _buildPrivacyTab(l10n, controller),
          _SettingsTab.binding => _buildBindingTab(l10n, controller),
          _SettingsTab.security => _buildSecurityTab(
            l10n,
            controller,
            isDark: isDark,
          ),
        },
      ),
    );
  }

  /// 资料:昵称/简介/头像(web profile tab)。
  Widget _buildProfileTab(AppLocalizations l10n, ScrollController controller) {
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: <Widget>[
        _settingsSection(
          context,
          title: l10n.settingsSectionProfile,
          child: Column(
            children: [
              GfSettingRow(
                icon: Icons.badge_outlined,
                title: l10n.settingsNickname,
                description: l10n.settingsNicknameEdit,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: () {
                  final u = _user.value;
                  if (u == null) {
                    _snack(l10n.settingsUserDataLoading);
                    return;
                  }
                  _editProfile(u);
                },
              ),
              const GfDivider(),
              GfSettingRow(
                icon: Icons.notes_outlined,
                title: l10n.settingsBio,
                description: l10n.settingsBioEdit,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: () {
                  final u = _user.value;
                  if (u == null) {
                    _snack(l10n.settingsUserDataLoading);
                    return;
                  }
                  _editProfile(u);
                },
              ),
              const GfDivider(),
              GfSettingRow(
                icon: Icons.photo_camera_outlined,
                title: l10n.settingsAvatar,
                description: _uploadingAvatar
                    ? l10n.settingsAvatarUploading
                    : l10n.settingsAvatarUpload,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _pickAvatar,
              ),
            ],
          ),
        ),
      ],
    );
  }

  /// 账户:邮箱/密码/徽章(web account tab)。
  Widget _buildAccountTab(AppLocalizations l10n, ScrollController controller) {
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: <Widget>[
        _settingsSection(
          context,
          title: l10n.settingsTabAccount,
          child: Column(
            children: [
              GfSettingRow(
                icon: Icons.email_outlined,
                title: l10n.settingsEmail,
                description: l10n.settingsEmailEdit,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _changeEmail,
              ),
              const GfDivider(),
              GfSettingRow(
                icon: Icons.lock_outline,
                title: l10n.settingsChangePassword,
                description: l10n.settingsChangePasswordSub,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _changePassword,
              ),
              const GfDivider(),
              GfSettingRow(
                icon: Icons.workspace_premium_outlined,
                title: l10n.settingsBadge,
                subtitleWidget: _user.when(
                  data: (u) => Text(
                    u.wornBadge == null
                        ? l10n.settingsBadgeNone
                        : l10n.settingsBadgeCurrent(u.wornBadge!.name),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  loading: () => Text(l10n.commonLoading),
                  error: (_, _) => Text(l10n.settingsBadge),
                ),
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: () {
                  final u = _user.value;
                  if (u == null) {
                    _snack(l10n.settingsUserDataLoading);
                    return;
                  }
                  _pickBadge(u);
                },
              ),
            ],
          ),
        ),
      ],
    );
  }

  /// 隐私:隐私设置(web privacy tab)。
  Widget _buildPrivacyTab(AppLocalizations l10n, ScrollController controller) {
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: <Widget>[
        _settingsSection(
          context,
          title: l10n.settingsTabPrivacy,
          child: Column(
            children: [
              GfSwitchRow(
                title: l10n.settingsPrivacyDirect,
                value: false,
                onChanged: (_) => _snack(l10n.settingsSecondPhase),
              ),
              const GfDivider(),
              GfSwitchRow(
                title: l10n.settingsPrivacyLikes,
                value: true,
                onChanged: (_) => _snack(l10n.settingsSecondPhase),
              ),
            ],
          ),
        ),
      ],
    );
  }

  /// 绑定:OAuth 绑定(web binding tab)。
  Widget _buildBindingTab(AppLocalizations l10n, ScrollController controller) {
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: <Widget>[
        _settingsSection(
          context,
          title: l10n.settingsTabBinding,
          child: Column(
            children: [
              GfSettingRow(
                icon: Icons.account_circle_outlined,
                title: l10n.settingsOAuth,
                description: l10n.settingsOAuthSub,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _manageOAuth,
              ),
            ],
          ),
        ),
      ],
    );
  }

  /// 安全:外观 + TOTP + 会话管理 + 关于(web security tab)。
  Widget _buildSecurityTab(
    AppLocalizations l10n,
    ScrollController controller, {
    required bool isDark,
  }) {
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: <Widget>[
        _settingsSection(
          context,
          title: l10n.settingsAppearance,
          child: GfSwitchRow(
            title: l10n.settingsDarkMode,
            description: isDark
                ? l10n.settingsDarkCurrent
                : l10n.settingsLightCurrent,
            value: isDark,
            onChanged: _toggleDarkMode,
          ),
        ),
        const SizedBox(height: 12),
        _settingsSection(
          context,
          title: l10n.settingsTotpTitle,
          child: GfSettingRow(
            icon: Icons.shield_outlined,
            title: l10n.settingsTotpEnable,
            description: l10n.settingsTotpSetupSecret,
            trailing: const Icon(Icons.chevron_right, size: 18),
            onTap: _manageTotp,
          ),
        ),
        const SizedBox(height: 12),
        _settingsSection(
          context,
          title: l10n.settingsSessions,
          child: _sessions.when(
            loading: () => const _SettingsSessionsSkeleton(),
            error: (e, _) => GfErrorRetry(
              message: resolveErrorMessage(l10n, e),
              onRetry: _loadSessions,
            ),
            data: (sessions) {
              if (sessions.isEmpty) {
                return Padding(
                  padding: const EdgeInsets.all(24),
                  child: GfEmpty(message: l10n.settingsSessionsEmpty),
                );
              }
              return Column(
                children: [
                  for (final s in sessions)
                    GfSettingRow(
                      leading: Icon(
                        s.isCurrent ? Icons.smartphone : Icons.devices_other,
                        size: 20,
                        color: s.isCurrent
                            ? GfTheme.colorsOf(context).primary
                            : GfTheme.colorsOf(context).iconMuted,
                      ),
                      title: s.userAgent,
                      subtitleWidget: Text(
                        '${s.ipMasked} · ${_formatTs(s.createdAt)}',
                        style: GfTheme.typographyOf(context).caption,
                      ),
                      trailing: s.isCurrent
                          ? Text(
                              l10n.commonCurrent,
                              style: GfTheme.typographyOf(context).caption
                                  .copyWith(
                                    color: GfTheme.colorsOf(context).iconMuted,
                                  ),
                            )
                          : GfIconButton(
                              icon: Icons.delete_outline,
                              iconSize: 18,
                              onPressed: () => _revokeSession(s.id),
                            ),
                    ),
                  const GfDivider(),
                  Padding(
                    padding: const EdgeInsets.all(8),
                    child: GfButton(
                      label: l10n.settingsRevokeAll,
                      variant: GfButtonVariant.danger,
                      size: GfButtonSize.small,
                      onPressed: _revokeAll,
                    ),
                  ),
                ],
              );
            },
          ),
        ),
        const SizedBox(height: 12),
        _settingsSection(
          context,
          title: l10n.settingsAbout,
          child: GfSettingRow(
            icon: Icons.info_outline,
            title: l10n.appTitle,
            description: l10n.settingsAboutVersion,
          ),
        ),
        const SizedBox(height: 12),
        // 登出(web AppShell logout 语义):服务端失效 + 清本地 token + 回登录页。
        GfButton(
          label: l10n.settingsLogout,
          variant: GfButtonVariant.danger,
          expanded: true,
          onPressed: _logout,
        ),
        const SizedBox(height: 24),
      ],
    );
  }

  /// 登出:确认 → 服务端失效 → 清 token → 清离线缓存 → 跳登录页。
  ///
  /// 离线缓存(drift)保存私信与已浏览话题,登出必须清空,否则同一设备
  /// 换账号后仍可读到上一账号的缓存数据(跨账号数据泄漏)。
  Future<void> _logout() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool? ok = await showGfAlertDialog<bool>(
      context,
      builder: (ctx) => GfAlertDialog(
        title: Text(l10n.settingsLogoutConfirm),
        actions: [
          GfButton(
            label: l10n.commonCancel,
            variant: GfButtonVariant.ghost,
            onPressed: () => Navigator.pop(ctx, false),
          ),
          GfButton(
            label: l10n.settingsLogout,
            variant: GfButtonVariant.danger,
            onPressed: () => Navigator.pop(ctx, true),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    try {
      await ref.read(authRepositoryProvider).logout();
    } catch (_) {
      // 服务端失效失败不阻塞本地登出(会话已不可信)。
    }
    await _signOutLocally();
  }

  /// 统一本地登出:清 token → 失效会话世代 → 清离线缓存 → 跳转登录页。
  ///
  /// 登出/吊销全部会话后当前 JWT 已不可信,必须清空本地 token 与离线
  /// 缓存(否则同一设备换账号后仍可读到上一账号数据,造成跨账号泄漏)。
  Future<void> _signOutLocally({String? successMessage}) async {
    await ref.read(tokenStorageProvider).clear();
    // 会话边界:先使旧会话在途写入失效、当前用户身份失效,再清空缓存。
    ref.read(offlineCacheEpochProvider.notifier).invalidate();
    ref.invalidate(currentUserProvider);
    // 清空话题/会话/私信离线缓存;失败静默(下次登出/登录会重试)。
    await clearOfflineCacheQuietly(
      ref.read(offlineTopicCacheProvider),
      ref.read(offlineChatCacheProvider),
    );
    if (successMessage != null && mounted) {
      showGfToast(context, successMessage);
    }
    if (mounted) context.go('/login');
  }

  void _snack(String message) {
    showGfToast(context, message);
  }

  /// 会话创建时间渲染为 `YYYY-MM-DD`。
  ///
  /// 契约(`components/schemas.yaml#/UserSession` 与后端
  /// `toSessionVO` 的 `UnixMilli()`)规定 createdAt 为 Unix **毫秒**,
  /// 直接使用,不得再乘 1000(否则会变成微秒级,日期溢出)。
  String _formatTs(int createdAtMs) {
    final DateTime t = DateTime.fromMillisecondsSinceEpoch(
      createdAtMs,
    ).toLocal();
    return '${t.year}-${t.month.toString().padLeft(2, '0')}-${t.day.toString().padLeft(2, '0')}';
  }
}

class _SettingsSessionsSkeleton extends StatelessWidget {
  const _SettingsSessionsSkeleton();

  @override
  Widget build(BuildContext context) {
    return const ExcludeSemantics(
      child: Padding(
        padding: EdgeInsets.all(16),
        child: Column(
          children: <Widget>[
            Row(
              children: <Widget>[
                GfSkeleton(width: 36, height: 36, radius: 8),
                SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      GfSkeleton(width: 176, height: 14, radius: 5),
                      SizedBox(height: 8),
                      GfSkeleton(width: 124, height: 12, radius: 5),
                    ],
                  ),
                ),
              ],
            ),
            SizedBox(height: 16),
            GfDivider(),
            SizedBox(height: 16),
            Row(
              children: <Widget>[
                GfSkeleton(width: 36, height: 36, radius: 8),
                SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      GfSkeleton(width: 148, height: 14, radius: 5),
                      SizedBox(height: 8),
                      GfSkeleton(width: 108, height: 12, radius: 5),
                    ],
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 设置分组:标题 + [GfPanel] 容器(web `gf-panel` 语义,移动端全宽无边框)。
Widget _settingsSection(
  BuildContext context, {
  required String title,
  required Widget child,
}) {
  final GfColors colors = GfTheme.colorsOf(context);
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Padding(
        padding: const EdgeInsets.only(left: 4, bottom: 6),
        child: Text(
          title,
          style: GfTheme.typographyOf(context).small.copyWith(
            color: colors.iconMuted,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      GfPanel(child: child),
    ],
  );
}
