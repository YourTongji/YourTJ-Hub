import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import '../../asset_url.dart';
import '../../server_messages.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../widgets/status_views.dart';

/// 聚合搜索页。结构与 Web SearchPage.vue 保持一致：页面头搜索框、
/// 四列范围胶囊，以及同一结果卡片中的用户/帖子/分类分组。
class SearchPage extends ConsumerStatefulWidget {
  const SearchPage({super.key});

  @override
  ConsumerState<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends ConsumerState<SearchPage> {
  final TextEditingController _query = TextEditingController();
  AsyncValue<SearchPageProps>? _result;
  String _scope = 'all';
  int _page = 1;
  bool _loadingMore = false;
  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();
  late final GfTabScrollRegistry _tabScrollRegistry;

  @override
  void initState() {
    super.initState();
    _tabScrollRegistry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.search, _scrollToTopController);
  }

  @override
  void dispose() {
    _tabScrollRegistry.unregister(
      GfShellDestination.search,
      _scrollToTopController,
    );
    _query.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final String q = _query.text.trim();
    if (q.isEmpty) return;
    setState(() {
      _result = const AsyncValue.loading();
      _page = 1;
    });
    try {
      final SearchPageProps props = await ref
          .read(topicRepositoryProvider)
          .search(query: q, scope: _scope == 'all' ? '' : _scope, page: 1);
      if (mounted) setState(() => _result = AsyncValue.data(props));
    } catch (e, st) {
      if (mounted) setState(() => _result = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadMore() async {
    final SearchPageProps? props = _result?.value;
    if (props == null || _loadingMore || _page >= props.totalPages) return;
    setState(() => _loadingMore = true);
    try {
      final SearchPageProps next = await ref
          .read(topicRepositoryProvider)
          .search(
            query: props.query,
            scope: _scope == 'all' ? '' : _scope,
            page: _page + 1,
          );
      if (mounted) {
        setState(() {
          _page += 1;
          _result = AsyncValue.data(
            next.copyWith(
              topics: <TopicPayload>[...props.topics, ...next.topics],
            ),
          );
        });
      }
    } catch (_) {
      // 保留当前结果，用户可再次触发加载。
    } finally {
      if (mounted) setState(() => _loadingMore = false);
    }
  }

  void _setScope(String scope) {
    if (scope == _scope) return;
    setState(() => _scope = scope);
    _search();
  }

  Future<void> _refresh() async {
    final String q = _query.text.trim();
    if (q.isEmpty) return;
    try {
      final SearchPageProps props = await ref
          .read(topicRepositoryProvider)
          .search(query: q, scope: _scope == 'all' ? '' : _scope, page: 1);
      if (mounted) {
        setState(() {
          _page = 1;
          _result = AsyncValue.data(props);
        });
      }
    } catch (e, st) {
      if (mounted) setState(() => _result = AsyncValue.error(e, st));
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final SearchPageProps? props = _result?.value;
    final String description = props == null
        ? l10n.searchEmpty
        : '${props.query} · ${formatNumber(_scope == 'users'
              ? props.usersTotal
              : _scope == 'categories'
              ? props.categoriesTotal
              : props.total)}';

    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.searchTitle)),
      body: Column(
        children: <Widget>[
          GfPageHeader(
            title: l10n.searchTitle,
            description: description,
            badge: _HeaderBadge(label: l10n.searchTitle),
            actions: GfInput(
              controller: _query,
              hintText: l10n.searchHint,
              textInputAction: TextInputAction.search,
              prefixIcon: const Icon(Icons.manage_search_outlined, size: 18),
              suffixIcon: GfIconButton(
                icon: Icons.search,
                tooltip: l10n.searchTitle,
                onPressed: _search,
              ),
              onSubmitted: (_) => _search(),
            ),
          ),
          if (props != null && props.searchUnavailable != true)
            _SearchScopeBar(
              selected: _scope,
              onSelected: _setScope,
              tabs: <_ScopeTab>[
                _ScopeTab('all', l10n.searchAll, props.total),
                _ScopeTab('topics', l10n.searchTopics, props.total),
                _ScopeTab('users', l10n.searchUsers, props.usersTotal),
                _ScopeTab(
                  'categories',
                  l10n.searchCategories,
                  props.categoriesTotal,
                ),
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
      return GfEmpty(message: l10n.searchEmpty);
    }
    return result.when(
      loading: () => const GfLoading(),
      error: (Object e, _) =>
          GfErrorRetry(message: resolveErrorMessage(l10n, e), onRetry: _search),
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

class _HeaderBadge extends StatelessWidget {
  const _HeaderBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Container(
      height: 20,
      padding: const EdgeInsets.symmetric(horizontal: 7),
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: colors.base300,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label.toUpperCase(),
        style: TextStyle(
          color: colors.baseContent.withValues(alpha: 0.55),
          fontSize: 11,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.3,
        ),
      ),
    );
  }
}

class _ScopeTab {
  const _ScopeTab(this.value, this.label, this.count);

  final String value;
  final String label;
  final int count;
}

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
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 12),
      child: Row(
        children: <Widget>[
          for (int index = 0; index < tabs.length; index++) ...<Widget>[
            if (index > 0) const SizedBox(width: 6),
            Expanded(
              child: _ScopeButton(
                tab: tabs[index],
                active: tabs[index].value == selected,
                colors: colors,
                onTap: () => onSelected(tabs[index].value),
              ),
            ),
          ],
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
          child: SizedBox(
            height: 36,
            child: Row(
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
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                const SizedBox(width: 3),
                Text(
                  formatNumber(tab.count),
                  style: TextStyle(
                    color: active
                        ? colors.neutralContent.withValues(alpha: 0.7)
                        : colors.baseContent.withValues(alpha: 0.4),
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
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
    required this.hasMore,
    required this.onLoadMore,
    required this.onRefresh,
    this.controller,
  });

  final SearchPageProps props;
  final String scope;
  final bool loadingMore;
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

    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView(
        controller: controller,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(12, 0, 12, 24),
        children: <Widget>[
          if ((props.failedScopes ?? const <String>[]).isNotEmpty)
            _PartialFailure(scopes: props.failedScopes!),
          GfCard(
            emphasized: true,
            child: hasResults
                ? Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: <Widget>[
                      if (showUsers) ...<Widget>[
                        GfSectionHeader(
                          title: l10n.searchUsers,
                          description: formatNumber(props.usersTotal),
                          icon: Icons.people_outline,
                        ),
                        _UserRows(users: props.users),
                      ],
                      if (showTopics) ...<Widget>[
                        if (scope == 'all')
                          GfSectionHeader(
                            title: l10n.searchTopics,
                            description: formatNumber(props.total),
                            icon: Icons.forum_outlined,
                          ),
                        _TopicRows(topics: props.topics),
                        if (props.totalPages > 1)
                          GfListFooter(
                            loading: loadingMore,
                            hasMore: hasMore,
                            onLoadMore: onLoadMore,
                          ),
                      ],
                      if (showCategories) ...<Widget>[
                        GfSectionHeader(
                          title: l10n.searchCategories,
                          description: formatNumber(props.categoriesTotal),
                          icon: Icons.folder_open_outlined,
                        ),
                        _CategoryGrid(categories: props.categories),
                      ],
                    ],
                  )
                : Padding(
                    padding: const EdgeInsets.symmetric(vertical: 44),
                    child: GfEmpty(
                      message: scope == 'users'
                          ? l10n.searchNoUsers
                          : scope == 'categories'
                          ? l10n.searchNoCategories
                          : '“${props.query}” · ${l10n.commonEmpty}',
                    ),
                  ),
          ),
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
          Icon(Icons.warning_amber_rounded, size: 18, color: colors.warning),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              '${AppLocalizations.of(context).searchUnavailable}: ${scopes.join(', ')}',
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

class _UserRows extends StatelessWidget {
  const _UserRows({required this.users});

  final List<UserSearchPayload> users;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Column(
      children: <Widget>[
        for (int index = 0; index < users.length; index++) ...<Widget>[
          InkWell(
            onTap: () => context.push('/u/${users[index].id}'),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              child: Row(
                children: <Widget>[
                  GfAvatar(
                    src: resolveApiAssetUrl(users[index].avatarUrl),
                    size: 40,
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text(
                          users[index].nickname.isEmpty
                              ? users[index].username
                              : users[index].nickname,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          '@${users[index].username}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: colors.baseContent.withValues(alpha: 0.55),
                            fontSize: 12,
                          ),
                        ),
                        if (users[index].bio.isNotEmpty) ...<Widget>[
                          const SizedBox(height: 2),
                          Text(
                            users[index].bio,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: colors.baseContent.withValues(alpha: 0.45),
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
          ),
          if (index < users.length - 1) Divider(height: 1, color: colors.line),
        ],
      ],
    );
  }
}

class _TopicRows extends StatelessWidget {
  const _TopicRows({required this.topics});

  final List<TopicPayload> topics;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Column(
      children: <Widget>[
        for (int index = 0; index < topics.length; index++)
          GfTopicRow(
            title: topics[index].title,
            description: topics[index].description,
            categories: <GfTopicCategory>[
              for (final CategoryBriefPayload category
                  in topics[index].categories)
                GfTopicCategory(
                  name: category.name,
                  color: colorFromHex(category.color),
                ),
            ],
            participantAvatarUrls: <String>[
              for (final UserBriefPayload participant
                  in topics[index].participants)
                resolveApiAssetUrl(participant.avatarUrl),
            ],
            activityText: timeAgo(topics[index].activityText, l10n: l10n),
            replyCount: topics[index].replyCount,
            viewCount: topics[index].viewCount,
            hot: topics[index].viewCount > 500,
            pinned: topics[index].pinWeight > 0,
            unseen: topics[index].unseen == true,
            showDivider: index < topics.length - 1,
            onTap: () => context.push('/p/${topics[index].id}'),
          ),
      ],
    );
  }
}

class _CategoryGrid extends StatelessWidget {
  const _CategoryGrid({required this.categories});

  final List<CategorySearchPayload> categories;

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      padding: const EdgeInsets.all(12),
      itemCount: categories.length,
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        crossAxisSpacing: 8,
        mainAxisSpacing: 8,
        mainAxisExtent: 82,
      ),
      itemBuilder: (BuildContext context, int index) {
        final CategorySearchPayload category = categories[index];
        final GfColors colors = GfTheme.colorsOf(context);
        return Material(
          color: colors.base100,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).field),
            side: BorderSide(color: colors.line),
          ),
          clipBehavior: Clip.antiAlias,
          child: InkWell(
            onTap: () => context.push('/c/${category.slug}/${category.id}'),
            child: Padding(
              padding: const EdgeInsets.all(10),
              child: Row(
                children: <Widget>[
                  Container(
                    width: 36,
                    height: 36,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: colorFromHex(category.color),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      category.icon.isEmpty ? '#' : category.icon,
                      style: const TextStyle(fontSize: 17),
                    ),
                  ),
                  const SizedBox(width: 9),
                  Expanded(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text(
                          category.name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        if (category.desc.isNotEmpty) ...<Widget>[
                          const SizedBox(height: 3),
                          Text(
                            category.desc,
                            maxLines: 1,
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
          ),
        );
      },
    );
  }
}
