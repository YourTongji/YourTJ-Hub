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
    appBar: AppBar(title: Text(AppLocalizations.of(context).anonymousIdentity)),
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
    return PopScope(
      canPop: !busy,
      child: Column(
        children: [
          if (widget.sheet) ...[
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 12, 12, 12),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Text(l.anonymousIdentity, style: type.title3),
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
                                  onPressed: () =>
                                      ref.invalidate(anonymousIdentityProvider),
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
                        style: type.body.copyWith(
                          color: colors.baseContent.withValues(alpha: .65),
                        ),
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
                          variant: GfButtonVariant.link,
                          onPressed: busy
                              ? null
                              : () => ref.invalidate(anonymousIdentityProvider),
                        ),
                        const SizedBox(height: 16),
                      ],
                      if (s.persona case final p?) ...[
                        GfSettingRow(
                          leading: GfAvatar(
                            src: resolveApiAssetUrl(p.avatarUrl),
                            size: 44,
                          ),
                          title: p.name,
                          description: s.governanceDisabled
                              ? l.anonymousUnavailable
                              : s.disabled
                              ? l.anonymousInactive
                              : l.anonymousReady,
                          trailing: const GfSymbol('chevron-right', size: 18),
                          onTap: busy
                              ? null
                              : () {
                                  if (widget.sheet) {
                                    Navigator.of(
                                      context,
                                    ).pop((persona: p, openProfile: true));
                                  } else {
                                    GoRouter.maybeOf(
                                      context,
                                    )?.push(p.profileUrl);
                                  }
                                },
                        ),
                        if (s.locked) ...[
                          const SizedBox(height: 12),
                          Text(
                            l.anonymousLockedUntil(date(s.availableAt!)),
                            style: type.caption.copyWith(
                              color: colors.baseContent.withValues(alpha: .65),
                            ),
                          ),
                        ],
                        const SizedBox(height: 16),
                        GfSettingRow(
                          title: l.anonymousShowContent,
                          description: l.anonymousShowContentDescription,
                          trailing: Switch.adaptive(
                            value: s.showContent,
                            onChanged: busy
                                ? null
                                : (value) =>
                                      run(() => repo.setProfileContent(value)),
                          ),
                        ),
                        const GfDivider(),
                        if (!s.governanceDisabled) ...[
                          const SizedBox(height: 16),
                          GfButton(
                            label: s.disabled
                                ? l.anonymousEnable
                                : l.anonymousDisable,
                            variant: GfButtonVariant.secondary,
                            loading: busy,
                            onPressed: busy
                                ? null
                                : () =>
                                      run(() => repo.setDisabled(!s.disabled)),
                          ),
                        ],
                        const SizedBox(height: 24),
                      ],
                      if (s.governanceDisabled) ...[
                        Text(
                          l.anonymousUnavailable,
                          style: type.body.copyWith(color: colors.error),
                        ),
                        const SizedBox(height: 16),
                      ],
                      if (canChoose) ...[
                        if (batch == null)
                          Container(
                            padding: const EdgeInsets.all(20),
                            decoration: BoxDecoration(
                              color: colors.base200,
                              borderRadius: BorderRadius.circular(16),
                            ),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                GfSymbol(
                                  'eye-off',
                                  size: 28,
                                  color: colors.baseContent.withValues(
                                    alpha: .65,
                                  ),
                                ),
                                const SizedBox(height: 16),
                                Text(
                                  l.anonymousIntroTitle,
                                  style: type.bodyStrong,
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  l.anonymousHistoryHint,
                                  style: type.body.copyWith(
                                    color: colors.baseContent.withValues(
                                      alpha: .65,
                                    ),
                                  ),
                                ),
                              ],
                            ),
                          )
                        else ...[
                          Row(
                            children: [
                              Expanded(
                                child: Text(
                                  l.anonymousChooseName,
                                  style: type.bodyStrong,
                                ),
                              ),
                              if (s.batches.length > 1)
                                PopupMenuButton<String>(
                                  tooltip: l.anonymousPreviousBatches,
                                  initialValue: batch.id,
                                  enabled: !busy,
                                  onSelected: (id) => setState(() {
                                    activeBatchId = id;
                                    choice = null;
                                  }),
                                  itemBuilder: (_) => [
                                    for (var i = 0; i < s.batches.length; i++)
                                      CheckedPopupMenuItem(
                                        value: s.batches[i].id,
                                        checked: s.batches[i].id == batch.id,
                                        child: Text(
                                          l.anonymousBatchNumber(i + 1),
                                        ),
                                      ),
                                  ],
                                  child: Padding(
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 8,
                                      vertical: 12,
                                    ),
                                    child: Text(
                                      l.anonymousBatchNumber(
                                        s.batches.indexOf(batch) + 1,
                                      ),
                                      style: type.caption,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                          const SizedBox(height: 12),
                          LayoutBuilder(
                            builder: (context, constraints) => Wrap(
                              spacing: 8,
                              runSpacing: 8,
                              children: [
                                for (var i = 0; i < batch.words.length; i++)
                                  SizedBox(
                                    width: (constraints.maxWidth - 8) / 2,
                                    child: Semantics(
                                      selected: selected?.index == i,
                                      child: OutlinedButton(
                                        onPressed: busy
                                            ? null
                                            : () => setState(
                                                () => choice = (
                                                  batchId: batch.id,
                                                  index: i,
                                                  word: batch.words[i],
                                                ),
                                              ),
                                        style: OutlinedButton.styleFrom(
                                          minimumSize: const Size(44, 48),
                                          padding: const EdgeInsets.symmetric(
                                            horizontal: 12,
                                            vertical: 12,
                                          ),
                                          foregroundColor: selected?.index == i
                                              ? colors.primary
                                              : colors.baseContent,
                                          backgroundColor: selected?.index == i
                                              ? colors.primary.withValues(
                                                  alpha: .08,
                                                )
                                              : colors.base100,
                                          side: BorderSide(
                                            color: selected?.index == i
                                                ? colors.primary
                                                : colors.line,
                                          ),
                                          shape: RoundedRectangleBorder(
                                            borderRadius: BorderRadius.circular(
                                              12,
                                            ),
                                          ),
                                        ),
                                        child: Row(
                                          mainAxisAlignment:
                                              MainAxisAlignment.center,
                                          children: [
                                            Flexible(
                                              child: Text(
                                                batch.words[i],
                                                textAlign: TextAlign.center,
                                              ),
                                            ),
                                            if (selected?.index == i) ...[
                                              const SizedBox(width: 4),
                                              GfSymbol(
                                                'check',
                                                size: 16,
                                                color: colors.primary,
                                              ),
                                            ],
                                          ],
                                        ),
                                      ),
                                    ),
                                  ),
                              ],
                            ),
                          ),
                        ],
                        const SizedBox(height: 20),
                        Wrap(
                          spacing: 12,
                          runSpacing: 8,
                          crossAxisAlignment: WrapCrossAlignment.center,
                          children: [
                            GfButton(
                              label: batch == null
                                  ? l.anonymousChooseName
                                  : l.anonymousRefreshNames,
                              icon: batch == null
                                  ? null
                                  : const GfSymbol('refresh-cw', size: 18),
                              variant: batch == null
                                  ? GfButtonVariant.primary
                                  : GfButtonVariant.secondary,
                              loading: busy,
                              onPressed: busy || s.remaining == 0
                                  ? null
                                  : () => draw(s),
                            ),
                            Text(
                              l.anonymousDrawsRemaining(s.remaining),
                              style: type.caption.copyWith(
                                color: colors.baseContent.withValues(
                                  alpha: .65,
                                ),
                              ),
                            ),
                          ],
                        ),
                        if (s.remaining == 0) ...[
                          const SizedBox(height: 8),
                          Text(
                            l.anonymousResetAt(date(s.resetsAt)),
                            style: type.caption.copyWith(
                              color: colors.baseContent.withValues(alpha: .65),
                            ),
                          ),
                        ],
                        if (selected != null) ...[
                          const SizedBox(height: 20),
                          Container(
                            padding: const EdgeInsets.all(16),
                            decoration: BoxDecoration(
                              color: colors.primary.withValues(alpha: .06),
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Row(
                              children: [
                                GfSymbol(
                                  'eye-off',
                                  size: 20,
                                  color: colors.primary,
                                ),
                                const SizedBox(width: 12),
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
                                    children: [
                                      Text(
                                        l.anonymousNamePreview,
                                        style: type.caption.copyWith(
                                          color: colors.baseContent.withValues(
                                            alpha: .65,
                                          ),
                                        ),
                                      ),
                                      const SizedBox(height: 4),
                                      Text(
                                        selected.word,
                                        style: type.bodyStrong,
                                      ),
                                    ],
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ],
                        const SizedBox(height: 24),
                      ],
                      const GfDivider(),
                      const SizedBox(height: 16),
                      if (canChoose && batch == null) ...[
                        Text(l.anonymousConfirmHint, style: type.caption),
                        const SizedBox(height: 8),
                      ],
                      Text(
                        l.anonymousPrivacySummary,
                        style: type.caption.copyWith(
                          color: colors.baseContent.withValues(alpha: .65),
                        ),
                      ),
                      ExpansionTile(
                        tilePadding: EdgeInsets.zero,
                        childrenPadding: const EdgeInsets.only(bottom: 12),
                        title: Text(l.anonymousRules, style: type.caption),
                        children: [
                          Text(l.anonymousBoundary, style: type.caption),
                          const SizedBox(height: 8),
                          Text(l.anonymousHistoryHint, style: type.caption),
                        ],
                      ),
                    ],
                  ),
          ),
          if (canChoose && batch != null)
            Container(
              key: const ValueKey('anonymous-confirmation-footer'),
              padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
              decoration: BoxDecoration(
                color: colors.base100,
                border: Border(top: BorderSide(color: colors.line)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(l.anonymousNameLockHint, style: type.caption),
                  const SizedBox(height: 12),
                  GfButton(
                    label: l.anonymousConfirmName,
                    expanded: true,
                    loading: busy,
                    onPressed: selected == null || busy ? null : confirm,
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
