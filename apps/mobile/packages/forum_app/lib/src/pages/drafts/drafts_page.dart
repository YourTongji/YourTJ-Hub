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
import '../topic/post_editor.dart';
import '../content/content_password_dialog.dart';

enum _DraftFilter { all, local, cloud, replies }

/// One scroll surface for account-scoped device recovery copies and cloud drafts.
class DraftsPage extends ConsumerStatefulWidget {
  const DraftsPage({super.key});
  @override
  ConsumerState<DraftsPage> createState() => _DraftsPageState();
}

class _DraftsPageState extends ConsumerState<DraftsPage> {
  static const double _wideBreakpoint = 900;
  static const double _sidebarWidth = 280;
  AsyncValue<DraftsPageProps> _page = const AsyncValue.loading();
  final _search = TextEditingController();
  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();
  _DraftFilter _filter = _DraftFilter.all;
  List<LocalDraft> _localDrafts = [];
  bool _localLoaded = false;
  Object? _localError;
  Object? _refreshError;
  String? _owner;
  int _loadSequence = 0;
  int _localLoadSequence = 0;
  String? _deletingKey;
  int? _deletingCloudId;
  int _cloudRevision = 0;
  String? _cloudDeletedTitle;
  bool _restoring = false;
  ({LocalDraft draft, String owner, int epoch})? _deleted;

  bool get _managing =>
      _deletingKey != null || _deletingCloudId != null || _restoring;
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
    if (_deletingCloudId != null) return;
    final sequence = ++_loadSequence;
    final cloudRevision = _cloudRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    final previous = _page.valueOrNull;
    final local = _loadLocal(sequence, epoch);
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final payload = await ref.read(pageRepositoryProvider).fetch('/drafts');
      if (!mounted ||
          sequence != _loadSequence ||
          cloudRevision != _cloudRevision ||
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
          cloudRevision == _cloudRevision &&
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
    setState(() {
      _deleted = null;
      _cloudDeletedTitle = null;
    });
    FocusManager.instance.primaryFocus?.unfocus();
    await context.push(route);
    if (mounted) await _load(silent: true);
  }

  Future<void> _openLocal(LocalDraft draft) async {
    if (draft.kind == DraftKind.postEdit) {
      setState(() {
        _deleted = null;
        _cloudDeletedTitle = null;
      });
      FocusManager.instance.primaryFocus?.unfocus();
      await showPostEditor(
        context,
        postId: draft.postId,
        content: draft.postOriginalContent ?? '',
      );
      if (mounted) await _load(silent: true);
      return;
    }
    await _open(
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
  }

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
    final confirmed = await showGfAlertDialog<bool>(
      context,
      builder: (context) => GfAlertDialog(
        title: GfText(l10n.draftDeleteLocal),
        content: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            GfText(
              _title(draft, l10n),
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 12),
            GfText(l10n.draftDeleteLocalConfirm),
          ],
        ),
        actions: [
          GfDialogAction(
            onPressed: () => Navigator.pop(context, false),
            child: GfText(l10n.commonCancel),
          ),
          GfDialogAction(
            destructive: true,
            onPressed: () => Navigator.pop(context, true),
            child: GfText(l10n.draftDeleteLocal),
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
          _cloudDeletedTitle = null;
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

  Future<void> _deleteCloud(DraftPayload draft) async {
    if (_managing) return;
    final session = ref.read(offlineCacheEpochProvider.notifier);
    final epoch = ref.read(offlineCacheEpochProvider);
    final l10n = AppLocalizations.of(context);
    final title = draft.title.isEmpty ? l10n.topicNoTitle : draft.title;
    final confirmed = await showGfAlertDialog<bool>(
      context,
      builder: (context) => GfAlertDialog(
        title: GfText(l10n.draftDeleteCloud),
        content: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            GfText(title, maxLines: 3, overflow: TextOverflow.ellipsis),
            const SizedBox(height: 12),
            GfText(l10n.draftDeleteCloudConfirm),
          ],
        ),
        actions: [
          GfDialogAction(
            onPressed: () => Navigator.pop(context, false),
            child: GfText(l10n.commonCancel),
          ),
          GfDialogAction(
            destructive: true,
            onPressed: () => Navigator.pop(context, true),
            child: GfText(l10n.draftDeleteCloud),
          ),
        ],
      ),
    );
    if (confirmed != true ||
        !mounted ||
        !session.isCurrent(epoch) ||
        _managing) {
      return;
    }
    setState(() {
      _deletingCloudId = draft.id;
      _cloudRevision++;
    });
    var deleted = false;
    try {
      final repository = ref.read(contentRepositoryProvider);
      List<ContentDeletionResult> results;
      try {
        results = await repository.delete(
          contentType: 'topic',
          ids: [draft.id],
        );
      } on ApiException catch (error) {
        if (error.messageCode != 'content.batchDelete.confirmRequired') rethrow;
        if (!mounted || !session.isCurrent(epoch)) return;
        final password = await showContentPasswordDialog(context);
        if (!mounted ||
            !session.isCurrent(epoch) ||
            password == null ||
            password.isEmpty) {
          return;
        }
        results = await repository.delete(
          contentType: 'topic',
          ids: [draft.id],
          password: password,
        );
      }
      if (!mounted || !session.isCurrent(epoch)) return;
      // A successful HTTP response can still contain a failed (or missing) item.
      final matching = results.where((result) => result.contentId == draft.id);
      if (matching.length != 1 || !matching.single.success) {
        showGfToast(context, l10n.draftDeleteCloudFailed, error: true);
        return;
      }
      deleted = true;
      setState(() {
        _cloudRevision++;
        final page = _page.valueOrNull;
        if (page != null) {
          final drafts = page.drafts
              .where((item) => item.id != draft.id)
              .toList();
          _page = AsyncValue.data(
            page.copyWith(drafts: drafts, total: drafts.length),
          );
        }
        _refreshError = null;
        _deleted = null;
        _cloudDeletedTitle = title;
      });
    } catch (error) {
      if (mounted && session.isCurrent(epoch)) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
      }
    } finally {
      if (mounted && session.isCurrent(epoch)) {
        setState(() => _deletingCloudId = null);
      }
    }
    // Refresh the bounded cloud window so its next draft can become visible.
    if (deleted && mounted && session.isCurrent(epoch)) {
      await _load(silent: true);
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
        DraftKind.postEdit => l10n.topicEditReply,
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
    child: GfText(title, style: GfTheme.typographyOf(context).title2),
  );

  Widget _retry(Object error, VoidCallback onRetry) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16),
    child: Row(
      children: [
        Expanded(
          child: GfText(
            resolveErrorMessage(AppLocalizations.of(context), error),
          ),
        ),
        TextButton(
          onPressed: onRetry,
          child: GfText(AppLocalizations.of(context).commonRetry),
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
      leading:
          (draft.kind == DraftKind.reply || draft.kind == DraftKind.postEdit)
          ? const GfSymbol('corner-down-left')
          : PublishTypeIcon(PublishType.fromValue(draft.contentType), size: 32),
      title: GfText(
        _title(draft, l10n),
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (draft.title.isNotEmpty && draft.content.isNotEmpty)
            GfText(draft.content, maxLines: 2, overflow: TextOverflow.ellipsis),
          const SizedBox(height: 4),
          GfText(
            [
              _kindLabel(draft, l10n),
              if (draft.kind != DraftKind.reply &&
                  draft.kind != DraftKind.postEdit)
                PublishType.fromValue(draft.contentType).label(l10n),
              timeAgo(updated, l10n: l10n),
            ].join(' · '),
          ),
          const SizedBox(height: 4),
          Wrap(
            spacing: 8,
            children: [
              TextButton(
                onPressed: _managing ? null : () => _openLocal(draft),
                child: GfText(l10n.publishContinue),
              ),
              Tooltip(
                message: l10n.draftDeleteLocal,
                child: TextButton.icon(
                  onPressed: _managing ? null : () => _delete(draft),
                  icon: const GfSymbol('trash-2', size: 18),
                  label: GfText(l10n.draftDeleteLocal),
                ),
              ),
            ],
          ),
        ],
      ),
      onTap: _managing ? null : () => _openLocal(draft),
    );
  }

  Widget _filterPanel(
    BuildContext context,
    AppLocalizations l10n, {
    bool includeBottomPadding = false,
  }) {
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
                        (draft.kind == DraftKind.reply ||
                            draft.kind == DraftKind.postEdit)) &&
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
    return Padding(
      padding: EdgeInsets.fromLTRB(16, 12, 16, includeBottomPadding ? 16 : 0),
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
          GfText(
            l10n.draftSearchScope,
            style: GfTheme.typographyOf(context).caption,
          ),
          const SizedBox(height: 8),
          SingleChildScrollView(
            scrollDirection: includeBottomPadding
                ? Axis.vertical
                : Axis.horizontal,
            child: Wrap(
              children: [
                for (final filter in _DraftFilter.values)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: ChoiceChip(
                      label: GfText(_filterLabel(filter, l10n)),
                      selected: _filter == filter,
                      onSelected: (_) => setState(() => _filter = filter),
                    ),
                  ),
              ],
            ),
          ),
          if (ready) GfText(l10n.draftMatchCount(count)),
          if (_filtered && count == 0 && ready) ...[
            const SizedBox(height: 12),
            GfText(l10n.draftNoMatches),
            TextButton(
              onPressed: _clearFilters,
              child: GfText(l10n.draftClearFilters),
            ),
          ],
        ],
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
        _deletingCloudId = null;
        _cloudDeletedTitle = null;
        _cloudRevision++;
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
                        (draft.kind == DraftKind.reply ||
                            draft.kind == DraftKind.postEdit)) &&
                    matches(draft.title, draft.content),
              )
              .toList()
        : <LocalDraft>[];
    final cloud = showCloud
        ? (_page.valueOrNull?.drafts ?? <DraftPayload>[])
              .where((draft) => matches(draft.title, draft.description))
              .toList()
        : <DraftPayload>[];
    final scaffold = Scaffold(
      appBar: GfAppBar(
        title: GfText(l10n.draftsTitle),
        actions: [
          GfIconButton(
            symbol: 'plus',
            tooltip: l10n.navPublish,
            onPressed: () => showComposeMenu(
              context,
              bottom: 16,
              onCompose: (type) => _open('/publish?type=${type.value}'),
            ),
          ),
        ],
      ),
      bottomNavigationBar: _cloudDeletedTitle != null
          ? SafeArea(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Semantics(
                      liveRegion: true,
                      child: GfText(l10n.draftDeleteCloudDone),
                    ),
                    GfText(
                      _cloudDeletedTitle!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    Align(
                      alignment: AlignmentDirectional.centerEnd,
                      child: TextButton(
                        onPressed: _managing
                            ? null
                            : () => _open('/recycle-bin'),
                        child: GfText(l10n.draftOpenRecycleBin),
                      ),
                    ),
                  ],
                ),
              ),
            )
          : _deleted == null
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
                      child: GfText(l10n.draftDeleteDone),
                    ),
                    GfText(
                      _title(_deleted!.draft, l10n),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    Align(
                      alignment: AlignmentDirectional.centerEnd,
                      child: TextButton.icon(
                        onPressed: _managing ? null : _undoDelete,
                        icon: const GfSymbol('undo-2'),
                        label: GfText(l10n.publishUndo),
                      ),
                    ),
                  ],
                ),
              ),
            ),
      body: GfScrollToTop(
        controller: _scrollToTopController,
        semanticLabel: l10n.commonBackToTop,
        threshold: 360,
        builder: (context, controller) => LayoutBuilder(
          builder: (context, constraints) {
            final wide =
                constraints.maxWidth >= _wideBreakpoint &&
                MediaQuery.textScalerOf(context).scale(17) <= 25.5;
            final list = AppRefreshIndicator(
              onRefresh: () => _load(silent: true),
              child: CustomScrollView(
                controller: controller,
                keyboardDismissBehavior:
                    ScrollViewKeyboardDismissBehavior.onDrag,
                physics: const AlwaysScrollableScrollPhysics(),
                slivers: [
                  SliverToBoxAdapter(
                    child: Offstage(
                      offstage: wide,
                      child: _filterPanel(context, l10n),
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
                          child: GfText(l10n.draftLocalEmpty),
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
                    if ((_page.valueOrNull?.drafts.length ?? 0) >= 100)
                      SliverToBoxAdapter(
                        child: Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 16),
                          child: GfText(l10n.draftCloudLimit),
                        ),
                      ),
                    if (_refreshError != null)
                      SliverToBoxAdapter(
                        child: _retry(
                          _refreshError!,
                          () => _load(silent: true),
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
                      data: (props) => cloud.isEmpty
                          ? SliverToBoxAdapter(
                              child: !_filtered && props.drafts.isEmpty
                                  ? Padding(
                                      padding: const EdgeInsets.all(16),
                                      child: GfText(l10n.draftsEmpty),
                                    )
                                  : const SizedBox.shrink(),
                            )
                          : SliverList.builder(
                              itemCount: cloud.length,
                              itemBuilder: (context, index) {
                                final draft = cloud[index];
                                return GfDraftRow(
                                  key: ValueKey('cloud-${draft.id}'),
                                  editLabel: l10n.publishContinue,
                                  deleteLabel: l10n.draftDeleteCloud,
                                  onDelete: _managing
                                      ? null
                                      : () => _deleteCloud(draft),
                                  showDelete: true,
                                  onEdit: _managing
                                      ? null
                                      : () => _open('/publish?id=${draft.id}'),
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
                                  onTap: _managing
                                      ? null
                                      : () => _open('/publish?id=${draft.id}'),
                                );
                              },
                            ),
                    ),
                  ],
                  const SliverToBoxAdapter(child: SizedBox(height: 32)),
                ],
              ),
            );
            return Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Offstage(
                  offstage: !wide,
                  child: SizedBox(
                    width: _sidebarWidth,
                    child: ColoredBox(
                      color: GfTheme.colorsOf(context).base200,
                      child: SingleChildScrollView(
                        key: const Key('draft-filter-sidebar'),
                        child: _filterPanel(
                          context,
                          l10n,
                          includeBottomPadding: true,
                        ),
                      ),
                    ),
                  ),
                ),
                Offstage(
                  offstage: !wide,
                  child: SizedBox(
                    width: 1,
                    child: ColoredBox(color: GfTheme.colorsOf(context).line),
                  ),
                ),
                Expanded(child: list),
              ],
            );
          },
        ),
      ),
    );
    return GfStatusBarScrollToTop(
      controller: _scrollToTopController,
      child: scaffold,
    );
  }
}
