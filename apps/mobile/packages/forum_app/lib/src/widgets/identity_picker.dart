import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
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
class IdentityPicker extends ConsumerStatefulWidget {
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
  ConsumerState<IdentityPicker> createState() => _IdentityPickerState();
}

class _IdentityPickerState extends ConsumerState<IdentityPicker> {
  Future<void> choose(String choice) async {
    if (widget.disabled) return;
    if (choice != 'setup') {
      widget.onChanged(choice);
      return;
    }
    final epoch = ref.read(offlineCacheEpochProvider);
    final previousFocus = FocusManager.instance.primaryFocus;
    previousFocus?.unfocus();
    final persona = await showAnonymousIdentitySheet(context);
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    if (!widget.disabled && persona != null) widget.onChanged('persona');
    if (previousFocus?.context != null) previousFocus!.requestFocus();
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final state = ref.watch(anonymousIdentityProvider);
    final p = state.valueOrNull?.persona;
    final user = ref.watch(composerMemberProvider).valueOrNull;
    final name = widget.value == 'persona'
        ? p?.name ?? l.anonymousUnavailable
        : user?.nickname.isNotEmpty == true
        ? user!.nickname
        : user?.username ?? l.anonymousMember;
    final kind = widget.value == 'persona'
        ? l.anonymousPersonaLabel
        : l.anonymousMember;
    return Row(
      children: [
        Expanded(
          child: PopupMenuButton<String>(
            tooltip: l.anonymousPublishAs,
            enabled: !widget.disabled,
            onSelected: choose,
            itemBuilder: (_) => [
              CheckedPopupMenuItem(
                value: 'member',
                checked: widget.value == 'member',
                child: Text(l.anonymousMember),
              ),
              if (p != null)
                CheckedPopupMenuItem(
                  value: 'persona',
                  checked: widget.value == 'persona',
                  enabled: state.valueOrNull?.usable == true,
                  child: Text(l.anonymousPersonaLabel),
                ),
              PopupMenuItem(
                value: 'setup',
                child: Text(p == null ? l.anonymousSetup : l.anonymousManage),
              ),
            ],
            child: Semantics(
              label: '${l.anonymousPublishAs}: $name · $kind',
              child: Tooltip(
                message: name,
                child: Container(
                  constraints: const BoxConstraints(minHeight: 48),
                  padding: const EdgeInsets.symmetric(horizontal: 8),
                  child: Row(
                    children: [
                      GfAvatar(
                        src: resolveApiAssetUrl(
                          widget.value == 'persona'
                              ? p?.avatarUrl ?? ''
                              : user?.avatarUrl ?? '',
                        ),
                        size: 28,
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: type.bodyStrong,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Flexible(
                        child: Text(
                          kind,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: type.caption.copyWith(
                            color: colors.baseContent.withValues(alpha: .65),
                          ),
                        ),
                      ),
                      if (!widget.disabled) ...[
                        const SizedBox(width: 8),
                        GfSymbol(
                          'chevron-down',
                          size: 16,
                          color: colors.iconMuted,
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
        if (state.hasError)
          GfIconButton(
            symbol: 'refresh-cw',
            tooltip: l.anonymousRetry,
            onPressed: () => ref.invalidate(anonymousIdentityProvider),
          ),
      ],
    );
  }
}
