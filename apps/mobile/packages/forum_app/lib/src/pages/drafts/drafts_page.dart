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
import '../publish/publish_type.dart';

enum _DraftFilter { all, local, cloud, replies }

/// One scroll surface for account-scoped device recovery copies and cloud drafts.
class DraftsPage extends ConsumerStatefulWidget {
  const DraftsPage({super.key});
  @override
  ConsumerState<DraftsPage> createState() => _DraftsPageState();
}

class _DraftsPageState extends ConsumerState<DraftsPage> {
  AsyncValue<DraftsPageProps> _page = const AsyncValue.loading();
  final _search = TextEditingController();
  _DraftFilter _filter = _DraftFilter.all;
  List<LocalDraft> _localDrafts = [];
  bool _localLoaded = false;
  Object? _localError;
  Object? _refreshError;
  String? _owner;
  int _loadSequence = 0;
  int _localLoadSequence = 0;
  String? _deletingKey;
  bool _restoring = false;
  ({LocalDraft draft, String owner, int epoch})? _deleted;

  bool get _managing => _deletingKey != null || _restoring;
  bool get _filtered =>
      _search.text.trim().isNotEmpty || _filter != _DraftFilter.all;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  Future<void> _loadLocal(int sequence, int epoch) async {
    final localSequence = ++_localLoadSequence;
    try {
      final owner = await ref.read(writingScopeProvider.future);
      final drafts = await ref.read(writingStoreProvider).drafts(owner);
      if (!mounted ||
          sequence != _loadSequence ||
          localSequence != _localLoadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        _owner = owner;
        _localDrafts = drafts;
        _localLoaded = true;
        _localError = null;
      });
    } catch (error) {
      if (mounted &&
          sequence == _loadSequence &&
          localSequence == _localLoadSequence &&
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

  Future<void> _open(String route) async {
    setState(() => _deleted = null);
    FocusManager.instance.primaryFocus?.unfocus();
    await context.push(route);
    if (mounted) await _load(silent: true);
  }

  Future<void> _openLocal(LocalDraft draft) => _open(
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

  String _title(LocalDraft draft, AppLocalizations l10n) =>
      draft.title.trim().isNotEmpty
      ? draft.title
      : draft.content.trim().isNotEmpty
      ? draft.content
      : draft.images.isNotEmpty
      ? l10n.publishImageOnlyTitle
      : l10n.topicNoTitle;

  Future<void> _delete(LocalDraft draft) async {
    final owner = _owner;
    if (owner == null || _managing) return;
    final session = ref.read(offlineCacheEpochProvider.notifier);
    final epoch = ref.read(offlineCacheEpochProvider);
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l10n.draftDeleteLocal),
        content: Text(
          _title(draft, l10n),
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
    setState(() => _deletingKey = draft.key);
    try {
      await ref
          .read(writingStoreProvider)
          .delete(owner, draft.key, isCurrent: () => session.isCurrent(epoch));
      if (mounted && session.isCurrent(epoch)) {
        setState(() {
          _localDrafts = _localDrafts
              .where((item) => item.key != draft.key)
              .toList();
          _deleted = (draft: draft, owner: owner, epoch: epoch);
        });
        await _loadLocal(_loadSequence, epoch);
      }
    } catch (error) {
      if (mounted && session.isCurrent(epoch)) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
      }
    } finally {
      if (mounted && session.isCurrent(epoch)) {
        setState(() => _deletingKey = null);
      }
    }
  }

  Future<void> _undoDelete() async {
    final deleted = _deleted;
    if (deleted == null || _managing) return;
    final session = ref.read(offlineCacheEpochProvider.notifier);
    if (!session.isCurrent(deleted.epoch) || deleted.owner != _owner) return;
    final l10n = AppLocalizations.of(context);
    setState(() => _restoring = true);
    try {
      final restored = await ref
          .read(writingStoreProvider)
          .restoreIfAbsent(
            deleted.owner,
            deleted.draft,
            isCurrent: () => session.isCurrent(deleted.epoch),
          );
      if (!mounted || !session.isCurrent(deleted.epoch)) return;
      setState(() => _deleted = null);
      await _loadLocal(_loadSequence, deleted.epoch);
      if (mounted && session.isCurrent(deleted.epoch)) {
        showGfToast(
          context,
          restored ? l10n.draftDeleteRestored : l10n.draftRestoreConflict,
        );
      }
    } catch (error) {
      if (mounted && session.isCurrent(deleted.epoch)) {
        showGfToast(context, l10n.draftRestoreFailed, error: true);
      }
    } finally {
      if (mounted && session.isCurrent(deleted.epoch)) {
        setState(() => _restoring = false);
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

  String _filterLabel(_DraftFilter filter, AppLocalizations l10n) =>
      switch (filter) {
        _DraftFilter.all => l10n.searchAll,
        _DraftFilter.local => l10n.draftLocalSection,
        _DraftFilter.cloud => l10n.draftCloudSection,
        _DraftFilter.replies => l10n.draftKindReply,
      };

  void _clearFilters() => setState(() {
    _search.clear();
    _filter = _DraftFilter.all;
  });

  Widget _section(String title) => Padding(
    padding: const EdgeInsets.fromLTRB(16, 20, 16, 8),
    child: Text(title, style: GfTheme.typographyOf(context).title2),
  );

  Widget _retry(Object error, VoidCallback onRetry) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16),
    child: Row(
      children: [
        Expanded(
          child: Text(resolveErrorMessage(AppLocalizations.of(context), error)),
        ),
        TextButton(
          onPressed: onRetry,
          child: Text(AppLocalizations.of(context).commonRetry),
        ),
      ],
    ),
  );

  Widget _localRow(LocalDraft draft, AppLocalizations l10n) {
    final updated = DateTime.fromMillisecondsSinceEpoch(
      draft.updatedAt,
    ).toIso8601String();
    return ListTile(
      key: ValueKey(draft.key),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      leading: draft.kind == DraftKind.reply
          ? const Icon(Icons.reply_outlined)
          : PublishTypeIcon(PublishType.fromValue(draft.contentType), size: 32),
      title: Text(
        _title(draft, l10n),
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (draft.title.isNotEmpty && draft.content.isNotEmpty)
            Text(draft.content, maxLines: 2, overflow: TextOverflow.ellipsis),
          const SizedBox(height: 4),
          Text(
            [
              _kindLabel(draft, l10n),
              if (draft.kind != DraftKind.reply)
                PublishType.fromValue(draft.contentType).label(l10n),
              timeAgo(updated, l10n: l10n),
            ].join(' · '),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.publishContinue,
            style: TextStyle(color: GfTheme.colorsOf(context).primary),
          ),
        ],
      ),
      onTap: _managing ? null : () => _openLocal(draft),
      trailing: IconButton(
        tooltip: l10n.draftDeleteLocal,
        icon: const GfSymbol('trash-2'),
        onPressed: _managing ? null : () => _delete(draft),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      _loadSequence++;
      _search.clear();
      setState(() {
        _owner = null;
        _localDrafts = [];
        _localLoaded = false;
        _localError = null;
        _refreshError = null;
        _deleted = null;
        _deletingKey = null;
        _restoring = false;
        _filter = _DraftFilter.all;
        _page = const AsyncValue.loading();
      });
    });
    final l10n = AppLocalizations.of(context);
    final query = _search.text.trim().toLowerCase();
    bool matches(String title, String content) =>
        query.isEmpty ||
        title.toLowerCase().contains(query) ||
        content.toLowerCase().contains(query);
    final showLocal = _filter != _DraftFilter.cloud;
    final showCloud =
        _filter == _DraftFilter.all || _filter == _DraftFilter.cloud;
    final local = showLocal
        ? _localDrafts
              .where(
                (draft) =>
                    (_filter != _DraftFilter.replies ||
                        draft.kind == DraftKind.reply) &&
                    matches(draft.title, draft.content),
              )
              .toList()
        : <LocalDraft>[];
    final cloud = showCloud
        ? (_page.valueOrNull?.drafts ?? <DraftPayload>[])
              .where((draft) => matches(draft.title, draft.description))
              .toList()
        : <DraftPayload>[];
    final count = local.length + cloud.length;
    final ready =
        (!showLocal || _localLoaded) && (!showCloud || _page.hasValue);
    return Scaffold(
      appBar: GfAppBar(
        title: Text(l10n.draftsTitle),
        actions: [
          GfIconButton(
            icon: Icons.add,
            tooltip: l10n.navPublish,
            onPressed: () => showComposeMenu(
              context,
              bottom: 16,
              onCompose: (type) => _open('/publish?type=${type.value}'),
            ),
          ),
        ],
      ),
      bottomNavigationBar: _deleted == null
          ? null
          : SafeArea(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Semantics(
                      liveRegion: true,
                      child: Text(l10n.draftDeleteDone),
                    ),
                    Text(
                      _title(_deleted!.draft, l10n),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    Align(
                      alignment: AlignmentDirectional.centerEnd,
                      child: TextButton.icon(
                        onPressed: _managing ? null : _undoDelete,
                        icon: const GfSymbol('undo-2'),
                        label: Text(l10n.publishUndo),
                      ),
                    ),
                  ],
                ),
              ),
            ),
      body: GfScrollToTop(
        semanticLabel: l10n.commonBackToTop,
        threshold: 360,
        builder: (context, controller) => AppRefreshIndicator(
          onRefresh: () => _load(silent: true),
          child: CustomScrollView(
            controller: controller,
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            physics: const AlwaysScrollableScrollPhysics(),
            slivers: [
              SliverToBoxAdapter(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      GfSearchField(
                        controller: _search,
                        hintText: l10n.draftSearchHint,
                        clearLabel: l10n.courseCopyClearSearch,
                        onChanged: (_) => setState(() {}),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        l10n.draftSearchScope,
                        style: GfTheme.typographyOf(context).caption,
                      ),
                      const SizedBox(height: 8),
                      SingleChildScrollView(
                        scrollDirection: Axis.horizontal,
                        child: Row(
                          children: [
                            for (final filter in _DraftFilter.values)
                              Padding(
                                padding: const EdgeInsets.only(right: 8),
                                child: ChoiceChip(
                                  label: Text(_filterLabel(filter, l10n)),
                                  selected: _filter == filter,
                                  onSelected: (_) =>
                                      setState(() => _filter = filter),
                                ),
                              ),
                          ],
                        ),
                      ),
                      if (ready) Text(l10n.draftMatchCount(count)),
                      if (_filtered && count == 0 && ready) ...[
                        const SizedBox(height: 12),
                        Text(l10n.draftNoMatches),
                        TextButton(
                          onPressed: _clearFilters,
                          child: Text(l10n.draftClearFilters),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
              if (showLocal) ...[
                SliverToBoxAdapter(child: _section(l10n.draftLocalSection)),
                if (_localError != null)
                  SliverToBoxAdapter(
                    child: _retry(
                      _localError!,
                      () => _loadLocal(
                        _loadSequence,
                        ref.read(offlineCacheEpochProvider),
                      ),
                    ),
                  ),
                if (!_localLoaded && _localError == null)
                  const SliverToBoxAdapter(
                    child: Padding(
                      padding: EdgeInsets.all(24),
                      child: GfLoading(),
                    ),
                  )
                else if (_localLoaded && _localDrafts.isEmpty && !_filtered)
                  SliverToBoxAdapter(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Text(l10n.draftLocalEmpty),
                    ),
                  ),
                SliverList.builder(
                  itemCount: local.length,
                  itemBuilder: (context, index) =>
                      _localRow(local[index], l10n),
                ),
              ],
              if (showCloud) ...[
                SliverToBoxAdapter(child: _section(l10n.draftCloudSection)),
                if (_refreshError != null)
                  SliverToBoxAdapter(
                    child: _retry(_refreshError!, () => _load(silent: true)),
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
                  data: (props) => cloud.isEmpty
                      ? SliverToBoxAdapter(
                          child: !_filtered && props.drafts.isEmpty
                              ? Padding(
                                  padding: const EdgeInsets.all(16),
                                  child: Text(l10n.draftsEmpty),
                                )
                              : const SizedBox.shrink(),
                        )
                      : SliverList.builder(
                          itemCount: cloud.length,
                          itemBuilder: (context, index) {
                            final draft = cloud[index];
                            return GfDraftRow(
                              editLabel: l10n.publishContinue,
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
                              onTap: () => _open('/publish?id=${draft.id}'),
                            );
                          },
                        ),
                ),
              ],
              const SliverToBoxAdapter(child: SizedBox(height: 32)),
            ],
          ),
        ),
      ),
    );
  }
}
