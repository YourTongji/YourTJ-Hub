import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:dio/dio.dart';
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
import '../../widgets/announcement_banner.dart';
import '../../app_config.dart';
import '../../current_user.dart';
import '../../offline/drift_cache.dart';
import '../../startup_metrics.dart';

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

/// Each visited sort owns its request generation, pagination and reading
/// position. Switching tabs never supersedes another tab's in-flight read.
class _HomeFeedState {
  _HomeFeedState(this.sort, {this.category});

  final String sort;
  final CategoryNavPayload? category;
  AsyncValue<HomeProps> page = const AsyncValue.loading();
  final List<TopicPayload> topics = [];
  bool loadingMore = false;
  String? loadMoreError;
  int loadSequence = 0;
  CancelToken? loadCancel;
  CancelToken? loadMoreCancel;

  void cancel() {
    loadCancel?.cancel("home feed disposed");
    loadMoreCancel?.cancel("home feed disposed");
  }

  final scrollToTop = GfScrollToTopController();
}

class _HomePageState extends ConsumerState<HomePage> {
  static const String _feedModeKey = 'goose:home-feed-mode';

  String _sort = '';
  CategoryNavPayload? _category;
  String _allSort = '';
  final _categorySorts = <int, String>{};
  String _key(String sort, CategoryNavPayload? category) =>
      category == null ? sort : 'category:${category.id}:$sort';
  String get _activeKey => _key(_sort, _category);
  final _feeds = <String, _HomeFeedState>{'': _HomeFeedState('')};
  _HomeFeedState get _activeFeed => _feeds[_activeKey]!;
  HomeProps? _navigationProps;
  bool _announcementCollapsed = true;
  int _interactionRevision = 0;
  bool _firstCardFrameRecorded = false;
  final _pendingInteractions = <(int, bool)>{};
  final _interactionOverrides = <(int, bool), _InteractionOverride>{};
  final _returnedTopicOverrides =
      <int, ({int revision, int epoch, TopicReturnState state})>{};

  // Pending writes and writes completed after a read started override that
  // response. A refresh started after completion remains authoritative.
  List<TopicPayload> _mergeInteractions(
    List<TopicPayload> incoming,
    int readRevision,
  ) => [for (final topic in incoming) _mergeInteraction(topic, readRevision)];

  TopicPayload _mergeInteraction(TopicPayload topic, int readRevision) {
    var result = topic;
    final returned = _returnedTopicOverrides[topic.id];
    final returnRevision =
        returned != null &&
            returned.epoch == ref.read(offlineCacheEpochProvider) &&
            returned.revision > readRevision
        ? returned.revision
        : 0;
    if (returnRevision > 0) {
      final state = returned!.state;
      result = result.copyWith(
        unseen: state.unseen,
        liked: state.liked,
        bookmarked: state.bookmarked,
        likeCount: state.likeCount,
        replyCount: state.replyCount,
        viewCount: state.viewCount,
      );
    }
    for (final bookmark in [false, true]) {
      final key = (topic.id, bookmark);
      final update = _interactionOverrides[key];
      if (update == null) continue;
      if (update.epoch != ref.read(offlineCacheEpochProvider)) {
        _interactionOverrides.remove(key);
        continue;
      }
      // Keep the revision fence for reads still running in other sorts.
      if (!_pendingInteractions.contains(key) &&
          update.revision <= math.max(readRevision, returnRevision)) {
        continue;
      }
      result = bookmark
          ? result.copyWith(bookmarked: update.value)
          : result.copyWith(liked: update.value, likeCount: update.likeCount!);
    }
    return result;
  }

  GfTopicFeedMode _feedMode = GfTopicFeedMode.card;
  List<CategoryNavPayload> _categories = const <CategoryNavPayload>[];
  GfScrollToTopController get _scrollToTopController => _activeFeed.scrollToTop;
  late final GfTabScrollRegistry _tabScrollRegistry;

  @override
  void initState() {
    super.initState();
    _tabScrollRegistry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.home, _scrollToTopController);
    _restoreFeedMode();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        recordStartupMilestone('startup.home_skeleton_frame_submitted');
      }
    });
    _load();
  }

  @override
  void dispose() {
    for (final feed in _feeds.values) {
      feed.cancel();
    }
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

  void _setAnnouncementCollapsed(bool collapsed) {
    if (_announcementCollapsed == collapsed) return;
    setState(() => _announcementCollapsed = collapsed);
  }

  Future<PagePayload> _fetchFeed(_HomeFeedState feed, CancelToken cancel) {
    final repository = ref.read(pageRepositoryProvider);
    final category = feed.category;
    if (category == null) {
      return repository.home(sort: feed.sort, cancelToken: cancel);
    }
    final uri = Uri.parse(category.url);
    final path = feed.sort.isEmpty || feed.sort == 'latest'
        ? uri.path
        : '${uri.path}/l/${Uri.encodeComponent(feed.sort)}';
    return repository.fetch(
      uri.replace(path: path).toString(),
      cancelToken: cancel,
    );
  }

  HomeProps? _feedProps(PagePayload payload, _HomeFeedState feed) {
    if (feed.category == null) return parsePageProps<HomeProps>(payload);
    final category = parsePageProps<CategoryPageProps>(payload);
    if (category == null || category.category.id != feed.category!.id) {
      return null;
    }
    return HomeProps(
      sort: category.sort,
      tabs: category.tabs,
      topics: category.topics,
      pagination: category.pagination,
      announcement: const AnnouncementPayload(enabled: false, html: ''),
    );
  }

  Future<void> _load({bool silent = false, _HomeFeedState? target}) async {
    final feed = target ?? _activeFeed;
    if (!mounted) return;
    final sequence = ++feed.loadSequence;
    final revision = _interactionRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    feed.loadCancel?.cancel('home request superseded');
    feed.loadMoreCancel?.cancel('home refresh superseded');
    feed.loadingMore = false;
    feed.loadMoreError = null;
    final cancel = feed.loadCancel = CancelToken();
    final requestedSort = feed.sort;
    final cacheScopeFuture = feed.category == null
        ? _homeCacheScope()
        : Future<(int, String)?>.value(null);
    var cachedPageShown = false;
    var networkPageShown = false;
    if (!silent) setState(() => feed.page = const AsyncValue.loading());
    final cacheShownFuture =
        (!silent
                ? () async {
                    try {
                      final scope = await cacheScopeFuture;
                      if (scope == null) return false;
                      final cache = ref.read(offlineTopicCacheProvider);
                      if (cache is! OfflineHomeCache) return false;
                      final cached = await (cache as OfflineHomeCache)
                          .getHomePage(
                            accountId: scope.$1,
                            baseUrl: scope.$2,
                            sort: requestedSort,
                          );
                      if (cached == null ||
                          networkPageShown ||
                          !mounted ||
                          sequence != feed.loadSequence ||
                          epoch != ref.read(offlineCacheEpochProvider)) {
                        return false;
                      }
                      final cachedProps = parsePageProps<HomeProps>(cached);
                      if (cachedProps == null) return false;
                      setState(() {
                        feed.page = AsyncValue.data(cachedProps);
                        _navigationProps ??= cachedProps;
                        _categories = cached.layout.sidebar.categories;
                        feed.topics
                          ..clear()
                          ..addAll(
                            _mergeInteractions(cachedProps.topics, revision),
                          );
                      });
                      _recordFirstHomeContent(cachedProps.topics.isNotEmpty);
                      return true;
                    } catch (_) {
                      return false;
                    }
                  }()
                : Future<bool>.value(false))
            .then((shown) => cachedPageShown = shown);
    try {
      final PagePayload payload = await _fetchFeed(feed, cancel);
      if (!mounted ||
          sequence != feed.loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final HomeProps? props = _feedProps(payload, feed);
      if (props == null) throw const FormatException('home props');
      networkPageShown = true;
      setState(() {
        feed.page = AsyncValue.data(props);
        if (feed.category == null) {
          _navigationProps = props;
          _categories = payload.layout.sidebar.categories;
        }
        feed.topics.clear();
        feed.topics.addAll(_mergeInteractions(props.topics, revision));
      });
      recordStartupMilestone('startup.home_data_parsed');
      _recordFirstHomeContent(props.topics.isNotEmpty);
      unawaited(() async {
        try {
          final scope = await cacheScopeFuture;
          if (sequence != feed.loadSequence ||
              epoch != ref.read(offlineCacheEpochProvider) ||
              scope == null) {
            return;
          }
          final cache = ref.read(offlineTopicCacheProvider);
          if (cache is OfflineHomeCache) {
            await (cache as OfflineHomeCache).putHomePage(
              accountId: scope.$1,
              baseUrl: scope.$2,
              sort: requestedSort,
              payload: payload,
            );
          }
        } catch (_) {
          // Persistent SWR is optional; the network result is already visible.
        }
      }());
    } catch (e, st) {
      if (!mounted ||
          sequence != feed.loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if (cancel.isCancelled) return;
      await cacheShownFuture;
      if (!mounted ||
          sequence != feed.loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if ((silent || cachedPageShown) && feed.page.hasValue) {
        showGfToast(
          context,
          AppLocalizations.of(context).refreshFailedRetained,
          error: true,
        );
      } else {
        setState(() => feed.page = AsyncValue.error(e, st));
      }
    } finally {
      if (identical(feed.loadCancel, cancel)) feed.loadCancel = null;
    }
  }

  Future<(int, String)?> _homeCacheScope() async {
    CurrentUser? user;
    try {
      user = await ref.read(currentUserProvider.future);
    } catch (_) {
      // A cache identity failure must not block the network request.
    }
    final hasToken = await hasSessionToken(ref.read(tokenStorageProvider));
    final accountId = user?.id ?? (hasToken ? null : 0);
    if (accountId == null) return null;
    final baseUrl = AppConfig.apiBaseUrl.isNotEmpty
        ? AppConfig.apiBaseUrl
        : GfApiClient.defaultBaseUrl;
    return (accountId, baseUrl);
  }

  void _recordFirstHomeContent(bool hasTopics) {
    if (_firstCardFrameRecorded) return;
    _firstCardFrameRecorded = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        recordFirstHomeContentAndFullyDrawn(hasTopics: hasTopics);
      }
    });
  }

  /// 从话题详情返回:先消费详情页带回的话题增量(unseen/点赞/收藏/计数,
  /// 条目落在任何已加载页都生效),再后台请求第一页,按 id 原位更新已加载
  /// 话题的最新状态。不把第一页 payload 写入 feed.page——分页游标
  /// (nextUrl/hasNext)保留,「加载更多」从原进度继续;列表长度不回缩,
  /// 滚动位置不跳顶(MADR 0012 Preserve scroll position)。下拉刷新仍走
  /// _load(silent: true) 的整页重置语义,两条静默路径分开接线。
  Future<void> _refreshAfterReturn(_HomeFeedState feed) async {
    if (!mounted) return;
    // Replies are only needed by retained profile activity pages. Once back at
    // the home feed, discard their detail handoff along with the topic handoff.
    ref.read(postReturnStatesProvider).clear();
    final returnReadRevision = _interactionRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    final Map<int, TopicReturnState> returned = Map.of(
      ref.read(topicReturnStatesProvider),
    );
    if (returned.isNotEmpty) {
      ref.read(topicReturnStatesProvider).clear();
      setState(() {
        for (final entry in returned.entries) {
          // A not-yet-loaded sort can still return an older read after this
          // detail handoff. Preserve its revision just like a local mutation.
          _returnedTopicOverrides[entry.key] = (
            revision: ++_interactionRevision,
            epoch: epoch,
            state: entry.value,
          );
          _updateLoadedTopic(
            entry.key,
            (topic) => _mergeInteraction(topic, returnReadRevision),
          );
        }
      });
    }
    if (feed.topics.isEmpty) return;
    final sequence = ++feed.loadSequence;
    final revision = _interactionRevision;
    feed.loadCancel?.cancel('home return refresh superseded');
    feed.loadMoreCancel?.cancel('home return refresh superseded');
    final cancel = feed.loadCancel = CancelToken();
    // 在途「加载更多」已随序号失效,其 finally 的同序号守卫不会清理加载态,
    // 这里必须像 _load 一样接管,否则 feed.loadingMore 卡死、分页失效。
    feed.loadingMore = false;
    feed.loadMoreError = null;
    try {
      final PagePayload payload = await _fetchFeed(feed, cancel);
      if (!mounted ||
          sequence != feed.loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final HomeProps? props = _feedProps(payload, feed);
      if (props == null) throw const FormatException('home props');
      final Map<int, TopicPayload> incoming = {
        for (final TopicPayload topic in props.topics) topic.id: topic,
      };
      setState(() {
        // 只原位替换已加载条目:不追加新热帖、不移除已消失条目,
        // 保证列表形状与滚动位置稳定。
        for (var i = 0; i < feed.topics.length; i++) {
          final TopicPayload? fresh = incoming[feed.topics[i].id];
          if (fresh != null) {
            feed.topics[i] = _mergeInteraction(fresh, revision);
          }
        }
      });
    } catch (_) {
      // 返回刷新失败:保留当前列表与分页进度,并按 Home 失败刷新的
      // 产品约定轻提示(docs/product/mobile-experience.md)。
      if (!mounted ||
          sequence != feed.loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if (cancel.isCancelled) return;
      showGfToast(
        context,
        AppLocalizations.of(context).refreshFailedRetained,
        error: true,
      );
    } finally {
      if (identical(feed.loadCancel, cancel)) feed.loadCancel = null;
    }
  }

  Future<void> _loadMore(_HomeFeedState feed) async {
    final HomeProps? props = feed.page.value;
    if (props == null || !props.pagination.hasNext || feed.loadingMore) return;
    final String nextUrl = props.pagination.nextUrl;
    if (nextUrl.isEmpty) return;
    final sequence = feed.loadSequence;
    final revision = _interactionRevision;
    final epoch = ref.read(offlineCacheEpochProvider);
    final cancel = feed.loadMoreCancel = CancelToken();
    setState(() {
      feed.loadingMore = true;
      feed.loadMoreError = null;
    });
    try {
      // 真实分页:按后端 nextUrl 请求下一页(页面级数据通道)。
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch(nextUrl, cancelToken: cancel);
      if (!mounted ||
          sequence != feed.loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final HomeProps? next = _feedProps(payload, feed);
      if (next == null) throw const FormatException('home pagination');
      setState(() {
        final seen = feed.topics.map((topic) => topic.id).toSet();
        feed.topics.addAll(
          _mergeInteractions(
            next.topics.where((topic) => seen.add(topic.id)).toList(),
            revision,
          ),
        );
        feed.page = AsyncValue.data(next);
      });
    } catch (error) {
      if (mounted &&
          sequence == feed.loadSequence &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(
          () => feed.loadMoreError = resolveErrorMessage(
            AppLocalizations.of(context),
            error,
          ),
        );
      }
    } finally {
      if (identical(feed.loadMoreCancel, cancel)) feed.loadMoreCancel = null;
      if (mounted && sequence == feed.loadSequence) {
        setState(() => feed.loadingMore = false);
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
    final topics = _activeFeed.topics;
    final index = topics.indexWhere((t) => t.id == topic.id);
    final current = index >= 0 ? topics[index] : topic;
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
      _updateLoadedTopic(
        topic.id,
        (loaded) => bookmark
            ? loaded.copyWith(bookmarked: target)
            : loaded.copyWith(liked: target, likeCount: optimistic.likeCount!),
      );
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
      _updateLoadedTopic(
        topicId,
        (topic) => bookmark
            ? topic.copyWith(bookmarked: snapshot.bookmarked)
            : topic.copyWith(
                liked: snapshot.liked,
                likeCount: snapshot.likeCount,
              ),
      );
    });
  }

  void _updateLoadedTopic(
    int topicId,
    TopicPayload Function(TopicPayload) update,
  ) {
    for (final feed in _feeds.values) {
      final index = feed.topics.indexWhere((topic) => topic.id == topicId);
      if (index >= 0) {
        feed.topics[index] = update(feed.topics[index]);
      }
    }
  }

  void _activateFeed() {
    final key = _activeKey;
    final firstVisit = !_feeds.containsKey(key);
    setState(() {
      _feeds.putIfAbsent(key, () => _HomeFeedState(_sort, category: _category));
    });
    _tabScrollRegistry.register(
      GfShellDestination.home,
      _scrollToTopController,
    );
    if (firstVisit) _load();
  }

  void _switchSort(String sort) {
    if (sort == 'latest') sort = '';
    if (sort == _sort) return;
    _sort = sort;
    if (_category == null) _allSort = sort;
    _activateFeed();
  }

  void _switchCategory(CategoryNavPayload? category) {
    if (_category?.id == category?.id) return;
    if (_category == null) {
      _allSort = _sort;
    } else {
      _categorySorts[_category!.id] = _sort;
    }
    _category = category;
    _sort = category == null ? _allSort : (_categorySorts[category.id] ?? '');
    _activateFeed();
  }

  void _filterCategory(int id) {
    for (final category in _categories) {
      if (category.id == id) {
        _switchCategory(category);
        return;
      }
    }
    // Visible topic categories may be absent from a curated sidebar.
    for (final topic in _activeFeed.topics) {
      for (final category in topic.categories) {
        if (category.id == id && category.url.isNotEmpty) {
          _switchCategory(
            CategoryNavPayload(
              id: category.id,
              label: category.name,
              color: category.color,
              url: category.url,
            ),
          );
          return;
        }
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.listen<int>(offlineCacheEpochProvider, (_, _) {
      _pendingInteractions.clear();
      _interactionOverrides.clear();
      _returnedTopicOverrides.clear();
      for (final feed in _feeds.values) {
        feed.cancel();
      }
      _category = null;
      _sort = _allSort;
      _categorySorts.clear();
      _feeds
        ..clear()
        ..[_sort] = _HomeFeedState(_sort);
      _navigationProps = null;
      _categories = [];
      _tabScrollRegistry.register(
        GfShellDestination.home,
        _scrollToTopController,
      );
      _load();
    });
    final AppLocalizations l10n = AppLocalizations.of(context);
    return RootSurface(
      titleWidget: _category == null
          ? const GfLogo(size: 32)
          : Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                const GfLogo(size: 24),
                const SizedBox(width: 8),
                Flexible(
                  child: Text(
                    _category!.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                IconButton(
                  key: const ValueKey('home-clear-category'),
                  tooltip: l10n.homeAllCategories,
                  onPressed: () => _switchCategory(null),
                  icon: const GfSymbol('x', size: 18),
                ),
              ],
            ),
      actions: [
        IconButton(
          tooltip: l10n.commonSearch,
          icon: const GfSymbol('search'),
          onPressed: () => context.push('/search'),
        ),
      ],
      toolbarHeight: GfTabBar.heightFor(context),
      toolbar: _navigationProps != null
          ? _HomeToolbar(
              props:
                  _activeFeed.page.valueOrNull ??
                  (_category == null
                      ? _navigationProps!
                      : _navigationProps!.copyWith(
                          tabs: [
                            TabItemPayload(
                              key: 'latest',
                              label: '',
                              url: '',
                              active: true,
                            ),
                            TabItemPayload(
                              key: 'new',
                              label: '',
                              url: '',
                              active: false,
                            ),
                          ],
                        )),
              categories: _categories,
              activeCategory: _category,
              onCategorySelected: _switchCategory,
              selected: _sort.isEmpty ? 'latest' : _sort,
              feedMode: _feedMode,
              onSelected: _switchSort,
              onFeedModeSelected: _setFeedMode,
            )
          : const SizedBox.shrink(),
      body: (top, bottom) => IndexedStack(
        index: _feeds.keys.toList().indexOf(_activeKey),
        children: [
          for (final feed in _feeds.values)
            TickerMode(
              key: ObjectKey(feed),
              enabled:
                  feed == _activeFeed && TickerMode.valuesOf(context).enabled,
              child: GfScrollToTop(
                semanticLabel: l10n.commonBackToTop,
                controller: feed.scrollToTop,
                showButton: false,
                builder: (_, controller) =>
                    _buildFeed(feed, controller, top, bottom),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildFeed(
    _HomeFeedState feed,
    ScrollController controller,
    double top,
    double bottom,
  ) {
    final l10n = AppLocalizations.of(context);
    return feed.page.when(
      loading: () => Padding(
        padding: EdgeInsets.only(top: top),
        child: const GfTopicFeedSkeleton(),
      ),
      error: (e, _) => Padding(
        padding: EdgeInsets.only(top: top, bottom: bottom),
        child: GfErrorRetry(
          message: resolveErrorMessage(l10n, e),
          onRetry: () => _load(target: feed),
        ),
      ),
      data: (props) => AppRefreshIndicator(
        edgeOffset: top,
        onRefresh: () => _load(silent: true, target: feed),
        child: GfTopicList(
          loadMoreError: feed.loadMoreError,
          controller: controller,
          padding: EdgeInsets.only(top: top, bottom: bottom),
          header: AnnouncementBanner(
            announcement: props.announcement,
            collapsed: _announcementCollapsed,
            onCollapsedChanged: _setAnnouncementCollapsed,
          ),
          loading: feed.loadingMore,
          topics: feed.topics,
          hiddenCategoryId: feed.category?.id,
          onCategorySelected: _filterCategory,
          feedMode: _feedMode,
          onFirstMediaFrame: recordFirstHomeMediaFrame,
          onLikeTopic: _toggleTopicInteraction,
          onBookmarkTopic: (topic, target) =>
              _toggleTopicInteraction(topic, target, bookmark: true),
          onReturnFromTopic: () => _refreshAfterReturn(feed),
          hasMore: props.pagination.hasNext,
          onLoadMore: () => _loadMore(feed),
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
    required this.activeCategory,
    required this.onCategorySelected,
  });

  final HomeProps props;
  final CategoryNavPayload? activeCategory;
  final ValueChanged<CategoryNavPayload?> onCategorySelected;
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

    return ColoredBox(
      color: colors.base100,
      child: Row(
        children: [
          Expanded(
            child: GfTabBar(
              tabs: [
                for (final tab in props.tabs)
                  GfTab(
                    label: _sortTabLabel(context, tab.key, tab.label ?? ''),
                    value: tab.key,
                  ),
              ],
              selected: effective,
              onSelected: (value) => onSelected(value as String),
            ),
          ),
          IconButton(
            tooltip: l10n.homeFeedOptions,
            icon: const GfSymbol('sliders-horizontal', size: 20),
            onPressed: () => _showOptions(context),
          ),
          const SizedBox(width: 4),
        ],
      ),
    );
  }

  Future<void> _showOptions(BuildContext context) async {
    final l10n = AppLocalizations.of(context);
    final selected = await showGfBottomSheet<Object>(
      context,
      builder: (sheetContext) => SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 4, 8, 12),
              child: Text(
                l10n.homeFeedOptions,
                style: GfTheme.typographyOf(context).title2,
              ),
            ),
            for (final mode in GfTopicFeedMode.values)
              ListTile(
                leading: GfSymbol(
                  mode == GfTopicFeedMode.card ? 'layout-grid' : 'list',
                ),
                title: Text(
                  mode == GfTopicFeedMode.card
                      ? l10n.topicFeedModeCard
                      : l10n.topicFeedModeList,
                ),
                trailing: feedMode == mode
                    ? const GfSymbol('check', size: 20)
                    : null,
                selected: feedMode == mode,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
                onTap: () => Navigator.pop(sheetContext, mode),
              ),
            if (categories.isNotEmpty) ...[
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 8),
                child: Divider(),
              ),
              ListTile(
                leading: const GfSymbol('house', size: 20),
                title: Text(l10n.homeAllCategories),
                selected: activeCategory == null,
                trailing: activeCategory == null
                    ? const GfSymbol('check', size: 20)
                    : null,
                onTap: () => Navigator.pop(sheetContext, 'all-categories'),
              ),
              for (final category in categories)
                ListTile(
                  leading: Container(
                    width: 10,
                    height: 10,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: colorFromHex(category.color),
                    ),
                  ),
                  title: Text(category.label),
                  selected: activeCategory?.id == category.id,
                  trailing: activeCategory?.id == category.id
                      ? const GfSymbol('check', size: 20)
                      : null,
                  onTap: () => Navigator.pop(sheetContext, category),
                ),
            ],
          ],
        ),
      ),
    );
    if (!context.mounted) return;
    if (selected is GfTopicFeedMode) onFeedModeSelected(selected);
    if (selected is CategoryNavPayload) onCategorySelected(selected);
    if (selected == 'all-categories') onCategorySelected(null);
  }

  /// Known feed labels follow the app locale; custom server tabs retain their label.
  String _sortTabLabel(BuildContext context, String key, String label) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return switch (key) {
      'latest' => l10n.sortLatest,
      'hot' => l10n.sortHot,
      'popular' => l10n.sortPopular,
      'following' => l10n.sortFollowing,
      'new' => l10n.sortNew,
      _ => label.isNotEmpty ? label : key,
    };
  }
}
