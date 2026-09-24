import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../admin/admin_page.dart';
import '../settings/campus_cache_clear_tile.dart';
import 'campus_helpers.dart';
import 'campus_state.dart';

class CampusConnection extends ConsumerStatefulWidget {
  const CampusConnection({super.key, this.compact = false, this.onConfirmed});
  final bool compact;
  final VoidCallback? onConfirmed;
  @override
  ConsumerState<CampusConnection> createState() => _CampusConnectionState();
}

class _CampusConnectionState extends ConsumerState<CampusConnection> {
  final _cancel = CancelToken();
  @override
  void dispose() {
    _cancel.cancel();
    super.dispose();
  }

  bool _opening = false;
  Object? _error;
  Future<void> _authorize(String mode) async {
    if (_opening) return;
    setState(() {
      _opening = true;
      _error = null;
    });
    final epoch = ref.read(offlineCacheEpochProvider);
    try {
      final uri = await ref
          .read(campusRepositoryProvider)
          .start(mode, cancelToken: _cancel);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      await Navigator.of(context).push<bool>(
        MaterialPageRoute(
          builder: (_) => AdminPage(
            target: MobileWebTarget.campus,
            campusAuthorizationUrl: uri,
          ),
        ),
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      await ref.read(campusControllerProvider.notifier).refresh();
    } catch (e) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _error = e);
      }
    } finally {
      if (mounted) setState(() => _opening = false);
    }
  }

  Future<void> _unbind(CampusBinding binding) async {
    final l = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final accepted = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l.campusUnbind),
        content: Text(l.campusUnbindBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l.campusUnbind),
          ),
        ],
      ),
    );
    if (!mounted ||
        accepted != true ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    final controller = ref.read(campusControllerProvider.notifier);
    await controller.change(
      (cancel) =>
          controller.repository.unbind(binding.revision, cancelToken: cancel),
      widgetState: 'unbound',
    );
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(campusControllerProvider);
    final binding = state.status?.binding;
    final candidate = state.status?.candidate;
    final busy = state.busy || _opening;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (!widget.compact) ...[
          Text(l.campusConnection, style: GfTheme.typographyOf(context).title2),
          const SizedBox(height: 12),
          Text(l.campusPrivacy),
          const SizedBox(height: 16),
          if (binding != null)
            Text(binding.maskedId, style: GfTheme.typographyOf(context).title1),
          const SizedBox(height: 12),
        ],
        if (candidate != null) ...[
          Text(l.campusConfirmBody),
          const SizedBox(height: 8),
          Text(
            candidate.maskedId,
            style: GfTheme.typographyOf(context).heading,
          ),
          const SizedBox(height: 12),
          GfButton(
            label: candidate.mode == 'reauthorize'
                ? l.campusConfirmUpdate
                : l.campusConfirmBinding,
            onPressed: busy
                ? null
                : () async {
                    final c = ref.read(campusControllerProvider.notifier);
                    final ok = await c.change(
                      (cancel) => c.repository.confirm(cancelToken: cancel),
                    );
                    if (mounted && ok) widget.onConfirmed?.call();
                  },
          ),
        ] else if (state.status?.enabled == false)
          Text(l.campusDisabled)
        else if (binding == null)
          GfButton(
            label: l.campusBind,
            onPressed: busy ? null : () => _authorize('bind'),
          )
        else ...[
          if (state.needsAuthorization || widget.compact)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Text(l.campusAuthRequired),
            ),
          GfButton(
            label: l.campusReauthorize,
            onPressed: busy ? null : () => _authorize('reauthorize'),
          ),
          if (!widget.compact) ...[
            const SizedBox(height: 8),
            GfButton(
              label: l.campusReplace,
              variant: GfButtonVariant.outline,
              onPressed: busy ? null : () => _authorize('replace'),
            ),
            const SizedBox(height: 8),
            GfButton(
              label: l.campusUnbind,
              variant: GfButtonVariant.ghost,
              onPressed: busy ? null : () => _unbind(binding),
            ),
            const SizedBox(height: 24),
            Text(l.campusUpstreamGaps),
          ],
        ],
        if (!widget.compact) ...[
          const SizedBox(height: 16),
          const CampusCacheClearTile(),
        ],
        if (_error != null || state.error != null)
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: Text(
              campusError(l, _error ?? state.error),
              style: TextStyle(color: GfTheme.colorsOf(context).error),
            ),
          ),
      ],
    );
  }
}
