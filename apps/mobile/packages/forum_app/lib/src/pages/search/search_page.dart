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
import '../../navigation/tab_page_transition.dart';
import '../../navigation/tab_swipe_surface.dart';
import '../../widgets/skeletons.dart';
import '../../widgets/status_views.dart';
import '../../widgets/topic_list.dart';

/// 聚合搜索页：已提交查询与输入中的关键词分离，各类型结果逐项构建。
/// 空输入展示课程/Wiki 入口与最近搜索；首次提交前输入时给出明确的搜索去向；
/// 提交后以类型标签切换范围。
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

  Future<void> _forget(String query) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _recent = [..._recent]..remove(query));
    try {
      final owner = await ref.read(writingScopeProvider.future);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      await ref.read(writingStoreProvider).forget(owner, query);
    } catch (error) {
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    }
    if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
      await _loadHistory();
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
    final bool canPop = Navigator.canPop(context);

    return Scaffold(
      body: SafeArea(
        bottom: false,
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: _maxContentWidth),
            child: Column(
              children: <Widget>[
                Padding(
                  padding: EdgeInsetsDirectional.fromSTEB(
                    canPop ? 4 : 16,
                    8,
                    16,
                    8,
                  ),
                  child: Row(
                    children: <Widget>[
                      if (canPop) ...[
                        GfIconButton(
                          symbol: 'chevron-left',
                          iconSize: 24,
                          color: GfTheme.colorsOf(context).baseContent,
                          tooltip: MaterialLocalizations.of(
                            context,
                          ).backButtonTooltip,
                          onPressed: () => Navigator.maybePop(context),
                        ),
                        const SizedBox(width: 4),
                      ],
                      Expanded(
                        child: GfSearchField(
                          controller: _query,
                          autofocus: true,
                          hintText: l10n.searchHint,
                          clearLabel: l10n.courseCopyClearSearch,
                          onSubmitted: (_) => _search(),
                          onClear: _clearSearch,
                        ),
                      ),
                    ],
                  ),
                ),
                Expanded(
                  child: _submittedQuery.isEmpty
                      ? _buildBody(context)
                      : _buildScopedResults(context),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// Type tabs and results share the app-wide horizontal swipe: the body
  /// follows the finger and the tab indicator tracks the same progress.
  Widget _buildScopedResults(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final int index = _scopes.indexOf(_scope);
    return TabSwipeSurface(
      index: index,
      length: _scopes.length,
      onChanged: (next) => _setScope(_scopes[next]),
      tabSelectionDuration: GfMotion.duration(context, GfMotion.selection),
      child: Column(
        children: <Widget>[
          GfTabBar(
            distribute: true,
            selected: _scope,
            onSelected: (value) => _setScope(value as String),
            tabs: <GfTab>[
              for (final scope in _scopes)
                GfTab(
                  label: _scopeLabel(scope, l10n),
                  value: scope,
                  symbol: _scopeSymbol(scope),
                ),
            ],
          ),
          const GfDivider(),
          Expanded(
            child: TabPageTransition(
              index: index,
              length: _scopes.length,
              retainInactivePages: false,
              // Only the selected type is fetched; a neighbour revealed by a
              // drag shows the loading shape until it becomes current.
              pageBuilder: (page, _) => page == index
                  ? _buildBody(context)
                  : const GfTopicFeedSkeleton(),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final AsyncValue<SearchPageProps>? result = _result;
    if (result == null) {
      return ValueListenableBuilder<TextEditingValue>(
        valueListenable: _query,
        builder: (context, value, _) {
          final String typed = value.text.trim();
          if (typed.isEmpty) {
            return _Discovery(
              recent: _recent,
              onCourses: () => _searchElsewhere('/courses'),
              onWiki: () => _searchElsewhere('/wiki/search'),
              onRecent: _searchRecent,
              onForget: _forget,
              onClearRecent: _clearHistory,
            );
          }
          final String needle = typed.toLowerCase();
          return _Suggestions(
            query: typed,
            recent: <String>[
              for (final query in _recent)
                if (query != typed && query.toLowerCase().contains(needle))
                  query,
            ],
            onSearch: () => _search(query: typed),
            onCourses: () => _searchElsewhere('/courses'),
            onWiki: () => _searchElsewhere('/wiki/search'),
            onRecent: _searchRecent,
          );
        },
      );
    }
    return result.when(
      loading: () => const GfTopicFeedSkeleton(),
      error: (Object e, _) => GfErrorRetry(
        message: resolveErrorMessage(l10n, e),
        onRetry: () => _search(query: _submittedQuery),
      ),
      data: (SearchPageProps props) {
        if (props.searchUnavailable == true) {
          return GfEmpty(
            symbol: 'circle-alert',
            message: l10n.searchUnavailable,
            action: GfButton(
              label: l10n.commonRetry,
              variant: GfButtonVariant.outline,
              onPressed: () => _search(query: _submittedQuery),
            ),
          );
        }
        return GfScrollToTop(
          semanticLabel: l10n.commonBackToTop,
          controller: _scrollToTopController,
          builder: (_, ScrollController controller) => _SearchResults(
            props: props,
            scope: _scope,
            loadingMore: _loadingMore,
            loadMoreError: _loadMoreError,
            hasMore: _page < props.totalPages,
            onLoadMore: _loadMore,
            onRefresh: _refresh,
            onSeeAll: _setScope,
            onCourses: () => _searchElsewhere('/courses'),
            onWiki: () => _searchElsewhere('/wiki/search'),
            controller: controller,
          ),
        );
      },
    );
  }

  void _searchRecent(String query) {
    _query.value = TextEditingValue(
      text: query,
      selection: TextSelection.collapsed(offset: query.length),
    );
    _search(query: query);
  }
}

/// Keeps the search column readable on tablets and wide windows.
const double _maxContentWidth = 720;

const List<String> _scopes = <String>['all', 'topics', 'users', 'categories'];

String _scopeSymbol(String scope) => switch (scope) {
  'topics' => 'file-text',
  'users' => 'users-round',
  'categories' => 'folder',
  _ => 'layout-grid',
};

String _scopeLabel(String scope, AppLocalizations l10n) => switch (scope) {
  'topics' => l10n.searchTopics,
  'users' => l10n.searchUsers,
  'categories' => l10n.searchCategories,
  'courses' => l10n.coursesTitle,
  'all' => l10n.searchAll,
  _ => scope,
};

/// Course and Wiki search live on their own native pages; these tiles carry
/// the current input there. Equal halves keep the row balanced, and each tile
/// uses a faint wash of its destination's tone.
class _ElsewhereShortcuts extends StatelessWidget {
  const _ElsewhereShortcuts({
    required this.onCourses,
    required this.onWiki,
    this.compact = false,
  });

  final VoidCallback onCourses;
  final VoidCallback onWiki;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          Expanded(
            child: _ElsewhereTile(
              symbol: 'graduation-cap',
              tone: colors.primary,
              label: l10n.searchCourses,
              compact: compact,
              onTap: onCourses,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _ElsewhereTile(
              symbol: 'book-open',
              tone: colors.warning,
              label: l10n.searchWiki,
              compact: compact,
              onTap: onWiki,
            ),
          ),
        ],
      ),
    );
  }
}

class _ElsewhereTile extends StatelessWidget {
  const _ElsewhereTile({
    required this.symbol,
    required this.tone,
    required this.label,
    required this.compact,
    required this.onTap,
  });

  final String symbol;
  final Color tone;
  final String label;
  final bool compact;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool dark = Theme.of(context).brightness == Brightness.dark;
    return Semantics(
      button: true,
      child: Material(
        color: tone.withValues(alpha: dark ? 0.16 : 0.08),
        borderRadius: BorderRadius.circular(16),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          child: ConstrainedBox(
            constraints: BoxConstraints(minHeight: compact ? 48 : 56),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: <Widget>[
                  GfSymbol(symbol, size: 20, color: tone),
                  const SizedBox(width: 8),
                  Flexible(
                    child: Text(
                      label,
                      textAlign: TextAlign.center,
                      style: GfTheme.typographyOf(context).caption.copyWith(
                        fontSize: 15,
                        height: 1.3,
                        fontWeight: FontWeight.w600,
                        color: colors.baseContent,
                      ),
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

/// Empty input: destinations first, then the device-local history.
class _Discovery extends StatelessWidget {
  const _Discovery({
    required this.recent,
    required this.onCourses,
    required this.onWiki,
    required this.onRecent,
    required this.onForget,
    required this.onClearRecent,
  });

  final List<String> recent;
  final VoidCallback onCourses;
  final VoidCallback onWiki;
  final ValueChanged<String> onRecent;
  final ValueChanged<String> onForget;
  final VoidCallback onClearRecent;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.only(
        top: 4,
        bottom: 24 + MediaQuery.paddingOf(context).bottom,
      ),
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: _ElsewhereShortcuts(onCourses: onCourses, onWiki: onWiki),
        ),
        const SizedBox(height: 24),
        if (recent.isEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Text(
              l10n.searchIdleHint,
              style: type.small.copyWith(color: colors.iconMuted),
            ),
          )
        else ...<Widget>[
          Padding(
            padding: const EdgeInsetsDirectional.fromSTEB(16, 0, 4, 0),
            child: Row(
              children: <Widget>[
                Expanded(
                  child: Semantics(
                    header: true,
                    child: Text(l10n.searchRecent, style: type.heading),
                  ),
                ),
                TextButton(
                  onPressed: onClearRecent,
                  style: TextButton.styleFrom(
                    foregroundColor: colors.iconMuted,
                    textStyle: type.caption.copyWith(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  child: Text(l10n.searchClearRecent),
                ),
              ],
            ),
          ),
          for (final query in recent)
            _SuggestionRow(
              key: ValueKey('search-recent-$query'),
              symbol: 'history',
              label: query,
              onTap: () => onRecent(query),
              trailing: GfIconButton(
                symbol: 'x',
                iconSize: 16,
                tooltip: l10n.searchRemoveRecent,
                onPressed: () => onForget(query),
              ),
            ),
        ],
      ],
    );
  }
}

/// Typing before the first submission: explicit destinations for the typed
/// text plus matching history. Nothing is searched until a row is chosen.
class _Suggestions extends StatelessWidget {
  const _Suggestions({
    required this.query,
    required this.recent,
    required this.onSearch,
    required this.onCourses,
    required this.onWiki,
    required this.onRecent,
  });

  final String query;
  final List<String> recent;
  final VoidCallback onSearch;
  final VoidCallback onCourses;
  final VoidCallback onWiki;
  final ValueChanged<String> onRecent;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return ListView(
      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
      padding: EdgeInsets.only(
        top: 4,
        bottom: 24 + MediaQuery.paddingOf(context).bottom,
      ),
      children: <Widget>[
        _SuggestionRow(
          symbol: 'search',
          label: l10n.searchFor(query),
          strong: true,
          onTap: onSearch,
        ),
        _SuggestionRow(
          symbol: 'graduation-cap',
          label: l10n.searchCoursesFor(query),
          onTap: onCourses,
        ),
        _SuggestionRow(
          symbol: 'book-open',
          label: l10n.searchWikiFor(query),
          onTap: onWiki,
        ),
        if (recent.isNotEmpty) ...<Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 20, 16, 4),
            child: Semantics(
              header: true,
              child: Text(
                l10n.searchRecent,
                style: GfTheme.typographyOf(context).caption.copyWith(
                  fontWeight: FontWeight.w600,
                  color: GfTheme.colorsOf(context).iconMuted,
                ),
              ),
            ),
          ),
          for (final item in recent)
            _SuggestionRow(
              key: ValueKey('search-suggestion-$item'),
              symbol: 'history',
              label: item,
              onTap: () => onRecent(item),
            ),
        ],
      ],
    );
  }
}

class _SuggestionRow extends StatelessWidget {
  const _SuggestionRow({
    super.key,
    required this.symbol,
    required this.label,
    required this.onTap,
    this.strong = false,
    this.trailing,
  });

  final String symbol;
  final String label;
  final VoidCallback onTap;
  final bool strong;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return InkWell(
      onTap: onTap,
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 52),
        child: Padding(
          padding: EdgeInsetsDirectional.fromSTEB(
            16,
            6,
            trailing == null ? 16 : 4,
            6,
          ),
          child: Row(
            children: <Widget>[
              GfSymbol(
                symbol,
                size: 20,
                color: strong ? colors.baseContent : colors.iconMuted,
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Text(
                  label,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: GfTheme.typographyOf(context).body.copyWith(
                    fontSize: 16,
                    fontWeight: strong ? FontWeight.w600 : FontWeight.w400,
                  ),
                ),
              ),
              if (trailing != null) ...[const SizedBox(width: 4), trailing!],
            ],
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
    required this.onSeeAll,
    required this.onCourses,
    required this.onWiki,
    this.controller,
  });

  final SearchPageProps props;
  final String scope;
  final bool loadingMore;
  final String? loadMoreError;
  final bool hasMore;
  final VoidCallback onLoadMore;
  final Future<void> Function() onRefresh;
  final ValueChanged<String> onSeeAll;
  final VoidCallback onCourses;
  final VoidCallback onWiki;
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
    final bool showElsewhere = scope == 'all' && hasResults;

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
        (showElsewhere ? 1 : 0) +
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
          padding: const EdgeInsets.symmetric(vertical: 32),
          child: GfEmpty(
            symbol: 'search',
            message: scope == 'users'
                ? l10n.searchNoUsers
                : scope == 'categories'
                ? l10n.searchNoCategories
                : l10n.searchNoResults(props.query),
            description: l10n.searchNoResultsHint,
            action: _ElsewhereShortcuts(onCourses: onCourses, onWiki: onWiki),
          ),
        );
      }
      if (showElsewhere) {
        if (index == 0) {
          return Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
            child: _ElsewhereShortcuts(
              onCourses: onCourses,
              onWiki: onWiki,
              compact: true,
            ),
          );
        }
        index--;
      }
      for (final (position, section) in sections.indexed) {
        if (index == 0) {
          return _SectionHeader(
            title: _scopeLabel(section.scope, l10n),
            count: l10n.searchResultCount(section.length, section.total),
            first: position == 0 && !showElsewhere,
            onSeeAll: scope == 'all' && section.total > section.length
                ? () => onSeeAll(section.scope)
                : null,
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
        keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
        padding: EdgeInsets.only(
          bottom: 24 + MediaQuery.paddingOf(context).bottom,
        ),
        itemCount: itemCount,
        itemBuilder: (context, index) => Material(
          color: GfTheme.colorsOf(context).base100,
          child: buildRow(index),
        ),
      ),
    );
  }
}

/// Result group label: type and counts on one baseline; "See all" narrows the
/// aggregate view to that type.
class _SectionHeader extends StatelessWidget {
  const _SectionHeader({
    required this.title,
    required this.count,
    required this.first,
    this.onSeeAll,
  });

  final String title;
  final String count;
  final bool first;
  final VoidCallback? onSeeAll;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    return Padding(
      padding: EdgeInsetsDirectional.fromSTEB(
        16,
        first ? 12 : 24,
        onSeeAll == null ? 16 : 4,
        4,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: <Widget>[
          Flexible(
            child: Wrap(
              spacing: 8,
              crossAxisAlignment: WrapCrossAlignment.end,
              children: <Widget>[
                Semantics(header: true, child: Text(title, style: type.title3)),
                Padding(
                  padding: const EdgeInsets.only(bottom: 1),
                  child: Text(
                    count,
                    style: type.caption.copyWith(color: colors.iconMuted),
                  ),
                ),
              ],
            ),
          ),
          if (onSeeAll != null) ...<Widget>[
            const Spacer(),
            TextButton(
              onPressed: onSeeAll,
              style: TextButton.styleFrom(
                foregroundColor: colors.primary,
                textStyle: type.caption.copyWith(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                ),
              ),
              child: Text(AppLocalizations.of(context).searchSeeAll),
            ),
          ],
        ],
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
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: colors.warning.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).field),
        ),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Row(
            children: <Widget>[
              GfSymbol('circle-alert', size: 18, color: colors.warning),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  '${l10n.searchUnavailable}: ${scopes.map((scope) => _scopeLabel(scope, l10n)).join(', ')}',
                  style: GfTheme.typographyOf(context).caption.copyWith(
                    fontSize: 14,
                    color: colors.baseContent.withValues(alpha: 0.8),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _UserRow extends StatelessWidget {
  const _UserRow({super.key, required this.user});
  final UserSearchPayload user;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    return InkWell(
      onTap: () => context.push('/u/${user.id}'),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            GfAvatar(src: resolveApiAssetUrl(user.avatarUrl), size: 44),
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
                    style: type.bodyStrong.copyWith(height: 1.35),
                  ),
                  Text(
                    '@${user.username}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: type.caption.copyWith(color: colors.iconMuted),
                  ),
                  if (user.bio.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      user.bio,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: type.caption.copyWith(
                        fontSize: 14,
                        color: colors.baseContent.withValues(alpha: 0.8),
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
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final Color tone = colorFromHex(category.color);
    return InkWell(
      onTap: () => context.push('/c/${category.slug}/${category.id}'),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        child: Row(
          children: [
            Container(
              width: 44,
              height: 44,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: tone.withValues(alpha: 0.14),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Text(
                category.icon.isEmpty ? '#' : category.icon,
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                  color: tone,
                ),
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
                    style: type.bodyStrong.copyWith(height: 1.35),
                  ),
                  if (category.desc.isNotEmpty)
                    Text(
                      category.desc,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: type.caption.copyWith(
                        fontSize: 14,
                        color: colors.iconMuted,
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
