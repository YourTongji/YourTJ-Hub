import 'package:flutter/material.dart';

import '../../l10n/app_localizations.dart';
import 'release_notes.dart';
import 'release_notes_view.dart';

class UpdateDialogBody extends StatelessWidget {
  const UpdateDialogBody({
    super.key,
    this.sizeBytes,
    required this.notes,
    required this.historyComplete,
    required this.working,
    required this.failed,
    required this.needsPermission,
    required this.ready,
    required this.receivedBytes,
    this.footer,
  });

  final int? sizeBytes;
  final List<ReleaseNote> notes;
  final bool historyComplete;
  final bool working;
  final bool failed;
  final bool needsPermission;
  final bool ready;
  final int receivedBytes;
  final Widget? footer;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final reduceMotion = MediaQuery.disableAnimationsOf(context);
    final statusText = working
        ? (receivedBytes == 0 ? l10n.updatePreparing : l10n.updateDownloading)
        : failed
        ? l10n.updateFailed
        : needsPermission
        ? l10n.updatePermission
        : ready
        ? l10n.updateReady
        : null;
    return ConstrainedBox(
      constraints: BoxConstraints(
        maxHeight: MediaQuery.sizeOf(context).height * .68,
      ),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (sizeBytes != null)
              Text('${(sizeBytes! / 1024 / 1024).toStringAsFixed(1)} MB'),
            if (notes.isNotEmpty || !historyComplete) ...[
              const SizedBox(height: 14),
              ReleaseNotesView(notes: notes, historyComplete: historyComplete),
            ],
            const SizedBox(height: 12),
            if (statusText != null)
              Semantics(
                key: const ValueKey('update-status'),
                container: true,
                liveRegion: true,
                label: statusText,
                excludeSemantics: true,
                child: Text(statusText),
              ),
            if (working) ...[
              const SizedBox(height: 12),
              ExcludeSemantics(
                child: LinearProgressIndicator(
                  value: receivedBytes == 0
                      ? (reduceMotion ? 0 : null)
                      : receivedBytes / (sizeBytes ?? 1),
                ),
              ),
            ],
            ?footer,
          ],
        ),
      ),
    );
  }
}
