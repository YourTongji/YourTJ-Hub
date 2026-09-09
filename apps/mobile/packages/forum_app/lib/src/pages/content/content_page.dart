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
  const ContentPage({super.key, this.deleted = false});
  final bool deleted;
  @override
  ConsumerState<ContentPage> createState() => _ContentPageState();
}

class _ContentPageState extends ConsumerState<ContentPage> {
  final List<UserContentItem> _items = [];
  final Set<int> _selected = {};
  String _type = 'topic';
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
      await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(title),
          content: Text(message),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(AppLocalizations.of(context).commonCancel),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(title),
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
        if (!error.messageKey.contains('confirmRequired')) rethrow;
        if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
        String enteredPassword = '';
        final password = await showDialog<String>(
          context: context,
          builder: (context) => AlertDialog(
            title: Text(AppLocalizations.of(context).contentPassword),
            content: TextField(
              onChanged: (value) => enteredPassword = value,
              obscureText: true,
              autofocus: true,
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: Text(AppLocalizations.of(context).commonCancel),
              ),
              TextButton(
                onPressed: () => Navigator.pop(context, enteredPassword),
                child: Text(AppLocalizations.of(context).commonConfirm),
              ),
            ],
          ),
        );
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

  Future<void> _act(UserContentItem item, String action) async {
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    if (action == 'restore') {
      await _run((_) => ref.read(contentRepositoryProvider).restore(item));
    } else {
      if (!await _confirm(
            l10n.contentPurge,
            l10n.contentPurgeConfirm,
          ) ||
          !mounted ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      await _run(
        (password) => ref
            .read(contentRepositoryProvider)
            .purge(item, password: password),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
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
        title: Text(widget.deleted ? l10n.profileTrash : l10n.profileContent),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: GfSegmented<String>(
              segments: [
                (l10n.profileTopics, 'topic'),
                (l10n.profileReplies, 'post'),
              ],
              selected: _type,
              onSelected: _busy
                  ? (_) {}
                  : (type) {
                      if (_type == type) return;
                      setState(() {
                        _type = type;
                        _items.clear();
                        _selected.clear();
                      });
                      _load();
                    },
            ),
          ),
          if (widget.deleted)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Text(
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
                  child: Text(
                    '${l10n.contentSelected} ${_selected.length} / 50',
                  ),
                ),
                TextButton(
                  onPressed: _busy || _selected.isEmpty
                      ? null
                      : _deleteSelected,
                  child: Text(l10n.contentDelete),
                ),
              ],
            ),
          if (_error != null) GfErrorRetry(message: _error!, onRetry: _load),
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
                                  loading: _loading,
                                  hasMore: _hasMore,
                                  onLoadMore: () => _load(more: true),
                                );
                        }
                        final item = _items[index];
                        return ListTile(
                          leading: widget.deleted
                              ? const Icon(Icons.article_outlined)
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
                          title: Text(
                            item.title,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                          ),
                          subtitle: Text(
                            item.excerpt.isEmpty
                                ? (widget.deleted
                                      ? item.deletedAt
                                      : item.createdAt)
                                : item.excerpt,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                          ),
                          onTap: widget.deleted
                              ? null
                              : () => context.push(
                                  Uri(
                                    path: '/p/${item.topicId ?? item.id}',
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
                              : PopupMenuButton<String>(
                                  enabled: !_busy,
                                  onSelected: (action) => _act(item, action),
                                  itemBuilder: (_) => [
                                    if (item.canRestore)
                                      PopupMenuItem(
                                        value: 'restore',
                                        child: Text(l10n.contentRestore),
                                      ),
                                    if (item.canPermanent)
                                      PopupMenuItem(
                                        value: 'purge',
                                        child: Text(l10n.contentPurge),
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
    );
  }
}
