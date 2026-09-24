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
import '../../local/writing_store.dart';
import '../../messages/chat_drafts.dart';
import '../../format.dart';
import '../../asset_url.dart';
import '../../server_messages.dart';
import '../../theme_mode.dart';
import '../../app_locale.dart';
import '../../widgets/language_picker.dart';
import '../../site_theme.dart';
import '../../campus_widget/schedule_widget_bridge.dart';
import '../../push/push_service.dart';
import '../../widgets/status_views.dart';
import '../../current_user.dart';
import '../../navigation/auth_navigation.dart';
import 'account_closure_dialog.dart';
import 'campus_cache_clear_tile.dart';
import 'profile_edit_dialog.dart';
import 'username_edit_dialog.dart';
import 'badge_display_dialog.dart';
import 'profile_image_editor.dart';
import 'oauth_bindings_sheet.dart';
import '../../widgets/profile_image_crop.dart';
import '../../widgets/skeletons.dart';

enum _SettingsTab {
  profile,
  account,
  privacy,
  binding,
  security,
  appearance,
  notifications;

  String label(AppLocalizations l10n) => switch (this) {
    _SettingsTab.profile => l10n.settingsTabProfile,
    _SettingsTab.account => l10n.settingsTabAccount,
    _SettingsTab.privacy => l10n.settingsTabPrivacy,
    _SettingsTab.binding => l10n.settingsTabBinding,
    _SettingsTab.security => l10n.settingsTabSecurity,
    _SettingsTab.appearance => l10n.settingsAppearance,
    _SettingsTab.notifications => l10n.settingsPush,
  };
}

/// 设置页(web settings.index 的移动端形态)。
///
/// Device preferences remain available to guests. Existing section deep links
/// retain their names; the index opens each category on a normal back stack.
class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({super.key, this.initialSection});
  final String? initialSection;

  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends ConsumerState<SettingsPage> {
  _SettingsTab? _tab;
  bool? _signedIn;
  int _sessionRequest = 0;
  int _userRequest = 0;
  int _sessionsRequest = 0;
  AsyncValue<List<UserSessionPayload>> _sessions = const AsyncValue.loading();
  AsyncValue<SettingsUserPayload> _user = const AsyncValue.loading();
  bool _uploadingAvatar = false;
  bool _accountClosing = false;
  bool _googleOAuthReady = false;
  int _widgetTransparency = ScheduleWidgetBridge.defaultTransparencyPercent;
  int _savedWidgetTransparency =
      ScheduleWidgetBridge.defaultTransparencyPercent;
  bool _widgetTransparencyLoaded = false;
  final ImagePicker _imagePicker = ImagePicker();

  @override
  void initState() {
    super.initState();
    for (final section in _SettingsTab.values) {
      if (section.name == widget.initialSection) _tab = section;
    }
    _loadSession();
    _loadWidgetTransparency();
  }

  bool get _needsUser =>
      _tab == _SettingsTab.profile ||
      _tab == _SettingsTab.account ||
      _tab == _SettingsTab.binding;

  Future<void> _loadSession() async {
    final request = ++_sessionRequest;
    final epoch = ref.read(offlineCacheEpochProvider);
    bool signedIn = false;
    try {
      signedIn = await hasSessionToken(ref.read(tokenStorageProvider));
    } catch (_) {
      // Device preferences work even when session storage is unavailable.
    }
    if (!mounted ||
        request != _sessionRequest ||
        ref.read(offlineCacheEpochProvider) != epoch) {
      return;
    }
    setState(() => _signedIn = signedIn);
    if (!signedIn) return;
    if (_needsUser) unawaited(_loadUser());
    if (_tab == _SettingsTab.security) unawaited(_loadSessions());
  }

  Future<void> _loadWidgetTransparency() async {
    try {
      final value = await ref
          .read(scheduleWidgetBridgeProvider)
          .readTransparency();
      if (!mounted) return;
      setState(() {
        _widgetTransparency = value;
        _savedWidgetTransparency = value;
        _widgetTransparencyLoaded = true;
      });
    } catch (_) {
      if (mounted) setState(() => _widgetTransparencyLoaded = true);
    }
  }

  Future<void> _saveWidgetTransparency(int value) async {
    try {
      await ref.read(scheduleWidgetBridgeProvider).setTransparency(value);
      if (mounted) setState(() => _savedWidgetTransparency = value);
    } catch (error) {
      if (!mounted) return;
      setState(() => _widgetTransparency = _savedWidgetTransparency);
      final l10n = AppLocalizations.of(context);
      showGfToast(
        context,
        l10n.settingsOpFailed(resolveErrorMessage(l10n, error)),
        error: true,
      );
    }
  }

  /// 加载设置页账户数据(settings.index 数据通道 → 徽章等)。
  Future<void> _loadUser({bool silent = false}) async {
    if (_signedIn != true) return;
    final request = ++_userRequest;
    final epoch = ref.read(offlineCacheEpochProvider);
    final SettingsUserPayload? previous = _user.valueOrNull;
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
      if (!mounted ||
          request != _userRequest ||
          ref.read(offlineCacheEpochProvider) != epoch) {
        return;
      }
      if (props == null) {
        throw FormatException(AppLocalizations.of(context).commonParseFailed);
      }
      setState(() {
        _googleOAuthReady = props.googleOAuthReady;
        _user = AsyncValue.data(props.user);
      });
    } catch (e, st) {
      if (!mounted ||
          request != _userRequest ||
          ref.read(offlineCacheEpochProvider) != epoch) {
        return;
      }
      if (silent && previous != null) {
        showGfToast(context, '$e', error: true);
        return;
      }
      setState(() => _user = AsyncValue.error(e, st));
    }
  }

  Future<void> _refresh() async {
    await Future.wait<void>(<Future<void>>[
      if (_signedIn == true && _needsUser) _loadUser(silent: true),
      if (_signedIn == true && _tab == _SettingsTab.security)
        _loadSessions(silent: true),
    ]);
  }

  void _leaveSettings() {
    final NavigatorState navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    if (_tab != null) {
      setState(() => _tab = null);
      return;
    }
    context.go(_signedIn == true ? '/profile' : '/');
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

  Future<void> _displayBadges(SettingsUserPayload user) async {
    final saved = await showDialog<bool>(
      context: context,
      builder: (_) => BadgeDisplayDialog(
        badges: user.badges,
        selected: user.displayBadges ?? user.badges.take(5).toList(),
        onSave: (codes) async {
          await ref.read(userRepositoryProvider).displayBadges(codes);
        },
      ),
    );
    if (saved == true && mounted) _loadUser(silent: true);
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

  Future<void> _editProfile(
    SettingsUserPayload user, {
    ProfileEditSection section = ProfileEditSection.all,
  }) async {
    final l10n = AppLocalizations.of(context);
    final updated = await showDialog<SettingsUserPayload>(
      context: context,
      builder: (_) => ProfileEditDialog(user: user, section: section),
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

  Future<void> _chooseAvatar() async {
    final l10n = AppLocalizations.of(context);
    final preset = await showGfBottomSheet<bool>(
      context,
      builder: (context) => Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(
            leading: const Icon(Icons.photo_library_outlined),
            title: Text(l10n.settingsAvatarUpload),
            onTap: () => Navigator.pop(context, false),
          ),
          ListTile(
            leading: const Icon(Icons.face_outlined),
            title: Text(l10n.settingsPresetAvatar),
            onTap: () => Navigator.pop(context, true),
          ),
        ],
      ),
    );
    if (!mounted || preset == null) return;
    if (preset) {
      await _pickPresetAvatar();
    } else {
      await _pickProfileImage();
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
        // The API also supports an immediate switch when verification is off.
        // Reload the server state to show whether confirmation is pending.
        showGfToast(context, l10n.settingsEmailChangeStaged);
        await _loadUser(silent: true);
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
    if (_signedIn != true) return;
    final request = ++_sessionsRequest;
    final epoch = ref.read(offlineCacheEpochProvider);
    final List<UserSessionPayload>? previous = _sessions.valueOrNull;
    if (!silent || previous == null) {
      setState(() => _sessions = const AsyncValue.loading());
    }
    try {
      final sessions = await ref.read(userRepositoryProvider).listSessions();
      if (!mounted ||
          request != _sessionsRequest ||
          ref.read(offlineCacheEpochProvider) != epoch) {
        return;
      }
      setState(() => _sessions = AsyncValue.data(sessions));
    } catch (e, st) {
      if (!mounted ||
          request != _sessionsRequest ||
          ref.read(offlineCacheEpochProvider) != epoch) {
        return;
      }
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
    ref.listen(offlineCacheEpochProvider, (previous, next) {
      if (previous == next) return;
      _userRequest++;
      _sessionsRequest++;
      setState(() {
        _signedIn = null;
        _user = const AsyncValue.loading();
        _sessions = const AsyncValue.loading();
        _googleOAuthReady = false;
      });
      unawaited(_loadSession());
    });
    final l10n = AppLocalizations.of(context);
    final title = _tab?.label(l10n) ?? l10n.settingsTitle;
    final colors = GfTheme.colorsOf(context);
    final titleStyle = GfTheme.typographyOf(
      context,
    ).heading.copyWith(fontSize: 18, height: 1.4, fontWeight: FontWeight.w700);
    final titleLayout = TextPainter(
      text: TextSpan(text: title, style: titleStyle),
      textDirection: Directionality.of(context),
      textScaler: MediaQuery.textScalerOf(context),
      maxLines: 2,
    )..layout(maxWidth: MediaQuery.sizeOf(context).width - 88);
    final toolbarHeight = (titleLayout.height + 16).clamp(
      56.0,
      double.infinity,
    );
    titleLayout.dispose();
    return Scaffold(
      backgroundColor: colors.base200,
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          tooltip: l10n.commonBack,
          onPressed: _leaveSettings,
        ),
        title: Text(title, maxLines: 2),
        centerTitle: false,
        titleTextStyle: titleStyle,
        toolbarHeight: toolbarHeight,
        backgroundColor: colors.base100,
        scrolledUnderElevation: 0,
        shape: Border(bottom: BorderSide(color: colors.line)),
      ),
      body: SafeArea(
        top: false,
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 720),
            child: _tab == null ? _buildIndex(l10n) : _buildTabBody(l10n),
          ),
        ),
      ),
    );
  }

  void _openSection(_SettingsTab section) {
    Navigator.of(context).push<void>(
      MaterialPageRoute(
        builder: (_) => SettingsPage(initialSection: section.name),
      ),
    );
  }

  Widget _buildIndex(AppLocalizations l10n) => ListView(
    padding: const EdgeInsets.all(16),
    children: [
      _settingsSection(
        context,
        title: l10n.settingsDevice,
        child: Column(
          children: [
            _categoryRow(
              key: const ValueKey('settings-category-appearance'),
              icon: Icons.palette_outlined,
              title: l10n.settingsAppearance,
              onTap: () => _openSection(_SettingsTab.appearance),
            ),
            const GfDivider(),
            _categoryRow(
              key: const ValueKey('settings-category-language'),
              icon: Icons.language,
              title: l10n.settingsAppLanguage,
              description:
                  appLanguageNames[ref
                      .watch(appLocaleProvider)
                      ?.languageCode] ??
                  l10n.settingsLanguageSystem,
              onTap: () => showAppLanguagePicker(context),
            ),
            const GfDivider(),
            _categoryRow(
              icon: Icons.info_outline,
              title: l10n.settingsAbout,
              onTap: () => context.push('/about'),
            ),
          ],
        ),
      ),
      const SizedBox(height: 24),
      _settingsSection(
        context,
        title: l10n.settingsYourAccount,
        child: Column(
          children: [
            if (_signedIn == true)
              for (final section in const [
                _SettingsTab.profile,
                _SettingsTab.account,
                _SettingsTab.privacy,
                _SettingsTab.binding,
                _SettingsTab.notifications,
                _SettingsTab.security,
              ]) ...[
                if (section != _SettingsTab.profile) const GfDivider(),
                _categoryRow(
                  key: ValueKey('settings-category-${section.name}'),
                  icon: switch (section) {
                    _SettingsTab.profile => Icons.person_outline,
                    _SettingsTab.account => Icons.manage_accounts_outlined,
                    _SettingsTab.privacy => Icons.privacy_tip_outlined,
                    _SettingsTab.binding => Icons.link,
                    _SettingsTab.notifications => Icons.notifications_outlined,
                    _ => Icons.shield_outlined,
                  },
                  title: section.label(l10n),
                  onTap: () => _openSection(section),
                ),
              ]
            else if (_signedIn == false)
              _categoryRow(
                icon: Icons.login,
                title: l10n.authLoginTitle,
                onTap: () => context.push(
                  authLoginLocation(
                    returnTo: _tab == null
                        ? '/settings'
                        : '/settings/${_tab!.name}',
                  ),
                ),
              )
            else
              const Padding(
                padding: EdgeInsets.all(16),
                child: LinearProgressIndicator(),
              ),
          ],
        ),
      ),
    ],
  );

  // Material's focusable InkWell gives category rows keyboard and screen-reader
  // activation. Wrapping text has no fixed row height, including at 200% scaling.
  Widget _categoryRow({
    Key? key,
    required IconData icon,
    required String title,
    String? description,
    required VoidCallback onTap,
  }) => ListTile(
    key: key,
    leading: Icon(icon),
    title: Text(title),
    subtitle: description == null ? null : Text(description),
    trailing: const Icon(Icons.chevron_right),
    contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
    onTap: onTap,
  );

  Widget _buildTabBody(AppLocalizations l10n) {
    if (_tab != _SettingsTab.appearance) {
      if (_signedIn == null) return const GfSettingsSkeleton();
      if (_signedIn == false) return _buildIndex(l10n);
    }
    if (_needsUser && _user.isLoading && !_user.hasValue) {
      return const GfSettingsSkeleton();
    }
    if (_needsUser && _user.hasError && !_user.hasValue) {
      return GfErrorRetry(
        message: resolveErrorMessage(l10n, _user.error!),
        onRetry: _loadUser,
      );
    }
    return GfScrollToTop(
      semanticLabel: l10n.commonBackToTop,
      key: ValueKey<_SettingsTab?>(_tab),
      builder: (_, ScrollController controller) {
        final content = switch (_tab!) {
          _SettingsTab.profile => _buildProfileTab(l10n, controller),
          _SettingsTab.account => _buildAccountTab(l10n, controller),
          _SettingsTab.privacy => _buildPrivacyTab(l10n, controller),
          _SettingsTab.binding => _buildBindingTab(l10n, controller),
          _SettingsTab.security => _buildSecurityTab(l10n, controller),
          _SettingsTab.appearance => _buildAppearance(l10n, controller),
          _SettingsTab.notifications => _buildNotifications(l10n, controller),
        };
        if (_tab == _SettingsTab.appearance ||
            _tab == _SettingsTab.notifications) {
          return content;
        }
        return AppRefreshIndicator(onRefresh: _refresh, child: content);
      },
    );
  }

  Widget _buildAppearance(AppLocalizations l10n, ScrollController controller) {
    final mode = ref.watch(themeModeProvider);
    final siteTheme = ref.watch(siteThemeProvider);
    return ListView(
      controller: controller,
      padding: const EdgeInsets.all(16),
      children: [
        _settingsSection(
          context,
          title: l10n.settingsAppearance,
          child: RadioGroup<ThemeMode>(
            groupValue: mode,
            onChanged: (value) {
              if (value != null) {
                ref.read(themeModeProvider.notifier).setMode(value);
              }
            },
            child: Column(
              children: [
                for (final choice in ThemeMode.values)
                  RadioListTile<ThemeMode>(
                    key: ValueKey('settings-theme-${choice.name}'),
                    title: Text(switch (choice) {
                      ThemeMode.system => l10n.settingsLanguageSystem,
                      ThemeMode.light => l10n.settingsThemeLight,
                      ThemeMode.dark => l10n.settingsThemeDark,
                    }),
                    value: choice,
                  ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 12),
        _settingsSection(
          context,
          title: l10n.scheduleWidgetSettingsTitle,
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 12,
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            l10n.scheduleWidgetTransparencyTitle,
                            style: Theme.of(context).textTheme.titleSmall,
                          ),
                        ),
                        Text(
                          '$_widgetTransparency%',
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ],
                    ),
                    Slider(
                      value: _widgetTransparency.toDouble(),
                      min: ScheduleWidgetBridge.minTransparencyPercent
                          .toDouble(),
                      max: ScheduleWidgetBridge.maxTransparencyPercent
                          .toDouble(),
                      divisions:
                          ScheduleWidgetBridge.maxTransparencyPercent -
                          ScheduleWidgetBridge.minTransparencyPercent,
                      label: '$_widgetTransparency%',
                      semanticFormatterCallback: (value) =>
                          '${l10n.scheduleWidgetTransparencyTitle}, ${value.round()}%',
                      onChanged: !_widgetTransparencyLoaded
                          ? null
                          : (value) => setState(
                              () => _widgetTransparency = value.round(),
                            ),
                      onChangeEnd: !_widgetTransparencyLoaded
                          ? null
                          : (value) => _saveWidgetTransparency(value.round()),
                    ),
                    Text(
                      l10n.scheduleWidgetTransparencyDescription,
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: GfTheme.colorsOf(context).iconMuted,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        if (siteTheme.available) ...[
          const SizedBox(height: 12),
          _settingsSection(
            context,
            title: l10n.settingsFollowSiteTheme,
            child: SwitchListTile(
              title: Text(l10n.settingsFollowSiteTheme),
              subtitle: Text(l10n.settingsFollowSiteThemeDesc),
              value: siteTheme.following,
              onChanged: (value) =>
                  ref.read(siteThemeProvider.notifier).setFollowing(value),
            ),
          ),
        ],
      ],
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
                  _editProfile(u, section: ProfileEditSection.nickname);
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
                  _editProfile(u, section: ProfileEditSection.bio);
                },
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'camera',
                title: l10n.settingsAvatar,
                description: _uploadingAvatar
                    ? l10n.settingsAvatarUploading
                    : l10n.settingsAvatarSources,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _uploadingAvatar ? null : _chooseAvatar,
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'user-round',
                title: l10n.settingsEditProfile,
                description: l10n.settingsSignature,
                onTap: _user.value == null
                    ? null
                    : () => _editProfile(_user.value!),
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'link',
                title: l10n.settingsProfileLinks,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: _user.value == null
                    ? null
                    : () => _editProfile(
                        _user.value!,
                        section: ProfileEditSection.links,
                      ),
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
                // 两阶段换绑（issue #678）：暂存期内展示待确认的新邮箱与当前邮箱，
                // 否则展示常规的「修改绑定邮箱」入口描述。
                subtitleWidget: _user.when(
                  data: (u) => Text(
                    u.pendingEmail.isNotEmpty
                        ? '${u.email}\n${l10n.settingsEmailPending(u.pendingEmail)}'
                        : u.email.isNotEmpty
                        ? u.email
                        : l10n.settingsEmailEdit,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                  loading: () => Text(l10n.commonLoading),
                  error: (_, _) => Text(l10n.settingsEmailEdit),
                ),
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
              const GfDivider(),
              GfSettingRow(
                symbol: 'award',
                title: l10n.badgeDisplayTitle,
                description: l10n.badgeDisplayHint,
                trailing: const Icon(Icons.chevron_right, size: 18),
                onTap: () {
                  final user = _user.value;
                  if (user != null) _displayBadges(user);
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
        const CampusCacheClearTile(),
        const GfDivider(),
        GfSettingRow(
          symbol: 'calendar-days',
          title: l10n.scheduleWidgetSettingsTitle,
          description: l10n.scheduleWidgetPrivacyDescription,
          trailing: const Icon(Icons.chevron_right, size: 18),
          onTap: () => context.push('/settings/widgets'),
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

  Widget _buildNotifications(
    AppLocalizations l10n,
    ScrollController controller,
  ) => ListView(
    controller: controller,
    padding: const EdgeInsets.all(16),
    children: [
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
    ],
  );

  /// Account security: two-factor authentication, sessions and sign-out.
  Widget _buildSecurityTab(AppLocalizations l10n, ScrollController controller) {
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: <Widget>[
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
                              tooltip: l10n.settingsRevokeSession,
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
    final store = ref.read(writingStoreProvider);
    final chatDraftStore = ref.read(chatDraftStoreProvider);
    final epoch = ref.read(offlineCacheEpochProvider);
    String? scope;
    try {
      scope = await ref.read(writingScopeProvider.future);
    } catch (_) {
      /* Account closure remains available. */
    }
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    try {
      await ref
          .read(contentRepositoryProvider)
          .closeAccount(mode: choice.mode, password: choice.password);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      ref.read(offlineCacheEpochProvider.notifier).invalidate();
      try {
        if (scope != null) {
          await Future.wait([
            chatDraftStore.clearAccount(scope),
            store.clearAccount(scope),
          ]);
        }
      } catch (_) {
        /* A storage failure must not keep a closed account signed in. */
      }
      if (mounted) await _signOutLocally();
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
      ref.read(scheduleWidgetBridgeProvider),
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

/// Settings sections keep related rows together on an inset surface.
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
        padding: const EdgeInsets.only(left: 4, bottom: 8),
        child: Text(
          title,
          style: GfTheme.typographyOf(context).small.copyWith(
            color: colors.iconMuted,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      Material(
        color: colors.base100,
        borderRadius: BorderRadius.circular(20),
        clipBehavior: Clip.antiAlias,
        child: child,
      ),
    ],
  );
}
