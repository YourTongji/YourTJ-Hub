import 'user_decoration.dart';
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
    this.compact = false,
    this.onMenuChanged,
  });
  final String value;
  final ValueChanged<String> onChanged;
  final bool disabled;

  /// Avatar-only 44px button beside an input. A small swap badge in the
  /// primary color marks it as switchable; the name stays in the tooltip and
  /// semantics label.
  final bool compact;

  /// Reports `true` while the choice sheet, or the setup sheet it leads to,
  /// holds focus and `false` once the flow ends, so a host can keep its
  /// keyboard space in the meantime.
  final ValueChanged<bool>? onMenuChanged;
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
    final onMenuChanged = widget.onMenuChanged;
    onMenuChanged?.call(true);
    try {
      final value = await showGfBottomSheet<String>(
        context,
        showDragHandle: true,
        builder: (_) => _IdentityMenu(selected: widget.value),
      );
      if (value != null && mounted) await choose(value);
    } finally {
      onMenuChanged?.call(false);
    }
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
    final avatar = GfAvatar(
      frameColors: widget.value == 'persona'
          ? const []
          : decorationColors(context, user?.decoration?.avatarFrame),
      frameAssetUrl: widget.value == 'persona'
          ? ''
          : decorationFrameAssetUrl(user?.decoration?.avatarFrame),
      frameScale: user?.decoration?.avatarFrame?.assetScale ?? 1.25,
      src: resolveApiAssetUrl(
        widget.value == 'persona' ? p?.avatarUrl ?? '' : user?.avatarUrl ?? '',
      ),
      size: widget.compact ? 32 : 28,
    );
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Flexible(
          child: Semantics(
            button: true,
            enabled: !widget.disabled,
            label: '${l.anonymousPublishAs}: $name · $kind',
            excludeSemantics: true,
            onTap: widget.disabled ? null : openMenu,
            child: Tooltip(
              message: widget.compact ? '$name · $kind' : name,
              child: InkWell(
                key: const Key('identity-picker'),
                customBorder: widget.compact
                    ? const CircleBorder()
                    : const StadiumBorder(),
                onTap: widget.disabled ? null : openMenu,
                child: widget.compact
                    ? SizedBox.square(
                        dimension: 44,
                        child: Center(
                          child: Stack(
                            clipBehavior: Clip.none,
                            children: [
                              AnimatedSwitcher(
                                duration: GfMotion.duration(
                                  context,
                                  GfMotion.selection,
                                ),
                                switchInCurve: GfMotion.enterCurve,
                                switchOutCurve: GfMotion.enterCurve,
                                transitionBuilder: (child, animation) =>
                                    FadeTransition(
                                      opacity: animation,
                                      child: ScaleTransition(
                                        scale: Tween<double>(
                                          begin: .25,
                                          end: 1,
                                        ).animate(animation),
                                        child: child,
                                      ),
                                    ),
                                child: KeyedSubtree(
                                  key: ValueKey<String>(widget.value),
                                  child: avatar,
                                ),
                              ),
                              if (!widget.disabled)
                                PositionedDirectional(
                                  end: -3,
                                  bottom: -2,
                                  child: Container(
                                    key: const Key('identity-switch-badge'),
                                    width: 16,
                                    height: 16,
                                    // Concentric with the glyph's own square.
                                    decoration: BoxDecoration(
                                      color: colors.base100,
                                      borderRadius: BorderRadius.circular(5),
                                    ),
                                    alignment: Alignment.center,
                                    child: GfSymbol(
                                      'arrow-swap-horizontal-filled',
                                      size: 14,
                                      color: colors.primary,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                        ),
                      )
                    : Container(
                        constraints: const BoxConstraints(minHeight: 48),
                        padding: const EdgeInsetsDirectional.fromSTEB(
                          6,
                          4,
                          10,
                          4,
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            avatar,
                            const SizedBox(width: 8),
                            Flexible(
                              child: GfDecoratedName(
                                name,
                                colors: widget.value == 'persona'
                                    ? const []
                                    : decorationColors(
                                        context,
                                        user?.decoration?.nicknameStyle,
                                      ),
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
                                child: GfText(
                                  kind,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: type.caption.copyWith(
                                    color: colors.baseContent.withValues(
                                      alpha: .7,
                                    ),
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
        subtitleWidget: GfText(
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
              child: GfText(l.anonymousPublishAs, style: type.title3),
            ),
            option(
              key: const Key('identity-option-member'),
              leading: GfAvatar(
                frameColors: decorationColors(
                  context,
                  user?.decoration?.avatarFrame,
                ),
                frameAssetUrl: decorationFrameAssetUrl(
                  user?.decoration?.avatarFrame,
                ),
                frameScale: user?.decoration?.avatarFrame?.assetScale ?? 1.25,
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
