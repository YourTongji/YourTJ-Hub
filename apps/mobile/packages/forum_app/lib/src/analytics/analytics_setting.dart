import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import 'analytics_consent.dart';

class AnalyticsSetting extends ConsumerStatefulWidget {
  const AnalyticsSetting({super.key});

  @override
  ConsumerState<AnalyticsSetting> createState() => _AnalyticsSettingState();
}

class _AnalyticsSettingState extends ConsumerState<AnalyticsSetting> {
  bool _saving = false;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return GfSwitchRow(
      symbol: 'monitor',
      title: l10n.settingsVisitorAnalytics,
      value: ref.watch(analyticsConsentProvider),
      onChanged: (enabled) async {
        if (_saving) return;
        _saving = true;
        if (enabled) {
          final accepted = await showDialog<bool>(
            context: context,
            builder: (dialogContext) => AlertDialog(
              title: Text(l10n.settingsVisitorAnalytics),
              content: SingleChildScrollView(
                child: Text(l10n.settingsVisitorAnalyticsDescription),
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.pop(dialogContext, false),
                  child: Text(l10n.commonCancel),
                ),
                TextButton(
                  onPressed: () => Navigator.pop(dialogContext, true),
                  child: Text(l10n.commonConfirm),
                ),
              ],
            ),
          );
          if (!context.mounted || accepted != true) {
            _saving = false;
            return;
          }
        }
        final saved = await ref
            .read(analyticsConsentProvider.notifier)
            .setEnabled(enabled);
        _saving = false;
        if (!context.mounted || saved) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.settingsVisitorAnalyticsSaveFailed)),
        );
      },
    );
  }
}
