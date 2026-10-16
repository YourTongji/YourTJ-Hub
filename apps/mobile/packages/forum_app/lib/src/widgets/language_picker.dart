import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../app_locale.dart';

Future<void> showAppLanguagePicker(BuildContext context) async {
  if (Theme.of(context).platform == TargetPlatform.iOS) {
    final container = ProviderScope.containerOf(context, listen: false);
    final l10n = AppLocalizations.of(context);
    final selected =
        container.read(appLocaleProvider)?.languageCode ?? 'system';
    final choice = await showGfActionSheet<String>(
      context,
      title: GfText(l10n.settingsAppLanguage),
      initialValue: selected,
      items: [
        for (final entry in {
          'system': l10n.settingsLanguageSystem,
          ...appLanguageNames,
        }.entries)
          GfPopupMenuItem(value: entry.key, child: GfText(entry.value)),
      ],
    );
    if (context.mounted && choice != null) {
      container
          .read(appLocaleProvider.notifier)
          .setLocale(normalizeAppLocale(choice));
    }
    return;
  }
  await showGfBottomSheet<void>(
    context,
    builder: (_) => Consumer(
      builder: (context, ref, _) {
        final l10n = AppLocalizations.of(context);
        final selected = ref.watch(appLocaleProvider)?.languageCode ?? 'system';
        return SingleChildScrollView(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 20, 16, 12),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                GfText(
                  l10n.settingsAppLanguage,
                  style: GfTheme.typographyOf(context).title2,
                ),
                const SizedBox(height: 12),
                for (final entry in {
                  'system': l10n.settingsLanguageSystem,
                  ...appLanguageNames,
                }.entries)
                  ListTile(
                    title: GfText(entry.value),
                    selected: selected == entry.key,
                    trailing: selected == entry.key
                        ? const GfSymbol('check')
                        : null,
                    onTap: () {
                      ref
                          .read(appLocaleProvider.notifier)
                          .setLocale(normalizeAppLocale(entry.key));
                      Navigator.pop(context);
                    },
                  ),
              ],
            ),
          ),
        );
      },
    ),
  );
}
