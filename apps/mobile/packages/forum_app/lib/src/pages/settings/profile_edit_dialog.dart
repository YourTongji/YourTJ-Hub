import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../profile_links.dart';
import '../../app_locale.dart';

/// Own the form controllers for the entire dialog route, including its exit
/// animation. The profile endpoint replaces every field, so preserve unknown
/// social providers as well as the visible fields.
enum ProfileEditSection { all, nickname, bio, links }

class ProfileEditDialog extends StatefulWidget {
  const ProfileEditDialog({
    super.key,
    required this.user,
    this.section = ProfileEditSection.all,
  });
  final SettingsUserPayload user;
  final ProfileEditSection section;

  @override
  State<ProfileEditDialog> createState() => _ProfileEditDialogState();
}

class _ProfileEditDialogState extends State<ProfileEditDialog> {
  final _form = GlobalKey<FormState>();
  late final Map<String, TextEditingController> _fields = {
    'nickname': TextEditingController(text: widget.user.nickname),
    'bio': TextEditingController(text: widget.user.bio),
    'signature': TextEditingController(text: widget.user.signature),
    'websiteName': TextEditingController(text: widget.user.websiteName),
    'website': TextEditingController(text: widget.user.website),
    for (final key in profileSocialProviders.keys)
      key: TextEditingController(
        text: widget.user.externalInformation[key]?.link ?? '',
      ),
  };
  late String _locale =
      normalizeAppLocale(widget.user.locale)?.languageCode ?? 'zh';

  @override
  void dispose() {
    for (final controller in _fields.values) {
      controller.dispose();
    }
    super.dispose();
  }

  void _save() {
    if (!_form.currentState!.validate()) return;
    if (widget.section == ProfileEditSection.nickname ||
        widget.section == ProfileEditSection.bio) {
      Navigator.pop(
        context,
        widget.section == ProfileEditSection.nickname
            ? widget.user.copyWith(nickname: _fields['nickname']!.text.trim())
            : widget.user.copyWith(bio: _fields['bio']!.text.trim()),
      );
      return;
    }
    Navigator.pop(
      context,
      widget.user.copyWith(
        nickname: widget.section == ProfileEditSection.links
            ? widget.user.nickname
            : _fields['nickname']!.text.trim(),
        bio: widget.section == ProfileEditSection.links
            ? widget.user.bio
            : _fields['bio']!.text.trim(),
        signature: widget.section == ProfileEditSection.links
            ? widget.user.signature
            : _fields['signature']!.text.trim(),
        websiteName: _fields['websiteName']!.text.trim(),
        website: normalizeProfileLink(_fields['website']!.text)!,
        locale: widget.section == ProfileEditSection.links
            ? widget.user.locale
            : _locale,
        externalInformation: {
          ...widget.user.externalInformation,
          for (final entry in profileSocialProviders.entries)
            entry.key: ExternalLinkPayload(
              link: normalizeProfileLink(
                _fields[entry.key]!.text,
                prefix: entry.value.$2,
              ),
            ),
        },
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    Widget field(
      String key,
      String label, {
      int lines = 1,
      bool url = false,
      String? prefix,
    }) => Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: TextFormField(
        key: ValueKey('profile-$key'),
        controller: _fields[key],
        maxLines: lines,
        keyboardType: url
            ? TextInputType.url
            : (lines > 1 ? TextInputType.multiline : TextInputType.text),
        autocorrect: !url,
        decoration: InputDecoration(
          labelText: label,
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
                  ? l10n.settingsInvalidLink
                  : null
            : null,
      ),
    );
    return AlertDialog(
      title: Text(switch (widget.section) {
        ProfileEditSection.nickname => l10n.settingsNickname,
        ProfileEditSection.bio => l10n.settingsBio,
        ProfileEditSection.links => l10n.settingsProfileLinks,
        ProfileEditSection.all => l10n.settingsEditProfile,
      }),
      content: SizedBox(
        width: 440,
        child: SingleChildScrollView(
          child: Form(
            key: _form,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (widget.section == ProfileEditSection.all ||
                    widget.section == ProfileEditSection.nickname)
                  field('nickname', l10n.settingsNickname),
                if (widget.section == ProfileEditSection.all ||
                    widget.section == ProfileEditSection.bio)
                  field('bio', l10n.settingsBio, lines: 3),
                if (widget.section == ProfileEditSection.all) ...[
                  field('signature', l10n.settingsSignature, lines: 2),
                  DropdownButtonFormField<String>(
                    initialValue: _locale,
                    decoration: InputDecoration(
                      labelText: l10n.settingsProfileLanguage,
                    ),
                    items: [
                      for (final entry in appLanguageNames.entries)
                        DropdownMenuItem(
                          value: entry.key,
                          child: Text(entry.value),
                        ),
                    ],
                    onChanged: (value) {
                      if (value != null) setState(() => _locale = value);
                    },
                  ),
                  const SizedBox(height: 16),
                ],
                if (widget.section == ProfileEditSection.all ||
                    widget.section == ProfileEditSection.links) ...[
                  field('websiteName', l10n.settingsWebsiteName),
                  field('website', l10n.settingsWebsite, url: true),
                  ExpansionTile(
                    tilePadding: EdgeInsets.zero,
                    title: Text(l10n.settingsSocialLinks),
                    children: [
                      for (final entry in profileSocialProviders.entries)
                        field(
                          entry.key,
                          entry.value.$1,
                          url: true,
                          prefix: entry.value.$2,
                        ),
                    ],
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.commonCancel),
        ),
        FilledButton(onPressed: _save, child: Text(l10n.commonSave)),
      ],
    );
  }
}
