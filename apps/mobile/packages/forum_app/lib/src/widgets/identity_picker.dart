import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../pages/settings/anonymous_identity_page.dart';
import '../current_user.dart';
import '../providers.dart';
import '../asset_url.dart';

final composerMemberProvider = FutureProvider.autoDispose<UserCardPayload?>((
  ref,
) async {
  ref.watch(offlineCacheEpochProvider);
  final user = await ref.watch(currentUserProvider.future);
  if (user == null) return null;
  return ref.watch(userRepositoryProvider).getUserCard(user.id);
});

/// A choice belongs to this composer; errors never switch a persona to member.
class IdentityPicker extends ConsumerWidget {
  const IdentityPicker({
    super.key,
    required this.value,
    required this.onChanged,
    this.disabled = false,
  });
  final String value;
  final ValueChanged<String> onChanged;
  final bool disabled;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(anonymousIdentityProvider);
    final p = state.valueOrNull?.persona;
    final user = ref.watch(composerMemberProvider).valueOrNull;
    final name = value == 'persona'
        ? (p?.name ?? l.anonymousUnavailable)
        : (user?.nickname.isNotEmpty == true
              ? user!.nickname
              : user?.username ?? l.anonymousMember);
    return Row(
      children: [
        GfAvatar(
          src: resolveApiAssetUrl(
            value == 'persona' ? p?.avatarUrl ?? '' : user?.avatarUrl ?? '',
          ),
          size: 28,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Tooltip(
            message: name,
            child: Text(
              '${l.anonymousPublishAs} · ${value == 'persona' ? l.anonymousPersonaLabel : l.anonymousMember}: $name',
              // Unrestricted THUOCL words must not consume the reply viewport.
              // Text semantics and the tooltip retain the complete chosen name.
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ),
        if (state.hasError)
          IconButton(
            tooltip: l.anonymousRetry,
            onPressed: () => ref.invalidate(anonymousIdentityProvider),
            icon: const Icon(Icons.refresh),
          ),
        if (!disabled)
          PopupMenuButton<String>(
            tooltip: l.anonymousPublishAs,
            onSelected: (choice) {
              if (choice == 'setup') {
                context.push('/settings/anonymous-identity');
              } else {
                onChanged(choice);
              }
            },
            itemBuilder: (_) => [
              CheckedPopupMenuItem(
                value: 'member',
                checked: value == 'member',
                child: Text(l.anonymousMember),
              ),
              CheckedPopupMenuItem(
                value: 'persona',
                checked: value == 'persona',
                enabled: state.valueOrNull?.usable == true,
                child: Text(l.anonymousPersonaLabel),
              ),
              PopupMenuItem(value: 'setup', child: Text(l.anonymousSetup)),
            ],
          ),
      ],
    );
  }
}
