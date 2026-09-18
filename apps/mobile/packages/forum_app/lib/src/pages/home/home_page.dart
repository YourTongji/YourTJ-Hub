import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../format.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../server_messages.dart';
import '../../widgets/skeletons.dart';
import '../../widgets/status_views.dart';
import '../../widgets/topic_list.dart';
import '../../widgets/root_surface.dart';
import '../../widgets/brand_mark.dart';
import '../../widgets/announcement_banner.dart';

/// 首页:公告 + 话题流(web HomePage.vue 的移动端形态)。
class HomePage extends ConsumerStatefulWidget {
  const HomePage({super.key});

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

/// Per-action overrides keep a failed like independent of a concurrent bookmark.
typedef _InteractionOverride = ({
  int revision,
  int epoch,
  bool value,
  int? likeCount,
});

class _HomePageState extends ConsumerState<HomePage> {
  static const String _feedModeKey = 'goose:home-feed-mode';

  AsyncValue<HomeProps> _page = const AsyncValue.loading();
  String _sort = '';
  int _loadSequence = 0;
  int _interactionRevision = 0;
  final _pendingInteractions = <(int, bool)>{};
  final _interactionOverrides = <(int, bool), _InteractionOverride>{};

  // Pending writes and writes completed after a read started override that
  // response. A refresh started after completion remains authoritative.
  List<TopicPayload> _mergeInteractions(
    List<TopicPayload> incoming,
    int readRevision,
  ) => [for (final topic in incoming) _mergeInteraction(topic, readRevision)];

  TopicPayload _mergeInteraction(TopicPayload topic, int readRevision) {
    var result = topic;
    for (final bookmark in [false, true]) {
      final key = (topic.id, bookmark);
      final update = _interactionOverrides[key];
      if (update == null) continue;
      if (update.epoch != ref.read(offlineCacheEpochProvider) ||
          (!_pendingInteractions.contains(key) &&
              update.revision <= readRevision)) {
        _interactionOverrides.remove(key);
        continue;
      }
      result = bookmark
          ? result.copyWith(bookmarked: update.value)
          : result.copyWith(liked: update.value, likeCount: update.likeCount!);
    }
    return result;
  }

  final List<TopicPayload> _topics = <TopicPayload>[];
  bool _loadingMore = false;
  String? _loadMoreError;
  GfTopicFeedMode _feedMode = GfTopicFeedMode.card;
  List<CategoryNavPayload> _categories = const <CategoryNavPayload>[];
  final GfScrollToTopController _scrollToTopController =
      GfScrollToTopController();
  late final GfTabScrollRegistry _tabScrollRegistry;

  @override
  void initState() {
    super.initState();
    _tabScrollRegistry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.home, _scrollToTopController);
    _restoreFeedMode();
    _load();
  }

  @override
  void dispose() {
    _tabScrollRegistry.unregister(
      GfShellDestination.home,
      _scrollToTopController,
    );
    super.dispose();
  }

  Future<void> _restoreFeedMode() async {
    try {
      final SharedPreferences preferences =
          await SharedPreferences.getInstance();
      final String? stored = preferences.getString(_feedModeKey);
      if (!mounted || stored == null) return;
      setState(() {
        _feedMode = GfTopicFeedMode.values.firstWhere(
          (GfTopicFeedMode mode) => mode.name == stored,
          orElse: () => GfTopicFeedMode.card,
        );
      });
    } catch (_) {
      // Restricted storage keeps the web-compatible mobile default: card.
    }
  }

  Future<void> _setFeedMode(GfTopicFeedMode mode) async {
    if (_feedMode == mode) return;
    setState(() => _feedMode = mode);
    try {
      final SharedPreferences preferences =
          await SharedPreferences.getInstance();
      await preferences.setString(_feedModeKey, mode.name);
    } catch (_) {
      // The in-memory choice remains valid for this session.
    }
  }

  Future<void> _load({bool silent = false}) async {
    if (!mounted) return;
    final sequence = ++_loadSequence;
    final revision = _interactionRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    _loadingMore = false;
    _loadMoreError = null;
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .home(sort: _sort);
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final HomeProps? props = parsePageProps<HomeProps>(payload);
      if (props == null) throw const FormatException('home props');
      setState(() {
        _page = AsyncValue.data(props);
        _categories = payload.layout.sidebar.categories;
        _topics.clear();
        _topics.addAll(_mergeInteractions(props.topics, revision));
      });
    } catch (e, st) {
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if (silent && _page.hasValue) {
        showGfToast(
          context,
          AppLocalizations.of(context).refreshFailedRetained,
          error: true,
        );
      } else {
        setState(() => _page = AsyncValue.error(e, st));
      }
    }
  }

  /// 从话题详情返回:先消费详情页带回的话题增量(unseen/点赞/收藏/计数,
  /// 条目落在任何已加载页都生效),再后台请求第一页,按 id 原位更新已加载
  /// 话题的最新状态。不把第一页 payload 写入 _page——分页游标
  /// (nextUrl/hasNext)保留,「加载更多」从原进度继续;列表长度不回缩,
  /// 滚动位置不跳顶(MADR 0012 Preserve scroll position)。下拉刷新仍走
  /// _load(silent: true) 的整页重置语义,两条静默路径分开接线。
  Future<void> _refreshAfterReturn() async {
    if (!mounted) return;
    final revision = _interactionRevision;
    final Map<int, TopicReturnState> returned = Map.of(
      ref.read(topicReturnStatesProvider),
    );
    if (returned.isNotEmpty) {
      ref.read(topicReturnStatesProvider).clear();
      setState(() {
        for (var i = 0; i < _topics.length; i++) {
          final TopicReturnState? state = returned[_topics[i].id];
          if (state == null) continue;
          _topics[i] = _mergeInteraction(
            _topics[i].copyWith(
              unseen: state.unseen,
              liked: state.liked,
              bookmarked: state.bookmarked,
              likeCount: state.likeCount,
              replyCount: state.replyCount,
              viewCount: state.viewCount,
            ),
            revision,
          );
        }
      });
    }
    if (_topics.isEmpty) return;
    final sequence = ++_loadSequence;
    final epoch = ref.read(offlineCacheEpochProvider);
    // 在途「加载更多」已随序号失效,其 finally 的同序号守卫不会清理加载态,
    // 这里必须像 _load 一样接管,否则 _loadingMore 卡死、分页失效。
    _loadingMore = false;
    _loadMoreError = null;
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .home(sort: _sort);
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final HomeProps? props = parsePageProps<HomeProps>(payload);
      if (props == null) throw const FormatException('home props');
      final Map<int, TopicPayload> incoming = {
        for (final TopicPayload topic in props.topics) topic.id: topic,
      };
      setState(() {
        // 只原位替换已加载条目:不追加新热帖、不移除已消失条目,
        // 保证列表形状与滚动位置稳定。
        for (var i = 0; i < _topics.length; i++) {
          final TopicPayload? fresh = incoming[_topics[i].id];
          if (fresh != null) _topics[i] = _mergeInteraction(fresh, revision);
        }
      });
    } catch (_) {
      // 返回刷新失败:保留当前列表与分页进度,并按 Home 失败刷新的
      // 产品约定轻提示(docs/product/mobile-experience.md)。
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      showGfToast(
        context,
        AppLocalizations.of(context).refreshFailedRetained,
        error: true,
      );
    }
  }

  Future<void> _loadMore() async {
    final HomeProps? props = _page.value;
    if (props == null || !props.pagination.hasNext || _loadingMore) return;
    final String nextUrl = props.pagination.nextUrl;
    if (nextUrl.isEmpty) return;
    final sequence = _loadSequence;
    final revision = _interactionRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      _loadingMore = true;
      _loadMoreError = null;
    });
    try {
      // 真实分页:按后端 nextUrl 请求下一页(页面级数据通道)。
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch(nextUrl);
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final HomeProps? next = parsePageProps<HomeProps>(payload);
      if (next == null) throw const FormatException('home pagination');
      setState(() {
        final seen = _topics.map((topic) => topic.id).toSet();
        _topics.addAll(
          _mergeInteractions(
            next.topics.where((topic) => seen.add(topic.id)).toList(),
            revision,
          ),
        );
        _page = AsyncValue.data(next);
      });
    } catch (error) {
      if (mounted &&
          sequence == _loadSequence &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(
          () => _loadMoreError = resolveErrorMessage(
            AppLocalizations.of(context),
            error,
          ),
        );
      }
    } finally {
      if (mounted && sequence == _loadSequence) {
        setState(() => _loadingMore = false);
      }
    }
  }

  Future<bool> _toggleTopicInteraction(
    TopicPayload topic,
    bool target, {
    bool bookmark = false,
  }) async {
    if (!mounted) return false;
    final key = (topic.id, bookmark);
    if (!_pendingInteractions.add(key)) return false;
    final epoch = ref.read(offlineCacheEpochProvider);
    final index = _topics.indexWhere((t) => t.id == topic.id);
    final current = index >= 0 ? _topics[index] : topic;
    // 乐观更新先于请求落盘：图标与点赞计数立即切换并记录 override（并发刷新
    // 据此折叠）；失败按字段回滚，过期会话丢弃结果，避免跨会话污染。
    final snapshot = (
      liked: current.liked,
      bookmarked: current.bookmarked,
      likeCount: current.likeCount,
      override: _interactionOverrides[key],
    );
    final optimistic = (
      revision: ++_interactionRevision,
      epoch: epoch,
      value: target,
      likeCount: bookmark
          ? null
          : math.max(0, snapshot.likeCount + (target ? 1 : -1)),
    );
    setState(() {
      _interactionOverrides[key] = optimistic;
      if (index >= 0) {
        _topics[index] = bookmark
            ? current.copyWith(bookmarked: target)
            : current.copyWith(liked: target, likeCount: optimistic.likeCount!);
      }
    });
    try {
      final repository = ref.read(topicRepositoryProvider);
      final success = bookmark
          ? await repository.bookmarkTopic(
              topicId: topic.id,
              action: target ? 1 : 2,
            )
          : await repository.likeTopic(
              topicId: topic.id,
              action: target ? 1 : 2,
            );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
        return false;
      }
      if (!success) {
        _rollbackInteraction(topic.id, bookmark, snapshot);
        showGfToast(
          context,
          AppLocalizations.of(context).commonLoadFailed,
          error: true,
        );
        return false;
      }
      // A read begun before completion must still retain this action, even if
      // it began after the optimistic update. A subsequent refresh is authoritative.
      _interactionOverrides[key] = (
        revision: ++_interactionRevision,
        epoch: epoch,
        value: optimistic.value,
        likeCount: optimistic.likeCount,
      );
      return true;
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        _rollbackInteraction(topic.id, bookmark, snapshot);
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
      return false;
    } finally {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        _pendingInteractions.remove(key);
      }
    }
  }

  void _rollbackInteraction(
    int topicId,
    bool bookmark,
    ({
      bool? liked,
      bool? bookmarked,
      int likeCount,
      _InteractionOverride? override,
    })
    snapshot,
  ) {
    if (!mounted) return;
    setState(() {
      final key = (topicId, bookmark);
      if (snapshot.override == null) {
        _interactionOverrides.remove(key);
      } else {
        _interactionOverrides[key] = snapshot.override!;
      }
      final index = _topics.indexWhere((t) => t.id == topicId);
      if (index >= 0) {
        _topics[index] = bookmark
            ? _topics[index].copyWith(bookmarked: snapshot.bookmarked)
            : _topics[index].copyWith(
                liked: snapshot.liked,
                likeCount: snapshot.likeCount,
              );
      }
    });
  }

  void _switchSort(String sort) {
    if (sort == _sort) return;
    _sort = sort;
    _load();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen<int>(offlineCacheEpochProvider, (_, _) {
      _pendingInteractions.clear();
      _interactionOverrides.clear();
      _load();
    });
    final AppLocalizations l10n = AppLocalizations.of(context);
    return RootSurface(
      title: 'YourTJ',
      titleWidget: const YourTjMark(),
      actions: [
        IconButton(
          tooltip: l10n.commonSearch,
          icon: const GfSymbol('search'),
          onPressed: () => context.push('/search'),
        ),
      ],
      toolbarHeight:
          GfTabBar.heightFor(context) + (_categories.isEmpty ? 0 : 56),
      toolbar: _page.hasValue
          ? _HomeToolbar(
              props: _page.requireValue,
              categories: _categories,
              selected: _sort,
              feedMode: _feedMode,
              onSelected: _switchSort,
              onFeedModeSelected: _setFeedMode,
            )
          : const SizedBox.shrink(),
      body: (top, bottom) => _page.when(
        loading: () => Padding(
          padding: EdgeInsets.only(top: top),
          child: const GfTopicFeedSkeleton(),
        ),
        error: (e, _) =>
            GfErrorRetry(message: resolveErrorMessage(l10n, e), onRetry: _load),
        data: (props) => GfScrollToTop(
          semanticLabel: l10n.commonBackToTop,
          controller: _scrollToTopController,
          showButton: false,
          builder: (_, controller) => AppRefreshIndicator(
            edgeOffset: top,
            onRefresh: () => _load(silent: true),
            child: GfTopicList(
              loadMoreError: _loadMoreError,
              controller: controller,
              padding: EdgeInsets.only(top: top, bottom: bottom),
              header: AnnouncementBanner(announcement: props.announcement),
              loading: _loadingMore,
              topics: _topics,
              feedMode: _feedMode,
              onLikeTopic: _toggleTopicInteraction,
              onBookmarkTopic: (topic, target) =>
                  _toggleTopicInteraction(topic, target, bookmark: true),
              onReturnFromTopic: _refreshAfterReturn,
              hasMore: props.pagination.hasNext,
              onLoadMore: _loadMore,
            ),
          ),
        ),
      ),
    );
  }
}

class _HomeToolbar extends ConsumerWidget {
  const _HomeToolbar({
    required this.props,
    required this.categories,
    required this.selected,
    required this.feedMode,
    required this.onSelected,
    required this.onFeedModeSelected,
  });

  final HomeProps props;
  final List<CategoryNavPayload> categories;
  final String selected;
  final GfTopicFeedMode feedMode;
  final ValueChanged<String> onSelected;
  final ValueChanged<GfTopicFeedMode> onFeedModeSelected;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // 选中项:显式 selected 优先;为空时回退到服务端标记的 active tab。
    String effective = selected;
    if (effective.isEmpty) {
      for (final tab in props.tabs) {
        if (tab.active) {
          effective = tab.key;
          break;
        }
      }
    }
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);

    // Mobile keeps the Web information hierarchy in one compact row: sort
    // tabs on the left and the list/card view switch on the right. Publishing
    // already has a persistent center entry in the bottom navigation.
    return Container(
      color: colors.base100,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          // Keep the overlay inset and visible tabs at the same scaled height.
          SizedBox(
            height: GfTabBar.heightFor(context),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: <Widget>[
                  Expanded(
                    child: GfTabBar(
                      tabs: <GfTab>[
                        for (final tab in props.tabs)
                          GfTab(
                            // 后端 tabs[].label 可能为空(web 端按 key fallback 到
                            // i18n),空 label 会让选中态深色底渲染成黑块,必须兜底。
                            label: _sortTabLabel(
                              context,
                              tab.key,
                              tab.label ?? '',
                            ),
                            value: tab.key,
                          ),
                      ],
                      selected: effective,
                      onSelected: (Object value) => onSelected(value as String),
                    ),
                  ),
                  const SizedBox(width: 8),
                  PopupMenuButton<GfTopicFeedMode>(
                    tooltip: l10n.topicFeedModeList,
                    icon: const GfSymbol('sliders-horizontal', size: 20),
                    onSelected: onFeedModeSelected,
                    itemBuilder: (_) => [
                      PopupMenuItem(
                        value: GfTopicFeedMode.list,
                        child: Text(l10n.topicFeedModeList),
                      ),
                      PopupMenuItem(
                        value: GfTopicFeedMode.card,
                        child: Text(l10n.topicFeedModeCard),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          // 分类快捷入口(与 web 侧边栏 categories 同源):横向滑动 pills,
          // 点击跳转分类页;后端未配置分类时整行不占位。
          if (categories.isNotEmpty)
            SizedBox(
              height: 56,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 6,
                ),
                itemCount: categories.length,
                separatorBuilder: (_, _) => const SizedBox(width: 8),
                itemBuilder: (BuildContext context, int index) {
                  final CategoryNavPayload category = categories[index];
                  return _CategoryPill(
                    label: category.label,
                    color: colorFromHex(category.color),
                    onTap: () => context.push(category.url),
                  );
                },
              ),
            ),
        ],
      ),
    );
  }

  /// 排序 tab 文案:与 web `sortTabLabel(key, label)` 一致——后端 label
  /// 为空时按 key 回退到 i18n(最新/热门/流行)。
  String _sortTabLabel(BuildContext context, String key, String label) {
    if (label.isNotEmpty) return label;
    final AppLocalizations l10n = AppLocalizations.of(context);
    return switch (key) {
      'latest' => l10n.sortLatest,
      'hot' => l10n.sortHot,
      'popular' => l10n.sortPopular,
      _ => key,
    };
  }
}

/// 首页顶栏分类入口 pill:色点 + 分类名,镜像 GfChip 的视觉规格,
/// 但可点击并按内容自适应宽度,横向滑动承载多个分类。
class _CategoryPill extends StatelessWidget {
  const _CategoryPill({
    required this.label,
    required this.color,
    required this.onTap,
  });

  final String label;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Material(
      color: colors.base300,
      borderRadius: BorderRadius.circular(999),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(999),
        child: Container(
          constraints: const BoxConstraints(minHeight: 44),
          padding: const EdgeInsets.symmetric(horizontal: 12),
          alignment: Alignment.center,
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              Container(
                width: 6,
                height: 6,
                decoration: BoxDecoration(color: color, shape: BoxShape.circle),
              ),
              const SizedBox(width: 5),
              Text(
                label,
                style: TextStyle(
                  color: colors.baseContent.withValues(alpha: 0.75),
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
