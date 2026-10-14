import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../providers.dart';
import '../server_messages.dart';
import '../widgets/status_views.dart';

final pushPreferencesProvider = FutureProvider.autoDispose<PushPreferences>((
  ref,
) {
  ref.watch(offlineCacheEpochProvider);
  return ref.watch(pushRepositoryProvider).preferences();
});

class PushPreferencesPanel extends ConsumerStatefulWidget {
  const PushPreferencesPanel({super.key});
  @override
  ConsumerState<PushPreferencesPanel> createState() =>
      _PushPreferencesPanelState();
}

class _PushPreferencesPanelState extends ConsumerState<PushPreferencesPanel> {
  bool _saving = false;

  Future<void> _save(String group, bool value) async {
    if (_saving) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _saving = true);
    try {
      final repository = await ref.read(pushRepositoryProvider).forSession();
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      await repository.updatePreferences({group: value});
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        ref.invalidate(pushPreferencesProvider);
      }
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final labels = {
      'chat': l10n.pushChat,
      'mentionsAndReplies': l10n.pushMentionsAndReplies,
      'activity': l10n.pushActivity,
      'following': l10n.pushFollowing,
      'system': l10n.pushSystem,
    };
    return ref
        .watch(pushPreferencesProvider)
        .when(
          loading: () => const Padding(
            padding: EdgeInsets.all(16),
            child: GfLoadingIndicator(small: true),
          ),
          error: (error, _) => GfErrorRetry(
            message: resolveErrorMessage(l10n, error),
            onRetry: () => ref.invalidate(pushPreferencesProvider),
          ),
          data: (preferences) => AbsorbPointer(
            absorbing: _saving,
            child: Column(
              children: [
                for (final entry in preferences.toJson().entries) ...[
                  const GfDivider(),
                  GfSwitchRow(
                    title: labels[entry.key]!,
                    value: entry.value,
                    onChanged: (value) => _save(entry.key, value),
                  ),
                ],
              ],
            ),
          ),
        );
  }
}
