import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:core/core.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../local/writing_store.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import '../../widgets/compose_menu.dart';

/// One scroll surface for account-scoped device recovery copies and cloud drafts.
class DraftsPage extends ConsumerStatefulWidget {
  const DraftsPage({super.key});
  @override
  ConsumerState<DraftsPage> createState() => _DraftsPageState();
}

class _DraftsPageState extends ConsumerState<DraftsPage> {
  AsyncValue<DraftsPageProps> _page = const AsyncValue.loading();
  List<LocalDraft> _localDrafts = [];
  Object? _localError;
  Object? _refreshError;
  String? _owner;
  int _loadSequence = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _loadLocal(int sequence, int epoch) async {
    try {
      final owner = await ref.read(writingScopeProvider.future);
      final drafts = await ref.read(writingStoreProvider).drafts(owner);
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        _owner = owner;
        _localDrafts = drafts;
        _localError = null;
      });
    } catch (error) {
      if (mounted &&
          sequence == _loadSequence &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _localError = error);
      }
    }
  }

  Future<void> _load({bool silent = false}) async {
    final sequence = ++_loadSequence;
    final epoch = ref.read(offlineCacheEpochProvider);
    final previous = _page.valueOrNull;
    final local = _loadLocal(sequence, epoch);
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final payload = await ref.read(pageRepositoryProvider).fetch('/drafts');
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final props = parsePageProps<DraftsPageProps>(payload);
      if (props == null) throw const FormatException('drafts props');
      setState(() {
        _page = AsyncValue.data(props);
        _refreshError = null;
      });
    } catch (error, stack) {
      if (mounted &&
          sequence == _loadSequence &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() {
          if (previous != null) {
            _page = AsyncValue.data(previous);
            _refreshError = error;
          } else {
            _page = AsyncValue.error(error, stack);
          }
        });
      }
    } finally {
      await local;
    }
  }

  Future<void> _openLocal(LocalDraft draft) async {
    await context.push(
      draft.kind == DraftKind.reply
          ? '/p/${draft.topicId}'
          : Uri(
              path: '/publish',
              queryParameters: {
                'local': draft.key,
                'type': '${draft.contentType}',
                if (draft.topicId > 0) 'id': '${draft.topicId}',
              },
            ).toString(),
    );
    if (mounted) await _load(silent: true);
  }

  Future<void> _delete(LocalDraft draft) async {
    final owner = _owner;
    if (owner == null) return;
    final session = ref.read(offlineCacheEpochProvider.notifier);
    final epoch = ref.read(offlineCacheEpochProvider);
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.draftDeleteLocal),
        content: Text(
          draft.title.isEmpty ? draft.content : draft.title,
          maxLines: 3,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l10n.draftDeleteLocal),
          ),
        ],
      ),
    );
    if (confirmed != true ||
        !mounted ||
        !session.isCurrent(epoch) ||
        owner != _owner) {
      return;
    }
    try {
      await ref
          .read(writingStoreProvider)
          .delete(owner, draft.key, isCurrent: () => session.isCurrent(epoch));
      if (mounted && session.isCurrent(epoch)) {
        await _loadLocal(_loadSequence, epoch);
      }
    } catch (error) {
      if (mounted && session.isCurrent(epoch)) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
      }
    }
  }

  String _kindLabel(LocalDraft draft, AppLocalizations l10n) =>
      switch (draft.kind) {
        DraftKind.newTopic => l10n.draftKindNew,
        DraftKind.serverDraft => l10n.draftKindServer,
        DraftKind.topicEdit => l10n.draftKindEdit,
        DraftKind.reply => l10n.draftKindReply,
      };

  Widget _section(String title) => Padding(
    padding: const EdgeInsets.fromLTRB(16, 24, 16, 8),
    child: Text(title, style: GfTheme.typographyOf(context).title2),
  );

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      _loadSequence++;
      setState(() {
        _owner = null;
        _localDrafts = [];
        _localError = null;
        _refreshError = null;
        _page = const AsyncValue.loading();
      });
    });
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(
        title: Text(l10n.draftsTitle),
        actions: [
          GfIconButton(
            icon: Icons.add,
            tooltip: l10n.navPublish,
            onPressed: () => showComposeMenu(context, bottom: 16),
          ),
        ],
      ),
      body: GfScrollToTop(
        semanticLabel: l10n.commonBackToTop,
        threshold: 360,
        builder: (context, controller) => AppRefreshIndicator(
          onRefresh: () => _load(silent: true),
          child: CustomScrollView(
            controller: controller,
            physics: const AlwaysScrollableScrollPhysics(),
            slivers: [
              SliverToBoxAdapter(child: _section(l10n.draftLocalSection)),
              if (_localError != null)
                SliverToBoxAdapter(
                  child: GfErrorRetry(
                    message: resolveErrorMessage(l10n, _localError!),
                    onRetry: () => _loadLocal(
                      _loadSequence,
                      ref.read(offlineCacheEpochProvider),
                    ),
                  ),
                )
              else if (_localDrafts.isEmpty)
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 8,
                    ),
                    child: Text(l10n.draftLocalEmpty),
                  ),
                ),
              SliverList.builder(
                itemCount: _localDrafts.length,
                itemBuilder: (context, index) {
                  final draft = _localDrafts[index];
                  final updated = DateTime.fromMillisecondsSinceEpoch(
                    draft.updatedAt,
                  ).toIso8601String();
                  return ListTile(
                    key: ValueKey(draft.key),
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 8,
                    ),
                    leading: Icon(
                      draft.kind == DraftKind.reply
                          ? Icons.reply_outlined
                          : Icons.edit_note_outlined,
                    ),
                    title: Text(
                      draft.title.isNotEmpty
                          ? draft.title
                          : (draft.content.isNotEmpty
                                ? draft.content
                                : l10n.topicNoTitle),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (draft.title.isNotEmpty && draft.content.isNotEmpty)
                          Text(
                            draft.content,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                          ),
                        const SizedBox(height: 4),
                        Text(
                          '${_kindLabel(draft, l10n)} · ${timeAgo(updated, l10n: l10n)}',
                        ),
                      ],
                    ),
                    onTap: () => _openLocal(draft),
                    trailing: IconButton(
                      tooltip: l10n.draftDeleteLocal,
                      icon: const Icon(Icons.delete_outline),
                      onPressed: () => _delete(draft),
                    ),
                  );
                },
              ),
              SliverToBoxAdapter(child: _section(l10n.draftCloudSection)),
              if (_refreshError != null)
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    child: Row(
                      children: [
                        Expanded(
                          child: Text(
                            resolveErrorMessage(l10n, _refreshError!),
                          ),
                        ),
                        TextButton(
                          onPressed: () => _load(silent: true),
                          child: Text(l10n.commonRetry),
                        ),
                      ],
                    ),
                  ),
                ),
              _page.when(
                loading: () => const SliverToBoxAdapter(
                  child: Padding(
                    padding: EdgeInsets.all(24),
                    child: GfLoading(),
                  ),
                ),
                error: (error, _) => SliverToBoxAdapter(
                  child: GfErrorRetry(
                    message: resolveErrorMessage(l10n, error),
                    onRetry: _load,
                  ),
                ),
                data: (props) => props.drafts.isEmpty
                    ? SliverToBoxAdapter(
                        child: Padding(
                          padding: const EdgeInsets.all(16),
                          child: Text(l10n.draftsEmpty),
                        ),
                      )
                    : SliverList.builder(
                        itemCount: props.drafts.length,
                        itemBuilder: (context, index) {
                          final draft = props.drafts[index];
                          return GfDraftRow(
                            editLabel: l10n.commonEdit,
                            blockedLabel: l10n.draftsBlocked,
                            title: draft.title.isEmpty
                                ? l10n.topicNoTitle
                                : draft.title,
                            description: draft.description,
                            categories: [
                              for (final cat in draft.categories)
                                GfTopicCategory(
                                  name: cat.name,
                                  color: colorFromHex(cat.color),
                                ),
                            ],
                            blocked: draft.processStatus == 1,
                            meta: l10n.draftsMetaCreated(
                              formatDateTime(draft.createdAt),
                            ),
                            updatedTime: formatDate(draft.updatedAt),
                            onTap: () async {
                              await context.push('/publish?id=${draft.id}');
                              if (mounted) await _load(silent: true);
                            },
                          );
                        },
                      ),
              ),
              const SliverToBoxAdapter(child: SizedBox(height: 32)),
            ],
          ),
        ),
      ),
    );
  }
}
