import 'package:flutter/material.dart';

import '../../l10n/app_localizations.dart';
import 'release_notes.dart';

/// The same bounded, semantic note presentation is used by update prompts and history.
class ReleaseNotesView extends StatelessWidget {
  const ReleaseNotesView({
    super.key,
    required this.notes,
    this.historyComplete = true,
  });

  final List<ReleaseNote> notes;
  final bool historyComplete;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (notes.isNotEmpty) ...[
          Text(
            l10n.releaseNotes,
            style: Theme.of(context).textTheme.titleSmall,
          ),
          const SizedBox(height: 8),
          for (final note in notes)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Semantics(
                container: true,
                excludeSemantics: true,
                label:
                    '${note.required ? '${l10n.releaseNotesRequired}. ' : ''}${note.title.isEmpty ? '' : '${note.title}. '}${note.summary}',
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if (note.required)
                      Text(
                        l10n.releaseNotesRequired,
                        style: Theme.of(context).textTheme.labelSmall,
                      ),
                    if (note.title.isNotEmpty)
                      Text(
                        note.title,
                        style: Theme.of(context).textTheme.titleSmall,
                      ),
                    Text(note.summary),
                  ],
                ),
              ),
            ),
        ],
        if (!historyComplete)
          Text(
            l10n.releaseNotesIncomplete,
            style: Theme.of(context).textTheme.bodySmall,
          ),
      ],
    );
  }
}
