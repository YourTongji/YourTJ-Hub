import '../../navigation/reading_window.dart';
import '../../navigation/account_return.dart';
import 'decoration_page.dart';
import '../../user_blocks.dart';
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
import '../../widgets/activation_email_action.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../local/writing_store.dart';
import '../../messages/chat_drafts.dart';
import '../../asset_url.dart';
import '../../server_messages.dart';
import '../../theme_mode.dart';
import '../../app_locale.dart';
import '../../widgets/language_picker.dart';
import '../../site_theme.dart';
import '../../push/push_service.dart';
import '../../push/push_preferences_panel.dart';
import '../../widgets/status_views.dart';
import '../../current_user.dart';
import '../../navigation/auth_navigation.dart';
import 'account_closure_dialog.dart';
import 'storage_settings_panel.dart';
import 'profile_edit_dialog.dart';
import 'password_edit_page.dart';
import 'session_device_label.dart';
import 'text_size_settings_page.dart';
import 'schedule_widget_settings_page.dart';
import 'username_edit_dialog.dart';
import 'passkey_settings_page.dart';
import 'badge_display_dialog.dart';
import 'profile_image_editor.dart';
import 'oauth_bindings_sheet.dart';
import '../../widgets/profile_image_crop.dart';
import '../../widgets/skeletons.dart';

enum _SettingsTab {
  profile,
  account,
  privacy,
  appearance,
  textSize,
  widgets,
  data,
  notifications;

  String label(AppLocalizations l10n) => switch (this) {
    _SettingsTab.profile => l10n.settingsProfileDisplay,
    _SettingsTab.account => l10n.settingsTabAccount,
    _SettingsTab.privacy => l10n.settingsTabPrivacy,
    _SettingsTab.data => l10n.settingsDataStorage,
    _SettingsTab.appearance => l10n.settingsAppearance,
    _SettingsTab.textSize => l10n.textSizeTitle,
    _SettingsTab.widgets => l10n.scheduleWidgetSettingsTitle,
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
  bool _requiresEmailVerification = false;
  int _sessionRequest = 0;
  int _userRequest = 0;
  int _sessionsRequest = 0;
  int _activationRevision = 0;
  AsyncValue<List<UserSessionPayload>> _sessions = const AsyncValue.loading();
  AsyncValue<SettingsUserPayload> _user = const AsyncValue.loading();
  bool _accountClosing = false;
  bool _googleOAuthReady = false;
  bool _appleOAuthReady = false;
  bool _autoEditProfileActive = false;
  bool _autoEditBadges = false;
  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();
  _ProfileEditSession _autoEditSession = _ProfileEditSession();
  final ImagePicker _imagePicker = ImagePicker();
  final Set<Route<dynamic>> _profileEditRoutes = {};
  final ScrollController _directoryScrollController = ScrollController();

  double get _layoutTextScale =>
      (MediaQuery.textScalerOf(context).scale(16) / 16).clamp(
        1.0,
        double.infinity,
      );
  bool get _hasDirectory =>
      MediaQuery.sizeOf(context).width >= 800 * _layoutTextScale;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // Wide settings opens a useful device preference without fetching an account.
    if (_hasDirectory && _tab == null) {
      _tab = _SettingsTab.appearance;
    }
  }

  @override
  void initState() {
    super.initState();
    _autoEditProfileActive =
        widget.autoEditProfile || widget.initialSection == 'profile';
    _autoEditBadges = widget.initialSection == 'badges';
    for (final section in _SettingsTab.values) {
      final incoming = switch (widget.initialSection) {
        'security' || 'binding' => 'account',
        'badges' => 'profile',
        'privacy' => 'data',
        'privacy-controls' => 'privacy',
        _ => widget.initialSection,
      };
      if (section.name == incoming) _tab = section;
    }
    _loadSession();
  }

  @override
  void dispose() {
    _directoryScrollController.dispose();
    _scrollToTopController.dispose();
    super.dispose();
  }

  bool get _needsUser =>
      _tab == null ||
      _tab == _SettingsTab.profile ||
      _tab == _SettingsTab.account ||
      _hasDirectory;

  bool get _devicePage =>
      _tab == _SettingsTab.appearance ||
      _tab == _SettingsTab.data ||
      _tab == _SettingsTab.textSize ||
      _tab == _SettingsTab.widgets;

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
    if (_tab == _SettingsTab.account) unawaited(_loadSessions());
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
        _requiresEmailVerification =
            payload.layout.viewer.requiresEmailVerification;
        _googleOAuthReady = props.googleOAuthReady;
        _appleOAuthReady = props.appleOAuthReady;
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
      if (_signedIn == true && _tab == _SettingsTab.account)
        _loadSessions(silent: true),
    ]);
  }

  void _leaveSettings() {
    final NavigatorState navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    if (_tab != null && !_hasDirectory) {
      setState(() => _tab = null);
      return;
    }
    context.go(_signedIn == true ? '/profile' : '/');
  }

  BadgeDisplayDialog _badgeEditor(SettingsUserPayload user, int epoch) =>
      BadgeDisplayDialog(
        badges: user.badges,
        onClose: _closeAccountEditor,
        wearableBadges: user.wearableBadges,
        wornBadgeCode: user.wornBadgeCode,
        onWear: (code) async {
          if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
            throw const UnauthorizedException();
          }
          await ref.read(userRepositoryProvider).wearBadge(code);
        },
        selected: user.displayBadges ?? user.badges.take(5).toList(),
        onSave: (codes) async {
          if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
            throw const UnauthorizedException();
          }
          await ref.read(userRepositoryProvider).displayBadges(codes);
          if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
            ref.invalidate(currentUserProvider);
          }
        },
      );

  Future<void> _displayBadges(SettingsUserPayload user) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    await _pushProfileEditRoute<bool>(
      (_) => ReadingPageWindow(child: _badgeEditor(user, epoch)),
    );
    if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
      await _loadUser(silent: true);
      ref.invalidate(currentUserProvider);
    }
  }

  Future<void> _changePassword() async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => ReadingPageWindow(
          child: PasswordEditPage(
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
      ),
    );
    if (saved == true && mounted) {
      showGfToast(
        context,
        AppLocalizations.of(context).settingsPasswordUpdated,
      );
    }
  }

  void _closeAccountEditor(bool saved) {
    final navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop(saved);
    } else {
      final router = GoRouter.maybeOf(context);
      if (router != null) {
        router.go('/settings');
      } else {
        setState(() {
          _tab = null;
          _autoEditProfileActive = false;
          _autoEditBadges = false;
        });
        if (_signedIn == true) unawaited(_loadUser(silent: true));
      }
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
      onClose: _closeAccountEditor,
      onPickImage: (cover, source) =>
          _pickProfileImageDraft(cover: cover, epoch: epoch, source: source),
      onSavePresetAvatar: (path) async {
        requireCurrentSession();
        await ref.read(userRepositoryProvider).savePresetAvatar(path);
        requireCurrentSession();
        session.changed = true;
        ref.invalidate(currentUserProvider);
      },
      onEditUsername: _changeUsername,
      onReloadUser: () async {
        requireCurrentSession();
        return _user.valueOrNull;
      },
      onPushLinks: (builder) =>
          _pushProfileEditRoute<Map<String, String>>(builder),
      onOpenDecorations: () async {
        await _pushProfileEditRoute<void>(
          (_) => DecorationPage(
            avatarUrl: user.avatarUrl,
            name: user.nickname.isEmpty ? user.username : user.nickname,
            username: user.username,
          ),
        );
        if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
          await _loadUser(silent: true);
        }
      },
      onOpenBadges: () => _displayBadges(user),
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
      (_) => ReadingPageWindow(child: _profileEditor(user, epoch, session)),
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
    required ProfileImageSource source,
  }) async {
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return null;
    final picked = await _selectProfileImageSource(
      cover: cover,
      imageSource: source,
    );
    if (picked == null ||
        !mounted ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return null;
    }
    Uint8List? cropped;
    final l = AppLocalizations.of(context);
    await _pushProfileEditRoute<bool>(
      (_) => ProfileImageEditor(
        source: picked,
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
    final epoch = ref.read(offlineCacheEpochProvider);
    final repository = ref.read(userRepositoryProvider);
    final saved = await showDialog<bool>(
      context: context,
      animationStyle: GfMotion.dialogStyle(context),
      barrierDismissible: false,
      builder: (_) => _SettingsAccountDialog(
        epoch: epoch,
        child: UsernameEditDialog(
          username: user.username,
          onSave: (username) async {
            if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
              throw const UnauthorizedException();
            }
            await repository.saveUserName(username);
            if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
              throw const UnauthorizedException();
            }
          },
        ),
      ),
    );
    if (saved != true ||
        !mounted ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    ref.invalidate(currentUserProvider);
    await _loadUser(silent: true);
    if (mounted) {
      showGfToast(
        context,
        AppLocalizations.of(context).settingsUsernameUpdated,
      );
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
    // Recreate the owner status reader immediately: loading /settings alone does
    // not invalidate its verified/pending snapshot or cancel an old status read.
    setState(() => _activationRevision++);
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
        appleReady: _appleOAuthReady,
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
              // Obscure it so a typed password never enters the keyboard
              // dictionary; the credential dialog then keeps suggestions off
              // for this field (issue #1111).
              _SettingsCredentialField(
                label: l10n.settingsTotpCode,
                obscureText: true,
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
          introduction: GfSelectableText(setup.secret),
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
            title: GfText(l10n.settingsTotpEnabled),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                GfText(l10n.settingsTotpRecoveryCodes),
                const SizedBox(height: 8),
                for (final code in enabled.recoveryCodes)
                  GfSelectableText(
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
    ProfileImageSource imageSource = ProfileImageSource.gallery,
  }) async {
    final l = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final picked = await _imagePicker.pickImage(
      source: imageSource == ProfileImageSource.camera
          ? ImageSource.camera
          : ImageSource.gallery,
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
        _appleOAuthReady = false;
        if (_autoEditProfileActive || _autoEditBadges) _tab = null;
        _autoEditProfileActive = false;
        _autoEditBadges = false;
        _autoEditSession = _ProfileEditSession();
      });
      unawaited(_loadSession());
    });
    ref.listen(accountReturnRefreshProvider, (previous, next) {
      if (previous == next ||
          _signedIn != true ||
          _tab != _SettingsTab.account) {
        return;
      }
      setState(() => _activationRevision++);
      unawaited(_refresh());
    });
    final l10n = AppLocalizations.of(context);
    if (_autoEditBadges && _signedIn != false) {
      final user = _user.valueOrNull;
      if (user != null) {
        return _badgeEditor(user, ref.read(offlineCacheEpochProvider));
      }
      return Scaffold(
        appBar: GfAppBar(title: GfText(l10n.settingsBadge)),
        body: _user.hasError
            ? GfErrorRetry(
                message: resolveErrorMessage(l10n, _user.error!),
                onRetry: _loadUser,
              )
            : const GfSettingsSkeleton(),
      );
    }
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
        appBar: GfAppBar(title: GfText(l10n.settingsEditProfile)),
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
    final hasDirectory = _hasDirectory;
    final scaffold = Scaffold(
      backgroundColor: colors.canvasGrouped,
      appBar: GfAppBar(
        backgroundColor: colors.canvasGrouped,
        showDivider: false,
        leading: IconButton(
          icon: const GfSymbol('chevron-left'),
          tooltip: l10n.commonBack,
          onPressed: _leaveSettings,
        ),
        title: GfText(hasDirectory ? l10n.settingsTitle : title),
      ),
      body: SafeArea(
        top: false,
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 1120),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Offstage(
                  offstage: !hasDirectory,
                  child: SizedBox(
                    key: const ValueKey('settings-directory'),
                    width: 292 * _layoutTextScale,
                    child: Scrollbar(
                      controller: _directoryScrollController,
                      thumbVisibility: true,
                      child: PrimaryScrollController(
                        controller: _directoryScrollController,
                        child: _buildIndexList(
                          l10n,
                          _directoryScrollController,
                        ),
                      ),
                    ),
                  ),
                ),
                SizedBox(
                  width: hasDirectory ? 1 : 0,
                  child: ColoredBox(color: colors.line),
                ),
                Expanded(
                  child: Align(
                    alignment: Alignment.topCenter,
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 720),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Padding(
                            padding: EdgeInsets.fromLTRB(
                              16,
                              hasDirectory ? 24 : 0,
                              16,
                              hasDirectory ? 12 : 0,
                            ),
                            child: hasDirectory
                                ? GfText(
                                    title,
                                    key: const ValueKey(
                                      'settings-detail-title',
                                    ),
                                    style: GfTheme.typographyOf(context).title2
                                        .copyWith(fontWeight: FontWeight.w700),
                                  )
                                : const SizedBox.shrink(),
                          ),
                          Expanded(
                            key: const ValueKey('settings-detail'),
                            child: _tab == null
                                ? _buildIndex(l10n)
                                : _buildTabBody(l10n),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    return GfStatusBarScrollToTop(
      controller: _scrollToTopController,
      child: scaffold,
    );
  }

  void _openSection(_SettingsTab section) {
    if (_hasDirectory) {
      setState(() => _tab = section);
      if (_signedIn == true) {
        if (_needsUser && !_user.hasValue) unawaited(_loadUser());
        if (section == _SettingsTab.account && !_sessions.hasValue) {
          unawaited(_loadSessions());
        }
      }
      return;
    }
    Navigator.of(context).push<void>(
      MaterialPageRoute(
        builder: (_) => ReadingPageWindow(
          maxContentWidth: 1120,
          child: SettingsPage(
            initialSection: section == _SettingsTab.privacy
                ? 'privacy-controls'
                : section.name,
          ),
        ),
      ),
    );
  }

  Widget _buildIndex(AppLocalizations l10n) => GfScrollToTop(
    controller: _scrollToTopController,
    semanticLabel: l10n.commonBackToTop,
    showButton: false,
    builder: (_, ScrollController controller) =>
        _buildIndexList(l10n, controller),
  );

  Widget _buildIndexList(
    AppLocalizations l10n,
    ScrollController controller,
  ) => ListView(
    controller: controller,
    padding: const EdgeInsets.all(16),
    children: [
      if (_signedIn == true) ...[
        _settingsSection(
          context,
          child: GfSettingRow(
            key: const Key('settings-profile-card'),
            symbol: 'user-round',
            title:
                _user.valueOrNull?.nickname ??
                (_user.hasError ? l10n.commonRetry : l10n.settingsEditProfile),
            description: _user.valueOrNull == null
                ? (_user.hasError
                      ? resolveErrorMessage(l10n, _user.error!)
                      : l10n.commonLoading)
                : '@${_user.valueOrNull!.username}',
            trailing: _user.valueOrNull == null
                ? null
                : GfAvatar(
                    src: resolveApiAssetUrl(_user.valueOrNull!.avatarUrl),
                    size: 40,
                  ),
            onTap: _user.valueOrNull == null
                ? (_user.hasError ? () => _loadUser() : null)
                : () => _editProfile(_user.valueOrNull!),
          ),
        ),
        const SizedBox(height: 20),
        _settingsSection(
          context,
          title: l10n.settingsYourAccount,
          child: Column(
            children: [
              for (final section in [
                _SettingsTab.account,
                _SettingsTab.privacy,
                _SettingsTab.notifications,
              ]) ...[
                if (section != _SettingsTab.account) const GfDivider(),
                _categoryRow(
                  key: ValueKey('settings-category-${section.name}'),
                  section: section,
                  symbol: switch (section) {
                    _SettingsTab.account => 'shield-check',
                    _SettingsTab.privacy => 'eye',
                    _ => 'bell',
                  },
                  title: section.label(l10n),
                  onTap: () => _openSection(section),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(height: 20),
      ],
      _settingsSection(
        context,
        title: l10n.settingsDevice,
        child: Column(
          children: [
            for (final section in [
              _SettingsTab.appearance,
              _SettingsTab.textSize,
              _SettingsTab.widgets,
              _SettingsTab.data,
            ]) ...[
              if (section != _SettingsTab.appearance) const GfDivider(),
              _categoryRow(
                key: ValueKey('settings-category-${section.name}'),
                section: section,
                symbol: switch (section) {
                  _SettingsTab.appearance => 'palette',
                  _SettingsTab.textSize => 'type',
                  _SettingsTab.widgets => 'calendar-days',
                  _ => 'folder',
                },
                title: section.label(l10n),
                onTap: () => _openSection(section),
              ),
            ],
            const GfDivider(),
            _categoryRow(
              key: const Key('settings-category-language'),
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
              key: const Key('settings-category-about'),
              symbol: 'info',
              title: l10n.siteInfoTitle,
              onTap: () => context.push('/about'),
            ),
          ],
        ),
      ),
      if (_signedIn == false) ...[
        const SizedBox(height: 20),
        GfButton(
          label: l10n.authLoginTitle,
          onPressed: () => context.push(
            authLoginLocation(
              returnTo: _tab == null ? '/settings' : '/settings/${_tab!.name}',
            ),
          ),
        ),
      ],
      if (_signedIn == true) ...[
        const SizedBox(height: 24),
        GfButton(
          label: l10n.settingsLogout,
          variant: GfButtonVariant.outline,
          expanded: true,
          onPressed: _logout,
        ),
      ],
    ],
  );

  Widget _categoryRow({
    Key? key,
    _SettingsTab? section,
    required String symbol,
    required String title,
    String? description,
    required VoidCallback onTap,
  }) {
    final selected = _hasDirectory && section != null && _tab == section;
    return Semantics(
      selected: section == null ? null : selected,
      child: ColoredBox(
        color: selected
            ? GfTheme.colorsOf(context).primary.withValues(alpha: .08)
            : Colors.transparent,
        child: GfSettingRow(
          key: key,
          symbol: symbol,
          title: title,
          description: description,
          trailing: selected
              ? GfSymbol(
                  'check',
                  size: 18,
                  color: GfTheme.colorsOf(context).primary,
                )
              : null,
          onTap: onTap,
        ),
      ),
    );
  }

  Widget _buildTabBody(AppLocalizations l10n) {
    if (_tab == _SettingsTab.textSize) {
      return const TextSizeSettingsPage(embedded: true);
    }
    if (_tab == _SettingsTab.widgets) {
      return const ScheduleWidgetSettingsPage(embedded: true);
    }
    if (!_devicePage) {
      if (_signedIn == null) return const GfSettingsSkeleton();
      if (_signedIn == false) {
        if (!_hasDirectory) return _buildIndex(l10n);
        return Center(
          child: GfButton(
            label: l10n.authLoginTitle,
            onPressed: () => context.push(
              authLoginLocation(returnTo: '/settings/${_tab!.name}'),
            ),
          ),
        );
      }
    }
    if (_signedIn == true &&
        _tab == _SettingsTab.profile &&
        _user.isLoading &&
        !_user.hasValue) {
      return const GfSettingsSkeleton();
    }
    if (_signedIn == true &&
        _tab == _SettingsTab.profile &&
        _user.hasError &&
        !_user.hasValue) {
      return GfErrorRetry(
        message: resolveErrorMessage(l10n, _user.error!),
        onRetry: _loadUser,
      );
    }
    return GfScrollToTop(
      controller: _scrollToTopController,
      semanticLabel: l10n.commonBackToTop,
      key: ValueKey<_SettingsTab?>(_tab),
      builder: (_, ScrollController controller) {
        final content = switch (_tab!) {
          _SettingsTab.profile => const GfSettingsSkeleton(),
          _SettingsTab.account => _buildAccountTab(l10n, controller),
          _SettingsTab.privacy => _buildPrivacyTab(l10n, controller),
          _SettingsTab.data => StorageSettingsPanel(
            controller: controller,
            signedIn: _signedIn == true,
          ),
          _SettingsTab.appearance => _buildAppearance(l10n, controller),
          _SettingsTab.textSize || _SettingsTab.widgets => throw StateError(
            'Device pages use their own scroll views',
          ),
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
                  GfRadioListTile<ThemeMode>(
                    key: ValueKey('settings-theme-${choice.name}'),
                    title: GfText(switch (choice) {
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
        if (siteTheme.available) ...[
          const SizedBox(height: 12),
          _settingsSection(
            context,
            child: SwitchListTile(
              title: GfText(l10n.settingsFollowSiteTheme),
              subtitle: GfText(l10n.settingsFollowSiteThemeDesc),
              value: siteTheme.following,
              onChanged: (value) =>
                  ref.read(siteThemeProvider.notifier).setFollowing(value),
            ),
          ),
        ],
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
        if (_user.hasError && !_user.hasValue)
          _settingsSection(
            context,
            child: GfErrorRetry(
              message: resolveErrorMessage(l10n, _user.error!),
              onRetry: _loadUser,
            ),
          )
        else if (_user.isLoading && !_user.hasValue)
          const GfSettingsSkeleton()
        else
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
                    data: (u) => GfText(
                      u.pendingEmail.isNotEmpty
                          ? '${u.email}\n${l10n.settingsEmailPending(u.pendingEmail)}'
                          : u.email.isNotEmpty
                          ? u.email
                          : l10n.settingsEmailEdit,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                    loading: () => GfText(l10n.commonLoading),
                    error: (_, _) => GfText(l10n.settingsEmailEdit),
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
        const SizedBox(height: 16),
        _settingsSection(
          context,
          child: GfSettingRow(
            symbol: 'link',
            title: l10n.settingsOAuth,
            description: l10n.settingsOAuthSub,
            onTap: _manageOAuth,
          ),
        ),
        if (_requiresEmailVerification && _user.valueOrNull != null) ...[
          _settingsSection(
            context,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                key: const ValueKey('account-activation-pending'),
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.authVerificationPending,
                    style: GfTheme.typographyOf(context).title3,
                  ),
                  const SizedBox(height: 8),
                  Text(_user.value!.email),
                  const SizedBox(height: 8),
                  Text(l10n.authVerificationPendingHint),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),
        ],
        ActivationEmailAction(key: ValueKey(_activationRevision)),
        const SizedBox(height: 12),
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
          title: l10n.settingsPasskeyTitle,
          child: GfSettingRow(
            symbol: 'key-round',
            title: l10n.settingsPasskeyManage,
            description: l10n.settingsPasskeyDescription,
            trailing: const GfSymbol('chevron-right', size: 18),
            onTap: () => Navigator.of(context).push<void>(
              MaterialPageRoute<void>(
                builder: (_) => const PasskeySettingsPage(),
              ),
            ),
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
                      subtitleWidget: GfText(
                        '${s.ipMasked} · ${_formatTs(s.createdAt)}',
                        style: GfTheme.typographyOf(context).caption,
                      ),
                      trailing: s.isCurrent
                          ? GfText(
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
        const SizedBox(height: 24),
        _settingsSection(
          context,
          child: GfSettingRow(
            key: const Key('settings-close-account'),
            symbol: 'trash-2',
            title: l10n.settingsCloseAccount,
            destructive: true,
            onTap: _accountClosing ? null : _closeAccount,
          ),
        ),
      ],
    );
  }

  Widget _buildPrivacyTab(AppLocalizations l10n, ScrollController controller) =>
      ListView(
        controller: controller,
        padding: const EdgeInsets.all(16),
        children: [
          _settingsSection(
            context,
            child: Column(
              children: [
                GfSettingRow(
                  key: const ValueKey('settings-account-profile-privacy'),
                  symbol: 'eye',
                  title: l10n.profilePrivacyTitle,
                  description: l10n.profilePrivacyDescription,
                  onTap: () => context.push('/settings/profile-privacy'),
                ),
                const GfDivider(),
                GfSettingRow(
                  key: const ValueKey('settings-chat-permissions'),
                  symbol: 'message-circle',
                  title: l10n.chatMessagePolicyTitle,
                  description: l10n.chatMessagePolicyHint,
                  onTap: () => context.push('/settings/chat-permissions'),
                ),
                const GfDivider(),
                GfSettingRow(
                  symbol: 'eye-off',
                  title: l10n.userBlocks,
                  onTap: () => showBlockedUsers(context),
                ),
                const GfDivider(),
                GfSettingRow(
                  symbol: 'circle-user-round',
                  title: l10n.anonymousIdentity,
                  description: l10n.anonymousSettingsHint,
                  onTap: () => context.push('/settings/anonymous-identity'),
                ),
              ],
            ),
          ),
        ],
      );

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
                const PushPreferencesPanel(),
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
  /// 登出:确认 → 服务端失效 → 清 token → 清离线缓存 → 跳登录页。
  ///
  /// 离线缓存(drift)保存私信与已浏览话题,登出必须清空,否则同一设备
  /// 换账号后仍可读到上一账号的缓存数据(跨账号数据泄漏)。
  Future<void> _closeAccount() async {
    final choice = await showDialog<({String mode, String password})>(
      context: context,
      animationStyle: GfMotion.dialogStyle(context),
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
        title: GfText(l10n.settingsLogoutConfirm),
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
    // Fence requests before waiting for secure storage or disk cleanup.
    ref.read(offlineCacheEpochProvider.notifier).invalidate();
    await ref.read(tokenStorageProvider).clear();
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

/// Settings share the same grouped surface as profile and account forms.
Widget _settingsSection(
  BuildContext context, {
  String? title,
  required Widget child,
}) => GfListSection(header: title, margin: EdgeInsets.zero, children: [child]);

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
        title: GfText(widget.title),
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
                  // 仅密码字段保留 enableSuggestions: false(原因见 login_page.dart
                  // 的同款注释);邮箱、TOTP 码等非密码字段保持默认,避免 ColorOS
                  // 将其误判为凭据输入而强制安全键盘(issue #1111)。
                  enableSuggestions: !widget.fields[index].obscureText,
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
