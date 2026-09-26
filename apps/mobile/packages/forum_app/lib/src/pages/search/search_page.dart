import '../../private_notes.dart';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:dio/dio.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../asset_url.dart';
import '../../server_messages.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../local/writing_store.dart';
import '../../widgets/status_views.dart';
import '../../widgets/topic_list.dart';
import '../../widgets/campus_shortcuts.dart';

/// 聚合搜索页：已提交查询与输入中的关键词分离，各类型结果逐项构建。
class SearchPage extends ConsumerStatefulWidget {
  const SearchPage({super.key});

  @override
  ConsumerState<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends ConsumerState<SearchPage> {
  final TextEditingController _query = TextEditingController();
  AsyncValue<SearchPageProps>? _result;
  String _scope = 'all';
  String _submittedQuery = '';
  int _page = 1;
  bool _loadingMore = false;
  String? _loadMoreError;
  int _generation = 0;
  CancelToken? _searchCancel;
  CancelToken? _loadMoreCancel;
  List<String> _recent = [];
  int _historyGeneration = 0;

  Future<void> _loadHistory() async {
    final generation = ++_historyGeneration;
    final epoch = ref.read(offlineCacheEpochProvider);
    try {
      final owner = await ref.read(writingScopeProvider.future);
      final recent = await ref.read(writingStoreProvider).history(owner);
      if (mounted &&
          generation == _historyGeneration &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _recent = recent);
      }
    } catch (_) {
      /* Search works without local storage. */
    }
  }

  Future<void> _remember(String query) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    try {
      final owner = await ref.read(writingScopeProvider.future);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      await ref.read(writingStoreProvider).remember(owner, query);
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        await _loadHistory();
      }
    } catch (_) {
      /* History is optional, never block a search. */
    }
  }

  Future<void> _clearHistory() async {
    final epoch = ref.read(offlineCacheEpochProvider);
    try {
      final owner = await ref.read(writingScopeProvider.future);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      await ref.read(writingStoreProvider).clearHistory(owner);
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        _historyGeneration++;
        setState(() => _recent = []);
      }
    } catch (error) {
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    }
  }

  void _searchElsewhere(String path) {
    final query = _query.text.trim();
    _remember(query);
    context.push(
      Uri(
        path: path,
        queryParameters: query.isEmpty ? null : {'q': query},
      ).toString(),
    );
  }

  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();

  @override
  void initState() {
    super.initState();
    _loadHistory();
  }

  @override
  void dispose() {
    _searchCancel?.cancel('search page disposed');
    _loadMoreCancel?.cancel('search page disposed');
    _query.dispose();
    super.dispose();
  }

  void _clearSearch() {
    // Invalidates initial, refresh and pagination requests already in flight.
    _generation++;
    _searchCancel?.cancel('search cleared');
    _loadMoreCancel?.cancel('search cleared');
    _query.clear();
    setState(() {
      _result = null;
      _submittedQuery = '';
      _scope = 'all';
      _page = 1;
      _loadingMore = false;
      _loadMoreError = null;
    });
  }

  Future<void> _search({String? query}) async {
    final String q = query ?? _query.text.trim();
    if (q.isEmpty) return;
    FocusManager.instance.primaryFocus?.unfocus();
    _remember(q);
    final epoch = ref.read(offlineCacheEpochProvider);
    final generation = ++_generation;
    _searchCancel?.cancel('search request superseded');
    _loadMoreCancel?.cancel('search query changed');
    final cancel = _searchCancel = CancelToken();
    setState(() {
      _submittedQuery = q;
      _result = const AsyncValue.loading();
      _page = 1;
      _loadingMore = false;
      _loadMoreError = null;
    });
    try {
      final SearchPageProps props = await ref
          .read(topicRepositoryProvider)
          .search(
            query: q,
            scope: _scope == 'all' ? '' : _scope,
            page: 1,
            cancelToken: cancel,
          );
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _result = AsyncValue.data(props));
      }
    } catch (e, st) {
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _result = AsyncValue.error(e, st));
      }
    } finally {
      if (identical(_searchCancel, cancel)) _searchCancel = null;
    }
  }

  Future<void> _loadMore() async {
    final SearchPageProps? props = _result?.value;
    if (props == null ||
        (_scope != 'all' && _scope != 'topics') ||
        _loadingMore ||
        _page >= props.totalPages) {
      return;
    }
    final epoch = ref.read(offlineCacheEpochProvider);
    final generation = _generation;
    final cancel = _loadMoreCancel = CancelToken();
    setState(() {
      _loadingMore = true;
      _loadMoreError = null;
    });
    try {
      final SearchPageProps next = await ref
          .read(topicRepositoryProvider)
          .search(
            query: props.query,
            // Only topics paginate; keep the other aggregate groups intact.
            scope: 'topics',
            page: _page + 1,
            cancelToken: cancel,
          );
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        if (next.searchUnavailable == true ||
            (next.failedScopes?.contains('topics') ?? false)) {
          setState(
            () =>
                _loadMoreError = AppLocalizations.of(context).searchUnavailable,
          );
          return;
        }
        setState(() {
          _page += 1;
          _result = AsyncValue.data(
            props.copyWith(
              topics: <TopicPayload>[...props.topics, ...next.topics],
              total: next.total,
              totalPages: next.totalPages,
              pagination: next.pagination,
            ),
          );
        });
      }
    } catch (error) {
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(
          () => _loadMoreError = resolveErrorMessage(
            AppLocalizations.of(context),
            error,
          ),
        );
      }
    } finally {
      if (identical(_loadMoreCancel, cancel)) _loadMoreCancel = null;
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _loadingMore = false);
      }
    }
  }

  void _setScope(String scope) {
    if (scope == _scope) return;
    setState(() => _scope = scope);
    _search(query: _submittedQuery);
  }

  Future<void> _refresh() async {
    final String q = _submittedQuery;
    if (q.isEmpty) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    final generation = ++_generation;
    _searchCancel?.cancel('search refresh superseded');
    _loadMoreCancel?.cancel('search refresh superseded');
    final cancel = _searchCancel = CancelToken();
    setState(() => _loadingMore = false);
    try {
      final SearchPageProps props = await ref
          .read(topicRepositoryProvider)
          .search(
            query: q,
            scope: _scope == 'all' ? '' : _scope,
            page: 1,
            cancelToken: cancel,
          );
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() {
          _page = 1;
          _result = AsyncValue.data(props);
          _loadMoreError = null;
        });
      }
    } catch (e, st) {
      if (mounted &&
          generation == _generation &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        if (_result?.hasValue == true) {
          showGfToast(
            context,
            AppLocalizations.of(context).refreshFailedRetained,
            error: true,
          );
        } else {
          setState(() => _result = AsyncValue.error(e, st));
        }
      }
    } finally {
      if (identical(_searchCancel, cancel)) _searchCancel = null;
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      _generation++;
      _searchCancel?.cancel('search session changed');
      _loadMoreCancel?.cancel('search session changed');
      _historyGeneration++;
      setState(() {
        _recent = [];
        _result = null;
        _submittedQuery = '';
        _scope = 'all';
        _page = 1;
        _loadingMore = false;
        _loadMoreError = null;
      });
      _loadHistory();
    });
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String description =
        '$_submittedQuery · ${_scopeLabel(_scope, l10n)}';

    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.searchTitle)),
      body: Column(
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                GfSearchField(
                  controller: _query,
                  hintText: l10n.searchHint,
                  clearLabel: l10n.courseCopyClearSearch,
                  onSubmitted: (_) => _search(),
                  onClear: _clearSearch,
                ),
                if (MediaQuery.viewInsetsOf(context).bottom == 0)
                  Wrap(
                    spacing: 8,
                    runSpacing: 0,
                    children: [
                      TextButton.icon(
                        onPressed: () => _searchElsewhere('/courses'),
                        icon: const GfSymbol('graduation-cap', size: 18),
                        label: Text(l10n.searchCourses),
                      ),
                      TextButton.icon(
                        onPressed: () => _searchElsewhere('/wiki/search'),
                        icon: const GfSymbol('book-open', size: 18),
                        label: Text(l10n.searchWiki),
                      ),
                    ],
                  ),
                if (_submittedQuery.isNotEmpty) ...[
                  const SizedBox(height: 10),
                  Text(
                    description,
                    style: GfTheme.typographyOf(context).caption.copyWith(
                      color: GfTheme.colorsOf(context).iconMuted,
                    ),
                  ),
                ],
              ],
            ),
          ),
          if (_submittedQuery.isNotEmpty)
            _SearchScopeBar(
              selected: _scope,
              onSelected: _setScope,
              tabs: <_ScopeTab>[
                _ScopeTab('all', l10n.searchAll),
                _ScopeTab('topics', l10n.searchTopics),
                _ScopeTab('users', l10n.searchUsers),
                _ScopeTab('categories', l10n.searchCategories),
              ],
            ),
          Expanded(child: _buildResult(context)),
        ],
      ),
    );
  }

  Widget _buildResult(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final AsyncValue<SearchPageProps>? result = _result;
    if (result == null) {
      return ListView(
        padding: const EdgeInsets.fromLTRB(20, 24, 20, 32),
        children: [
          if (_recent.isNotEmpty) ...[
            Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.searchRecent,
                    style: GfTheme.typographyOf(context).title2,
                  ),
                ),
                TextButton(
                  onPressed: _clearHistory,
                  child: Text(l10n.searchClearRecent),
                ),
              ],
            ),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final query in _recent)
                  ActionChip(
                    label: Text(
                      query,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    onPressed: () {
                      _query.text = query;
                      _search();
                    },
                  ),
              ],
            ),
            const SizedBox(height: 24),
          ],
          Align(
            alignment: Alignment.centerLeft,
            child: GfIconTile('search', size: 56),
          ),
          const SizedBox(height: 24),
          Text(l10n.searchEmpty, style: GfTheme.typographyOf(context).title1),
          const SizedBox(height: 8),
          Text(
            l10n.searchDiscoveryDescription,
            style: GfTheme.typographyOf(context).body.copyWith(
              color: GfTheme.colorsOf(context).iconMuted,
              height: 1.5,
            ),
          ),
          const SizedBox(height: 32),
          Text(l10n.campusTools, style: GfTheme.typographyOf(context).title2),
          const SizedBox(height: 16),
          const CampusShortcuts(),
        ],
      );
    }
    return result.when(
      loading: () => const GfLoading(),
      error: (Object e, _) => GfErrorRetry(
        message: resolveErrorMessage(l10n, e),
        onRetry: () => _search(query: _submittedQuery),
      ),
      data: (SearchPageProps props) {
        if (props.searchUnavailable == true) {
          return GfEmpty(message: l10n.searchUnavailable);
        }
        return GfScrollToTop(
          semanticLabel: AppLocalizations.of(context).commonBackToTop,
          controller: _scrollToTopController,
          builder: (_, ScrollController controller) => _SearchResults(
            props: props,
            scope: _scope,
            loadingMore: _loadingMore,
            loadMoreError: _loadMoreError,
            hasMore: _page < props.totalPages,
            onLoadMore: _loadMore,
            onRefresh: _refresh,
            controller: controller,
          ),
        );
      },
    );
  }
}

class _ScopeTab {
  const _ScopeTab(this.value, this.label);

  final String value;
  final String label;
}

String _scopeLabel(String scope, AppLocalizations l10n) => switch (scope) {
  'topics' => l10n.searchTopics,
  'users' => l10n.searchUsers,
  'categories' => l10n.searchCategories,
  'courses' => l10n.coursesTitle,
  'all' => l10n.searchAll,
  _ => scope,
};

class _SearchScopeBar extends StatelessWidget {
  const _SearchScopeBar({
    required this.tabs,
    required this.selected,
    required this.onSelected,
  });

  final List<_ScopeTab> tabs;
  final String selected;
  final ValueChanged<String> onSelected;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
      child: Row(
        children: [
          for (final tab in tabs)
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: _ScopeButton(
                tab: tab,
                active: selected == tab.value,
                colors: colors,
                onTap: () => onSelected(tab.value),
              ),
            ),
        ],
      ),
    );
  }
}

class _ScopeButton extends StatelessWidget {
  const _ScopeButton({
    required this.tab,
    required this.active,
    required this.colors,
    required this.onTap,
  });

  final _ScopeTab tab;
  final bool active;
  final GfColors colors;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      selected: active,
      child: Material(
        color: active ? colors.neutral : colors.base200,
        borderRadius: BorderRadius.circular(999),
        child: InkWell(
          borderRadius: BorderRadius.circular(999),
          onTap: onTap,
          child: Container(
            constraints: const BoxConstraints(minHeight: 44),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              mainAxisAlignment: MainAxisAlignment.center,
              children: <Widget>[
                Flexible(
                  child: Text(
                    tab.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: active
                          ? colors.neutralContent
                          : colors.baseContent.withValues(alpha: 0.65),
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _SearchResults extends StatelessWidget {
  const _SearchResults({
    required this.props,
    required this.scope,
    required this.loadingMore,
    this.loadMoreError,
    required this.hasMore,
    required this.onLoadMore,
    required this.onRefresh,
    this.controller,
  });

  final SearchPageProps props;
  final String scope;
  final bool loadingMore;
  final String? loadMoreError;
  final bool hasMore;
  final VoidCallback onLoadMore;
  final Future<void> Function() onRefresh;
  final ScrollController? controller;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool showUsers =
        (scope == 'all' || scope == 'users') && props.users.isNotEmpty;
    final bool showTopics =
        (scope == 'all' || scope == 'topics') && props.topics.isNotEmpty;
    final bool showCategories =
        (scope == 'all' || scope == 'categories') &&
        props.categories.isNotEmpty;
    final bool hasResults = showUsers || showTopics || showCategories;

    final failed = props.failedScopes ?? const <String>[];
    final sections = <({String scope, int length, int total})>[
      if (showUsers)
        (scope: 'users', length: props.users.length, total: props.usersTotal),
      if (showTopics)
        (scope: 'topics', length: props.topics.length, total: props.total),
      if (showCategories)
        (
          scope: 'categories',
          length: props.categories.length,
          total: props.categoriesTotal,
        ),
    ];
    final showFooter = showTopics && props.totalPages > 1;
    final itemCount =
        (failed.isEmpty ? 0 : 1) +
        (hasResults
            ? sections.fold<int>(
                    0,
                    (count, section) => count + 1 + section.length,
                  ) +
                  (showFooter ? 1 : 0)
            : 1);

    // The delegate constructs individual rows, including after topic pages append.
    // No shrink-wrapped grid or whole-section Column expands the viewport cache.
    Widget buildRow(int index) {
      if (failed.isNotEmpty) {
        if (index == 0) return _PartialFailure(scopes: failed);
        index--;
      }
      if (!hasResults) {
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 44),
          child: GfEmpty(
            message: scope == 'users'
                ? l10n.searchNoUsers
                : scope == 'categories'
                ? l10n.searchNoCategories
                : '“${props.query}” · ${l10n.commonEmpty}',
          ),
        );
      }
      for (final section in sections) {
        if (index == 0) {
          return Semantics(
            header: true,
            child: GfSectionHeader(
              title: _scopeLabel(section.scope, l10n),
              description: l10n.searchResultCount(
                section.length,
                section.total,
              ),
              icon: switch (section.scope) {
                'users' => Icons.people_outline,
                'topics' => Icons.forum_outlined,
                _ => Icons.folder_open_outlined,
              },
            ),
          );
        }
        index--;
        if (index < section.length) {
          return switch (section.scope) {
            'users' => _UserRow(
              key: ValueKey('search-user-${props.users[index].id}'),
              user: props.users[index],
            ),
            'topics' => _TopicRow(
              key: ValueKey('search-topic-${props.topics[index].id}'),
              topic: props.topics[index],
            ),
            _ => _CategoryRow(
              key: ValueKey('search-category-${props.categories[index].id}'),
              category: props.categories[index],
            ),
          };
        }
        index -= section.length;
        if (section.scope == 'topics' && showFooter) {
          if (index == 0) {
            return GfListFooter(
              progressKey: props.topics.length,
              loading: loadingMore,
              error: loadMoreError,
              hasMore: hasMore,
              onLoadMore: onLoadMore,
            );
          }
          index--;
        }
      }
      throw RangeError.index(index, sections);
    }

    return AppRefreshIndicator(
      onRefresh: onRefresh,
      child: ListView.builder(
        controller: controller,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(12, 0, 12, 24),
        itemCount: itemCount,
        itemBuilder: (context, index) => Material(
          color: GfTheme.colorsOf(context).base100,
          child: buildRow(index),
        ),
      ),
    );
  }
}

class _PartialFailure extends StatelessWidget {
  const _PartialFailure({required this.scopes});

  final List<String> scopes;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 11),
      decoration: BoxDecoration(
        color: colors.warning.withValues(alpha: 0.05),
        border: Border.all(color: colors.warning.withValues(alpha: 0.3)),
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).field),
      ),
      child: Row(
        children: <Widget>[
          GfSymbol('circle-alert', size: 18, color: colors.warning),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              '${AppLocalizations.of(context).searchUnavailable}: ${scopes.map((scope) => _scopeLabel(scope, AppLocalizations.of(context))).join(', ')}',
              style: TextStyle(
                color: colors.baseContent.withValues(alpha: 0.75),
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _UserRow extends StatelessWidget {
  const _UserRow({super.key, required this.user});
  final UserSearchPayload user;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return InkWell(
      onTap: () => context.push('/u/${user.id}'),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border(bottom: BorderSide(color: colors.line)),
        ),
        child: Row(
          children: [
            GfAvatar(src: resolveApiAssetUrl(user.avatarUrl), size: 40),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    privateDisplayName(
                      context,
                      user.id,
                      user.username,
                      user.nickname,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    '@${user.username}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: colors.baseContent.withValues(alpha: 0.55),
                      fontSize: 12,
                    ),
                  ),
                  if (user.bio.isNotEmpty) ...[
                    const SizedBox(height: 2),
                    Text(
                      user.bio,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.55),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _TopicRow extends StatelessWidget {
  const _TopicRow({super.key, required this.topic});
  final TopicPayload topic;

  @override
  Widget build(BuildContext context) {
    return buildTopicFeedCard(context, topic);
  }
}

class _CategoryRow extends StatelessWidget {
  const _CategoryRow({super.key, required this.category});
  final CategorySearchPayload category;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return InkWell(
      onTap: () => context.push('/c/${category.slug}/${category.id}'),
      child: Container(
        constraints: const BoxConstraints(minHeight: 72),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border(bottom: BorderSide(color: colors.line)),
        ),
        child: Row(
          children: [
            Container(
              width: 36,
              height: 36,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: colorFromHex(category.color),
                borderRadius: BorderRadius.circular(
                  GfTheme.radiiOf(context).field,
                ),
              ),
              child: Text(
                category.icon.isEmpty ? '#' : category.icon,
                style: const TextStyle(fontSize: 17),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    category.name,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  if (category.desc.isNotEmpty) ...[
                    const SizedBox(height: 3),
                    Text(
                      category.desc,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.55),
                        fontSize: 12,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
