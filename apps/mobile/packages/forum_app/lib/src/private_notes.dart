import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../l10n/app_localizations.dart';
import 'current_user.dart';
import 'providers.dart';
import 'server_messages.dart';

// Every session epoch gets a new provider instance. Nothing is persisted offline.
final privateNotesProvider = FutureProvider.autoDispose
    .family<Map<int, PrivateNotePayload>, (int, int)>((ref, key) async {
      if (key.$1 <= 0) return {};
      final result = await ref.read(userRepositoryProvider).getPrivateNotes();
      if (result.ownerId != key.$1) {
        throw StateError('Private notes owner changed');
      }
      return {for (final note in result.notes) note.targetUserId: note};
    });

class PrivateNotesScope extends InheritedWidget {
  const PrivateNotesScope({
    super.key,
    required this.ownerId,
    required this.notes,
    this.ready = true,
    this.failed = false,
    required super.child,
  });
  final bool ready;
  final bool failed;
  final int ownerId;
  final Map<int, PrivateNotePayload> notes;
  static PrivateNotesScope? of(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<PrivateNotesScope>();
  @override
  bool updateShouldNotify(PrivateNotesScope oldWidget) =>
      ownerId != oldWidget.ownerId ||
      notes != oldWidget.notes ||
      ready != oldWidget.ready ||
      failed != oldWidget.failed;
}

String privateDisplayName(
  BuildContext context,
  int id,
  String username, [
  String? nickname,
]) {
  final note = PrivateNotesScope.of(context)?.notes[id];
  if (note == null) return nickname?.isNotEmpty == true ? nickname! : username;

  final displayName = nickname?.isNotEmpty == true
      ? nickname!
      : username.isNotEmpty
      ? username
      : note.username;
  return '${note.note}($displayName)';
}

class PrivateNotesHost extends ConsumerStatefulWidget {
  const PrivateNotesHost({super.key, required this.child});
  final Widget child;
  @override
  ConsumerState<PrivateNotesHost> createState() => _PrivateNotesHostState();
}

class _PrivateNotesHostState extends ConsumerState<PrivateNotesHost>
    with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      ref.invalidate(privateNotesProvider);
    }
  }

  @override
  Widget build(BuildContext context) {
    final epoch = ref.watch(offlineCacheEpochProvider);
    final owner = ref.watch(currentUserProvider).valueOrNull?.id ?? 0;
    final result = ref.watch(privateNotesProvider((owner, epoch)));
    return PrivateNotesScope(
      ownerId: owner,
      notes: result.valueOrNull ?? const {},
      ready: result.hasValue && !result.isLoading && !result.hasError,
      failed: result.hasError,
      child: widget.child,
    );
  }
}

class PrivateNoteButton extends ConsumerWidget {
  const PrivateNoteButton({
    super.key,
    required this.userId,
    required this.username,
  });
  final int userId;
  final String username;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scope = PrivateNotesScope.of(context);
    if (scope == null || scope.ownerId <= 0 || scope.ownerId == userId) {
      return const SizedBox.shrink();
    }
    if (scope.failed) {
      return TextButton(
        onPressed: () => ref.invalidate(privateNotesProvider),
        child: Text(AppLocalizations.of(context).commonRetry),
      );
    }
    return TextButton(
      onPressed: !scope.ready
          ? null
          : () => showDialog<void>(
              context: context,
              builder: (_) => _PrivateNoteDialog(
                userId: userId,
                initial: scope.notes[userId]?.note ?? '',
              ),
            ),
      child: Text(AppLocalizations.of(context).privateNoteEdit),
    );
  }
}

class _PrivateNoteDialog extends ConsumerStatefulWidget {
  const _PrivateNoteDialog({required this.userId, required this.initial});
  final int userId;
  final String initial;
  @override
  ConsumerState<_PrivateNoteDialog> createState() => _PrivateNoteDialogState();
}

class _PrivateNoteDialogState extends ConsumerState<_PrivateNoteDialog> {
  late final _text = TextEditingController(text: widget.initial);
  bool _busy = false;
  String? _error;
  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_busy || _text.text.trim().runes.length > 64) return;
    final l10n = AppLocalizations.of(context);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref
          .read(userRepositoryProvider)
          .setPrivateNote(widget.userId, _text.text);
      ref.invalidate(privateNotesProvider);
      if (mounted) Navigator.pop(context);
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = resolveErrorMessage(l10n, error);
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _busy = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return PopScope(
      canPop: !_busy,
      child: AlertDialog(
        scrollable: true,
        title: Text(l10n.privateNoteLabel),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(l10n.privateNoteHint),
            TextField(
              controller: _text,
              enabled: !_busy,
              autofocus: true,
              decoration: InputDecoration(labelText: l10n.privateNoteLabel),
              onChanged: (_) => setState(() {}),
            ),
            Text('${_text.text.trim().runes.length}/64'),
            if (_error != null)
              Text(
                _error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: _busy ? null : () => Navigator.pop(context),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: _busy || _text.text.trim().runes.length > 64
                ? null
                : _save,
            child: Text(l10n.commonSave),
          ),
        ],
      ),
    );
  }
}
