import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:flutter/foundation.dart';
import 'package:image_picker/image_picker.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../format.dart';
import '../../asset_url.dart';
import '../../server_messages.dart';
import '../../theme_mode.dart';
import '../../app_locale.dart';
import '../../widgets/language_picker.dart';
import '../../site_theme.dart';
import '../../push/push_service.dart';
import '../../widgets/status_views.dart';
import '../../current_user.dart';
import 'account_closure_dialog.dart';
import 'profile_edit_dialog.dart';
import 'username_edit_dialog.dart';
import 'profile_image_editor.dart';
import 'oauth_bindings_sheet.dart';
import '../../widgets/profile_image_crop.dart';
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
  const SettingsPage({super.key, this.initialSection});
  final String? initialSection;

  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends ConsumerState<SettingsPage> {
  _SettingsTab _tab = _SettingsTab.profile;
  AsyncValue<List<UserSessionPayload>> _sessions = const AsyncValue.loading();
  AsyncValue<SettingsUserPayload> _user = const AsyncValue.loading();
  bool _uploadingAvatar = false;
  bool _accountClosing = false;
  bool _googleOAuthReady = false;
  final ImagePicker _imagePicker = ImagePicker();

  @override
  void initState() {
    super.initState();
    _tab = _SettingsTab.values.firstWhere(
      (tab) => tab.name == widget.initialSection,
      orElse: () => _SettingsTab.profile,
    );
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
        _googleOAuthReady = props?.googleOAuthReady ?? false;
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
        showGfToast(
          context,
          l10n.settingsBadgeFailed(resolveErrorMessage(l10n, e)),
          error: true,
        );
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
        showGfToast(
          context,
          l10n.settingsPasswordFailed(resolveErrorMessage(l10n, e)),
          error: true,
        );
      }
    }
  }

  Future<void> _editProfile(SettingsUserPayload user) async {
    final l10n = AppLocalizations.of(context);
    final updated = await showDialog<SettingsUserPayload>(
      context: context,
      builder: (_) => ProfileEditDialog(user: user),
    );
    if (updated == null || !mounted) return;
    try {
      await ref
          .read(userRepositoryProvider)
          .saveUserInfo(
            nickname: updated.nickname,
            bio: updated.bio,
            signature: updated.signature,
            websiteName: updated.websiteName,
            website: updated.website,
            locale: updated.locale,
            externalInformation: updated.externalInformation,
          );
      if (mounted) {
        showGfToast(context, l10n.settingsInfoSaved);
      }
      _loadUser(silent: true);
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          l10n.settingsInfoFailed(resolveErrorMessage(l10n, e)),
          error: true,
        );
      }
    }
  }

  Future<void> _changeUsername() async {
    final user = _user.value;
    if (user == null) return;
    final saved = await showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (_) => UsernameEditDialog(
        username: user.username,
        onSave: ref.read(userRepositoryProvider).saveUserName,
      ),
    );
    if (saved != true || !mounted) return;
    ref.invalidate(currentUserProvider);
    await _loadUser(silent: true);
    if (mounted) {
      showGfToast(
        context,
        AppLocalizations.of(context).settingsUsernameUpdated,
      );
    }
  }

  Future<void> _pickPresetAvatar() async {
    final l10n = AppLocalizations.of(context);
    var selected = _user.value?.avatarUrl ?? '';
    final choice = await showGfBottomSheet<String>(
      context,
      builder: (context) => StatefulBuilder(
        builder: (context, setSheetState) => SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  l10n.settingsPresetAvatar,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 16),
                Flexible(
                  child: GridView.builder(
                    shrinkWrap: true,
                    gridDelegate:
                        const SliverGridDelegateWithFixedCrossAxisCount(
                          crossAxisCount: 4,
                          mainAxisSpacing: 12,
                          crossAxisSpacing: 12,
                        ),
                    itemCount: 12,
                    itemBuilder: (context, index) {
                      final path = '/static/pic/${index + 1}.webp';
                      return Semantics(
                        label: '${l10n.settingsPresetAvatar} ${index + 1}',
                        selected: selected == path,
                        button: true,
                        child: InkWell(
                          onTap: () => setSheetState(() => selected = path),
                          borderRadius: BorderRadius.circular(16),
                          child: Container(
                            padding: const EdgeInsets.all(4),
                            decoration: BoxDecoration(
                              borderRadius: BorderRadius.circular(16),
                              border: Border.all(
                                color: selected == path
                                    ? GfTheme.colorsOf(context).primary
                                    : Colors.transparent,
                                width: 3,
                              ),
                            ),
                            child: GfAvatar(
                              src: resolveApiAssetUrl(path),
                              size: 64,
                            ),
                          ),
                        ),
                      );
                    },
                  ),
                ),
                const SizedBox(height: 16),
                GfButton(
                  label: l10n.commonSave,
                  onPressed: selected.startsWith('/static/pic/')
                      ? () => Navigator.pop(context, selected)
                      : null,
                ),
              ],
            ),
          ),
        ),
      ),
    );
    if (choice == null || !mounted) return;
    try {
      await ref.read(userRepositoryProvider).savePresetAvatar(choice);
      ref.invalidate(currentUserProvider);
      await _loadUser(silent: true);
      if (mounted) showGfToast(context, l10n.settingsInfoSaved);
    } catch (error) {
      if (mounted) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
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
        showGfToast(
          context,
          l10n.settingsEmailFailed(resolveErrorMessage(l10n, e)),
          error: true,
        );
      }
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          l10n.settingsEmailFailed(resolveErrorMessage(l10n, e)),
          error: true,
        );
      }
    }
  }

  Future<void> _manageOAuth() async {
    if (_user.value == null) await _loadUser();
    final user = _user.value;
    if (!mounted || user == null) return;
    await showGfBottomSheet<void>(
      context,
      builder: (_) => OAuthBindingsSheet(
        username: user.username,
        googleReady: _googleOAuthReady,
      ),
    );
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
            showGfToast(
              context,
              l10n.settingsTotpFailed(resolveErrorMessage(l10n, e)),
              error: true,
            );
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
      if (mounted) {
        _snack(l10n.settingsTotpFailed(resolveErrorMessage(l10n, e)));
      }
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
        showGfToast(
          context,
          l10n.settingsRevokeFailed(resolveErrorMessage(l10n, e)),
          error: true,
        );
      }
    }
  }

  /// 吊销全部会话:后端原子删除全部会话并递增 tokenVersion,当前 JWT
  /// 立即失效,必须走统一登出清理(token/认证状态/离线缓存)并跳转登录页,
  /// 否则本地继续持有已被后端撤销的 JWT。
  Future<void> _revokeAll() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    try {
      await ref.read(pushControllerProvider.notifier).handleLogout();
      await ref.read(userRepositoryProvider).revokeAllSessions();
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          l10n.settingsOpFailed(resolveErrorMessage(l10n, e)),
          error: true,
        );
      }
      return;
    }
    await _signOutLocally(successMessage: l10n.settingsRevokeAllDone);
  }

  void _toggleDarkMode(bool value) {
    ref.read(themeModeProvider.notifier).toggleDark(value);
  }

  Future<void> _pickProfileImage({bool cover = false}) async {
    if (_uploadingAvatar) return;
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _uploadingAvatar = true);
    try {
      final picked = await _imagePicker.pickImage(
        source: ImageSource.gallery,
        maxWidth: 4096,
        maxHeight: 4096,
      );
      if (picked == null || !mounted) return;
      final maxMb = cover ? 10 : 5;
      if (await picked.length() > maxMb * 1024 * 1024) {
        if (mounted) _snack(l10n.settingsImageTooLarge(maxMb));
        return;
      }
      final source = await compute(
        prepareProfileCrop,
        await picked.readAsBytes(),
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (cover && (source.width < 1200 || source.height < 240)) {
        _snack(l10n.settingsCoverMinSize);
        return;
      }
      final saved = await Navigator.of(context).push<bool>(
        MaterialPageRoute(
          builder: (_) => ProfileImageEditor(
            source: source,
            cover: cover,
            onSave: (bytes) async {
              if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
                throw const UnauthorizedException();
              }
              final files = ref.read(fileRepositoryProvider);
              if (cover) {
                final url = await files.uploadImage(
                  bytes: bytes,
                  filename: 'cover.webp',
                );
                if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
                  throw const UnauthorizedException();
                }
                await ref
                    .read(userRepositoryProvider)
                    .saveUserProfileCover(url);
              } else {
                await files.uploadAvatar(bytes: bytes, filename: 'avatar.webp');
              }
            },
          ),
        ),
      );
      if (mounted && saved == true) {
        ref.invalidate(currentUserProvider);
        await _loadUser(silent: true);
        if (mounted) _snack(l10n.settingsImageSaved);
      }
    } catch (error) {
      if (mounted) {
        _snack(
          error is FormatException
              ? l10n.settingsImageDecodeFailed
              : resolveErrorMessage(l10n, error),
        );
      }
    } finally {
      if (mounted) setState(() => _uploadingAvatar = false);
    }
  }

  Future<void> _removeCover() async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.settingsCoverRemove),
        content: Text(l10n.settingsCoverRemoveConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l10n.commonConfirm),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted || _uploadingAvatar) return;
    setState(() => _uploadingAvatar = true);
    try {
      await ref.read(userRepositoryProvider).saveUserProfileCover('');
      if (!mounted) return;
      ref.invalidate(currentUserProvider);
      if (mounted) await _loadUser(silent: true);
    } catch (error) {
      if (mounted) _snack(resolveErrorMessage(l10n, error));
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
        actions: [
          IconButton(
            icon: const GfSymbol('languages'),
            tooltip: l10n.settingsAppLanguage,
            onPressed: () => showAppLanguagePicker(context),
          ),
          IconButton(
            icon: const GfSymbol('info'),
            tooltip: l10n.siteInfoTitle,
            onPressed: () => context.push('/about'),
          ),
        ],
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
      builder: (_, ScrollController controller) => AppRefreshIndicator(
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
        GfSettingRow(
          symbol: 'languages',
          title: l10n.settingsAppLanguage,
          description:
              appLanguageNames[ref.watch(appLocaleProvider)?.languageCode] ??
              l10n.settingsLanguageSystem,
          onTap: () => showAppLanguagePicker(context),
        ),
        const SizedBox(height: 12),
        _settingsSection(
          context,
          title: l10n.settingsSectionProfile,
          child: Column(
            children: [
              GfSettingRow(
                symbol: 'id-card',
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
                symbol: 'feather',
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
                symbol: 'camera',
                title: l10n.settingsAvatar,
                description: _uploadingAvatar
                    ? l10n.settingsAvatarUploading
                    : l10n.settingsAvatarUpload,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _uploadingAvatar ? null : () => _pickProfileImage(),
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'smile',
                title: l10n.settingsPresetAvatar,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _uploadingAvatar ? null : _pickPresetAvatar,
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'image',
                title: l10n.settingsCover,
                description: l10n.settingsCoverDescription,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _uploadingAvatar
                    ? null
                    : () => _pickProfileImage(cover: true),
              ),
              if (_user.value?.profileCoverUrl.isNotEmpty == true) ...[
                const GfDivider(),
                GfSettingRow(
                  symbol: 'image-off',
                  title: l10n.settingsCoverRemove,
                  onTap: _uploadingAvatar ? null : _removeCover,
                ),
              ],
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
                symbol: 'at-sign',
                title: l10n.authUsername,
                description: _user.value?.username ?? '',
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _user.value == null ? null : _changeUsername,
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'mail',
                title: l10n.settingsEmail,
                description: l10n.settingsEmailEdit,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _changeEmail,
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'key-round',
                title: l10n.settingsChangePassword,
                description: l10n.settingsChangePasswordSub,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _changePassword,
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'award',
                iconColor: const Color(0xFFD97706),
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
        GfSettingRow(
          symbol: 'folder',
          title: l10n.profileContent,
          onTap: () => context.push('/my-content'),
        ),
        GfSettingRow(
          symbol: 'trash-2',
          iconColor: const Color(0xFFE11D48),
          title: l10n.profileTrash,
          onTap: () => context.push('/recycle-bin'),
        ),
        const SizedBox(height: 24),
        Text(l10n.settingsCloseAccountWarning),
        const SizedBox(height: 12),
        GfButton(
          label: l10n.settingsCloseAccount,
          variant: GfButtonVariant.danger,
          loading: _accountClosing,
          onPressed: _accountClosing ? null : _closeAccount,
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
                symbol: 'circle-user-round',
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
          child: Column(
            children: [
              GfSwitchRow(
                symbol: isDark ? 'moon' : 'sun',
                iconColor: const Color(0xFF7C3AED),
                title: l10n.settingsDarkMode,
                description: isDark
                    ? l10n.settingsDarkCurrent
                    : l10n.settingsLightCurrent,
                value: isDark,
                onChanged: _toggleDarkMode,
              ),
              // 站点主题同步（Route A）：仅服务端启用站点主题时展示。
              Consumer(
                builder: (BuildContext context, WidgetRef ref, _) {
                  final SiteThemeState siteTheme = ref.watch(siteThemeProvider);
                  if (!siteTheme.available) return const SizedBox.shrink();
                  return Column(
                    children: [
                      const GfDivider(),
                      GfSwitchRow(
                        symbol: 'palette',
                        title: l10n.settingsFollowSiteTheme,
                        description: l10n.settingsFollowSiteThemeDesc,
                        value: siteTheme.following,
                        onChanged: (bool value) => ref
                            .read(siteThemeProvider.notifier)
                            .setFollowing(value),
                      ),
                    ],
                  );
                },
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        // Keep delivery status visible even when configuration is incomplete.
        Consumer(
          builder: (BuildContext context, WidgetRef ref, _) {
            final PushChannelStatus push = ref.watch(pushControllerProvider);
            // permissionDenied = 用户已开启但系统权限被拒：开关保持开，
            // 下方给出跳系统设置引导行（三态之二）。
            final bool switchOn =
                push == PushChannelStatus.enabled ||
                push == PushChannelStatus.permissionDenied ||
                push == PushChannelStatus.serverDisabled ||
                push == PushChannelStatus.registrationFailed;
            return _settingsSection(
              context,
              title: l10n.settingsPush,
              child: Column(
                children: [
                  GfSwitchRow(
                    symbol: 'bell',
                    title: l10n.settingsPush,
                    description: l10n.settingsPushConsent,
                    value: switchOn,
                    onChanged: (bool value) async {
                      final PushController controller = ref.read(
                        pushControllerProvider.notifier,
                      );
                      if (value) {
                        await controller.enable();
                      } else {
                        await controller.disable();
                      }
                    },
                  ),
                  const GfDivider(),
                  GfSettingRow(
                    title: l10n.settingsPushPrivacy,
                    onTap: () => launchUrl(
                      Uri.parse('https://www.jiguang.cn/license/privacy'),
                      mode: LaunchMode.externalApplication,
                    ),
                  ),
                  if (push == PushChannelStatus.unsupported ||
                      push == PushChannelStatus.serverDisabled ||
                      push == PushChannelStatus.registrationFailed) ...[
                    const GfDivider(),
                    GfSettingRow(
                      title: push == PushChannelStatus.unsupported
                          ? l10n.settingsPushUnsupported
                          : push == PushChannelStatus.serverDisabled
                          ? l10n.settingsPushServerDisabled
                          : l10n.settingsPushFailed,
                      onTap: () =>
                          ref.read(pushControllerProvider.notifier).enable(),
                    ),
                  ],
                  if (push == PushChannelStatus.permissionDenied) ...[
                    const GfDivider(),
                    GfSettingRow(
                      title: l10n.settingsPushDenied,
                      trailing: const Icon(Icons.chevron_right, size: 18),
                      onTap: () => ref
                          .read(pushControllerProvider.notifier)
                          .openSystemSettings(),
                    ),
                  ],
                ],
              ),
            );
          },
        ),
        const SizedBox(height: 12),
        _settingsSection(
          context,
          title: l10n.settingsTotpTitle,
          child: GfSettingRow(
            symbol: 'shield-check',
            iconColor: const Color(0xFF059669),
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
                      symbol: s.isCurrent ? 'smartphone' : 'monitor',
                      iconColor: s.isCurrent
                          ? GfTheme.colorsOf(context).primary
                          : GfTheme.colorsOf(context).iconMuted,
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
            symbol: 'info',
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
  Future<void> _closeAccount() async {
    final choice = await showDialog<({String mode, String password})>(
      context: context,
      builder: (_) => const AccountClosureDialog(),
    );
    if (choice == null || !mounted) return;
    setState(() => _accountClosing = true);
    try {
      await ref
          .read(contentRepositoryProvider)
          .closeAccount(mode: choice.mode, password: choice.password);
      await _signOutLocally();
    } catch (error) {
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _accountClosing = false);
    }
  }

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
      // Unbind while the session is still valid; logout revokes its JWT.
      await ref.read(pushControllerProvider.notifier).handleLogout();
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
    // 原生推送：尽力注销当前设备的用户绑定（幂等；失败不阻塞登出）。
    await ref.read(pushControllerProvider.notifier).handleLogout();
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
