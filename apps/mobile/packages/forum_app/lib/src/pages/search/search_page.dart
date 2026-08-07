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

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final String q = _query.text.trim();
    if (q.isEmpty) return;
    setState(() => _result = const AsyncValue.loading());
    try {
      final props = await ref
          .read(topicRepositoryProvider)
          .search(query: q, scope: _scope);
      setState(() {
        _result = AsyncValue.data(props);
        _searched = true;
      });
    } catch (e, st) {
      setState(() => _result = AsyncValue.error(e, st));
    }
  }

  void _setScope(String scope) {
    if (scope == _scope) return;
    _scope = scope;
    _search();
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
                  _ => GfTopicList(
                    loading: false,
                    topics: props.topics,
                    hasMore: props.totalPages > 1,
                    onLoadMore: () {},
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
        return ListTile(
          leading: CircleAvatar(
            radius: 18,
            backgroundImage: user.avatarUrl.isEmpty
                ? null
                : NetworkImage(user.avatarUrl),
            child: user.avatarUrl.isEmpty
                ? const Icon(Icons.person, size: 18)
                : null,
          ),
          title: Text(user.nickname.isEmpty ? user.username : user.nickname),
          subtitle: user.bio.isEmpty
              ? null
              : Text(user.bio, maxLines: 1, overflow: TextOverflow.ellipsis),
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
        return ListTile(
          leading: CircleAvatar(
            radius: 16,
            backgroundColor: colorFromHex(cat.color),
          ),
          title: Text(cat.name),
          subtitle: cat.desc.isEmpty
              ? null
              : Text(cat.desc, maxLines: 1, overflow: TextOverflow.ellipsis),
          onTap: () => context.push('/c/${cat.slug}/${cat.id}'),
        );
      },
    );
  }
}
