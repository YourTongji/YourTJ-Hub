import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../campus/campus_memory_cache.dart';

/// Clear only disposable campus data; drafts and schedule plans are not cache.
Future<void> clearCampusDeviceCache(WidgetRef ref) async {
  ref.read(campusCacheEpochProvider.notifier).invalidate();
  ref.read(campusMemoryCacheProvider).clear();
  final store = ref.read(campusSnapshotStoreProvider);
  final bridge = ref.read(scheduleWidgetBridgeProvider);
  store.invalidate();
  bridge.invalidate();
  Object? failure;
  try {
    await store.clear();
  } catch (error) {
    failure = error;
  }
  // Attempt both stores, even if one fails. A retry can finish partial cleanup.
  try {
    await bridge.clear();
  } catch (error) {
    failure ??= error;
  }
  if (failure != null) throw failure;
}

class CampusCacheClearTile extends ConsumerStatefulWidget {
  const CampusCacheClearTile({super.key});
  @override
  ConsumerState<CampusCacheClearTile> createState() =>
      _CampusCacheClearTileState();
}

class _CampusCacheClearTileState extends ConsumerState<CampusCacheClearTile> {
  bool _busy = false;
  bool _failed = false;

  Future<void> _clear() async {
    if (_busy) return;
    final l = AppLocalizations.of(context);
    final confirmed = await showGfAlertDialog<bool>(
      context,
      builder: (context) => GfAlertDialog(
        title: Text(l.campusCacheClear),
        content: Text(l.campusCacheClearDescription),
        actions: [
          GfButton(
            label: l.commonCancel,
            variant: GfButtonVariant.ghost,
            onPressed: () => Navigator.pop(context, false),
          ),
          GfButton(
            label: l.commonConfirm,
            onPressed: () => Navigator.pop(context, true),
          ),
        ],
      ),
    );
    if (!mounted || confirmed != true || _busy) return;
    setState(() {
      _busy = true;
      _failed = false;
    });
    try {
      await clearCampusDeviceCache(ref);
      if (mounted) showGfToast(context, l.campusCacheCleared);
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    return GfSettingRow(
      symbol: 'trash-2',
      title: l.campusCacheClear,
      description: _failed
          ? l.campusCacheClearFailed
          : l.campusCacheClearDescription,
      trailing: _busy
          ? const GfLoadingIndicator(small: true)
          : const Icon(Icons.chevron_right, size: 18),
      onTap: _busy ? null : _clear,
    );
  }
}
