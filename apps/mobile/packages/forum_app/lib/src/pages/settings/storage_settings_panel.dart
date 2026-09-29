import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../offline/drift_cache.dart';
import '../../storage/device_storage.dart';
import '../../storage/storage_providers.dart';
import '../../storage/storage_strings.dart';

class StorageSettingsPanel extends ConsumerStatefulWidget {
  const StorageSettingsPanel({
    super.key,
    this.controller,
    this.signedIn = false,
  });
  final ScrollController? controller;
  final bool signedIn;
  @override
  ConsumerState<StorageSettingsPanel> createState() =>
      _StorageSettingsPanelState();
}

class _StorageSettingsPanelState extends ConsumerState<StorageSettingsPanel> {
  DeviceStorageUsage? _usage;
  bool _loading = true, _busy = false, _failed = false;
  final _selected = {
    CacheCategory.forum,
    CacheCategory.chat,
    CacheCategory.media,
  };
  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    if (mounted) setState(() => _loading = true);
    try {
      final usage = await ref.read(deviceStorageProvider).usage();
      if (mounted) {
        setState(() {
          _usage = usage;
          _failed = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<bool> _confirm({
    required String title,
    required String content,
    bool danger = false,
  }) async =>
      await showGfAlertDialog<bool>(
        context,
        builder: (ctx) => GfAlertDialog(
          title: Text(title),
          content: SingleChildScrollView(child: Text(content)),
          actions: [
            GfButton(
              label: StorageStrings(context).cancel,
              variant: GfButtonVariant.ghost,
              onPressed: () => Navigator.pop(ctx, false),
            ),
            GfButton(
              label: StorageStrings(context).confirm,
              variant: danger
                  ? GfButtonVariant.danger
                  : GfButtonVariant.primary,
              onPressed: () => Navigator.pop(ctx, true),
            ),
          ],
        ),
      ) ==
      true;

  Future<void> _clear({bool retry = false}) async {
    if (_busy) return;
    final s = StorageStrings(context);
    if (!retry &&
        !await _confirm(
          title: s.clearConfirm,
          content:
              '${_selected.map(s.category).join('、')}\n\n${s.preserve}${_selected.contains(CacheCategory.campus) ? '\n\n${s.campusHint}' : ''}',
        )) {
      return;
    }
    if (!mounted) return;
    setState(() => _busy = true);
    try {
      final service = ref.read(deviceStorageProvider);
      final result = retry
          ? await service.resume()
          : await service.clear(Set.of(_selected));
      if (!mounted) return;
      showGfToast(
        context,
        result.succeeded ? s.cleared : s.failed,
        error: !result.succeeded,
      );
      await _load();
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _reset() async {
    if (_busy || _usage == null) return;
    final s = StorageStrings(context), u = _usage!;
    if (!await _confirm(
      title: s.reset,
      content: [
        s.workSummary(
          u.drafts,
          u.chatDrafts,
          u.plans,
          u.unsyncedPlans,
          u.recoveryPlans,
        ),
        if (u.legacyPlans > 0) s.recovery(u.legacyPlans),
        s.resetHint,
        s.resetConfirm,
      ].join('\n\n'),
      danger: true,
    )) {
      return;
    }
    if (!mounted) return;
    setState(() => _busy = true);
    try {
      await ref.read(deviceStorageProvider).reset();
      if (!mounted) return;
      showGfToast(context, s.resetDone);
      context.go('/settings');
      await _load();
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = StorageStrings(context), u = _usage;
    final heading = Theme.of(context).textTheme.titleMedium;
    return ListView(
      controller: widget.controller,
      padding: const EdgeInsets.all(16),
      children: [
        Text(s.total, style: heading),
        const SizedBox(height: 8),
        if (_loading)
          Text(s.calculating)
        else if (u != null)
          Text(
            storageSize(u.physicalCacheBytes),
            style: Theme.of(context).textTheme.headlineMedium,
          ),
        const SizedBox(height: 8),
        Text(s.accounting),
        if (_failed) ...[
          const SizedBox(height: 12),
          Text(s.unavailable),
          GfButton(label: s.retry, onPressed: _busy ? null : _load),
        ],
        if (u != null && u.pending.isNotEmpty) ...[
          const SizedBox(height: 12),
          Text(s.failed),
          GfButton(
            label: s.retry,
            onPressed: _busy ? null : () => _clear(retry: true),
          ),
        ],
        const SizedBox(height: 24),
        Text(s.cache, style: heading),
        const SizedBox(height: 8),
        for (final category in CacheCategory.values)
          CheckboxListTile(
            contentPadding: EdgeInsets.zero,
            controlAffinity: ListTileControlAffinity.leading,
            value: _selected.contains(category),
            title: Text(s.category(category)),
            subtitle: category == CacheCategory.campus
                ? Text(s.campusHint)
                : null,
            secondary: Text(
              u == null ? '—' : storageSize(u.cacheBytes[category] ?? 0),
            ),
            onChanged: _busy
                ? null
                : (value) => setState(() {
                    value == true
                        ? _selected.add(category)
                        : _selected.remove(category);
                  }),
          ),
        const SizedBox(height: 8),
        Text(s.preserve),
        const SizedBox(height: 16),
        GfButton(
          label: _busy ? s.calculating : s.clear,
          onPressed: _busy || _selected.isEmpty || _loading ? null : _clear,
        ),
        const SizedBox(height: 28),
        Text(s.localWork, style: heading),
        const SizedBox(height: 8),
        if (u != null) ...[
          Text(
            s.workSummary(
              u.drafts,
              u.chatDrafts,
              u.plans,
              u.unsyncedPlans,
              u.recoveryPlans,
            ),
          ),
          Text(storageSize(u.workBytes)),
          if (u.legacyPlans > 0) Text(s.recovery(u.legacyPlans)),
        ],
        Text(s.excluded),
        if (widget.signedIn)
          GfSettingRow(
            symbol: 'file-text',
            title: s.drafts,
            onTap: _busy ? null : () => context.push('/drafts'),
          ),
        GfSettingRow(
          symbol: 'calendar-days',
          title: s.plans,
          onTap: _busy ? null : () => context.push('/schedule'),
        ),
        const SizedBox(height: 28),
        Text(s.reset, style: heading),
        const SizedBox(height: 8),
        Text(s.resetHint),
        const SizedBox(height: 16),
        GfButton(
          label: s.reset,
          variant: GfButtonVariant.danger,
          onPressed: _busy || u == null ? null : _reset,
        ),
        const SizedBox(height: 24),
      ],
    );
  }
}
