import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../l10n/app_localizations.dart';
import '../../schedule/schedule_sync.dart';

class ScheduleSyncPanel extends ConsumerWidget {
  const ScheduleSyncPanel({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final sync = ref.watch(scheduleSyncControllerProvider);
    final l = AppLocalizations.of(context);
    return ListenableBuilder(
      listenable: Listenable.merge([
        sync.conflicts,
        sync.drafts,
        sync.needsAdoption,
        sync.blocked,
      ]),
      builder: (context, _) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (sync.needsAdoption.value)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  children: [
                    Text(l.planSyncAdoptHint),
                    TextButton(
                      onPressed: sync.adoptLocal,
                      child: Text(l.planSyncAdopt),
                    ),
                  ],
                ),
              ),
            ),
          if (sync.blocked.value)
            Text(
              l.planSyncCapacity,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          for (final conflict in sync.conflicts.value)
            _ConflictCard(
              key: ValueKey('${conflict.id}:${conflict.remote?.revision}'),
              sync: sync,
              conflict: conflict,
            ),
          if (sync.drafts.value.isNotEmpty)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l.planSyncDrafts,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    Text(l.planSyncDraftHint),
                    for (final entry in sync.drafts.value.entries)
                      Wrap(
                        crossAxisAlignment: WrapCrossAlignment.center,
                        children: [
                          Text(entry.value.name),
                          TextButton(
                            onPressed: () => sync.restoreDraft(entry.key),
                            child: Text(l.planSyncRestore),
                          ),
                        ],
                      ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _ConflictCard extends StatefulWidget {
  const _ConflictCard({super.key, required this.sync, required this.conflict});
  final ScheduleSyncController sync;
  final PlanSyncConflict conflict;
  @override
  State<_ConflictCard> createState() => _ConflictCardState();
}

class _ConflictCardState extends State<_ConflictCard> {
  final _choices = <String, String>{};
  bool _busy = false;
  String _describe(Object? value, AppLocalizations l) {
    if (value == null) return l.planSyncDeleted;
    if (value is List) return value.map((v) => _describe(v, l)).join(', ');
    if (value is Map) {
      if (value['course'] is Map) {
        return '${(value['course'] as Map)['courseName']}: ${_describe(value['selected'], l)}';
      }
      if (value['courses'] is Map) {
        return [
          value['name'],
          ...(value['courses'] as Map).values.map((v) => _describe(v, l)),
          ...(value['events'] as Map).values.map((v) => _describe(v, l)),
        ].join('\n');
      }
      return [
        value['label'],
        value['day'],
        value['sections'],
        value['weeks'],
      ].where((v) => v != null).map((v) => _describe(v, l)).join(' · ');
    }
    return '$value';
  }

  String _label(List<String> path, AppLocalizations l) {
    if (path.isEmpty) return l.planSyncPlan;
    if (path.first == 'courses') return '${l.planSyncCourse} ${path[1]}';
    return switch (path.last) {
      'name' => l.planSyncName,
      'createdAt' => l.planSyncCreatedAt,
      'label' => l.planSyncLabel,
      'day' => l.planSyncDay,
      'sections' => l.planSyncSections,
      'weeks' => l.planSyncWeeks,
      _ => l.planSyncEvent,
    };
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context), c = widget.conflict;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              '${l.planSyncTitle} · ${c.local?.name ?? c.remote?.plan.name ?? c.base?.name ?? ''}',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            Text(l.planSyncBody),
            for (final field in c.fields) ...[
              const SizedBox(height: 12),
              Text(
                _label(field.path, l),
                style: Theme.of(context).textTheme.titleSmall,
              ),
              for (final side in ['local', 'remote'])
                CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                  value: _choices[jsonEncode(field.path)] == side,
                  onChanged: _busy
                      ? null
                      : (_) => setState(
                          () => _choices[jsonEncode(field.path)] = side,
                        ),
                  title: Text(
                    side == 'local' ? l.planSyncLocal : l.planSyncRemote,
                  ),
                  subtitle: Text(
                    _describe(side == 'local' ? field.local : field.remote, l),
                  ),
                ),
            ],
            FilledButton(
              onPressed:
                  _busy ||
                      c.fields.any(
                        (f) => !_choices.containsKey(jsonEncode(f.path)),
                      )
                  ? null
                  : () async {
                      setState(() => _busy = true);
                      await widget.sync.resolveConflict(c.id, _choices);
                      if (mounted) setState(() => _busy = false);
                    },
              child: Text(l.planSyncApply),
            ),
          ],
        ),
      ),
    );
  }
}
