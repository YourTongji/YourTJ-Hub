import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../schedule/schedule_sync.dart';
import 'schedule_sync_panel.dart';
import 'schedule_storage_panel.dart';
import 'scheduler_web_tip.dart';

/// Collapse help and recovery controls while keeping actionable sync problems
/// visible in the summary, including updates received after the page opens.
class ScheduleStatusSummary extends ConsumerWidget {
  const ScheduleStatusSummary({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final sync = ref.watch(scheduleSyncControllerProvider);
    final l = AppLocalizations.of(context);
    return ListenableBuilder(
      listenable: Listenable.merge([
        sync.conflicts,
        sync.needsAdoption,
        sync.blocked,
        sync.drafts,
      ]),
      builder: (_, _) => ExpansionTile(
        title: GfText(l.scheduleStatusHelp),
        subtitle: sync.blocked.value
            ? GfText(
                sync.blockedReason.value == 'rejected'
                    ? l.planSyncRejected
                    : l.planSyncCapacity,
              )
            : sync.conflicts.value.isNotEmpty
            ? GfText(l.planSyncTitle)
            : sync.needsAdoption.value
            ? GfText(l.planSyncAdoptHint)
            : sync.drafts.value.isNotEmpty
            ? GfText(l.planSyncDrafts)
            : null,
        children: const [
          SchedulerWebTip(),
          ScheduleSyncPanel(),
          ScheduleStoragePanel(),
        ],
      ),
    );
  }
}
