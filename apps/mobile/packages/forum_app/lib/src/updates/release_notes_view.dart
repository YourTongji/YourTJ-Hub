import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import 'release_notes.dart';

/// The same bounded, semantic note presentation is used by update prompts and history.
/// Group labels and note titles stay below the surface title in the type scale,
/// so reviewed notes never render as banner-sized headings.
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
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final required = notes.where((note) => note.required).toList();
    final byKind = <String, List<ReleaseNote>>{};
    for (final note in notes.where((note) => !note.required)) {
      byKind.putIfAbsent(note.kind, () => []).add(note);
    }
    final groups = <Widget>[
      if (required.isNotEmpty)
        DecoratedBox(
          decoration: BoxDecoration(
            color: colors.base300,
            borderRadius: BorderRadius.circular(16),
          ),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 16),
            child: _NoteGroup(
              symbol: 'circle-alert',
              label: l10n.releaseNotesRequired,
              notes: required,
            ),
          ),
        ),
      for (final (kind, symbol, label) in [
        ('security', 'shield-check', l10n.releaseNotesKindSecurity),
        ('feature', 'sparkles', l10n.releaseNotesKindFeature),
        ('improvement', 'trend-up', l10n.releaseNotesKindImprovement),
        ('fix', 'bug', l10n.releaseNotesKindFix),
      ])
        if (byKind[kind] case final kindNotes?)
          _NoteGroup(symbol: symbol, label: label, notes: kindNotes),
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (var i = 0; i < groups.length; i++) ...[
          if (i > 0) const SizedBox(height: 20),
          groups[i],
        ],
        if (!historyComplete) ...[
          if (groups.isNotEmpty) const SizedBox(height: 16),
          Text(
            l10n.releaseNotesIncomplete,
            style: type.caption.copyWith(
              color: colors.baseContent.withValues(alpha: 0.6),
            ),
          ),
        ],
      ],
    );
  }
}

class _NoteGroup extends StatelessWidget {
  const _NoteGroup({
    required this.symbol,
    required this.label,
    required this.notes,
  });

  final String symbol;
  final String label;
  final List<ReleaseNote> notes;

  @override
  Widget build(BuildContext context) {
    final type = GfTheme.typographyOf(context);
    final muted = GfTheme.colorsOf(context).baseContent.withValues(alpha: 0.6);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Semantics(
          header: true,
          child: Row(
            children: [
              GfSymbol(symbol, size: 16, color: muted),
              const SizedBox(width: 6),
              Flexible(
                child: Text(
                  label,
                  style: type.caption.copyWith(
                    color: muted,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        for (var i = 0; i < notes.length; i++) ...[
          if (i > 0) const SizedBox(height: 12),
          _NoteItem(note: notes[i]),
        ],
      ],
    );
  }
}

class _NoteItem extends StatelessWidget {
  const _NoteItem({required this.note});

  final ReleaseNote note;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final hasTitle = note.title.isNotEmpty;
    // Drafted notes may restate a short title as the whole summary; show it once.
    final restated =
        hasTitle &&
        note.summary.replaceFirst(RegExp(r'[。．.！!？?]+$'), '').trim() ==
            note.title.trim();
    return Semantics(
      container: true,
      excludeSemantics: true,
      label:
          '${note.required ? '${l10n.releaseNotesRequired}. ' : ''}${restated ? note.title : '${hasTitle ? '${note.title}. ' : ''}${note.summary}'}',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (hasTitle) Text(note.title, style: type.bodyStrong),
          if (hasTitle && !restated) const SizedBox(height: 2),
          if (!restated)
            Text(
              note.summary,
              style: type.small.copyWith(
                height: 1.45,
                color: hasTitle
                    ? colors.baseContent.withValues(alpha: 0.72)
                    : colors.baseContent,
              ),
            ),
        ],
      ),
    );
  }
}
