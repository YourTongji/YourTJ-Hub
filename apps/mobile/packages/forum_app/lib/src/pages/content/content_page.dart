import '../../realtime/realtime_updates.dart';
import 'content_password_dialog.dart';
import 'content_review_sheet.dart';
import '../topic/post_editor.dart';
import '../../widgets/markdown_view.dart';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';

class ContentPage extends ConsumerStatefulWidget {
  const ContentPage({super.key, this.deleted = false, this.initialType});
  final bool deleted;

  /// Filter to open on: 'topic' or 'post'; anything else opens topics.
  final String? initialType;
  @override
  ConsumerState<ContentPage> createState() => _ContentPageState();
}

class _ContentPageState extends ConsumerState<ContentPage> {
  final List<UserContentItem> _items = [];
  final Set<int> _selected = {};
  late String _type = widget.initialType == 'post' ? 'post' : 'topic';
  String? _error;
  bool _loading = true;
  bool _busy = false;
  bool _hasMore = false;
  int _cursor = 0;
  int _generation = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  void _removeTopics(Set<int> removed) {
    if (widget.deleted) return;
    _items.removeWhere((item) {
      if (!removed.contains(item.topicId ?? item.id)) return false;
      _selected.remove(item.id);
      return true;
    });
  }

  Future<void> _load({bool more = false}) async {
    if (more && (_loading || !_hasMore)) return;
    final generation = ++_generation;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final page = await ref
          .read(contentRepositoryProvider)
          .list(
            contentType: _type,
            deleted: widget.deleted,
            cursor: more ? _cursor : 0,
          );
      if (!mounted ||
          generation != _generation ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        if (!more) {
          _items.clear();
          _selected.clear();
        }
        final ids = _items.map((item) => item.id).toSet();
        _items.addAll(page.items.where((item) => ids.add(item.id)));
        _removeTopics(ref.read(removedTopicsProvider));
        _cursor = page.nextCursorId;
        _hasMore = page.hasMore;
      });
    } catch (error) {
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _loading = false);
      }
    }
  }

  Future<bool> _confirm(String title, String message) async =>
      await showGfAlertDialog<bool>(
        context,
        builder: (context) => GfAlertDialog(
          title: GfText(title),
          content: GfText(message),
          actions: [
            GfDialogAction(
              onPressed: () => Navigator.pop(context, false),
              child: GfText(AppLocalizations.of(context).commonCancel),
            ),
            GfDialogAction(
              destructive: true,
              onPressed: () => Navigator.pop(context, true),
              child: GfText(title),
            ),
          ],
        ),
      ) ??
      false;

  Future<void> _run(Future<void> Function(String? password) action) async {
    if (_busy) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _busy = true);
    try {
      try {
        await action(null);
      } on ApiException catch (error) {
        if (error.messageCode != 'content.batchDelete.confirmRequired') rethrow;
        if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
        final password = await showContentPasswordDialog(context);
        if (!mounted ||
            epoch != ref.read(offlineCacheEpochProvider) ||
            password == null ||
            password.isEmpty) {
          return;
        }
        await action(password);
      }
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        await _load();
      }
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _deleteSelected() async {
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final ids = _selected.toList();
    if (ids.isEmpty ||
        !await _confirm(l10n.contentDelete, l10n.contentDeleteConfirm) ||
        !mounted ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    await _run((password) async {
      final results = await ref
          .read(contentRepositoryProvider)
          .delete(contentType: _type, ids: ids, password: password);
      final failures = results.where((result) => !result.success).toList();
      if (failures.isNotEmpty &&
          mounted &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          failures.map((result) => result.message).join('\n'),
          error: true,
        );
      }
    });
  }

  Future<void> _view(UserContentItem item) async {
    await showDialog<void>(
      context: context,
      builder: (context) => AlertDialog(
        title: GfText(item.title),
        content: SizedBox(
          width: 600,
          child: SingleChildScrollView(
            child: GfMarkdownView(data: item.content, images: item.images),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: GfText(AppLocalizations.of(context).commonCancel),
          ),
        ],
      ),
    );
  }

  /// Editing replaces the appealed revision, so a pending appeal would close.
  Future<bool> _confirmEditDuringAppeal(UserContentItem item) async {
    if (item.appeal?.status != 'pending') return true;
    final l10n = AppLocalizations.of(context);
    return await showGfAlertDialog<bool>(
          context,
          builder: (context) => GfAlertDialog(
            content: GfText(l10n.appealEditWarning),
            actions: [
              GfDialogAction(
                onPressed: () => Navigator.pop(context, false),
                child: GfText(l10n.commonCancel),
              ),
              GfDialogAction(
                onPressed: () => Navigator.pop(context, true),
                child: GfText(l10n.appealEditAnyway),
              ),
            ],
          ),
        ) ??
        false;
  }

  Future<void> _review(UserContentItem item, {bool appeal = false}) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final action = await showContentReviewSheet(
      context,
      item,
      startAppeal: appeal,
    );
    if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
    switch (action) {
      case ContentReviewAction.view:
        await _view(item);
      case ContentReviewAction.edit:
        await _edit(item);
      case ContentReviewAction.appealed:
        await _load();
      case null:
        break;
    }
  }

  Future<void> _edit(UserContentItem item) async {
    if (!await _confirmEditDuringAppeal(item) || !mounted) return;
    if (item.contentType == 'topic') {
      await context.push('/publish?id=${item.id}');
      if (mounted) await _load();
      return;
    }
    // Content kind is topic/post; the publish page selects the topic's format
    // (question, moment or article) from the latest editable version.
    if (item.contentType != 'post') return;
    final epoch = ref.read(offlineCacheEpochProvider);
    final saved = await showPostEditor(
      context,
      postId: item.id,
      content: item.content,
    );
    if (saved == true &&
        mounted &&
        epoch == ref.read(offlineCacheEpochProvider)) {
      await _load();
    }
  }

  Future<void> _act(UserContentItem item, String action) async {
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    if (action == 'edit') {
      await _edit(item);
      return;
    }
    if (action == 'view') {
      await _view(item);
      return;
    }
    if (action == 'appeal') {
      await _review(item, appeal: true);
      return;
    }
    if (action == 'restore') {
      await _run((_) => ref.read(contentRepositoryProvider).restore(item));
    } else {
      if (!await _confirm(l10n.contentPurge, l10n.contentPurgeConfirm) ||
          !mounted ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      await _run(
        (password) =>
            ref.read(contentRepositoryProvider).purge(item, password: password),
      );
    }
  }

  void _selectType(String type) {
    if (_type == type) return;
    setState(() {
      _type = type;
      _items.clear();
      _selected.clear();
    });
    _load();
  }

  Widget _typeSelector(AppLocalizations l10n) => GfSegmented<String>(
    segments: [(l10n.profileTopics, 'topic'), (l10n.profileReplies, 'post')],
    selected: _type,
    onSelected: _busy ? (_) {} : _selectType,
  );

  @override
  Widget build(BuildContext context) {
    ref.listen<int>(
      realtimeInvalidationsProvider.select((state) => state.contentRevision),
      (_, _) {
        if (mounted) _load();
      },
    );
    final l10n = AppLocalizations.of(context);
    ref.listen(removedTopicsProvider, (_, removed) {
      setState(() => _removeTopics(removed));
    });
    ref.listen(offlineCacheEpochProvider, (_, next) {
      setState(() {
        _generation++;
        _items.clear();
        _selected.clear();
        _loading = false;
        _hasMore = false;
        _error = null;
      });
    });
    return Scaffold(
      appBar: GfAppBar(
        title: GfText(widget.deleted ? l10n.profileTrash : l10n.profileContent),
      ),
      body: LayoutBuilder(
        builder: (context, constraints) {
          final wide =
              constraints.maxWidth >= 900 &&
              MediaQuery.textScalerOf(context).scale(17) <= 25.5;
          return Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Offstage(
                offstage: !wide,
                child: SizedBox(
                  width: 280,
                  child: Material(
                    color: GfTheme.colorsOf(context).base200,
                    child: SingleChildScrollView(
                      key: const Key('content-type-sidebar'),
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        children: [
                          for (final entry in [
                            (l10n.profileTopics, 'topic'),
                            (l10n.profileReplies, 'post'),
                          ])
                            ListTile(
                              title: GfText(entry.$1),
                              selected: _type == entry.$2,
                              onTap: _busy ? null : () => _selectType(entry.$2),
                            ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
              Expanded(
                child: Column(
                  children: [
                    Offstage(
                      offstage: wide,
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: _typeSelector(l10n),
                      ),
                    ),
                    if (widget.deleted)
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 16),
                        child: GfText(
                          l10n.contentTrashHint,
                          style: GfTheme.typographyOf(context).caption,
                        ),
                      ),
                    if (!widget.deleted && _items.isNotEmpty)
                      Row(
                        children: [
                          Checkbox(
                            value: _selected.length == _items.length,
                            onChanged: _busy
                                ? null
                                : (checked) => setState(() {
                                    _selected.clear();
                                    if (checked == true) {
                                      _selected.addAll(
                                        _items.take(50).map((item) => item.id),
                                      );
                                    }
                                  }),
                          ),
                          Expanded(
                            child: GfText(
                              '${l10n.contentSelected} ${_selected.length} / 50',
                            ),
                          ),
                          TextButton(
                            onPressed: _busy || _selected.isEmpty
                                ? null
                                : _deleteSelected,
                            child: GfText(l10n.contentDelete),
                          ),
                        ],
                      ),
                    if (_error != null)
                      GfErrorRetry(message: _error!, onRetry: _load),
                    Expanded(
                      child: _loading && _items.isEmpty
                          ? const GfLoading()
                          : AppRefreshIndicator(
                              onRefresh: _load,
                              child: ListView.builder(
                                physics: const AlwaysScrollableScrollPhysics(),
                                itemCount: _items.length + 1,
                                itemBuilder: (context, index) {
                                  if (index == _items.length) {
                                    return _items.isEmpty
                                        ? GfEmpty(message: l10n.contentEmpty)
                                        : GfListFooter(
                                            progressKey: _items.length,
                                            loading: _loading,
                                            hasMore: _hasMore,
                                            onLoadMore: () => _load(more: true),
                                          );
                                  }
                                  final item = _items[index];
                                  // 无标题瞬间（标题为空）用正文摘要作主文案，日期退回副文案，
                                  // 避免列表出现空标题行（与 Web TopicRow 的摘要回退一致）。
                                  final String dateText = widget.deleted
                                      ? item.deletedAt
                                      : item.createdAt;
                                  final String primaryText =
                                      item.title.isNotEmpty
                                      ? item.title
                                      : (item.excerpt.isNotEmpty
                                            ? item.excerpt
                                            : dateText);
                                  final String secondaryText =
                                      item.title.isNotEmpty &&
                                          item.excerpt.isNotEmpty
                                      ? item.excerpt
                                      : (primaryText == dateText
                                            ? ''
                                            : dateText);
                                  return ListTile(
                                    leading: widget.deleted
                                        ? const GfSymbol('file-text')
                                        : Checkbox(
                                            value: _selected.contains(item.id),
                                            onChanged: _busy
                                                ? null
                                                : (selected) => setState(() {
                                                    if (selected == true &&
                                                        _selected.length < 50) {
                                                      _selected.add(item.id);
                                                    } else {
                                                      _selected.remove(item.id);
                                                    }
                                                  }),
                                          ),
                                    title: GfText(
                                      primaryText,
                                      maxLines: 2,
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                    subtitle:
                                        !widget.deleted &&
                                            item.processStatus != 0
                                        ? _ReviewSummary(item: item)
                                        : secondaryText.isEmpty
                                        ? null
                                        : GfText(
                                            secondaryText,
                                            maxLines: 2,
                                            overflow: TextOverflow.ellipsis,
                                          ),
                                    onTap: widget.deleted
                                        ? null
                                        : item.processStatus == 1
                                        ? () => _review(item)
                                        : item.processStatus != 0
                                        ? () => _view(item)
                                        : () => context.push(
                                            Uri(
                                              path:
                                                  '/p/${item.topicId ?? item.id}',
                                              queryParameters:
                                                  item.contentType == 'post' &&
                                                      item.postNo != null
                                                  ? {'postNo': '${item.postNo}'}
                                                  : null,
                                            ).toString(),
                                          ),
                                    trailing:
                                        widget.deleted &&
                                            !item.canRestore &&
                                            !item.canPermanent
                                        ? null
                                        : GfPopupMenuButton<String>(
                                            icon: const GfSymbol(
                                              'ellipsis',
                                              size: 20,
                                            ),
                                            useRootNavigator: true,
                                            enabled: !_busy,
                                            onSelected: (action) =>
                                                _act(item, action),
                                            itemBuilder: (_) => [
                                              if (!widget.deleted) ...[
                                                GfPopupMenuItem(
                                                  value: 'view',
                                                  child: GfText(
                                                    l10n.contentReviewView,
                                                  ),
                                                ),
                                                GfPopupMenuItem(
                                                  value: 'edit',
                                                  child: GfText(
                                                    l10n.contentReviewRetry,
                                                  ),
                                                ),
                                                if (item.processStatus == 1 &&
                                                    item.canAppeal &&
                                                    item.appeal == null)
                                                  GfPopupMenuItem(
                                                    value: 'appeal',
                                                    child: GfText(
                                                      l10n.appealAction,
                                                    ),
                                                  ),
                                              ],
                                              if (item.canRestore)
                                                GfPopupMenuItem(
                                                  value: 'restore',
                                                  child: GfText(
                                                    l10n.contentRestore,
                                                  ),
                                                ),
                                              if (item.canPermanent)
                                                GfPopupMenuItem(
                                                  destructive: true,
                                                  value: 'purge',
                                                  child: GfText(
                                                    l10n.contentPurge,
                                                  ),
                                                ),
                                            ],
                                          ),
                                  );
                                },
                              ),
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

/// Review state in the list row: status, appeal progress and the reason,
/// so authors can tell rejected, appealed and pending items apart at a glance.
class _ReviewSummary extends StatelessWidget {
  const _ReviewSummary({required this.item});

  final UserContentItem item;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final type = GfTheme.typographyOf(context);
    final muted = GfTheme.colorsOf(context).baseContent.withValues(alpha: 0.6);
    final appeal = item.appeal;
    final (
      String? appealLabel,
      GfBadgeVariant appealVariant,
    ) = switch (appeal?.status) {
      'pending' => (l10n.appealStatePendingTitle, GfBadgeVariant.warning),
      'approved' => (l10n.appealStateApprovedTitle, GfBadgeVariant.success),
      'rejected' => (l10n.appealStateRejectedTitle, GfBadgeVariant.muted),
      'stale' => (l10n.appealStateStaleTitle, GfBadgeVariant.muted),
      _ => (null, GfBadgeVariant.muted),
    };
    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Wrap(
            spacing: 6,
            runSpacing: 4,
            children: [
              item.processStatus == 2
                  ? GfBadge(
                      label: l10n.contentReviewPending,
                      variant: GfBadgeVariant.warning,
                    )
                  : GfBadge(
                      label: l10n.contentReviewBlocked,
                      variant: GfBadgeVariant.error,
                    ),
              if (appealLabel != null)
                GfBadge(label: appealLabel, variant: appealVariant),
            ],
          ),
          if (item.processStatus == 1 && item.reviewReason.isNotEmpty) ...[
            const SizedBox(height: 4),
            GfText(
              item.reviewReason,
              style: type.small.copyWith(color: muted),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
          ],
          if (item.hasPublishedVersion) ...[
            const SizedBox(height: 2),
            GfText(
              l10n.contentReviewLive,
              style: type.caption.copyWith(color: muted),
            ),
          ],
        ],
      ),
    );
  }
}
