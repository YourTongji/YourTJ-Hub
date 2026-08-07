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
import '../../theme_mode.dart';
import '../../widgets/status_views.dart';

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
  Future<void> _loadUser() async {
    setState(() => _user = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/settings');
      final SettingsPageProps? props = parsePageProps<SettingsPageProps>(
        payload,
      );
      if (mounted) {
        setState(() {
          _user = props == null
              ? AsyncValue.error(
                  AppLocalizations.of(context).commonParseFailed,
                  StackTrace.empty,
                )
              : AsyncValue.data(props.user);
        });
      }
    } catch (e, st) {
      if (mounted) setState(() => _user = AsyncValue.error(e, st));
    }
  }

  /// 徽章佩戴选择:底部弹出可佩戴徽章列表,点选调 wear-badge。
  Future<void> _pickBadge(SettingsUserPayload user) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final wearable = user.wearableBadges;
    if (wearable.isEmpty) {
      _snack(l10n.settingsBadgeNoOptions);
      return;
    }
    final String? selected = await showModalBottomSheet<String>(
      context: context,
      builder: (ctx) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                l10n.settingsBadgePick,
                style: const TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            for (final b in wearable)
              ListTile(
                leading: Icon(
                  b.iconType == 'image'
                      ? Icons.image_outlined
                      : Icons.workspace_premium_outlined,
                  color: _hexColor(b.color),
                ),
                title: Text(b.name),
                subtitle: Text(
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
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsBadgeUpdated)));
      }
      _loadUser();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsBadgeFailed('$e'))));
      }
    }
  }

  Color _hexColor(String hex) {
    final value = int.tryParse(hex.replaceFirst('#', ''), radix: 16);
    return value == null ? Colors.amber : Color(0xFF000000 | value);
  }

  /// 修改密码:对话框输入旧/新密码,调 change-password。
  Future<void> _changePassword() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final oldCtrl = TextEditingController();
    final newCtrl = TextEditingController();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.settingsChangePassword),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: oldCtrl,
              obscureText: true,
              decoration: InputDecoration(
                labelText: l10n.settingsCurrentPassword,
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: newCtrl,
              obscureText: true,
              decoration: InputDecoration(labelText: l10n.authNewPassword),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(l10n.commonSave),
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
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsPasswordUpdated)));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.settingsPasswordFailed('$e'))),
        );
      }
    }
  }

  /// 资料编辑:昵称/简介/签名 → set-user-info。
  Future<void> _editProfile(SettingsUserPayload user) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final nickCtrl = TextEditingController(text: user.nickname);
    final bioCtrl = TextEditingController(text: user.bio);
    final sigCtrl = TextEditingController(text: user.signature);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.settingsEditProfile),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: nickCtrl,
              maxLength: 30,
              decoration: InputDecoration(labelText: l10n.settingsNickname),
            ),
            TextField(
              controller: bioCtrl,
              maxLines: 3,
              decoration: InputDecoration(labelText: l10n.settingsBio),
            ),
            TextField(
              controller: sigCtrl,
              maxLines: 2,
              decoration: InputDecoration(labelText: l10n.settingsSignature),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(l10n.commonSave),
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
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsInfoSaved)));
      }
      _loadUser();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsInfoFailed('$e'))));
      }
    }
  }

  /// 邮箱修改 → set-user-email。
  Future<void> _changeEmail() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ctrl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.settingsEmail),
        content: TextField(
          controller: ctrl,
          keyboardType: TextInputType.emailAddress,
          decoration: InputDecoration(labelText: l10n.settingsNewEmail),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(l10n.commonSave),
          ),
        ],
      ),
    );
    if (ok != true) return;
    final email = ctrl.text.trim();
    if (email.isEmpty) {
      _snack(l10n.settingsFillComplete);
      return;
    }
    try {
      await ref.read(userRepositoryProvider).setUserEmail(email);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsEmailUpdated)));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsEmailFailed('$e'))));
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
      await showModalBottomSheet<void>(
        context: context,
        builder: (ctx) => SafeArea(
          child: ListView(
            shrinkWrap: true,
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  l10n.settingsOAuthBindings,
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              for (final entry in bindings.entries)
                ListTile(
                  leading: const Icon(Icons.link, size: 20),
                  title: Text(entry.key),
                  subtitle: Text(
                    entry.value.bound
                        ? l10n.settingsBound
                        : l10n.settingsUnbound,
                  ),
                  trailing: entry.value.bound
                      ? TextButton(
                          onPressed: () async {
                            Navigator.pop(ctx);
                            try {
                              await ref
                                  .read(userRepositoryProvider)
                                  .unbindOAuth(entry.key);
                              if (mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(
                                  SnackBar(
                                    content: Text(l10n.settingsUnboundDone),
                                  ),
                                );
                              }
                            } catch (e) {
                              if (mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(
                                  SnackBar(
                                    content: Text(
                                      l10n.settingsUnbindFailed('$e'),
                                    ),
                                  ),
                                );
                              }
                            }
                          },
                          child: Text(l10n.settingsUnbind),
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
        final ok = await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: Text(l10n.settingsTotpDisableTitle),
            content: TextField(
              controller: codeCtrl,
              decoration: InputDecoration(labelText: l10n.settingsTotpCode),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx, false),
                child: Text(l10n.commonCancel),
              ),
              TextButton(
                onPressed: () => Navigator.pop(ctx, true),
                child: Text(l10n.settingsTotpDisable),
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
            ScaffoldMessenger.of(
              context,
            ).showSnackBar(SnackBar(content: Text(l10n.settingsTotpDisabled)));
          }
        } catch (e) {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text(l10n.settingsTotpFailed('$e'))),
            );
          }
        }
        return;
      }
      // 启用:先要密码 → setup → enable → 展示恢复码。
      final pwdCtrl = TextEditingController();
      final okPwd = await showDialog<bool>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Text(l10n.settingsTotpEnableTitle),
          content: TextField(
            controller: pwdCtrl,
            obscureText: true,
            decoration: InputDecoration(labelText: l10n.settingsTotpPassword),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(l10n.commonCancel),
            ),
            TextButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: Text(l10n.settingsTotpNext),
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
      final okCode = await showDialog<bool>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Text(l10n.settingsTotpScanSecret),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SelectableText(setup.secret),
              const SizedBox(height: 8),
              TextField(
                controller: codeCtrl,
                decoration: InputDecoration(labelText: l10n.settingsTotpCode),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(l10n.commonCancel),
            ),
            TextButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: Text(l10n.settingsTotpEnable),
            ),
          ],
        ),
      );
      if (okCode != true) return;
      final enabled = await ref
          .read(userRepositoryProvider)
          .enableTotp(code: codeCtrl.text.trim());
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (ctx) => AlertDialog(
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
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: Text(l10n.settingsTotpDone),
            ),
          ],
        ),
      );
    } catch (e) {
      if (mounted) _snack(l10n.settingsTotpFailed('$e'));
    }
  }

  Future<void> _loadSessions() async {
    setState(() => _sessions = const AsyncValue.loading());
    try {
      final sessions = await ref.read(userRepositoryProvider).listSessions();
      if (mounted) {
        setState(() => _sessions = AsyncValue.data(sessions));
      }
    } catch (e, st) {
      if (mounted) {
        setState(() => _sessions = AsyncValue.error(e, st));
      }
    }
  }

  Future<void> _revokeSession(int id) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    try {
      await ref.read(userRepositoryProvider).revokeSession(id);
      _loadSessions();
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsRevoked)));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.settingsRevokeFailed('$e'))),
        );
      }
    }
  }

  Future<void> _revokeAll() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    try {
      await ref.read(userRepositoryProvider).revokeAllSessions();
      _loadSessions();
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsRevokeAllDone)));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.settingsOpFailed('$e'))));
      }
    }
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
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.settingsAvatarUploaded(url))),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.settingsAvatarUploadFailed('$e'))),
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
      appBar: AppBar(title: Text(l10n.settingsTitle)),
      body: Column(
        children: [
          // Tab 栏(对齐 web settingsTabLabel: profile/account/privacy/binding/security)。
          Container(
            height: 46,
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
          const Divider(height: 1),
          Expanded(
            child: switch (_tab) {
              _SettingsTab.profile => _buildProfileTab(l10n),
              _SettingsTab.account => _buildAccountTab(l10n),
              _SettingsTab.privacy => _buildPrivacyTab(l10n),
              _SettingsTab.binding => _buildBindingTab(l10n),
              _SettingsTab.security => _buildSecurityTab(l10n, isDark: isDark),
            },
          ),
        ],
      ),
    );
  }

  /// 资料:昵称/简介/头像(web profile tab)。
  Widget _buildProfileTab(AppLocalizations l10n) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _SectionCard(
          title: l10n.settingsSectionProfile,
          child: Column(
            children: [
              ListTile(
                leading: const Icon(Icons.badge_outlined, size: 20),
                title: Text(l10n.settingsNickname),
                subtitle: Text(l10n.settingsNicknameEdit),
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
              const Divider(height: 1),
              ListTile(
                leading: const Icon(Icons.notes_outlined, size: 20),
                title: Text(l10n.settingsBio),
                subtitle: Text(l10n.settingsBioEdit),
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
              const Divider(height: 1),
              ListTile(
                leading: const Icon(Icons.photo_camera_outlined, size: 20),
                title: Text(l10n.settingsAvatar),
                subtitle: Text(
                  _uploadingAvatar
                      ? l10n.settingsAvatarUploading
                      : l10n.settingsAvatarUpload,
                ),
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
  Widget _buildAccountTab(AppLocalizations l10n) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _SectionCard(
          title: l10n.settingsTabAccount,
          child: Column(
            children: [
              ListTile(
                leading: const Icon(Icons.email_outlined, size: 20),
                title: Text(l10n.settingsEmail),
                subtitle: Text(l10n.settingsEmailEdit),
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _changeEmail,
              ),
              const Divider(height: 1),
              ListTile(
                leading: const Icon(Icons.lock_outline, size: 20),
                title: Text(l10n.settingsChangePassword),
                subtitle: Text(l10n.settingsChangePasswordSub),
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _changePassword,
              ),
              const Divider(height: 1),
              ListTile(
                leading: const Icon(Icons.workspace_premium_outlined, size: 20),
                title: Text(l10n.settingsBadge),
                subtitle: _user.when(
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
  Widget _buildPrivacyTab(AppLocalizations l10n) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _SectionCard(
          title: l10n.settingsTabPrivacy,
          child: Column(
            children: [
              SwitchListTile(
                title: Text(l10n.settingsPrivacyDirect),
                value: false,
                onChanged: (_) => _snack(l10n.settingsSecondPhase),
              ),
              const Divider(height: 1),
              SwitchListTile(
                title: Text(l10n.settingsPrivacyLikes),
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
  Widget _buildBindingTab(AppLocalizations l10n) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _SectionCard(
          title: l10n.settingsTabBinding,
          child: Column(
            children: [
              ListTile(
                leading: const Icon(Icons.account_circle_outlined, size: 20),
                title: Text(l10n.settingsOAuth),
                subtitle: Text(l10n.settingsOAuthSub),
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
  Widget _buildSecurityTab(AppLocalizations l10n, {required bool isDark}) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _SectionCard(
          title: l10n.settingsAppearance,
          child: SwitchListTile(
            title: Text(l10n.settingsDarkMode),
            subtitle: Text(
              isDark ? l10n.settingsDarkCurrent : l10n.settingsLightCurrent,
            ),
            value: isDark,
            onChanged: _toggleDarkMode,
          ),
        ),
        const SizedBox(height: 12),
        _SectionCard(
          title: l10n.settingsTotpTitle,
          child: ListTile(
            leading: const Icon(Icons.shield_outlined, size: 20),
            title: Text(l10n.settingsTotpEnable),
            subtitle: Text(l10n.settingsTotpSetupSecret),
            trailing: const Icon(Icons.chevron_right, size: 18),
            onTap: _manageTotp,
          ),
        ),
        const SizedBox(height: 12),
        _SectionCard(
          title: l10n.settingsSessions,
          child: _sessions.when(
            loading: () =>
                const Padding(padding: EdgeInsets.all(24), child: GfLoading()),
            error: (e, _) =>
                GfErrorRetry(message: '$e', onRetry: _loadSessions),
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
                    ListTile(
                      leading: Icon(
                        s.isCurrent ? Icons.smartphone : Icons.devices_other,
                        size: 20,
                        color: s.isCurrent
                            ? GfTheme.colorsOf(context).primary
                            : GfTheme.colorsOf(context).iconMuted,
                      ),
                      title: Text(
                        s.userAgent,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      subtitle: Text(
                        '${s.ipMasked} · ${_formatTs(s.createdAt)}',
                        style: const TextStyle(fontSize: 12),
                      ),
                      trailing: s.isCurrent
                          ? Text(
                              l10n.commonCurrent,
                              style: TextStyle(
                                fontSize: 12,
                                color: GfTheme.colorsOf(context).iconMuted,
                              ),
                            )
                          : IconButton(
                              icon: const Icon(Icons.delete_outline, size: 18),
                              onPressed: () => _revokeSession(s.id),
                            ),
                    ),
                  const Divider(height: 1),
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
        _SectionCard(
          title: l10n.settingsAbout,
          child: ListTile(
            leading: const Icon(Icons.info_outline, size: 20),
            title: Text(l10n.appTitle),
            subtitle: Text(l10n.settingsAboutVersion),
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

  /// 登出:确认 → 服务端失效 → 清 token → 跳登录页。
  Future<void> _logout() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool? ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.settingsLogoutConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(l10n.settingsLogout),
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
    await ref.read(tokenStorageProvider).clear();
    if (mounted) context.go('/login');
  }

  void _snack(String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  String _formatTs(int tsSeconds) {
    final DateTime t = DateTime.fromMillisecondsSinceEpoch(
      tsSeconds * 1000,
    ).toLocal();
    return '${t.year}-${t.month.toString().padLeft(2, '0')}-${t.day.toString().padLeft(2, '0')}';
  }
}

/// 设置分组卡片。
class _SectionCard extends StatelessWidget {
  const _SectionCard({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: 4, bottom: 6),
          child: Text(
            title,
            style: TextStyle(
              fontSize: 13,
              color: colors.iconMuted,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        Container(
          decoration: BoxDecoration(
            color: colors.base100,
            borderRadius: BorderRadius.circular(12),
          ),
          child: child,
        ),
      ],
    );
  }
}
