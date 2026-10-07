import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../asset_url.dart';
import '../../server_messages.dart';
import '../../local/writing_store.dart';

final anonymousIdentityProvider =
    FutureProvider.autoDispose<AnonymousIdentityState>((ref) {
      ref.watch(offlineCacheEpochProvider);
      return AnonymousIdentityRepository(ref.watch(apiClientProvider)).state();
    });

class AnonymousIdentityPage extends ConsumerStatefulWidget {
  const AnonymousIdentityPage({super.key});
  @override
  ConsumerState<AnonymousIdentityPage> createState() =>
      _AnonymousIdentityPageState();
}

class _AnonymousIdentityPageState extends ConsumerState<AnonymousIdentityPage> {
  bool busy = false;
  Object? error;
  String? pendingKey, pendingDay;
  late final int epoch;
  AnonymousIdentityRepository get repo =>
      AnonymousIdentityRepository(ref.read(apiClientProvider));
  @override
  void initState() {
    super.initState();
    epoch = ref.read(offlineCacheEpochProvider);
  }

  bool get current => mounted && epoch == ref.read(offlineCacheEpochProvider);
  Future<void> run(Future<void> Function() action) async {
    if (busy || !current) return;
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await action();
      if (current) ref.invalidate(anonymousIdentityProvider);
    } catch (e) {
      if (current) setState(() => error = e);
    } finally {
      if (current) setState(() => busy = false);
    }
  }

  Future<void> choose(AnonymousNameBatch batch, int index) async {
    final l = AppLocalizations.of(context);
    final yes = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        scrollable: true,
        title: Text(l.anonymousConfirm),
        content: Text('${batch.words[index]}\n\n${l.anonymousBoundary}'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l.anonymousCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l.anonymousConfirm),
          ),
        ],
      ),
    );
    if (yes == true && current) {
      await run(() async {
        await repo.confirm(batch.id, index);
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      if (next != epoch && mounted) context.go('/');
    });
    final l = AppLocalizations.of(context);
    final state = ref.watch(anonymousIdentityProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l.anonymousIdentity)),
      body: state.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: TextButton(
            onPressed: () => ref.invalidate(anonymousIdentityProvider),
            child: Text(resolveErrorMessage(l, e)),
          ),
        ),
        data: (s) => ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(l.anonymousBoundary),
            const SizedBox(height: 20),
            if (s.persona case final p?) ...[
              ListTile(
                leading: GfAvatar(src: resolveApiAssetUrl(p.avatarUrl)),
                title: Text(p.name),
                subtitle: Text(l.anonymousPersonaLabel),
                onTap: () => context.push(p.profileUrl),
              ),
              if (s.availableAt != null)
                Text('${l.anonymousLockDate}${s.availableAt!.toLocal()}'),
              FilledButton.tonal(
                onPressed: busy || s.governanceDisabled
                    ? null
                    : () => run(() => repo.setDisabled(!s.disabled)),
                child: Text(
                  s.disabled ? l.anonymousEnable : l.anonymousDisable,
                ),
              ),
            ],
            if (s.governanceDisabled) Text(l.anonymousUnavailable),
            if (error != null)
              TextButton(
                onPressed: () => ref.invalidate(anonymousIdentityProvider),
                child: Text(resolveErrorMessage(l, error!)),
              ),
            Text('${l.anonymousRemaining}${s.remaining}'),
            if (!s.locked && !s.disabled && !s.governanceDisabled) ...[
              FilledButton(
                onPressed: busy || s.remaining == 0
                    ? null
                    : () => run(() async {
                        if (pendingDay != s.day) {
                          pendingKey = null;
                          pendingDay = s.day;
                        }
                        pendingKey ??= newTopicDraftKey();
                        await repo.generate(s.day, pendingKey!);
                        if (current) pendingKey = null;
                      }),
                child: Text(busy ? l.commonLoading : l.anonymousRandomize),
              ),
              for (final batch in s.batches) ...[
                const Divider(),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (var i = 0; i < batch.words.length; i++)
                      OutlinedButton(
                        onPressed: busy ? null : () => choose(batch, i),
                        child: Text(batch.words[i]),
                      ),
                  ],
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }
}
