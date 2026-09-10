import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../l10n/app_localizations.dart';

/// Public destination; no native credential is passed to the browser.
final fullSchedulerUri = Uri.https('f.yourtj.de', '/schedule');

class SchedulerWebTip extends StatelessWidget {
  const SchedulerWebTip({super.key, this.onOpen});
  final VoidCallback? onOpen;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final l10n = AppLocalizations.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colors.primary.withValues(alpha: .06),
        border: Border.all(color: colors.primary.withValues(alpha: .4)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.schedulerWebTitle,
            style: GfTheme.typographyOf(
              context,
            ).bodyStrong.copyWith(color: colors.primary),
          ),
          TextButton.icon(
            icon: GfSymbol('external-link', size: 18, color: colors.primary),
            label: Text(l10n.schedulerWebAction),
            onPressed:
                onOpen ??
                () async {
                  try {
                    if (await launchUrl(
                      fullSchedulerUri,
                      mode: LaunchMode.externalApplication,
                    )) {
                      return;
                    }
                  } catch (_) {
                    /* Report a failed browser handoff below. */
                  }
                  if (context.mounted) {
                    ScaffoldMessenger.of(
                      context,
                    ).showSnackBar(SnackBar(content: Text(l10n.commonRetry)));
                  }
                },
          ),
        ],
      ),
    );
  }
}
