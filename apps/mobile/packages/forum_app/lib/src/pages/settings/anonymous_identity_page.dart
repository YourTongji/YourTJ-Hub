import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
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

/// The composer owns returned profile navigation, so its input stays unfocused.
Future<({AnonymousPersona persona, bool openProfile})?>
showAnonymousIdentitySheet(BuildContext context) =>
    showGfBottomSheet<({AnonymousPersona persona, bool openProfile})>(
      context,
      height: MediaQuery.sizeOf(context).height * .9,
      keyboardAware: true,
      barrierDismissible: false,
      enableDrag: false,
      showDragHandle: true,
      builder: (_) => const _AnonymousIdentityContent(sheet: true),
    );

class AnonymousIdentityPage extends StatelessWidget {
  const AnonymousIdentityPage({super.key});
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: GfAppBar(
      title: Text(AppLocalizations.of(context).anonymousIdentity),
    ),
    body: const SafeArea(top: false, child: _AnonymousIdentityContent()),
  );
}

class _AnonymousIdentityContent extends ConsumerStatefulWidget {
  const _AnonymousIdentityContent({this.sheet = false});
  final bool sheet;
  @override
  ConsumerState<_AnonymousIdentityContent> createState() =>
      _AnonymousIdentityContentState();
}

class _AnonymousIdentityContentState
    extends ConsumerState<_AnonymousIdentityContent> {
  bool busy = false;
  Object? error;
  String? pendingKey, pendingDay, activeBatchId;
  bool historyOpen = false;
  ({String batchId, int index, String word})? choice;
  late final int epoch;
  AnonymousIdentityRepository get repo =>
      AnonymousIdentityRepository(ref.read(apiClientProvider));
  @override
  void initState() {
    super.initState();
    epoch = ref.read(offlineCacheEpochProvider);
  }

  bool get current => mounted && epoch == ref.read(offlineCacheEpochProvider);
  String date(DateTime value) => DateFormat.yMMMd(
    AppLocalizations.of(context).localeName,
  ).add_Hm().format(value.toLocal());

  Future<void> run(
    Future<void> Function() action, {
    bool refresh = true,
  }) async {
    if (busy || !current) return;
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await action();
      if (current && refresh) ref.invalidate(anonymousIdentityProvider);
    } catch (e) {
      if (current) setState(() => error = e);
    } finally {
      if (current) setState(() => busy = false);
    }
  }

  Future<void> draw(AnonymousIdentityState state) => run(() async {
    if (pendingDay != state.day) {
      pendingKey = null;
      pendingDay = state.day;
    }
    pendingKey ??= newTopicDraftKey();
    final batch = await repo.generate(state.day, pendingKey!);
    if (current) {
      pendingKey = null;
      activeBatchId = batch.id;
      choice = null;
      historyOpen = false;
    }
  });

  Future<void> confirm() async {
    final selected = choice;
    if (selected == null) return;
    await run(() async {
      final persona = await repo.confirm(selected.batchId, selected.index);
      if (!mounted || !current) return;
      choice = null;
      if (widget.sheet) {
        ref.invalidate(anonymousIdentityProvider);
        Navigator.of(context).pop((persona: persona, openProfile: false));
      }
    }, refresh: !widget.sheet);
  }

  void close() {
    if (!busy) Navigator.of(context).maybePop();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      if (next != epoch && mounted) {
        if (widget.sheet) {
          Navigator.of(context).pop();
        } else {
          GoRouter.maybeOf(context)?.go('/');
        }
      }
    });
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final state = ref.watch(anonymousIdentityProvider);
    if (!current) return const SizedBox.shrink();
    final s = state.valueOrNull;
    final canChoose =
        s != null && !s.locked && !s.disabled && !s.governanceDisabled;
    final batch = s == null || s.batches.isEmpty
        ? null
        : s.batches.where((b) => b.id == activeBatchId).firstOrNull ??
              s.batches.last;
    // Midnight invalidates yesterday's preview as well as its request key.
    final selected = choice?.batchId == batch?.id ? choice : null;
    final muted = colors.baseContent.withValues(alpha: .65);
    return PopScope(
      canPop: !busy,
      child: LayoutBuilder(
        builder: (context, box) {
          // Short sheets (large text, open keyboard) keep only the button
          // pinned; the lock hint moves into the scrolling body.
          final compact = box.maxHeight < 560;
          return Column(
            children: [
              if (widget.sheet) ...[
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 12, 12, 12),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Text(
                          s?.persona == null
                              ? l.anonymousSetup
                              : l.anonymousManage,
                          style: type.title3,
                        ),
                      ),
                      GfIconButton(
                        symbol: 'x',
                        tooltip: l.commonClose,
                        onPressed: busy ? null : close,
                      ),
                    ],
                  ),
                ),
                const GfDivider(),
              ],
              Expanded(
                child: s == null
                    ? Center(
                        child: state.hasError
                            ? Padding(
                                padding: const EdgeInsets.all(20),
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    Text(
                                      resolveErrorMessage(l, state.error!),
                                      textAlign: TextAlign.center,
                                      style: type.body,
                                    ),
                                    const SizedBox(height: 16),
                                    GfButton(
                                      label: l.anonymousRetry,
                                      variant: GfButtonVariant.secondary,
                                      onPressed: () => ref.invalidate(
                                        anonymousIdentityProvider,
                                      ),
                                    ),
                                  ],
                                ),
                              )
                            : const CircularProgressIndicator(),
                      )
                    : ListView(
                        padding: const EdgeInsets.all(20),
                        children: [
                          Text(
                            l.anonymousPurpose,
                            style: type.body.copyWith(color: muted),
                          ),
                          const SizedBox(height: 20),
                          if (error != null || state.hasError) ...[
                            Semantics(
                              liveRegion: true,
                              child: Text(
                                resolveErrorMessage(l, error ?? state.error!),
                                style: type.body.copyWith(color: colors.error),
                              ),
                            ),
                            GfButton(
                              label: l.anonymousRetry,
                              variant: GfButtonVariant.ghost,
                              onPressed: busy
                                  ? null
                                  : () => ref.invalidate(
                                      anonymousIdentityProvider,
                                    ),
                            ),
                            const SizedBox(height: 16),
                          ],
                          if (s.persona case final p?) ...[
                            // One inset group keeps the persona and its
                            // switches on a shared edge.
                            Material(
                              key: const ValueKey('anonymous-persona-card'),
                              color: colors.base200.withValues(alpha: .6),
                              borderRadius: BorderRadius.circular(20),
                              clipBehavior: Clip.antiAlias,
                              child: Column(
                                children: [
                                  GfSettingRow(
                                    leading: GfAvatar(
                                      src: resolveApiAssetUrl(p.avatarUrl),
                                      size: 44,
                                    ),
                                    title: p.name,
                                    description: s.governanceDisabled
                                        ? l.anonymousRestricted
                                        : s.disabled
                                        ? l.anonymousInactive
                                        : l.anonymousReady,
                                    trailing: const GfSymbol(
                                      'chevron-right',
                                      size: 18,
                                    ),
                                    onTap: busy
                                        ? null
                                        : () {
                                            if (widget.sheet) {
                                              Navigator.of(context).pop((
                                                persona: p,
                                                openProfile: true,
                                              ));
                                            } else {
                                              GoRouter.maybeOf(
                                                context,
                                              )?.push(p.profileUrl);
                                            }
                                          },
                                  ),
                                  const GfDivider(inset: 16),
                                  GfSwitchRow(
                                    title: l.anonymousShowContent,
                                    description:
                                        l.anonymousShowContentDescription,
                                    value: s.showContent,
                                    onChanged: (value) {
                                      if (!busy) {
                                        run(
                                          () => repo.setProfileContent(value),
                                        );
                                      }
                                    },
                                  ),
                                  const GfDivider(inset: 16),
                                  if (s.governanceDisabled)
                                    GfSettingRow(
                                      title: l.anonymousActiveLabel,
                                      subtitleWidget: Text(
                                        l.anonymousStatusRestricted,
                                        style: type.caption.copyWith(
                                          color: colors.error,
                                        ),
                                      ),
                                    )
                                  else
                                    GfSwitchRow(
                                      title: l.anonymousActiveLabel,
                                      description: l.anonymousDisabledHint,
                                      value: !s.disabled,
                                      onChanged: (active) {
                                        if (!busy) {
                                          run(() => repo.setDisabled(!active));
                                        }
                                      },
                                    ),
                                ],
                              ),
                            ),
                            if (s.locked) ...[
                              const SizedBox(height: 8),
                              Padding(
                                padding: const EdgeInsetsDirectional.only(
                                  start: 16,
                                ),
                                child: Text(
                                  l.anonymousLockedUntil(date(s.availableAt!)),
                                  style: type.caption.copyWith(color: muted),
                                ),
                              ),
                            ],
                            const SizedBox(height: 24),
                          ],
                          if (canChoose) ...[
                            _NameStage(
                              caption: l.anonymousPreviewCaption,
                              name: selected?.word,
                              placeholder: l.anonymousPreviewPlaceholder,
                              tag: l.anonymousTag,
                            ),
                            const SizedBox(height: 16),
                            if (batch == null) ...[
                              Text(
                                l.anonymousIntroTitle,
                                style: type.bodyStrong,
                              ),
                              const SizedBox(height: 4),
                              Text(
                                l.anonymousIntroBody,
                                style: type.body.copyWith(color: muted),
                              ),
                            ] else ...[
                              Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      l.anonymousChooseName,
                                      style: type.bodyStrong,
                                    ),
                                  ),
                                  if (s.batches.length > 1)
                                    Flexible(
                                      child: _HistoryToggle(
                                        label: l.anonymousBatchOf(
                                          s.batches.indexOf(batch) + 1,
                                          s.batches.length,
                                        ),
                                        semanticsLabel:
                                            l.anonymousPreviousBatches,
                                        open: historyOpen,
                                        onPressed: busy
                                            ? null
                                            : () => setState(
                                                () =>
                                                    historyOpen = !historyOpen,
                                              ),
                                      ),
                                    ),
                                ],
                              ),
                              AnimatedSize(
                                duration:
                                    MediaQuery.disableAnimationsOf(context)
                                    ? Duration.zero
                                    : const Duration(milliseconds: 180),
                                curve: Curves.easeOut,
                                alignment: Alignment.topCenter,
                                child: historyOpen && s.batches.length > 1
                                    ? Padding(
                                        padding: const EdgeInsets.only(top: 8),
                                        child: _HistoryMenu(
                                          batches: s.batches,
                                          activeId: batch.id,
                                          title: l.anonymousPreviousBatches,
                                          numberLabel: l.anonymousBatchNumber,
                                          onSelected: (id) => setState(() {
                                            historyOpen = false;
                                            if (id == activeBatchId) return;
                                            activeBatchId = id;
                                            choice = null;
                                          }),
                                        ),
                                      )
                                    : const SizedBox(width: double.infinity),
                              ),
                              const SizedBox(height: 12),
                              // One fade per batch keeps a new draw noticeable
                              // without animating every option.
                              AnimatedSwitcher(
                                duration:
                                    MediaQuery.disableAnimationsOf(context)
                                    ? Duration.zero
                                    : const Duration(milliseconds: 200),
                                switchInCurve: const Cubic(.2, 0, 0, 1),
                                switchOutCurve: Curves.easeOut,
                                child: LayoutBuilder(
                                  key: ValueKey(batch.id),
                                  builder: (context, constraints) => Wrap(
                                    spacing: 8,
                                    runSpacing: 8,
                                    children: [
                                      for (
                                        var i = 0;
                                        i < batch.words.length;
                                        i++
                                      )
                                        SizedBox(
                                          width: (constraints.maxWidth - 8) / 2,
                                          child: _NameOption(
                                            word: batch.words[i],
                                            selected: selected?.index == i,
                                            onTap: busy
                                                ? null
                                                : () => setState(
                                                    () => choice = (
                                                      batchId: batch.id,
                                                      index: i,
                                                      word: batch.words[i],
                                                    ),
                                                  ),
                                          ),
                                        ),
                                    ],
                                  ),
                                ),
                              ),
                              const SizedBox(height: 12),
                              Text(
                                l.anonymousDrawsRemaining(s.remaining),
                                style: type.caption.copyWith(color: muted),
                              ),
                            ],
                            if (compact && batch != null) ...[
                              const SizedBox(height: 4),
                              Text(
                                l.anonymousNameLockHint,
                                style: type.caption.copyWith(color: muted),
                              ),
                            ],
                            if (s.remaining == 0) ...[
                              const SizedBox(height: 8),
                              Text(
                                l.anonymousQuota(date(s.resetsAt)),
                                style: type.caption.copyWith(color: muted),
                              ),
                            ],
                            const SizedBox(height: 24),
                          ],
                          const GfDivider(),
                          const SizedBox(height: 16),
                          Text(
                            l.anonymousPrivacySummary,
                            style: type.caption.copyWith(color: muted),
                          ),
                          ExpansionTile(
                            tilePadding: EdgeInsets.zero,
                            childrenPadding: const EdgeInsets.only(bottom: 12),
                            title: Text(l.anonymousRules, style: type.caption),
                            children: [
                              Text(l.anonymousBoundary, style: type.caption),
                              const SizedBox(height: 8),
                              Text(
                                l.anonymousLinkabilityHint,
                                style: type.caption,
                              ),
                            ],
                          ),
                        ],
                      ),
              ),
              if (canChoose)
                Container(
                  key: const ValueKey('anonymous-confirmation-footer'),
                  padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
                  decoration: BoxDecoration(
                    color: colors.base100,
                    border: Border(top: BorderSide(color: colors.line)),
                  ),
                  child: batch == null
                      ? GfButton(
                          label: l.anonymousGenerateNames,
                          icon: const GfSymbol('refresh-cw', size: 18),
                          expanded: true,
                          loading: busy,
                          onPressed: busy || s.remaining == 0
                              ? null
                              : () => draw(s),
                        )
                      : Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            if (!compact) ...[
                              Text(
                                l.anonymousNameLockHint,
                                style: type.caption.copyWith(color: muted),
                              ),
                              const SizedBox(height: 10),
                            ],
                            _FooterActions(
                              refresh: GfButton(
                                label: l.anonymousRefreshNames,
                                icon: const GfSymbol('refresh-cw', size: 18),
                                variant: GfButtonVariant.secondary,
                                onPressed: busy || s.remaining == 0
                                    ? null
                                    : () => draw(s),
                              ),
                              confirm: GfButton(
                                label: l.anonymousConfirmName,
                                expanded: true,
                                loading: busy,
                                onPressed: selected == null || busy
                                    ? null
                                    : confirm,
                              ),
                            ),
                          ],
                        ),
                ),
            ],
          );
        },
      ),
    );
  }
}

/// How the chosen name will read next to a post: mask avatar, name, tag.
class _NameStage extends StatelessWidget {
  const _NameStage({
    required this.caption,
    required this.name,
    required this.placeholder,
    required this.tag,
  });
  final String caption, placeholder, tag;
  final String? name;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(18),
        gradient: LinearGradient(
          begin: Alignment.topRight,
          end: Alignment.bottomLeft,
          colors: [
            Color.alphaBlend(
              colors.primary.withValues(alpha: .12),
              colors.base200,
            ),
            colors.base200,
          ],
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            caption,
            style: type.caption.copyWith(
              color: colors.baseContent.withValues(alpha: .7),
            ),
          ),
          const SizedBox(height: 8),
          Semantics(
            liveRegion: true,
            child: Row(
              children: [
                Container(
                  width: 36,
                  height: 36,
                  decoration: BoxDecoration(
                    color: colors.base100,
                    shape: BoxShape.circle,
                  ),
                  alignment: Alignment.center,
                  child: GfSymbol('eye-off', size: 16, color: colors.iconMuted),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      AnimatedSwitcher(
                        duration: MediaQuery.disableAnimationsOf(context)
                            ? Duration.zero
                            : const Duration(milliseconds: 150),
                        child: Text(
                          name ?? placeholder,
                          key: ValueKey(name),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: type.title3.copyWith(
                            color: name == null
                                ? colors.baseContent.withValues(alpha: .6)
                                : colors.baseContent,
                          ),
                        ),
                      ),
                      const SizedBox(height: 4),
                      GfBadge(label: tag, variant: GfBadgeVariant.muted),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Draw again beside confirm; stacks when large text leaves no room.
class _FooterActions extends StatelessWidget {
  const _FooterActions({required this.refresh, required this.confirm});
  final Widget refresh, confirm;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, box) {
      final roomy =
          box.maxWidth >= 300 &&
          MediaQuery.textScalerOf(context).scale(14) <= 18;
      if (!roomy) {
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [confirm, const SizedBox(height: 8), refresh],
        );
      }
      return Row(
        children: [
          refresh,
          const SizedBox(width: 12),
          Expanded(child: confirm),
        ],
      );
    },
  );
}

/// One drawn name. Selection shows as a primary outline plus a check, so it
/// never relies on color alone.
class _NameOption extends StatelessWidget {
  const _NameOption({
    required this.word,
    required this.selected,
    required this.onTap,
  });
  final String word;
  final bool selected;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return Semantics(
      button: true,
      selected: selected,
      enabled: onTap != null,
      label: word,
      excludeSemantics: true,
      onTap: onTap,
      child: Material(
        animationDuration: GfMotion.duration(context, GfMotion.selection),
        color: selected
            ? Color.alphaBlend(
                colors.primary.withValues(alpha: .08),
                colors.base100,
              )
            : colors.base100,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: BorderSide(
            color: selected ? colors.primary : colors.line,
            width: selected ? 1.5 : 1,
          ),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: onTap,
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 52),
            child: Padding(
              padding: const EdgeInsetsDirectional.fromSTEB(14, 12, 10, 12),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      word,
                      style: (selected ? type.bodyStrong : type.body).copyWith(
                        color: colors.baseContent,
                      ),
                    ),
                  ),
                  const SizedBox(width: 6),
                  AnimatedOpacity(
                    opacity: selected ? 1 : 0,
                    duration: GfMotion.duration(context, GfMotion.selection),
                    child: GfSymbol('check', size: 16, color: colors.primary),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _HistoryToggle extends StatelessWidget {
  const _HistoryToggle({
    required this.label,
    required this.semanticsLabel,
    required this.open,
    required this.onPressed,
  });
  final String label, semanticsLabel;
  final bool open;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return Semantics(
      button: true,
      expanded: open,
      label: '$semanticsLabel: $label',
      excludeSemantics: true,
      onTap: onPressed,
      child: Material(
        color: colors.base200,
        shape: const StadiumBorder(),
        child: InkWell(
          customBorder: const StadiumBorder(),
          onTap: onPressed,
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 40),
            child: Padding(
              padding: const EdgeInsetsDirectional.fromSTEB(12, 6, 8, 6),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  GfSymbol('history', size: 14, color: colors.iconMuted),
                  const SizedBox(width: 6),
                  Flexible(
                    child: Text(
                      label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: type.caption,
                    ),
                  ),
                  const SizedBox(width: 2),
                  AnimatedRotation(
                    turns: open ? .5 : 0,
                    duration: const Duration(milliseconds: 150),
                    child: GfSymbol(
                      'chevron-down',
                      size: 14,
                      color: colors.iconMuted,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Every generated batch, newest first, scrolling inside a capped menu.
class _HistoryMenu extends StatelessWidget {
  const _HistoryMenu({
    required this.batches,
    required this.activeId,
    required this.title,
    required this.numberLabel,
    required this.onSelected,
  });
  final List<AnonymousNameBatch> batches;
  final String activeId, title;
  final String Function(int number) numberLabel;
  final ValueChanged<String> onSelected;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return GfMenuSurface(
      key: const ValueKey('anonymous-batch-history'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(10, 8, 10, 4),
            child: Text(
              title,
              style: type.caption.copyWith(
                color: colors.baseContent.withValues(alpha: .65),
              ),
            ),
          ),
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 240),
            child: ListView.builder(
              shrinkWrap: true,
              primary: false,
              itemCount: batches.length,
              itemBuilder: (context, row) {
                final index = batches.length - 1 - row;
                final batch = batches[index];
                final active = batch.id == activeId;
                return Semantics(
                  selected: active,
                  button: true,
                  child: InkWell(
                    borderRadius: BorderRadius.circular(12),
                    onTap: () => onSelected(batch.id),
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(minHeight: 48),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 10,
                          vertical: 8,
                        ),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            SizedBox(
                              width: 56,
                              child: Text(
                                numberLabel(index + 1),
                                style: type.caption.copyWith(
                                  fontWeight: FontWeight.w600,
                                  color: active
                                      ? colors.primary
                                      : colors.baseContent.withValues(
                                          alpha: .65,
                                        ),
                                ),
                              ),
                            ),
                            Expanded(
                              child: Text(
                                batch.words.join(' · '),
                                style: type.body.copyWith(
                                  color: colors.baseContent.withValues(
                                    alpha: .85,
                                  ),
                                ),
                              ),
                            ),
                            if (active)
                              GfSymbol(
                                'check',
                                size: 16,
                                color: colors.primary,
                              ),
                          ],
                        ),
                      ),
                    ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
