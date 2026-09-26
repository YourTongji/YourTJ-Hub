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
import '../../asset_url.dart';
import '../../server_messages.dart';
import '../../theme_mode.dart';
import '../../app_locale.dart';
import '../../widgets/language_picker.dart';
import '../../widgets/user_badge.dart';
import '../../site_theme.dart';
import '../../push/push_service.dart';
import '../../widgets/status_views.dart';
import '../../current_user.dart';
import '../../navigation/auth_navigation.dart';
import 'account_closure_dialog.dart';
import 'campus_cache_clear_tile.dart';
import 'profile_edit_dialog.dart';
import 'password_edit_page.dart';
import 'session_device_label.dart';
import '../../widgets/stickers/sticker_library_page.dart';
import '../../widgets/stickers/sticker_strings.dart';
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
    _SettingsTab.profile => l10n.settingsProfileDisplay,
    _SettingsTab.account => l10n.settingsTabAccount,
    _SettingsTab.privacy => l10n.settingsDataStorage,
    _SettingsTab.binding => l10n.settingsTabBinding,
    _SettingsTab.security => l10n.settingsTabSecurity,
    _SettingsTab.appearance => l10n.settingsAppearance,
    _SettingsTab.notifications => l10n.settingsPush,
  };
}

class _ProfileEditSession {
  bool changed = false;
  Uint8List? uploadedCoverBytes;
  String? uploadedCoverUrl;
}

/// 设置页(web settings.index 的移动端形态)。
///
/// Device preferences remain available to guests. Existing section deep links
/// retain their names; the index opens each category on a normal back stack.
class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({
    super.key,
    this.initialSection,
    this.autoEditProfile = false,
  });
  final String? initialSection;
  final bool autoEditProfile;

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
  bool _autoEditProfileActive = false;
  _ProfileEditSession _autoEditSession = _ProfileEditSession();
  final ImagePicker _imagePicker = ImagePicker();
  final Set<Route<dynamic>> _profileEditRoutes = {};

  @override
  void initState() {
    super.initState();
    _autoEditProfileActive = widget.autoEditProfile;
    for (final section in _SettingsTab.values) {
      if (section.name == widget.initialSection) _tab = section;
    }
    _loadSession();
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
                leading: GfBadgeMedallion(
                  icon: UserBadgeArtwork(b, size: 24),
                  color: userBadgeColor(b),
                  size: 40,
                ),
                title: b.name,
                subtitleWidget: Text(
                  b.description,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: b.code == user.wornBadgeCode
                    ? const GfSymbol('check', size: 20)
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
    final epoch = ref.read(offlineCacheEpochProvider);
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => BadgeDisplayDialog(
          badges: user.badges,
          selected: user.displayBadges ?? user.badges.take(5).toList(),
          onSave: (codes) async {
            if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
              throw const UnauthorizedException();
            }
            await ref.read(userRepositoryProvider).displayBadges(codes);
          },
        ),
      ),
    );
    if (saved == true && mounted) _loadUser(silent: true);
  }

  Future<void> _changePassword() async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => PasswordEditPage(
          onSave: (oldPassword, newPassword) async {
            if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
              throw const UnauthorizedException();
            }
            await ref
                .read(userRepositoryProvider)
                .changePassword(
                  oldPassword: oldPassword,
                  newPassword: newPassword,
                );
          },
        ),
      ),
    );
    if (saved == true && mounted) {
      showGfToast(
        context,
        AppLocalizations.of(context).settingsPasswordUpdated,
      );
    }
  }

  Future<T?> _pushProfileEditRoute<T>(WidgetBuilder builder) async {
    final route = MaterialPageRoute<T>(builder: builder);
    _profileEditRoutes.add(route);
    try {
      return await Navigator.of(context).push<T>(route);
    } finally {
      _profileEditRoutes.remove(route);
    }
  }

  void _closeProfileEditRoutes() {
    // Discard another session's private image drafts immediately, even when a
    // crop route or an in-flight save currently prevents interactive dismissal.
    for (final route in _profileEditRoutes.toList().reversed) {
      if (route.isActive) route.navigator?.removeRoute(route);
    }
    _profileEditRoutes.clear();
  }

  ProfileEditPage _profileEditor(
    SettingsUserPayload user,
    int epoch,
    _ProfileEditSession session, {
    Key? key,
  }) {
    void requireCurrentSession() {
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
        throw const UnauthorizedException();
      }
    }

    return ProfileEditPage(
      key: key,
      user: user,
      onPickImage: (cover) =>
          _pickProfileImageDraft(cover: cover, epoch: epoch),
      onSave: (updated) async {
        requireCurrentSession();
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
        requireCurrentSession();
        session.changed = true;
        ref.invalidate(currentUserProvider);
      },
      onSaveAvatar: (bytes) async {
        requireCurrentSession();
        final url = await ref
            .read(fileRepositoryProvider)
            .uploadAvatar(bytes: bytes, filename: 'avatar.webp');
        requireCurrentSession();
        session.changed = true;
        ref.invalidate(currentUserProvider);
        return url;
      },
      onSaveCover: (bytes) async {
        requireCurrentSession();
        var url = '';
        if (bytes != null) {
          // Retry without uploading an already acknowledged cover again.
          if (!identical(bytes, session.uploadedCoverBytes) ||
              session.uploadedCoverUrl == null) {
            url = await ref
                .read(fileRepositoryProvider)
                .uploadImage(bytes: bytes, filename: 'cover.webp');
            requireCurrentSession();
            session.uploadedCoverBytes = bytes;
            session.uploadedCoverUrl = url;
          } else {
            url = session.uploadedCoverUrl!;
          }
        }
        await ref.read(userRepositoryProvider).saveUserProfileCover(url);
        requireCurrentSession();
        session.changed = true;
        ref.invalidate(currentUserProvider);
        return url;
      },
    );
  }

  Future<void> _editProfile(SettingsUserPayload user) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final session = _ProfileEditSession();
    final saved = await _pushProfileEditRoute<bool>(
      (_) => _profileEditor(user, epoch, session),
    );
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    // A cancelled retry may still have acknowledged earlier independent steps.
    // Refresh those changes, but only announce full success after every save.
    if (session.changed) await _loadUser(silent: true);
    if (saved == true &&
        mounted &&
        epoch == ref.read(offlineCacheEpochProvider)) {
      showGfToast(context, AppLocalizations.of(context).settingsInfoSaved);
    }
  }

  double _profileCoverAspectRatio() {
    final width = MediaQuery.sizeOf(context).width.clamp(0.0, 760.0);
    return width /
        GfUserCard.coverHeightFor(
          width,
          topInset: MediaQuery.viewPaddingOf(context).top,
        );
  }

  Future<Uint8List?> _pickProfileImageDraft({
    required bool cover,
    required int epoch,
  }) async {
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return null;
    final source = await _selectProfileImageSource(cover: cover);
    if (source == null ||
        !mounted ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return null;
    }
    Uint8List? cropped;
    final l = AppLocalizations.of(context);
    await _pushProfileEditRoute<bool>(
      (_) => ProfileImageEditor(
        source: source,
        cover: cover,
        coverPreviewAspectRatio: _profileCoverAspectRatio(),
        confirmLabel: l.commonConfirm,
        savingLabel: l.commonLoading,
        onSave: (bytes) async {
          if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
            throw const UnauthorizedException();
          }
          cropped = bytes;
        },
      ),
    );
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return null;
    return cropped;
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
            leading: const GfSymbol('image'),
            title: Text(l10n.settingsAvatarUpload),
            onTap: () => Navigator.pop(context, false),
          ),
          ListTile(
            leading: const GfSymbol('user-round'),
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

  Future<T?> _showAccountDialog<T>(int epoch, Widget child) =>
      showGfAlertDialog<T>(
        context,
        builder: (_) => _SettingsAccountDialog(epoch: epoch, child: child),
      );

  /// Keep validation and acknowledgements in the editor so failures retain
  /// the user's address and re-authentication input for a deliberate retry.
  Future<void> _changeEmail() async {
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final repository = ref.read(userRepositoryProvider);
    bool isCurrent() => mounted && epoch == ref.read(offlineCacheEpochProvider);
    final saved = await _showAccountDialog<bool>(
      epoch,
      _SettingsCredentialDialog<bool>(
        title: l10n.settingsEmail,
        submitLabel: l10n.commonSave,
        isCurrent: isCurrent,
        fields: [
          _SettingsCredentialField(
            label: l10n.settingsNewEmail,
            keyboardType: TextInputType.emailAddress,
            autofillHints: const [AutofillHints.email],
          ),
          _SettingsCredentialField(
            label: l10n.settingsCurrentPassword,
            obscureText: true,
            autofillHints: const [AutofillHints.password],
          ),
        ],
        onSubmit: (values) =>
            repository.setUserEmail(values[0].trim(), values[1]),
        errorMessage: (error) =>
            error is ApiException &&
                error.messageCode == 'auth.password.oauthRequired'
            ? l10n.settingsEmailOAuthReauthRequired
            : l10n.settingsEmailFailed(resolveErrorMessage(l10n, error)),
      ),
    );
    if (saved != true || !mounted || !isCurrent()) return;
    showGfToast(context, l10n.settingsEmailChangeStaged);
    // The API also supports an immediate switch when verification is off.
    await _loadUser(silent: true);
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
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final repository = ref.read(userRepositoryProvider);
    bool isCurrent() => mounted && epoch == ref.read(offlineCacheEpochProvider);
    String failure(Object error) =>
        l10n.settingsTotpFailed(resolveErrorMessage(l10n, error));
    final codeField = _SettingsCredentialField(
      label: l10n.settingsTotpCode,
      keyboardType: TextInputType.number,
      autofillHints: const [AutofillHints.oneTimeCode],
    );
    try {
      final status = await repository.getTotpStatus();
      if (!isCurrent()) return;
      if (status.enabled) {
        final disabled = await _showAccountDialog<bool>(
          epoch,
          _SettingsCredentialDialog<bool>(
            title: l10n.settingsTotpDisableTitle,
            submitLabel: l10n.settingsTotpDisable,
            submitVariant: GfButtonVariant.danger,
            isCurrent: isCurrent,
            fields: [
              // Disable also accepts the account password; retain its full
              // keyboard instead of forcing the numeric setup-code keypad.
              _SettingsCredentialField(
                label: l10n.settingsTotpCode,
                autofillHints: const [AutofillHints.oneTimeCode],
              ),
            ],
            onSubmit: (values) =>
                repository.disableTotp(code: values.single.trim()),
            errorMessage: failure,
          ),
        );
        if (disabled == true && mounted && isCurrent()) {
          showGfToast(context, l10n.settingsTotpDisabled);
        }
        return;
      }
      final setup = await _showAccountDialog<TotpSetupPayload>(
        epoch,
        _SettingsCredentialDialog<TotpSetupPayload>(
          title: l10n.settingsTotpEnableTitle,
          submitLabel: l10n.settingsTotpNext,
          isCurrent: isCurrent,
          fields: [
            _SettingsCredentialField(
              label: l10n.settingsTotpPassword,
              obscureText: true,
              autofillHints: const [AutofillHints.password],
            ),
          ],
          onSubmit: (values) =>
              repository.getTotpSetup(password: values.single),
          errorMessage: failure,
        ),
      );
      if (setup == null || !isCurrent()) return;
      final enabled = await _showAccountDialog<TotpEnablePayload>(
        epoch,
        _SettingsCredentialDialog<TotpEnablePayload>(
          title: l10n.settingsTotpScanSecret,
          submitLabel: l10n.settingsTotpEnable,
          isCurrent: isCurrent,
          introduction: SelectableText(setup.secret),
          fields: [codeField],
          onSubmit: (values) =>
              repository.enableTotp(code: values.single.trim()),
          errorMessage: failure,
        ),
      );
      if (enabled == null || !mounted || !isCurrent()) return;
      await showGfAlertDialog<void>(
        context,
        builder: (ctx) => _SettingsAccountDialog(
          epoch: epoch,
          child: GfAlertDialog(
            title: Text(l10n.settingsTotpEnabled),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(l10n.settingsTotpRecoveryCodes),
                const SizedBox(height: 8),
                for (final code in enabled.recoveryCodes)
                  SelectableText(
                    code,
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
        ),
      );
    } catch (error) {
      if (isCurrent()) _snack(failure(error));
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

  Future<ProfileCropSource?> _selectProfileImageSource({
    required bool cover,
  }) async {
    final l = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final picked = await _imagePicker.pickImage(
      source: ImageSource.gallery,
      maxWidth: 4096,
      maxHeight: 4096,
    );
    if (picked == null ||
        !mounted ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return null;
    }
    final maxMb = cover ? 10 : 5;
    if (await picked.length() > maxMb * 1024 * 1024) {
      throw ApiException(fallbackMessage: l.settingsImageTooLarge(maxMb));
    }
    ProfileCropSource source;
    try {
      source = await compute(prepareProfileCrop, await picked.readAsBytes());
    } on FormatException {
      throw ApiException(fallbackMessage: l.settingsImageDecodeFailed);
    }
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return null;
    if (cover && (source.width < 1200 || source.height < 240)) {
      throw ApiException(fallbackMessage: l.settingsCoverMinSize);
    }
    return source;
  }

  Future<void> _pickProfileImage({bool cover = false}) async {
    if (_uploadingAvatar) return;
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _uploadingAvatar = true);
    try {
      final source = await _selectProfileImageSource(cover: cover);
      if (source == null ||
          !mounted ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final saved = await _pushProfileEditRoute<bool>(
        (_) => ProfileImageEditor(
          source: source,
          cover: cover,
          coverPreviewAspectRatio: _profileCoverAspectRatio(),
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
              await ref.read(userRepositoryProvider).saveUserProfileCover(url);
            } else {
              await files.uploadAvatar(bytes: bytes, filename: 'avatar.webp');
            }
          },
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
      _closeProfileEditRoutes();
      _userRequest++;
      _sessionsRequest++;
      setState(() {
        _signedIn = null;
        _user = const AsyncValue.loading();
        _sessions = const AsyncValue.loading();
        _googleOAuthReady = false;
        _autoEditProfileActive = false;
        _autoEditSession = _ProfileEditSession();
      });
      unawaited(_loadSession());
    });
    final l10n = AppLocalizations.of(context);
    if (_autoEditProfileActive &&
        _tab == _SettingsTab.profile &&
        _signedIn != false) {
      final user = _user.valueOrNull;
      if (user != null) {
        return _profileEditor(
          user,
          ref.read(offlineCacheEpochProvider),
          _autoEditSession,
          key: ObjectKey(_autoEditSession),
        );
      }
      return Scaffold(
        appBar: GfAppBar(title: Text(l10n.settingsEditProfile)),
        body: _user.hasError
            ? GfErrorRetry(
                message: resolveErrorMessage(l10n, _user.error!),
                onRetry: _loadUser,
              )
            : const GfSettingsSkeleton(),
      );
    }
    final title = _tab?.label(l10n) ?? l10n.settingsTitle;
    final colors = GfTheme.colorsOf(context);
    return Scaffold(
      backgroundColor: colors.base200,
      appBar: GfAppBar(
        leading: IconButton(
          icon: const GfSymbol('chevron-left'),
          tooltip: l10n.commonBack,
          onPressed: _leaveSettings,
        ),
        title: Text(title),
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
              symbol: 'palette',
              title: l10n.settingsAppearance,
              description: switch (ref.watch(themeModeProvider)) {
                ThemeMode.system => l10n.settingsLanguageSystem,
                ThemeMode.light => l10n.settingsThemeLight,
                ThemeMode.dark => l10n.settingsThemeDark,
              },
              onTap: () => _openSection(_SettingsTab.appearance),
            ),
            const GfDivider(),
            _categoryRow(
              key: const ValueKey('settings-category-language'),
              symbol: 'languages',
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
              symbol: 'info',
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
                  symbol: switch (section) {
                    _SettingsTab.profile => 'user-round',
                    _SettingsTab.account => 'at-sign',
                    _SettingsTab.privacy => 'folder',
                    _SettingsTab.binding => 'link',
                    _SettingsTab.notifications => 'bell',
                    _ => 'shield-check',
                  },
                  title: section.label(l10n),
                  onTap: () => _openSection(section),
                ),
                if (section == _SettingsTab.profile) ...[
                  const GfDivider(),
                  GfSettingRow(
                    symbol: 'smile',
                    title: StickerStrings(context).title,
                    onTap: () => Navigator.of(context).push(
                      MaterialPageRoute<void>(
                        builder: (_) => const StickerLibraryPage(),
                      ),
                    ),
                  ),
                ],
              ]
            else if (_signedIn == false)
              _categoryRow(
                symbol: 'log-out',
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

  Widget _categoryRow({
    Key? key,
    required String symbol,
    required String title,
    String? description,
    required VoidCallback onTap,
  }) => GfSettingRow(
    key: key,
    symbol: symbol,
    title: title,
    description: description,
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
          child: GfSettingRow(
            symbol: 'calendar-days',
            title: l10n.scheduleWidgetSettingsTitle,
            onTap: () => context.push('/settings/widgets'),
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

  Widget _buildProfileTab(AppLocalizations l10n, ScrollController controller) {
    final user = _user.value;
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: [
        _settingsSection(
          context,
          child: Column(
            children: [
              GfSettingRow(
                symbol: 'user-round',
                title: l10n.settingsEditProfile,
                description: user == null
                    ? l10n.commonLoading
                    : [
                        user.nickname.isEmpty ? user.username : user.nickname,
                        user.bio,
                      ].where((value) => value.isNotEmpty).join(' · '),
                onTap: user == null ? null : () => _editProfile(user),
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'camera',
                title: l10n.settingsAvatar,
                description: _uploadingAvatar
                    ? l10n.settingsAvatarUploading
                    : null,
                leading: user == null
                    ? null
                    : GfAvatar(
                        src: resolveApiAssetUrl(user.avatarUrl),
                        size: 32,
                      ),
                onTap: _uploadingAvatar ? null : _chooseAvatar,
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'image',
                title: l10n.settingsCover,
                description: user?.profileCoverUrl.isNotEmpty == true
                    ? null
                    : l10n.settingsNotSet,
                onTap: _uploadingAvatar
                    ? null
                    : () => _pickProfileImage(cover: true),
              ),
              if (user?.profileCoverUrl.isNotEmpty == true) ...[
                const GfDivider(),
                GfSettingRow(
                  symbol: 'image-off',
                  title: l10n.settingsCoverRemove,
                  onTap: _uploadingAvatar ? null : _removeCover,
                ),
              ],
              const GfDivider(),
              GfSettingRow(
                symbol: 'award',
                title: l10n.settingsBadge,
                description: user?.wornBadge?.name ?? l10n.settingsBadgeNone,
                onTap: user == null ? null : () => _pickBadge(user),
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'award',
                title: l10n.badgeDisplayTitle,
                description: user == null
                    ? l10n.commonLoading
                    : l10n.badgeDisplaySelectedCount(
                        (user.displayBadges ?? user.badges.take(5).toList())
                            .length,
                      ),
                onTap: user == null ? null : () => _displayBadges(user),
              ),
            ],
          ),
        ),
      ],
    );
  }

  /// Account identity and security credentials.
  Widget _buildAccountTab(AppLocalizations l10n, ScrollController controller) {
    return ListView(
      controller: controller,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.all(16),
      children: <Widget>[
        _settingsSection(
          context,
          child: Column(
            children: [
              GfSettingRow(
                symbol: 'at-sign',
                title: l10n.authUsername,
                description: _user.value?.username ?? '',
                trailing: const GfSymbol('chevron-right', size: 18),
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
                trailing: const GfSymbol('chevron-right', size: 18),
                onTap: _changeEmail,
              ),
              const GfDivider(),
              GfSettingRow(
                symbol: 'key-round',
                title: l10n.settingsChangePassword,
                description: l10n.settingsChangePasswordSub,
                trailing: const GfSymbol('chevron-right', size: 18),
                onTap: _changePassword,
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),
        TextButton(
          onPressed: _accountClosing ? null : _closeAccount,
          child: Text(l10n.settingsCloseAccount),
        ),
      ],
    );
  }

  /// Legacy privacy deep links now open data and local-storage tools.
  Widget _buildPrivacyTab(AppLocalizations l10n, ScrollController controller) =>
      ListView(
        controller: controller,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(16),
        children: [
          _settingsSection(
            context,
            title: l10n.profileContent,
            child: Column(
              children: [
                GfSettingRow(
                  symbol: 'folder',
                  title: l10n.profileContent,
                  onTap: () => context.push('/my-content'),
                ),
                const GfDivider(),
                GfSettingRow(
                  symbol: 'trash-2',
                  title: l10n.profileTrash,
                  onTap: () => context.push('/recycle-bin'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          _settingsSection(
            context,
            title: l10n.settingsDataStorage,
            child: const CampusCacheClearTile(),
          ),
        ],
      );

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
                trailing: const GfSymbol('chevron-right', size: 18),
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
      Consumer(
        builder: (context, ref, _) {
          final push = ref.watch(pushControllerProvider);
          final isIOS = defaultTargetPlatform == TargetPlatform.iOS;
          final preferenceOn =
              push == PushChannelStatus.enabled ||
              push == PushChannelStatus.permissionDenied ||
              push == PushChannelStatus.serverDisabled ||
              push == PushChannelStatus.registrationFailed;
          final delivery = switch (push) {
            PushChannelStatus.enabled => l10n.settingsPushReady,
            PushChannelStatus.disabled => l10n.settingsPushOff,
            PushChannelStatus.unknown => l10n.commonLoading,
            PushChannelStatus.unsupported => l10n.settingsPushUnsupported,
            PushChannelStatus.serverDisabled => l10n.settingsPushServerDisabled,
            PushChannelStatus.registrationFailed => l10n.settingsPushFailed,
            PushChannelStatus.permissionDenied => l10n.settingsPushDenied,
          };
          return _settingsSection(
            context,
            title: l10n.settingsPush,
            child: Column(
              children: [
                GfSwitchRow(
                  symbol: 'bell',
                  title: l10n.settingsPushPreference,
                  description: isIOS
                      ? l10n.settingsPushIOSConsent
                      : l10n.settingsPushAndroidConsent,
                  value: preferenceOn,
                  onChanged: (value) async {
                    final controller = ref.read(
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
                  symbol: 'smartphone',
                  title: l10n.settingsPushDelivery,
                  description: delivery,
                  onTap: push == PushChannelStatus.permissionDenied
                      ? () => ref
                            .read(pushControllerProvider.notifier)
                            .openSystemSettings()
                      : push == PushChannelStatus.serverDisabled ||
                            push == PushChannelStatus.registrationFailed
                      ? () => ref.read(pushControllerProvider.notifier).enable()
                      : null,
                ),
                if (!isIOS) ...[
                  const GfDivider(),
                  GfSettingRow(
                    symbol: 'info',
                    title: l10n.settingsPushPrivacy,
                    onTap: () => launchUrl(
                      Uri.parse('https://www.jiguang.cn/license/privacy'),
                      mode: LaunchMode.externalApplication,
                    ),
                  ),
                ],
              ],
            ),
          );
        },
      ),
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
            title: l10n.settingsTotpEnable,
            description: l10n.settingsTotpSetupSecret,
            trailing: const GfSymbol('chevron-right', size: 18),
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
                      title: sessionDeviceLabel(
                        s.userAgent,
                        l10n,
                        isCurrent: s.isCurrent,
                      ),
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
                              symbol: 'trash-2',
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
  String? title,
  required Widget child,
}) {
  final GfColors colors = GfTheme.colorsOf(context);
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      if (title != null)
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
        borderRadius: BorderRadius.circular(16),
        clipBehavior: Clip.antiAlias,
        child: child,
      ),
    ],
  );
}

/// Dialog routes outlive their settings subtree. Remove exactly this route on
/// an account switch, including while a credential write is still pending.
class _SettingsAccountDialog extends ConsumerStatefulWidget {
  const _SettingsAccountDialog({required this.epoch, required this.child});
  final int epoch;
  final Widget child;

  @override
  ConsumerState<_SettingsAccountDialog> createState() =>
      _SettingsAccountDialogState();
}

class _SettingsAccountDialogState
    extends ConsumerState<_SettingsAccountDialog> {
  bool _closing = false;

  void _close() {
    if (_closing) return;
    _closing = true;
    final route = ModalRoute.of(context);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (route?.isActive == true) route!.navigator?.removeRoute(route);
    });
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      if (next != widget.epoch) _close();
    });
    if (ref.watch(offlineCacheEpochProvider) != widget.epoch) {
      _close();
      return const SizedBox.shrink();
    }
    return widget.child;
  }
}

class _SettingsCredentialField {
  const _SettingsCredentialField({
    required this.label,
    this.obscureText = false,
    this.keyboardType,
    this.autofillHints,
  });
  final String label;
  final bool obscureText;
  final TextInputType? keyboardType;
  final Iterable<String>? autofillHints;
}

/// A small credential step owns its input until the server acknowledges it.
class _SettingsCredentialDialog<T> extends StatefulWidget {
  const _SettingsCredentialDialog({
    required this.title,
    required this.submitLabel,
    required this.fields,
    required this.onSubmit,
    required this.errorMessage,
    required this.isCurrent,
    this.submitVariant = GfButtonVariant.primary,
    this.introduction,
  });
  final String title;
  final String submitLabel;
  final List<_SettingsCredentialField> fields;
  final Future<T> Function(List<String>) onSubmit;
  final String Function(Object) errorMessage;
  final bool Function() isCurrent;
  final GfButtonVariant submitVariant;
  final Widget? introduction;

  @override
  State<_SettingsCredentialDialog<T>> createState() =>
      _SettingsCredentialDialogState<T>();
}

class _SettingsCredentialDialogState<T>
    extends State<_SettingsCredentialDialog<T>> {
  late final _controllers = [
    for (final _ in widget.fields) TextEditingController(),
  ];
  bool _saving = false;
  String? _error;

  @override
  void dispose() {
    for (final controller in _controllers) {
      controller.dispose();
    }
    super.dispose();
  }

  Future<void> _submit() async {
    if (_saving || !widget.isCurrent()) return;
    final values = [for (final controller in _controllers) controller.text];
    if (values.any((value) => value.trim().isEmpty)) {
      setState(
        () => _error = AppLocalizations.of(context).settingsFillComplete,
      );
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final result = await widget.onSubmit(values);
      if (!mounted || !widget.isCurrent()) return;
      Navigator.pop(context, result);
    } catch (error) {
      if (mounted && widget.isCurrent()) {
        setState(() => _error = widget.errorMessage(error));
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return PopScope(
      canPop: !_saving,
      child: GfAlertDialog(
        title: Text(widget.title),
        content: AutofillGroup(
          onDisposeAction: AutofillContextAction.cancel,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (widget.introduction != null) ...[
                widget.introduction!,
                const SizedBox(height: 12),
              ],
              for (var index = 0; index < widget.fields.length; index++) ...[
                if (index > 0) const SizedBox(height: 16),
                GfInput(
                  controller: _controllers[index],
                  enabled: !_saving,
                  labelText: widget.fields[index].label,
                  obscureText: widget.fields[index].obscureText,
                  keyboardType: widget.fields[index].keyboardType,
                  autofillHints: widget.fields[index].autofillHints,
                  autocorrect: false,
                  enableSuggestions: false,
                  textInputAction: index == widget.fields.length - 1
                      ? TextInputAction.done
                      : TextInputAction.next,
                  onSubmitted: index == widget.fields.length - 1
                      ? (_) => _submit()
                      : null,
                ),
              ],
              if (_error != null) ...[
                const SizedBox(height: 12),
                GfStatusMessage(message: _error!),
              ],
            ],
          ),
        ),
        actions: [
          GfButton(
            label: l10n.commonCancel,
            variant: GfButtonVariant.ghost,
            onPressed: _saving ? null : () => Navigator.pop(context),
          ),
          GfButton(
            label: widget.submitLabel,
            variant: widget.submitVariant,
            loading: _saving,
            onPressed: _saving ? null : _submit,
          ),
        ],
      ),
    );
  }
}
