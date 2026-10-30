import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../providers.dart';
import '../server_messages.dart';
import '../widgets/status_views.dart';
import 'unread_display.dart';

class UnreadDisplaySettings extends ConsumerStatefulWidget {
  const UnreadDisplaySettings({super.key});
  @override
  ConsumerState<UnreadDisplaySettings> createState() =>
      _UnreadDisplaySettingsState();
}

class _UnreadDisplaySettingsState extends ConsumerState<UnreadDisplaySettings> {
  @override
  void initState() {
    super.initState();
    unawaited(ref.read(unreadDisplayProvider.notifier).load());
  }

  Future<void> _save(bool numbers) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final controller = ref.read(unreadDisplayProvider.notifier);
    final saved = await controller.update(
      numbers ? UnreadBadgeStyle.number : UnreadBadgeStyle.dot,
    );
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    final error = ref.read(unreadDisplayProvider).error;
    if (!saved && error != null) {
      showGfToast(
        context,
        resolveErrorMessage(AppLocalizations.of(context), error),
        error: true,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(unreadDisplayProvider);
    final l = AppLocalizations.of(context);
    if (!state.loaded) {
      if (state.error != null) {
        return GfErrorRetry(
          message: resolveErrorMessage(l, state.error!),
          onRetry: () =>
              unawaited(ref.read(unreadDisplayProvider.notifier).load()),
        );
      }
      return const Padding(
        padding: EdgeInsets.all(16),
        child: GfLoadingIndicator(small: true),
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfSwitchRow(
          key: const Key('unread-style-switch'),
          symbol: 'bell',
          title: l.unreadStyleNumbers,
          value: state.style == UnreadBadgeStyle.number,
          enabled: !state.saving,
          onChanged: _save,
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
          child: GfText(
            l.unreadStyleHint,
            style: TextStyle(
              fontSize: 14,
              color: GfTheme.colorsOf(context).textSecondary,
            ),
          ),
        ),
        if (state.saving)
          Semantics(
            liveRegion: true,
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: GfText(l.unreadStyleSaving),
            ),
          ),
        if (state.error != null)
          Semantics(
            liveRegion: true,
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: GfText(resolveErrorMessage(l, state.error!)),
            ),
          ),
      ],
    );
  }
}
