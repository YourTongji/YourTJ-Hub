import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../app_locale.dart';

Future<void> showAppLanguagePicker(BuildContext context) async {
  final container = ProviderScope.containerOf(context, listen: false);
  final l10n = AppLocalizations.of(context);
  final selected = container.read(appLocaleProvider)?.languageCode ?? 'system';
  final value = await showGfActionMenu<String>(
    context,
    semanticLabel: l10n.settingsAppLanguage,
    actions: [
      for (final entry in {
        'system': l10n.settingsLanguageSystem,
        ...appLanguageNames,
      }.entries)
        GfContextAction(
          value: entry.key,
          label: entry.value,
          selected: selected == entry.key,
        ),
    ],
  );
  if (context.mounted && value != null) {
    container
        .read(appLocaleProvider.notifier)
        .setLocale(normalizeAppLocale(value));
  }
}
