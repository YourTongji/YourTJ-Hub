import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../current_user.dart';
import '../../providers.dart';
import '../../schedule/schedule_store.dart';
import '../../schedule/schedule_sync.dart';
import 'schedule_storage_strings.dart';

/// Recovery is explicit because old preferences never recorded their API origin.
class ScheduleStoragePanel extends ConsumerStatefulWidget {
  const ScheduleStoragePanel({super.key});
  @override
  ConsumerState<ScheduleStoragePanel> createState() =>
      _ScheduleStoragePanelState();
}

class _ScheduleStoragePanelState extends ConsumerState<ScheduleStoragePanel> {
  Map<String, String>? _legacy;
  Map<String, String> _archive = {};
  bool _busy = false;
  bool _readFailed = false;
  int _loadRevision = 0;
  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final revision = ++_loadRevision;
    final store = ref.read(scheduleStoreProvider.notifier);
    final owner = ref.read(currentUserProvider).valueOrNull?.id ?? 0;
    try {
      final data = await store.readUnassignedLegacyPlans();
      final archive = await store.readRecoveryArchive(owner: owner);
      if (mounted && revision == _loadRevision) {
        setState(() {
          _legacy = data;
          _archive = archive;
          _readFailed = false;
        });
      }
    } catch (_) {
      if (mounted && revision == _loadRevision) {
        setState(() => _readFailed = true);
      }
    }
  }

  int get _count {
    try {
      return (jsonDecode(_legacy?['pk.plans'] ?? '[]') as List).length;
    } catch (_) {
      return 0;
    }
  }

  Future<void> _review() async {
    final strings = ScheduleStorageStrings(context);
    final notifier = ref.read(scheduleStoreProvider.notifier);
    final userState = ref.read(currentUserProvider);
    if (userState.isLoading || userState.hasError) return;
    final user = userState.valueOrNull;
    final owner = user?.id ?? 0;
    final epoch = ref.read(offlineCacheEpochProvider);
    final site = notifier.site;
    final oldOwner = int.tryParse(_legacy?['pk.syncOwner'] ?? '');
    final allowed = oldOwner == null || oldOwner == 0 || oldOwner == owner;
    final accepted = await showGfAlertDialog<bool>(
      context,
      builder: (context) => GfAlertDialog(
        title: GfText(strings.review),
        content: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              GfText(strings.explanation),
              const SizedBox(height: 12),
              GfSelectableText(
                '$site\n${user?.username ?? strings.guest} · ID $owner',
              ),
              const SizedBox(height: 12),
              GfText(strings.summary(_count)),
              if (!allowed)
                GfText(
                  strings.ownerMismatch(oldOwner),
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: GfText(strings.cancel),
          ),
          FilledButton(
            onPressed: allowed ? () => Navigator.pop(context, true) : null,
            child: GfText(strings.restore),
          ),
        ],
      ),
    );
    if (accepted != true || !mounted) return;
    bool current() =>
        mounted &&
        ref.read(offlineCacheEpochProvider) == epoch &&
        identical(ref.read(scheduleStoreProvider.notifier), notifier) &&
        ref.read(currentUserProvider).valueOrNull?.id == user?.id;
    if (!current()) return;
    setState(() => _busy = true);
    final restored = await notifier.restoreUnassignedLegacyPlans(
      owner: owner,
      canWrite: current,
    );
    if (!mounted) return;
    setState(() => _busy = false);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: GfText(restored ? strings.restored : strings.failed)),
    );
    if (restored) {
      // Rebuild CAS state only after the explicit move succeeds.
      await ref.read(scheduleSyncControllerProvider).reloadLocalAfterRecovery();
      await _load();
    }
  }

  Future<void> _export() async {
    final data = _legacy?.isNotEmpty == true ? _legacy! : _archive;
    if (data.isEmpty) return;
    final strings = ScheduleStorageStrings(context);
    final render = context.findRenderObject() as RenderBox?;
    try {
      await SharePlus.instance.share(
        ShareParams(
          files: [
            XFile.fromData(
              Uint8List.fromList(
                utf8.encode(const JsonEncoder.withIndent('  ').convert(data)),
              ),
              mimeType: 'application/json',
            ),
          ],
          fileNameOverrides: ['yourtj-legacy-schedule.json'],
          sharePositionOrigin: render == null
              ? null
              : render.localToGlobal(Offset.zero) & render.size,
        ),
      );
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: GfText(strings.failed)));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final strings = ScheduleStorageStrings(context);
    final notifier = ref.watch(scheduleStoreProvider.notifier);
    final identity = ref.watch(currentUserProvider);
    void reloadScope() {
      _legacy = null;
      _archive = {};
      _load();
    }

    ref.listen(scheduleStoreProvider.notifier, (previous, next) {
      if (!identical(previous, next)) reloadScope();
    });
    ref.listen(currentUserProvider, (previous, next) {
      if (previous?.valueOrNull?.id != next.valueOrNull?.id) reloadScope();
    });
    return ValueListenableBuilder<Object?>(
      valueListenable: notifier.persistenceError,
      builder: (context, error, _) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (error != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: GfText(
                strings.storageError,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ),
          if (error != null)
            TextButton(
              onPressed: notifier.retryPersistence,
              child: GfText(strings.retry),
            ),
          if (_readFailed)
            TextButton(onPressed: _load, child: GfText(strings.retry)),
          if (_legacy?.isNotEmpty != true && _archive.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: TextButton(
                onPressed: _busy ? null : _export,
                child: GfText(strings.exportArchive),
              ),
            ),
          if (_legacy?.isNotEmpty == true)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfText(
                      strings.found,
                      style: Theme.of(context).textTheme.titleSmall,
                    ),
                    const SizedBox(height: 6),
                    GfText(strings.summary(_count)),
                    GfText(strings.exportHint),
                    Wrap(
                      spacing: 8,
                      children: [
                        TextButton(
                          onPressed:
                              _busy || identity.isLoading || identity.hasError
                              ? null
                              : _review,
                          child: GfText(strings.review),
                        ),
                        TextButton(
                          onPressed: _busy ? null : _export,
                          child: GfText(strings.export),
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
