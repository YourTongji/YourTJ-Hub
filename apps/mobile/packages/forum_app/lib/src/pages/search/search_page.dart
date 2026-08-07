import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';
import '../../widgets/topic_list.dart';

/// 聚合搜索页(web SearchPage.vue 的移动端形态):
/// 帖子/用户/分类三域搜索(SearchJSON)。
class SearchPage extends ConsumerStatefulWidget {
  const SearchPage({super.key});

  @override
  ConsumerState<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends ConsumerState<SearchPage> {
  final TextEditingController _query = TextEditingController();
  AsyncValue<SearchPageProps> _result = const AsyncValue.loading();
  String _scope = '';
  bool _searched = false;
  int _page = 1;
  bool _loadingMore = false;

  @override
  void dispose() {
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
      final props = await ref
          .read(topicRepositoryProvider)
          .search(query: q, scope: _scope, page: _page);
      setState(() {
        _result = AsyncValue.data(props);
        _searched = true;
      });
    } catch (e, st) {
      setState(() => _result = AsyncValue.error(e, st));
    }
  }

  /// 真实分页:加载下一页追加到结果(web 无限滚动语义)。
  Future<void> _loadMore() async {
    final SearchPageProps? props = _result.value;
    if (props == null || _loadingMore || _page >= props.totalPages) return;
    setState(() => _loadingMore = true);
    try {
      final next = await ref
          .read(topicRepositoryProvider)
          .search(query: props.query, scope: _scope, page: _page + 1);
      setState(() {
        _page += 1;
        _result = AsyncValue.data(
          next.copyWith(
            topics: <TopicPayload>[...props.topics, ...next.topics],
          ),
        );
      });
    } catch (_) {
      // 静默,用户可再次触发。
    } finally {
      setState(() => _loadingMore = false);
    }
  }

  void _setScope(String scope) {
    if (scope == _scope) return;
    _scope = scope;
    _search();
  }

  /// 下拉刷新:重载第一页(不置全页 loading,保留列表)。
  Future<void> _refreshTopics() async {
    final String q = _query.text.trim();
    if (q.isEmpty) return;
    setState(() {
      _page = 1;
      _loadingMore = true;
    });
    try {
      final props = await ref
          .read(topicRepositoryProvider)
          .search(query: q, scope: _scope, page: 1);
      if (mounted) {
        setState(() {
          _result = AsyncValue.data(props);
          _searched = true;
        });
      }
    } catch (e, st) {
      if (mounted) {
        setState(() => _result = AsyncValue.error(e, st));
      }
    } finally {
      if (mounted) setState(() => _loadingMore = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(
        title: TextField(
          controller: _query,
          decoration: InputDecoration(
            hintText: l10n.searchHint,
            border: InputBorder.none,
            suffixIcon: IconButton(
              icon: const Icon(Icons.search),
              onPressed: _search,
            ),
          ),
          onSubmitted: (_) => _search(),
        ),
      ),
      body: _result.when(
        loading: () =>
            _searched ? const GfLoading() : GfEmpty(message: l10n.searchEmpty),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _search),
        data: (props) {
          if (props.searchUnavailable == true) {
            return GfEmpty(message: l10n.searchUnavailable);
          }
          return Column(
            children: [
              // Scope tabs。
              Container(
                height: 44,
                alignment: Alignment.centerLeft,
                padding: const EdgeInsets.symmetric(horizontal: 8),
                child: GfTabBar(
                  tabs: <GfTab>[
                    GfTab(
                      label: '${l10n.searchAll} (${props.total})',
                      value: '',
                    ),
                    GfTab(
                      label: '${l10n.searchTopics} (${props.topics.length})',
                      value: 'topic',
                    ),
                    GfTab(
                      label: '${l10n.searchUsers} (${props.usersTotal})',
                      value: 'user',
                    ),
                    GfTab(
                      label:
                          '${l10n.searchCategories} (${props.categoriesTotal})',
                      value: 'category',
                    ),
                  ],
                  selected: _scope,
                  onSelected: (Object value) => _setScope(value as String),
                ),
              ),
              Expanded(
                child: switch (_scope) {
                  'user' => _UserResults(users: props.users),
                  'category' => _CategoryResults(categories: props.categories),
                  _ => RefreshIndicator(
                    onRefresh: _refreshTopics,
                    child: GfTopicList(
                      loading: _loadingMore,
                      topics: props.topics,
                      hasMore: _page < props.totalPages,
                      onLoadMore: _loadMore,
                    ),
                  ),
                },
              ),
            ],
          );
        },
      ),
    );
  }
}

class _UserResults extends StatelessWidget {
  const _UserResults({required this.users});

  final List<UserSearchPayload> users;

  @override
  Widget build(BuildContext context) {
    if (users.isEmpty) {
      return GfEmpty(message: AppLocalizations.of(context).searchNoUsers);
    }
    return ListView.separated(
      itemCount: users.length,
      separatorBuilder: (_, _) => const Divider(height: 1),
      itemBuilder: (context, i) {
        final user = users[i];
        return GfSettingRow(
          leading: GfAvatar(src: user.avatarUrl, size: 36),
          title: user.nickname.isEmpty ? user.username : user.nickname,
          description: user.bio.isEmpty ? null : user.bio,
          onTap: () => context.push('/u/${user.id}'),
        );
      },
    );
  }
}

class _CategoryResults extends StatelessWidget {
  const _CategoryResults({required this.categories});

  final List<CategorySearchPayload> categories;

  @override
  Widget build(BuildContext context) {
    if (categories.isEmpty) {
      return GfEmpty(message: AppLocalizations.of(context).searchNoCategories);
    }
    return ListView.separated(
      itemCount: categories.length,
      separatorBuilder: (_, _) => const Divider(height: 1),
      itemBuilder: (context, i) {
        final cat = categories[i];
        return GfSettingRow(
          leading: CircleAvatar(
            radius: 16,
            backgroundColor: colorFromHex(cat.color),
          ),
          title: cat.name,
          description: cat.desc.isEmpty ? null : cat.desc,
          onTap: () => context.push('/c/${cat.slug}/${cat.id}'),
        );
      },
    );
  }
}
