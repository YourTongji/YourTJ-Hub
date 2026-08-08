import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';
import '../../widgets/topic_list.dart';

/// 分类页(web CategoryPage.vue 的移动端形态)。
class CategoryPage extends ConsumerStatefulWidget {
  const CategoryPage({super.key, required this.slug, required this.categoryId});

  final String slug;
  final int categoryId;

  @override
  ConsumerState<CategoryPage> createState() => _CategoryPageState();
}

class _CategoryPageState extends ConsumerState<CategoryPage> {
  AsyncValue<CategoryPageProps> _page = const AsyncValue.loading();
  final List<TopicPayload> _topics = <TopicPayload>[];
  bool _loadingMore = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .category(widget.slug, widget.categoryId);
      final props = parsePageProps<CategoryPageProps>(payload);
      if (props == null) {
        setState(
          () => _page = AsyncValue.error(
            AppLocalizations.of(context).commonParseFailed,
            StackTrace.current,
          ),
        );
        return;
      }
      setState(() {
        _page = AsyncValue.data(props);
        _topics.clear();
        _topics.addAll(props.topics);
      });
    } catch (e, st) {
      setState(() => _page = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadMore() async {
    final CategoryPageProps? props = _page.value;
    if (props == null || !props.pagination.hasNext || _loadingMore) return;
    final String nextUrl = props.pagination.nextUrl;
    if (nextUrl.isEmpty) return;
    setState(() => _loadingMore = true);
    try {
      // 真实分页:按后端 nextUrl 请求下一页(页面级数据通道)。
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch(nextUrl);
      final CategoryPageProps? next = parsePageProps<CategoryPageProps>(
        payload,
      );
      if (next != null && next.topics.isNotEmpty) {
        setState(() {
          _topics.addAll(next.topics);
          _page = AsyncValue.data(next);
        });
      }
    } catch (_) {
      // 加载更多失败静默(用户可再次点击)。
    } finally {
      setState(() => _loadingMore = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: GfAppBar(title: Text(AppLocalizations.of(context).categoryTitle)),
      body: _page.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (props) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(
                          Icons.folder_outlined,
                          size: 18,
                          color: colorFromHex(props.category.color),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          props.category.name,
                          style: GfTheme.typographyOf(context).title2,
                        ),
                      ],
                    ),
                    if (props.category.description.isNotEmpty) ...[
                      const SizedBox(height: 6),
                      Text(
                        props.category.description,
                        style: GfTheme.typographyOf(context).small.copyWith(
                          color: GfTheme.colorsOf(context).iconMuted,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () => _load(silent: true),
                  child: GfTopicList(
                    loading: _loadingMore,
                    topics: _topics,
                    hasMore: props.pagination.hasNext,
                    onLoadMore: _loadMore,
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
