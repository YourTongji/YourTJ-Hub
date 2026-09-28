import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../l10n/app_localizations.dart';
import 'current_user.dart';
import 'providers.dart';
import 'server_messages.dart';

// Kept in memory only, scoped to the current login epoch.
final userBlocksProvider = FutureProvider.autoDispose<UserBlocksPayload>((
  ref,
) async {
  ref.watch(offlineCacheEpochProvider);
  final owner = await ref.watch(currentUserProvider.future);
  if (owner == null) return const UserBlocksPayload(ownerId: 0, blocks: []);
  final result = await ref.read(userRepositoryProvider).getUserBlocks();
  if (result.ownerId != owner.id) throw StateError('Block list owner changed');
  return result;
});

class UserBlockButton extends ConsumerStatefulWidget {
  const UserBlockButton({super.key, required this.userId});
  final int userId;
  @override
  ConsumerState<UserBlockButton> createState() => _UserBlockButtonState();
}

class _UserBlockButtonState extends ConsumerState<UserBlockButton> {
  bool _busy = false;
  Future<void> _change(bool blocked) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final l10n = AppLocalizations.of(context);
    setState(() => _busy = true);
    try {
      await changeUserBlock(
        context,
        ref,
        userId: widget.userId,
        blocked: blocked,
      );
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = ref.watch(currentUserProvider);
    if (user.isLoading ||
        user.valueOrNull == null ||
        user.valueOrNull!.id == widget.userId) {
      return const SizedBox.shrink();
    }
    final list = ref.watch(userBlocksProvider);
    final l10n = AppLocalizations.of(context);
    final blocked =
        list.valueOrNull?.blocks.any(
          (item) => item.targetUserId == widget.userId,
        ) ??
        false;
    return IconButton(
      constraints: const BoxConstraints(minWidth: 48, minHeight: 48),
      tooltip: list.hasError
          ? l10n.commonRetry
          : blocked
          ? l10n.userUnblock
          : l10n.userBlock,
      icon: Icon(blocked ? Icons.person_off : Icons.block),
      onPressed: _busy || list.isLoading
          ? null
          : list.hasError
          ? () => ref.invalidate(userBlocksProvider)
          : () => _change(!blocked),
    );
  }
}

Future<void> changeUserBlock(
  BuildContext context,
  WidgetRef ref, {
  required int userId,
  required bool blocked,
}) async {
  final epoch = ref.read(offlineCacheEpochProvider);
  final l10n = AppLocalizations.of(context);
  final approved = await showDialog<bool>(
    context: context,
    animationStyle: GfMotion.dialogStyle(context),
    builder: (ctx) => AlertDialog(
      scrollable: true,
      title: Text(blocked ? l10n.userBlock : l10n.userUnblock),
      content: Text(l10n.userBlockExplanation),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(ctx, false),
          child: Text(l10n.commonCancel),
        ),
        FilledButton(
          onPressed: () => Navigator.pop(ctx, true),
          child: Text(blocked ? l10n.userBlock : l10n.userUnblock),
        ),
      ],
    ),
  );
  if (!context.mounted ||
      approved != true ||
      epoch != ref.read(offlineCacheEpochProvider)) {
    return;
  }
  await ref.read(userRepositoryProvider).setUserBlock(userId, blocked);
  if (!context.mounted || epoch != ref.read(offlineCacheEpochProvider)) {
    return;
  }
  ref.invalidate(userBlocksProvider);
}

Future<void> showBlockedUsers(BuildContext context) => showDialog<void>(
  context: context,
  animationStyle: GfMotion.dialogStyle(context),
  builder: (_) => const _BlockedUsersDialog(),
);

class _BlockedUsersDialog extends ConsumerWidget {
  const _BlockedUsersDialog();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return AlertDialog(
      scrollable: true,
      title: Text(l10n.userBlocks),
      content: SizedBox(
        width: 420,
        child: ref
            .watch(userBlocksProvider)
            .when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (error, _) => Column(
                children: [
                  Text(resolveErrorMessage(l10n, error)),
                  TextButton(
                    onPressed: () => ref.invalidate(userBlocksProvider),
                    child: Text(l10n.commonRetry),
                  ),
                ],
              ),
              data: (data) => data.blocks.isEmpty
                  ? Text(l10n.userBlocksEmpty)
                  : Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        for (final item in data.blocks)
                          ListTile(
                            title: Text(
                              item.username.isEmpty
                                  ? '#${item.targetUserId}'
                                  : item.username,
                            ),
                            trailing: UserBlockButton(
                              userId: item.targetUserId,
                            ),
                          ),
                      ],
                    ),
            ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.commonClose),
        ),
      ],
    );
  }
}
