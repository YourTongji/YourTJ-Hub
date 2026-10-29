import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../app_locale.dart';
import '../../asset_url.dart';
import '../../profile_links.dart';
import '../../server_messages.dart';
import '../../widgets/confirm_discard_edit.dart';
import '../../widgets/user_decoration.dart';

/// Where a newly picked avatar or cover comes from.
enum ProfileImageSource { gallery, camera }

/// Text capacities mirror the forum `users` columns (nickname/bio/signature/website_name).
const int profileNicknameMaxLength = 64;
const int profileBioMaxLength = 500;
const int profileSignatureMaxLength = 255;
const int profileWebsiteNameMaxLength = 64;

/// The twelve built-in avatars served by the site.
const int profilePresetAvatarCount = 12;
String profilePresetAvatarPath(int index) => '/static/pic/${index + 1}.webp';

/// The single profile editor: avatar first, then cover, basic information,
/// links and display options. Every avatar, cover and text change stays a
/// local draft until Save, which commits the independent API operations in
/// order and remembers each acknowledged step for a retry.
class ProfileEditPage extends StatefulWidget {
  const ProfileEditPage({
    super.key,
    required this.user,
    required this.onSave,
    this.onPickImage,
    this.onSaveAvatar,
    this.onSavePresetAvatar,
    this.onSaveCover,
    this.onEditUsername,
    this.onOpenDecorations,
    this.onOpenBadges,
    this.onReloadUser,
    this.onPushLinks,
    this.onClose,
  });

  final SettingsUserPayload user;
  final Future<void> Function(SettingsUserPayload) onSave;
  final Future<Uint8List?> Function(bool cover, ProfileImageSource source)?
  onPickImage;
  final Future<String> Function(Uint8List bytes)? onSaveAvatar;
  final Future<void> Function(String path)? onSavePresetAvatar;
  final Future<String> Function(Uint8List? bytes)? onSaveCover;

  /// Username, decorations and badges have their own editors and saves.
  final Future<void> Function()? onEditUsername;
  final Future<void> Function()? onOpenDecorations;
  final Future<void> Function()? onOpenBadges;
  final Future<SettingsUserPayload?> Function()? onReloadUser;
  final Future<Map<String, String>?> Function(WidgetBuilder)? onPushLinks;

  final ValueChanged<bool>? onClose;

  @override
  State<ProfileEditPage> createState() => _ProfileEditPageState();
}

class _ProfileEditPageState extends State<ProfileEditPage> {
  late final Map<String, String> _savedFields = {
    'nickname': widget.user.nickname,
    'bio': widget.user.bio,
    'signature': widget.user.signature,
    'websiteName': widget.user.websiteName,
    'website': widget.user.website,
    for (final key in profileSocialProviders.keys)
      key: widget.user.externalInformation[key]?.link ?? '',
  };
  late final _fields = {
    for (final e in _savedFields.entries)
      e.key: TextEditingController(text: e.value),
  };
  late String _savedLocale =
      normalizeAppLocale(widget.user.locale)?.languageCode ?? 'zh';
  late String _locale = _savedLocale;
  late SettingsUserPayload _displayUser = widget.user;
  late String _avatarUrl = widget.user.avatarUrl;
  late String _coverUrl = widget.user.profileCoverUrl;
  Uint8List? _avatarBytes, _coverBytes;
  String? _presetAvatar;
  bool _coverChanged = false;
  bool _saving = false, _picking = false, _allowPop = false, _closing = false;
  final Set<String> _savedParts = {};
  String? _error;

  bool get _fieldsDirty =>
      _locale != _savedLocale ||
      _fields.entries.any((e) => e.value.text != _savedFields[e.key]);
  bool get _dirty =>
      _fieldsDirty ||
      _avatarBytes != null ||
      _presetAvatar != null ||
      _coverChanged;
  bool get _busy => _saving || _picking;
  bool get _coverVisible =>
      _coverBytes != null || (!_coverChanged && _coverUrl.isNotEmpty);
  bool get _canUploadAvatar =>
      widget.onPickImage != null && widget.onSaveAvatar != null;
  bool get _canUploadCover =>
      widget.onPickImage != null && widget.onSaveCover != null;
  bool get _cameraAvailable =>
      !kIsWeb &&
      (defaultTargetPlatform == TargetPlatform.iOS ||
          defaultTargetPlatform == TargetPlatform.android);

  @override
  void initState() {
    super.initState();
    for (final controller in _fields.values) {
      controller.addListener(_edited);
    }
  }

  void _edited() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    for (final c in _fields.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _close() async {
    if (_busy || _closing) return;
    _closing = true;
    final leave = !_dirty || await confirmDiscardEdit(context);
    _closing = false;
    if (mounted && leave) _pop(false);
  }

  void _pop(bool saved) {
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        if (widget.onClose != null) {
          widget.onClose!(saved);
        } else {
          Navigator.pop(context, saved);
        }
      }
    });
  }

  PopupMenuEntry<String> _sheetItem(
    String value,
    String symbol,
    String label, {
    bool destructive = false,
  }) {
    final colors = GfTheme.colorsOf(context);
    return GfPopupMenuItem(
      value: value,
      destructive: destructive,
      child: Row(
        key: ValueKey('profile-sheet-$value'),
        children: [
          GfSymbol(
            symbol,
            size: 22,
            color: destructive ? colors.error : colors.baseContent,
          ),
          const SizedBox(width: 12),
          Expanded(child: GfText(label)),
        ],
      ),
    );
  }

  Widget _sheetTitle(String title) => GfText(
    title,
    style: GfTheme.typographyOf(
      context,
    ).heading.copyWith(fontWeight: FontWeight.w700),
  );

  Future<void> _chooseAvatar() async {
    if (_busy) return;
    final l = AppLocalizations.of(context);
    final choice = await showGfActionSheet<String>(
      context,
      title: _sheetTitle(l.settingsAvatar),
      items: [
        if (_canUploadAvatar) ...[
          _sheetItem('avatar-gallery', 'image', l.profileEditFromPhotos),
          if (_cameraAvailable)
            _sheetItem('avatar-camera', 'camera', l.profileEditTakePhoto),
        ],
        if (widget.onSavePresetAvatar != null)
          _sheetItem('avatar-preset', 'user-round', l.settingsPresetAvatar),
      ],
    );
    if (!mounted || choice == null) return;
    if (choice == 'avatar-preset') {
      final path = await _pickPresetAvatar();
      if (!mounted || path == null) return;
      setState(() {
        // Re-picking the current avatar is not a change.
        _presetAvatar = path == _avatarUrl ? null : path;
        _avatarBytes = null;
        _error = null;
      });
      return;
    }
    await _pickImage(
      false,
      choice == 'avatar-camera'
          ? ProfileImageSource.camera
          : ProfileImageSource.gallery,
    );
  }

  Future<void> _chooseCover() async {
    if (_busy) return;
    final l = AppLocalizations.of(context);
    final choice = await showGfActionSheet<String>(
      context,
      title: _sheetTitle(l.settingsCover),
      items: [
        if (_canUploadCover) ...[
          _sheetItem('cover-gallery', 'image', l.profileEditFromPhotos),
          if (_cameraAvailable)
            _sheetItem('cover-camera', 'camera', l.profileEditTakePhoto),
        ],
        if (_coverVisible && widget.onSaveCover != null)
          _sheetItem(
            'cover-remove',
            'image-off',
            l.settingsCoverRemove,
            destructive: true,
          ),
      ],
    );
    if (!mounted || choice == null) return;
    if (choice == 'cover-remove') {
      setState(() {
        _coverBytes = null;
        _coverChanged = _coverUrl.isNotEmpty;
      });
      return;
    }
    await _pickImage(
      true,
      choice == 'cover-camera'
          ? ProfileImageSource.camera
          : ProfileImageSource.gallery,
    );
  }

  Future<void> _pickImage(bool cover, ProfileImageSource source) async {
    if (_busy || widget.onPickImage == null) return;
    setState(() => _picking = true);
    try {
      final bytes = await widget.onPickImage!(cover, source);
      if (!mounted || bytes == null) return;
      setState(() {
        if (cover) {
          _coverBytes = bytes;
          _coverChanged = true;
        } else {
          _avatarBytes = bytes;
          _presetAvatar = null;
        }
        _error = null;
      });
    } catch (error) {
      if (mounted) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _picking = false);
    }
  }

  Future<String?> _pickPresetAvatar() {
    final l = AppLocalizations.of(context);
    var selected = _presetAvatar ?? _avatarUrl;
    return showGfBottomSheet<String>(
      context,
      builder: (sheetContext) => StatefulBuilder(
        builder: (context, setSheetState) => SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _sheetTitle(l.settingsPresetAvatar),
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
                    itemCount: profilePresetAvatarCount,
                    itemBuilder: (context, index) {
                      final path = profilePresetAvatarPath(index);
                      return Semantics(
                        label: '${l.settingsPresetAvatar} ${index + 1}',
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
                  key: const Key('profile-preset-confirm'),
                  label: l.commonConfirm,
                  expanded: true,
                  onPressed: selected.startsWith('/static/pic/')
                      ? () => Navigator.pop(sheetContext, selected)
                      : null,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _editLinks() async {
    if (_busy) return;
    Widget builder(BuildContext _) => _ProfileLinksPage(
      values: {
        for (final key in [
          'websiteName',
          'website',
          ...profileSocialProviders.keys,
        ])
          key: _fields[key]!.text,
      },
    );
    final values =
        await (widget.onPushLinks?.call(builder) ??
            Navigator.of(
              context,
            ).push<Map<String, String>>(MaterialPageRoute(builder: builder)));
    if (!mounted || values == null) return;
    for (final entry in values.entries) {
      _fields[entry.key]!.text = entry.value;
    }
  }

  Future<void> _openAccountEditor(Future<void> Function() open) async {
    if (_busy) return;
    await open();
    if (!mounted) return;
    SettingsUserPayload? updated;
    try {
      updated = await widget.onReloadUser?.call();
    } on UnauthorizedException {
      // 账号已切换（会话 epoch 失效），视为编辑器结果不再属于当前用户。
      return;
    }
    final reloaded = updated;
    if (mounted && reloaded != null) {
      setState(() => _displayUser = reloaded);
    }
  }

  Future<void> _chooseLocale() async {
    if (_busy) return;
    final l = AppLocalizations.of(context);
    final choice = await showGfActionSheet<String>(
      context,
      title: _sheetTitle(l.settingsProfileLanguage),
      initialValue: _locale,
      items: [
        for (final e in appLanguageNames.entries)
          GfPopupMenuItem(value: e.key, child: GfText(e.value)),
      ],
    );
    if (mounted && choice != null) setState(() => _locale = choice);
  }

  Future<void> _save() async {
    if (_busy || !_dirty) return;
    final l = AppLocalizations.of(context);
    final website = normalizeProfileLink(_fields['website']!.text);
    final social = {
      for (final e in profileSocialProviders.entries)
        e.key: normalizeProfileLink(_fields[e.key]!.text, prefix: e.value.$2),
    };
    if (_fieldsDirty && (website == null || social.values.contains(null))) {
      setState(() => _error = l.settingsInvalidLink);
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      // These endpoints cannot be one server transaction. A retry resumes at
      // the failed step and never resubmits an acknowledged image upload.
      if (_fieldsDirty) {
        await widget.onSave(
          widget.user.copyWith(
            nickname: _fields['nickname']!.text.trim(),
            bio: _fields['bio']!.text.trim(),
            signature: _fields['signature']!.text.trim(),
            websiteName: _fields['websiteName']!.text.trim(),
            website: website!,
            locale: _locale,
            externalInformation: {
              ...widget.user.externalInformation,
              for (final e in social.entries)
                e.key: ExternalLinkPayload(link: e.value),
            },
          ),
        );
        if (!mounted) return;
        _savedLocale = _locale;
        for (final e in _fields.entries) {
          _savedFields[e.key] = e.value.text;
        }
        _savedParts.add(l.settingsSectionProfile);
      }
      if (_coverChanged && widget.onSaveCover != null) {
        final url = await widget.onSaveCover!(_coverBytes);
        if (!mounted) return;
        _coverUrl = url;
        _coverBytes = null;
        _coverChanged = false;
        _savedParts.add(l.settingsCover);
      }
      if (_avatarBytes != null && widget.onSaveAvatar != null) {
        final url = await widget.onSaveAvatar!(_avatarBytes!);
        if (!mounted) return;
        _avatarUrl = url;
        _avatarBytes = null;
        _savedParts.add(l.settingsAvatar);
      } else if (_presetAvatar != null && widget.onSavePresetAvatar != null) {
        await widget.onSavePresetAvatar!(_presetAvatar!);
        if (!mounted) return;
        _avatarUrl = _presetAvatar!;
        _presetAvatar = null;
        _savedParts.add(l.settingsAvatar);
      }
      if (mounted) _pop(true);
    } catch (error) {
      if (mounted) {
        setState(() => _error = resolveErrorMessage(l, error));
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Widget _avatarHeader(AppLocalizations l) {
    const double size = 96;
    final colors = GfTheme.colorsOf(context);
    final frame = _displayUser.decoration?.avatarFrame;
    final frameColors = decorationColors(context, frame);
    final frameAssetUrl = decorationFrameAssetUrl(frame);
    final frameScale = frame?.assetScale ?? 1.25;
    final Widget avatar = _avatarBytes != null
        ? GfAvatarFrame(
            size: size,
            colors: frameColors,
            assetUrl: frameAssetUrl,
            assetScale: frameScale,
            child: ClipOval(
              child: Image.memory(
                _avatarBytes!,
                key: const Key('profile-local-avatar'),
                width: size,
                height: size,
                fit: BoxFit.cover,
                gaplessPlayback: true,
              ),
            ),
          )
        : GfAvatar(
            key: _presetAvatar == null
                ? null
                : const Key('profile-preset-avatar'),
            src: resolveApiAssetUrl(_presetAvatar ?? _avatarUrl),
            size: size,
            frameColors: frameColors,
            frameAssetUrl: frameAssetUrl,
            frameScale: frameScale,
          );
    final enabled =
        !_busy && (_canUploadAvatar || widget.onSavePresetAvatar != null);
    return Column(
      children: [
        // The labelled text button below is the accessible control; the
        // picture repeats it as a larger touch target.
        ExcludeSemantics(
          child: InkWell(
            key: const Key('profile-edit-avatar'),
            customBorder: const CircleBorder(),
            onTap: enabled ? _chooseAvatar : null,
            child: Padding(
              // Frame artwork paints outside the avatar box.
              padding: const EdgeInsets.all(size * .14),
              child: Stack(
                clipBehavior: Clip.none,
                children: [
                  avatar,
                  Positioned(
                    right: -2,
                    bottom: -2,
                    child: Container(
                      width: 32,
                      height: 32,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: colors.primary,
                        border: Border.all(color: colors.base200, width: 2),
                      ),
                      child: GfSymbol(
                        'camera',
                        size: 16,
                        color: colors.primaryContent,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        GfButton(
          key: const Key('profile-change-avatar'),
          label: l.profileEditChangeAvatar,
          variant: GfButtonVariant.link,
          size: GfButtonSize.small,
          onPressed: enabled ? _chooseAvatar : null,
        ),
      ],
    );
  }

  Widget _coverCard(AppLocalizations l) {
    final colors = GfTheme.colorsOf(context);
    final enabled =
        !_busy &&
        (_canUploadCover || (_coverVisible && widget.onSaveCover != null));
    return LayoutBuilder(
      builder: (context, constraints) {
        final height = (constraints.maxWidth / 5).clamp(
          MediaQuery.textScalerOf(context).scale(16) * 1.4 + 24,
          double.infinity,
        );
        final Widget image = _coverBytes != null
            ? Image.memory(
                _coverBytes!,
                key: const Key('profile-local-cover'),
                fit: BoxFit.cover,
                gaplessPlayback: true,
              )
            : _coverVisible
            ? GfNetworkImage(
                resolveApiAssetUrl(_coverUrl),
                fit: BoxFit.cover,
                errorBuilder: (_, _, _) => ColoredBox(color: colors.base300),
              )
            : Center(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      GfSymbol('image', size: 24, color: colors.iconMuted),
                      const SizedBox(width: 8),
                      Flexible(
                        child: GfText(
                          l.profileEditAddCover,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(color: colors.iconMuted),
                        ),
                      ),
                    ],
                  ),
                ),
              );
        return Semantics(
          button: true,
          enabled: enabled,
          label: _coverVisible
              ? l.profileEditChangeCover
              : l.profileEditAddCover,
          child: Material(
            color: colors.base300,
            borderRadius: BorderRadius.circular(16),
            clipBehavior: Clip.antiAlias,
            child: InkWell(
              key: const Key('profile-edit-cover'),
              onTap: enabled ? _chooseCover : null,
              child: SizedBox(
                height: height,
                child: Stack(
                  fit: StackFit.expand,
                  children: [
                    ExcludeSemantics(
                      child: SizedBox.expand(
                        key: const Key('profile-cover-preview'),
                        child: image,
                      ),
                    ),
                    if (_coverVisible)
                      Positioned(
                        left: 12,
                        right: 12,
                        bottom: 8,
                        child: ExcludeSemantics(
                          child: Align(
                            alignment: AlignmentDirectional.centerEnd,
                            child: Container(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 12,
                                vertical: 6,
                              ),
                              decoration: BoxDecoration(
                                color: Colors.black.withValues(alpha: .55),
                                borderRadius: BorderRadius.circular(20),
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  const GfSymbol(
                                    'camera',
                                    size: 16,
                                    color: Colors.white,
                                  ),
                                  const SizedBox(width: 6),
                                  Flexible(
                                    child: GfText(
                                      l.profileEditChangeCover,
                                      maxLines: 1,
                                      overflow: TextOverflow.ellipsis,
                                      style: const TextStyle(
                                        color: Colors.white,
                                        fontSize: 13,
                                        fontWeight: FontWeight.w600,
                                      ),
                                    ),
                                  ),
                                ],
                              ),
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
      },
    );
  }

  Widget _inlineField(
    String key,
    String label, {
    required int maxLength,
    int lines = 1,
    String? hint,
  }) {
    return _LabeledRow(
      label: label,
      alignTop: lines > 1,
      child: _ProfileTextField(
        fieldKey: key,
        controller: _fields[key]!,
        enabled: !_busy,
        maxLength: maxLength,
        lines: lines,
        inlineCounter: lines == 1,
        hint: hint,
      ),
    );
  }

  Widget _errorBanner(AppLocalizations l) {
    final colors = GfTheme.colorsOf(context);
    return Semantics(
      liveRegion: true,
      child: Container(
        width: double.infinity,
        color: colors.error.withValues(alpha: .1),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (_savedParts.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: GfText(
                  l.settingsProfilePartialSave(_savedParts.join(' · ')),
                  key: const Key('profile-partial-save'),
                  style: TextStyle(color: colors.baseContent),
                ),
              ),
            GfText(
              _error!,
              key: const Key('profile-save-error'),
              style: TextStyle(color: colors.error),
            ),
          ],
        ),
      ),
    );
  }

  Widget _editorToolbar(AppLocalizations l, bool stacked) {
    final cancel = Tooltip(
      message: l.commonCancel,
      child: TextButton(
        onPressed: _busy ? null : _close,
        child: GfText(l.commonCancel),
      ),
    );
    final title = GfText(
      l.settingsEditProfile,
      textAlign: TextAlign.center,
      maxLines: 1,
      overflow: TextOverflow.ellipsis,
      style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
    );
    final save = Padding(
      padding: const EdgeInsets.only(right: 8),
      child: GfButton(
        key: const Key('profile-save'),
        label: l.commonSave,
        variant: GfButtonVariant.link,
        size: GfButtonSize.small,
        loading: _saving,
        onPressed: !_dirty || _busy ? null : _save,
      ),
    );
    if (stacked) {
      return Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              Expanded(
                child: Align(
                  alignment: AlignmentDirectional.centerStart,
                  child: cancel,
                ),
              ),
              Expanded(
                child: Align(
                  alignment: AlignmentDirectional.centerEnd,
                  child: save,
                ),
              ),
            ],
          ),
          const SizedBox(height: 4),
          title,
        ],
      );
    }
    return Row(
      children: [
        cancel,
        Expanded(child: title),
        save,
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final stackedToolbar =
        MediaQuery.sizeOf(context).width < 600 &&
        MediaQuery.textScalerOf(context).scale(17) > 24;
    final colors = GfTheme.colorsOf(context);
    final user = _displayUser;
    final filledSocial = [
      for (final key in profileSocialProviders.keys)
        if (_fields[key]!.text.trim().isNotEmpty) key,
    ];
    final websiteName = _fields['websiteName']!.text.trim();
    final websiteUrl = _fields['website']!.text.trim();
    final shownBadges =
        (user.displayBadges ?? user.badges.take(5).toList()).length;
    return PopScope(
      canPop: _allowPop || (!_busy && !_dirty),
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: Scaffold(
        backgroundColor: colors.canvasGrouped,
        appBar: AppBar(
          backgroundColor: colors.canvasGrouped,
          surfaceTintColor: Colors.transparent,
          elevation: 0,
          automaticallyImplyLeading: false,
          titleSpacing: 8,
          toolbarHeight:
              (MediaQuery.textScalerOf(context).scale(17) *
                          1.4 *
                          (stackedToolbar ? 2 : 1) +
                      24)
                  .clamp(56.0, double.infinity),
          title: _editorToolbar(l, stackedToolbar),
        ),
        body: SafeArea(
          top: false,
          child: Column(
            children: [
              if (_picking) const LinearProgressIndicator(),
              if (_error != null) _errorBanner(l),
              Expanded(
                child: Align(
                  alignment: Alignment.topCenter,
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 640),
                    child: ListView(
                      keyboardDismissBehavior:
                          ScrollViewKeyboardDismissBehavior.onDrag,
                      padding: const EdgeInsets.fromLTRB(16, 12, 16, 32),
                      children: [
                        _avatarHeader(l),
                        const SizedBox(height: 20),
                        _editorSection(
                          context,
                          title: l.settingsCover,
                          child: _coverCard(l),
                        ),
                        const SizedBox(height: 24),
                        _editorSection(
                          context,
                          title: l.profileEditBasics,
                          child: Column(
                            children: [
                              _inlineField(
                                'nickname',
                                l.settingsNickname,
                                maxLength: profileNicknameMaxLength,
                                hint: l.profileEditNicknameHint,
                              ),
                              const GfDivider(),
                              _LabeledRow(
                                key: const Key('profile-username'),
                                label: l.authUsername,
                                onTap: widget.onEditUsername == null || _busy
                                    ? null
                                    : () => _openAccountEditor(
                                        widget.onEditUsername!,
                                      ),
                                child: GfText(
                                  '@${user.username}',
                                  style: TextStyle(color: colors.iconMuted),
                                ),
                              ),
                              const GfDivider(),
                              _inlineField(
                                'bio',
                                l.settingsBio,
                                maxLength: profileBioMaxLength,
                                lines: 5,
                                hint: l.profileEditBioHint,
                              ),
                              const GfDivider(),
                              _inlineField(
                                'signature',
                                l.settingsSignature,
                                maxLength: profileSignatureMaxLength,
                                lines: 2,
                                hint: l.profileEditSignatureHint,
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 24),
                        _editorSection(
                          context,
                          title: l.profileEditLinks,
                          child: Column(
                            children: [
                              GfSettingRow(
                                key: const Key('profile-website-row'),
                                symbol: 'globe-pointer-filled',
                                title: l.settingsWebsite,
                                description: websiteName.isNotEmpty
                                    ? websiteName
                                    : websiteUrl.isNotEmpty
                                    ? websiteUrl
                                    : l.profileEditAddWebsite,
                                onTap: _busy ? null : _editLinks,
                              ),
                              const GfDivider(),
                              GfSettingRow(
                                key: const Key('profile-social-row'),
                                symbol: 'link',
                                title: l.settingsSocialLinks,
                                subtitleWidget: filledSocial.isEmpty
                                    ? GfText(l.profileEditAddSocialLinks)
                                    : Wrap(
                                        spacing: 8,
                                        runSpacing: 4,
                                        children: [
                                          for (final key in filledSocial)
                                            Tooltip(
                                              message:
                                                  profileSocialProviders[key]!
                                                      .$1,
                                              child: GfSocialIcon(
                                                key,
                                                size: 18,
                                              ),
                                            ),
                                        ],
                                      ),
                                onTap: _busy ? null : _editLinks,
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 24),
                        _editorSection(
                          context,
                          title: l.profileEditDisplay,
                          footer: l.profileLanguageHint,
                          child: Column(
                            children: [
                              if (widget.onOpenDecorations != null) ...[
                                GfSettingRow(
                                  key: const Key('profile-decorations'),
                                  symbol: 'sparkles',
                                  title: l.decorationTitle,
                                  onTap: _busy
                                      ? null
                                      : () => _openAccountEditor(
                                          widget.onOpenDecorations!,
                                        ),
                                ),
                                const GfDivider(),
                              ],
                              if (widget.onOpenBadges != null) ...[
                                GfSettingRow(
                                  key: const Key('profile-badges'),
                                  symbol: 'award',
                                  title: l.settingsBadge,
                                  description: l.profileEditBadgeSummary(
                                    user.wornBadge?.name ?? l.settingsBadgeNone,
                                    shownBadges,
                                  ),
                                  onTap: _busy
                                      ? null
                                      : () => _openAccountEditor(
                                          widget.onOpenBadges!,
                                        ),
                                ),
                                const GfDivider(),
                              ],
                              GfSettingRow(
                                key: const Key('profile-locale'),
                                symbol: 'languages',
                                title: l.settingsProfileLanguage,
                                description:
                                    appLanguageNames[_locale] ?? _locale,
                                onTap: _busy ? null : _chooseLocale,
                              ),
                            ],
                          ),
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
    );
  }
}

/// Website and social links, edited together on one page above the editor.
/// Done validates every destination; the editor keeps them as drafts.
class _ProfileLinksPage extends StatefulWidget {
  const _ProfileLinksPage({required this.values});
  final Map<String, String> values;

  @override
  State<_ProfileLinksPage> createState() => _ProfileLinksPageState();
}

class _ProfileLinksPageState extends State<_ProfileLinksPage> {
  final _form = GlobalKey<FormState>();
  late final _fields = {
    for (final e in widget.values.entries)
      e.key: TextEditingController(text: e.value),
  };
  bool _allowPop = false, _closing = false;

  bool get _dirty =>
      _fields.entries.any((e) => e.value.text != widget.values[e.key]);

  @override
  void dispose() {
    for (final c in _fields.values) {
      c.dispose();
    }
    super.dispose();
  }

  void _pop(Map<String, String>? values) {
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) Navigator.pop(context, values);
    });
  }

  Future<void> _close() async {
    if (_closing) return;
    _closing = true;
    final leave = !_dirty || await confirmDiscardEdit(context);
    _closing = false;
    if (mounted && leave) _pop(null);
  }

  void _done() {
    if (!_form.currentState!.validate()) return;
    _pop({for (final e in _fields.entries) e.key: e.value.text});
  }

  Widget _linkField(String key, String label, {String? prefix, String? hint}) {
    final l = AppLocalizations.of(context);
    return _LabeledRow(
      label: label,
      leading: profileSocialProviders.containsKey(key)
          ? GfSocialIcon(key, size: 18)
          : null,
      child: _ProfileTextField(
        fieldKey: key,
        controller: _fields[key]!,
        hint: hint,
        url: true,
        validator: (value) =>
            normalizeProfileLink(value ?? '', prefix: prefix) == null
            ? l.settingsInvalidLink
            : null,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    return PopScope(
      canPop: _allowPop || !_dirty,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: Scaffold(
        backgroundColor: colors.canvasGrouped,
        appBar: GfAppBar(
          backgroundColor: colors.canvasGrouped,
          showDivider: false,
          leading: IconButton(
            tooltip: l.commonBack,
            onPressed: _close,
            icon: const GfSymbol('chevron-left'),
          ),
          title: GfText(l.profileEditLinks),
          actions: [
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: GfButton(
                key: const Key('profile-links-done'),
                label: l.profileEditDone,
                variant: GfButtonVariant.link,
                size: GfButtonSize.small,
                onPressed: _done,
              ),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: Form(
                key: _form,
                child: ListView(
                  keyboardDismissBehavior:
                      ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
                  children: [
                    _editorSection(
                      context,
                      title: l.settingsWebsite,
                      child: Column(
                        children: [
                          _LabeledRow(
                            label: l.settingsWebsiteName,
                            child: _ProfileTextField(
                              fieldKey: 'websiteName',
                              controller: _fields['websiteName']!,
                              maxLength: profileWebsiteNameMaxLength,
                              hint: l.settingsWebsiteName,
                            ),
                          ),
                          const GfDivider(),
                          _linkField(
                            'website',
                            l.profileEditWebsiteUrl,
                            hint: 'https://',
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 24),
                    _editorSection(
                      context,
                      title: l.settingsSocialLinks,
                      child: Column(
                        children: [
                          for (final (i, e)
                              in profileSocialProviders.entries.indexed) ...[
                            if (i > 0) const GfDivider(),
                            _linkField(
                              e.key,
                              e.value.$1,
                              prefix: e.value.$2,
                              hint: l.profileEditSocialHint,
                            ),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Borderless input for a labelled row. Single-line counters sit beside the
/// input so their reserved space cannot push the editable text above its label.
class _ProfileTextField extends StatelessWidget {
  const _ProfileTextField({
    required this.fieldKey,
    required this.controller,
    this.enabled = true,
    this.maxLength,
    this.lines = 1,
    this.inlineCounter = false,
    this.hint,
    this.url = false,
    this.validator,
  });

  final String fieldKey;
  final TextEditingController controller;
  final bool enabled;
  final int? maxLength;
  final int lines;
  final bool inlineCounter;
  final String? hint;
  final bool url;
  final FormFieldValidator<String>? validator;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final field = TextFormField(
      key: ValueKey('profile-$fieldKey'),
      controller: controller,
      enabled: enabled,
      minLines: 1,
      maxLines: lines,
      textAlignVertical: inlineCounter ? TextAlignVertical.center : null,
      maxLength: maxLength,
      maxLengthEnforcement: MaxLengthEnforcement.enforced,
      buildCounter:
          (context, {required currentLength, required isFocused, maxLength}) =>
              maxLength != null && !inlineCounter
              ? GfText(
                  '$currentLength/$maxLength',
                  style: TextStyle(fontSize: 12, color: colors.textSecondary),
                )
              : null,
      style: GfTheme.typographyOf(
        context,
      ).body.copyWith(fontSize: 16, height: 1.4),
      keyboardType: url
          ? TextInputType.url
          : (lines > 1 ? TextInputType.multiline : TextInputType.text),
      textInputAction: lines > 1
          ? TextInputAction.newline
          : TextInputAction.next,
      // 资料页没有密码字段:enableSuggestions 必须保持默认 true,否则
      // Flutter Android 引擎会附加密码类 inputType,ColorOS 据此强制
      // 安全键盘并禁用粘贴(issue #1111)。
      autocorrect: !url,
      validator: validator,
      decoration: InputDecoration(
        hintText: hint,
        hintStyle: TextStyle(color: colors.iconMuted),
        filled: false,
        isDense: true,
        constraints: inlineCounter ? const BoxConstraints(minHeight: 44) : null,
        contentPadding: EdgeInsets.symmetric(vertical: inlineCounter ? 14 : 6),
        border: InputBorder.none,
        enabledBorder: InputBorder.none,
        disabledBorder: InputBorder.none,
        focusedBorder: InputBorder.none,
        errorBorder: InputBorder.none,
        focusedErrorBorder: InputBorder.none,
        errorMaxLines: 3,
      ),
    );
    if (!inlineCounter || maxLength == null) return field;
    return Row(
      children: [
        Expanded(child: field),
        const SizedBox(width: 12),
        ValueListenableBuilder<TextEditingValue>(
          valueListenable: controller,
          builder: (context, value, _) => GfText(
            '${value.text.characters.length}/$maxLength',
            key: ValueKey('profile-$fieldKey-counter'),
            style: TextStyle(fontSize: 12, color: colors.textSecondary),
          ),
        ),
      ],
    );
  }
}

/// Label and value side by side; narrow windows and large text stack them so
/// neither is squeezed.
class _LabeledRow extends StatelessWidget {
  const _LabeledRow({
    super.key,
    required this.label,
    required this.child,
    this.leading,
    this.alignTop = false,
    this.onTap,
  });

  final String label;
  final Widget child;
  final Widget? leading;
  final bool alignTop;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final scale = (MediaQuery.textScalerOf(context).scale(16) / 16).clamp(
      1.0,
      double.infinity,
    );
    final labelText = GfText(
      label,
      style: TextStyle(
        fontSize: 15,
        fontWeight: FontWeight.w600,
        color: colors.baseContent,
      ),
    );
    final labelWidget = leading == null
        ? labelText
        : Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              leading!,
              const SizedBox(width: 8),
              Flexible(child: labelText),
            ],
          );
    final row = LayoutBuilder(
      builder: (context, constraints) {
        final labelWidth = (leading == null ? 76.0 : 112.0) * scale;
        final stacked = constraints.maxWidth - labelWidth < 160;
        final value = onTap == null
            ? child
            : Row(
                children: [
                  Expanded(child: child),
                  const SizedBox(width: 8),
                  GfSymbol('chevron-right', size: 18, color: colors.iconMuted),
                ],
              );
        if (stacked) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [labelWidget, const SizedBox(height: 4), value],
          );
        }
        return Row(
          crossAxisAlignment: alignTop
              ? CrossAxisAlignment.start
              : CrossAxisAlignment.center,
          children: [
            Padding(
              padding: EdgeInsets.only(top: alignTop ? 6 : 0),
              child: SizedBox(width: labelWidth, child: labelWidget),
            ),
            const SizedBox(width: 12),
            Expanded(child: value),
          ],
        );
      },
    );
    final content = ConstrainedBox(
      constraints: const BoxConstraints(minHeight: 52),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Align(alignment: Alignment.centerLeft, child: row),
      ),
    );
    if (onTap == null) return content;
    return Semantics(
      button: true,
      child: InkWell(onTap: onTap, child: content),
    );
  }
}

/// The shared mobile group keeps header, surface and footer styling aligned.
Widget _editorSection(
  BuildContext context, {
  required String title,
  required Widget child,
  String? footer,
}) => GfListSection(
  header: title,
  footer: footer,
  margin: EdgeInsets.zero,
  children: [child],
);
