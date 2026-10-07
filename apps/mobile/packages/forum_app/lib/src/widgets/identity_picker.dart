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
    final result = await showAnonymousIdentitySheet(context);
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    if (result?.openProfile == true) {
      // Let the sheet restore its route scope before clearing its edit focus.
      await WidgetsBinding.instance.endOfFrame;
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      FocusScope.of(context).unfocus();
      GoRouter.maybeOf(context)?.push(result!.persona.profileUrl);
      return;
    }
    if (!widget.disabled && result != null) widget.onChanged('persona');
    if (previousFocus?.context != null) previousFocus!.requestFocus();
  }

  Future<void> openMenu() async {
    if (widget.disabled) return;
    final value = await showGfBottomSheet<String>(
      context,
      showDragHandle: true,
      builder: (_) => _IdentityMenu(selected: widget.value),
    );
    if (value != null && mounted) await choose(value);
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
    final kind = widget.value == 'persona' ? l.anonymousTag : l.anonymousMember;
    return Row(
      children: [
        Flexible(
          child: Semantics(
            button: true,
            enabled: !widget.disabled,
            label: '${l.anonymousPublishAs}: $name · $kind',
            excludeSemantics: true,
            child: Tooltip(
              message: name,
              child: InkWell(
                key: const Key('identity-picker'),
                customBorder: const StadiumBorder(),
                onTap: widget.disabled ? null : openMenu,
                child: Container(
                  constraints: const BoxConstraints(minHeight: 48),
                  padding: const EdgeInsetsDirectional.fromSTEB(6, 4, 10, 4),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
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
                      Flexible(
                        child: Text(
                          name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: type.bodyStrong,
                        ),
                      ),
                      const SizedBox(width: 6),
                      Flexible(
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 6,
                            vertical: 2,
                          ),
                          decoration: ShapeDecoration(
                            color: colors.base200,
                            shape: const StadiumBorder(),
                          ),
                          child: Text(
                            kind,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: type.caption.copyWith(
                              color: colors.baseContent.withValues(alpha: .7),
                            ),
                          ),
                        ),
                      ),
                      if (!widget.disabled) ...[
                        const SizedBox(width: 4),
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

/// Bottom-sheet choice between the member account and the persona; returns
/// `member`, `persona` or `setup`.
class _IdentityMenu extends ConsumerWidget {
  const _IdentityMenu({required this.selected});
  final String selected;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final state = ref.watch(anonymousIdentityProvider).valueOrNull;
    final p = state?.persona;
    final user = ref.watch(composerMemberProvider).valueOrNull;
    void pick(String value) => Navigator.of(context).pop(value);
    Widget option({
      required Key key,
      required Widget leading,
      required String title,
      required String subtitle,
      required bool checked,
      Color? subtitleColor,
      VoidCallback? onTap,
    }) => Semantics(
      selected: checked,
      child: GfSettingRow(
        key: key,
        leading: leading,
        title: title,
        subtitleWidget: Text(
          subtitle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: type.caption.copyWith(
            color: subtitleColor ?? colors.baseContent.withValues(alpha: .65),
          ),
        ),
        trailing: checked
            ? GfSymbol('check', size: 18, color: colors.primary)
            : null,
        onTap: onTap,
      ),
    );
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(8, 0, 8, 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
              child: Text(l.anonymousPublishAs, style: type.title3),
            ),
            option(
              key: const Key('identity-option-member'),
              leading: GfAvatar(
                src: resolveApiAssetUrl(user?.avatarUrl ?? ''),
                size: 36,
              ),
              title: l.anonymousMember,
              subtitle: user == null ? '' : '@${user.username}',
              checked: selected == 'member',
              onTap: () => pick('member'),
            ),
            if (p == null)
              option(
                key: const Key('identity-option-setup'),
                leading: const GfSymbol('eye-off', size: 22),
                title: l.anonymousIdentity,
                subtitle: l.anonymousSetup,
                subtitleColor: colors.primary,
                checked: false,
                onTap: () => pick('setup'),
              )
            else
              option(
                key: const Key('identity-option-persona'),
                leading: GfAvatar(
                  src: resolveApiAssetUrl(p.avatarUrl),
                  size: 36,
                ),
                title: l.anonymousIdentity,
                subtitle: state!.governanceDisabled
                    ? l.anonymousStatusRestricted
                    : state.disabled
                    ? l.anonymousStatusDisabled
                    : p.name,
                subtitleColor: state.usable ? null : colors.warning,
                checked: selected == 'persona',
                onTap: state.usable ? () => pick('persona') : null,
              ),
            if (p != null) ...[
              const GfDivider(),
              GfSettingRow(
                key: const Key('identity-option-manage'),
                symbol: 'settings',
                title: l.anonymousManage,
                onTap: () => pick('setup'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
