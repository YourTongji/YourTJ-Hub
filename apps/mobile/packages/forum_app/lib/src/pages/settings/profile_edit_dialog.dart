import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../profile_links.dart';
import '../../app_locale.dart';
import '../../server_messages.dart';
import '../../widgets/confirm_discard_edit.dart';

/// Edit the profile in its reading layout. Image selection is local; the explicit
/// Save action commits independent API operations and remembers each success.
class ProfileEditPage extends StatefulWidget {
  const ProfileEditPage({
    super.key,
    required this.user,
    required this.onSave,
    this.onPickImage,
    this.onSaveAvatar,
    this.onSaveCover,
  });
  final SettingsUserPayload user;
  final Future<void> Function(SettingsUserPayload) onSave;
  final Future<Uint8List?> Function(bool cover)? onPickImage;
  final Future<String> Function(Uint8List bytes)? onSaveAvatar;
  final Future<String> Function(Uint8List? bytes)? onSaveCover;
  @override
  State<ProfileEditPage> createState() => _ProfileEditPageState();
}

class _ProfileEditPageState extends State<ProfileEditPage> {
  final _form = GlobalKey<FormState>();
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
  late String _avatarUrl = widget.user.avatarUrl;
  late String _coverUrl = widget.user.profileCoverUrl;
  Uint8List? _avatarBytes, _coverBytes;
  bool _coverChanged = false;
  bool _saving = false, _picking = false, _allowPop = false, _closing = false;
  final Set<String> _savedParts = {};
  String? _error;
  bool get _fieldsDirty =>
      _locale != _savedLocale ||
      _fields.entries.any((e) => e.value.text != _savedFields[e.key]);
  bool get _dirty => _fieldsDirty || _avatarBytes != null || _coverChanged;
  bool get _busy => _saving || _picking;

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
      if (mounted) Navigator.pop(context, saved);
    });
  }

  Future<void> _pickImage(bool cover) async {
    if (_busy || widget.onPickImage == null) return;
    setState(() => _picking = true);
    try {
      final bytes = await widget.onPickImage!(cover);
      if (!mounted || bytes == null) return;
      setState(() {
        if (cover) {
          _coverBytes = bytes;
          _coverChanged = true;
        } else {
          _avatarBytes = bytes;
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

  Future<void> _save() async {
    if (_busy || !_dirty || !_form.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    final l = AppLocalizations.of(context);
    final updated = widget.user.copyWith(
      nickname: _fields['nickname']!.text.trim(),
      bio: _fields['bio']!.text.trim(),
      signature: _fields['signature']!.text.trim(),
      websiteName: _fields['websiteName']!.text.trim(),
      website: normalizeProfileLink(_fields['website']!.text)!,
      locale: _locale,
      externalInformation: {
        ...widget.user.externalInformation,
        for (final e in profileSocialProviders.entries)
          e.key: ExternalLinkPayload(
            link: normalizeProfileLink(
              _fields[e.key]!.text,
              prefix: e.value.$2,
            ),
          ),
      },
    );
    try {
      // These endpoints cannot be one server transaction. A retry resumes at
      // the failed step and never resubmits an acknowledged image upload.
      if (_fieldsDirty) {
        await widget.onSave(updated);
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

  Widget _field(
    String key,
    String label, {
    int lines = 1,
    bool url = false,
    String? prefix,
    bool headline = false,
  }) {
    final colors = GfTheme.colorsOf(context);
    final l = AppLocalizations.of(context);
    return Padding(
      padding: EdgeInsets.only(bottom: headline ? 8 : 20),
      child: TextFormField(
        key: ValueKey('profile-$key'),
        controller: _fields[key],
        enabled: !_busy,
        minLines: 1,
        maxLines: lines,
        style: GfTheme.typographyOf(context).body.copyWith(
          fontSize: headline ? 22 : 16,
          height: headline ? 1.2 : 1.5,
          fontWeight: headline ? FontWeight.w700 : FontWeight.w400,
        ),
        keyboardType: url
            ? TextInputType.url
            : (lines > 1 ? TextInputType.multiline : TextInputType.text),
        textInputAction: lines > 1
            ? TextInputAction.newline
            : TextInputAction.next,
        autocorrect: !url,
        enableSuggestions: !url,
        decoration: InputDecoration(
          labelText: label,
          floatingLabelBehavior: FloatingLabelBehavior.always,
          labelStyle: TextStyle(fontSize: 16, color: colors.iconMuted),
          floatingLabelStyle: TextStyle(fontSize: 13, color: colors.iconMuted),
          filled: false,
          isDense: true,
          contentPadding: const EdgeInsets.symmetric(vertical: 10),
          border: UnderlineInputBorder(
            borderSide: BorderSide(color: colors.line),
          ),
          enabledBorder: UnderlineInputBorder(
            borderSide: BorderSide(color: colors.line),
          ),
          disabledBorder: UnderlineInputBorder(
            borderSide: BorderSide(color: colors.line),
          ),
          focusedBorder: UnderlineInputBorder(
            borderSide: BorderSide(color: colors.primary),
          ),
          errorBorder: UnderlineInputBorder(
            borderSide: BorderSide(color: colors.error),
          ),
          focusedErrorBorder: UnderlineInputBorder(
            borderSide: BorderSide(color: colors.error),
          ),
          prefixIcon: profileSocialProviders.containsKey(key)
              ? Padding(
                  padding: const EdgeInsets.all(12),
                  child: GfSocialIcon(key),
                )
              : null,
        ),
        validator: url
            ? (value) =>
                  normalizeProfileLink(value ?? '', prefix: prefix) == null
                  ? l.settingsInvalidLink
                  : null
            : null,
      ),
    );
  }

  Widget _photoButton({
    required Key key,
    required String label,
    required VoidCallback? onPressed,
    String symbol = 'camera',
  }) {
    return GfGlassIconButton(
      key: key,
      symbol: symbol,
      tooltip: label,
      onPressed: onPressed,
    );
  }

  Widget _profileImages(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final l = AppLocalizations.of(context);
    // Capture the page inset before Scaffold/SafeArea consume it for the body.
    final topInset = MediaQuery.viewPaddingOf(context).top;
    final coverVisible =
        _coverBytes != null || (!_coverChanged && _coverUrl.isNotEmpty);
    return LayoutBuilder(
      builder: (context, constraints) {
        final height = GfUserCard.coverHeightFor(
          constraints.maxWidth,
          topInset: topInset,
        );
        final cover = _coverBytes != null
            ? Image.memory(
                _coverBytes!,
                key: const Key('profile-local-cover'),
                fit: BoxFit.cover,
                width: double.infinity,
                height: height,
                gaplessPlayback: true,
              )
            : !_coverChanged && _coverUrl.isNotEmpty
            ? Image.network(
                resolveApiAssetUrl(_coverUrl),
                fit: BoxFit.cover,
                width: double.infinity,
                height: height,
                errorBuilder: (_, _, _) => ColoredBox(color: colors.base300),
              )
            : ColoredBox(color: colors.base300);
        return SizedBox(
          height: height + GfUserCardHeader.minimumActionHeight,
          child: Stack(
            children: [
              Positioned(
                top: 0,
                left: 0,
                right: 0,
                height: height,
                child: SizedBox.expand(
                  key: const Key('profile-cover-preview'),
                  child: cover,
                ),
              ),
              Positioned(
                top: height / 2 - 22,
                left: constraints.maxWidth / 2 - 22,
                child: _photoButton(
                  key: const Key('profile-edit-cover'),
                  label: l.settingsCover,
                  onPressed:
                      _busy ||
                          widget.onPickImage == null ||
                          widget.onSaveCover == null
                      ? null
                      : () => _pickImage(true),
                ),
              ),
              if (coverVisible && widget.onSaveCover != null)
                Positioned(
                  top: 12,
                  right: 12,
                  child: _photoButton(
                    key: const Key('profile-remove-cover'),
                    label: l.settingsCoverRemove,
                    symbol: 'x',
                    onPressed: _busy
                        ? null
                        : () => setState(() {
                            _coverBytes = null;
                            _coverChanged = _coverUrl.isNotEmpty;
                          }),
                  ),
                ),
              Positioned(
                top: height - 48,
                left: 16,
                child: Container(
                  padding: const EdgeInsets.all(4),
                  decoration: BoxDecoration(
                    color: colors.base100,
                    shape: BoxShape.circle,
                  ),
                  child: SizedBox.square(
                    dimension: 88,
                    child: Stack(
                      fit: StackFit.expand,
                      children: [
                        ClipOval(
                          child: _avatarBytes != null
                              ? Image.memory(
                                  _avatarBytes!,
                                  key: const Key('profile-local-avatar'),
                                  fit: BoxFit.cover,
                                  gaplessPlayback: true,
                                )
                              : GfAvatar(
                                  src: resolveApiAssetUrl(_avatarUrl),
                                  size: 88,
                                ),
                        ),
                        Center(
                          child: _photoButton(
                            key: const Key('profile-edit-avatar'),
                            label: l.settingsAvatar,
                            onPressed:
                                _busy ||
                                    widget.onPickImage == null ||
                                    widget.onSaveAvatar == null
                                ? null
                                : () => _pickImage(false),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    return PopScope(
      canPop: _allowPop || (!_busy && !_dirty),
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: Scaffold(
        appBar: AppBar(
          automaticallyImplyLeading: false,
          titleSpacing: 8,
          toolbarHeight: 56,
          title: Row(
            children: [
              Tooltip(
                message: l.commonCancel,
                child: TextButton(
                  onPressed: _busy ? null : _close,
                  child: Text(l.commonCancel),
                ),
              ),
              Expanded(
                child: Text(
                  l.settingsEditProfile,
                  textAlign: TextAlign.center,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 17,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              const SizedBox(width: 8),
              GfButton(
                key: const Key('profile-save'),
                label: l.commonSave,
                size: GfButtonSize.small,
                loading: _saving,
                onPressed: !_dirty || _busy ? null : _save,
              ),
            ],
          ),
        ),
        body: SafeArea(
          top: false,
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 760),
              child: Form(
                key: _form,
                child: ListView(
                  keyboardDismissBehavior:
                      ScrollViewKeyboardDismissBehavior.onDrag,
                  padding: const EdgeInsets.only(bottom: 24),
                  children: [
                    _profileImages(context),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          if (_picking)
                            const Padding(
                              padding: EdgeInsets.only(bottom: 16),
                              child: LinearProgressIndicator(),
                            ),
                          if (_error != null)
                            Padding(
                              padding: const EdgeInsets.only(bottom: 20),
                              child: Semantics(
                                liveRegion: true,
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    if (_savedParts.isNotEmpty)
                                      Padding(
                                        padding: const EdgeInsets.only(
                                          bottom: 8,
                                        ),
                                        child: Text(
                                          l.settingsProfilePartialSave(
                                            _savedParts.join(' · '),
                                          ),
                                          key: const Key(
                                            'profile-partial-save',
                                          ),
                                          style: TextStyle(
                                            color: colors.baseContent,
                                          ),
                                        ),
                                      ),
                                    Text(
                                      _error!,
                                      key: const Key('profile-save-error'),
                                      style: TextStyle(color: colors.error),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          _field(
                            'nickname',
                            l.settingsNickname,
                            lines: 2,
                            headline: true,
                          ),
                          Padding(
                            padding: const EdgeInsets.only(bottom: 24),
                            child: Text(
                              '@${widget.user.username}',
                              style: TextStyle(
                                fontSize: 15,
                                color: colors.iconMuted,
                              ),
                            ),
                          ),
                          _field('bio', l.settingsBio, lines: 5),
                          _field('signature', l.settingsSignature, lines: 3),
                          _field('websiteName', l.settingsWebsiteName),
                          _field('website', l.settingsWebsite, url: true),
                          ExpansionTile(
                            maintainState: true,
                            tilePadding: EdgeInsets.zero,
                            title: Text(l.settingsSocialLinks),
                            children: [
                              for (final e in profileSocialProviders.entries)
                                _field(
                                  e.key,
                                  e.value.$1,
                                  url: true,
                                  prefix: e.value.$2,
                                ),
                            ],
                          ),
                          const SizedBox(height: 20),
                          DropdownButtonFormField<String>(
                            initialValue: _locale,
                            isExpanded: true,
                            itemHeight: null,
                            borderRadius: BorderRadius.circular(16),
                            icon: const GfSymbol('chevron-down', size: 20),
                            decoration: InputDecoration(
                              labelText: l.settingsProfileLanguage,
                            ),
                            items: [
                              for (final e in appLanguageNames.entries)
                                DropdownMenuItem(
                                  value: e.key,
                                  child: Text(e.value),
                                ),
                            ],
                            onChanged: _busy
                                ? null
                                : (value) {
                                    if (value != null) {
                                      setState(() => _locale = value);
                                    }
                                  },
                          ),
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
