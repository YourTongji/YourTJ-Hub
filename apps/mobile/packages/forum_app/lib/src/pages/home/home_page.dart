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
import '../../navigation/tab_scroll_registry.dart';
import '../../server_messages.dart';
import '../../widgets/skeletons.dart';
import '../../widgets/status_views.dart';
import '../../widgets/topic_list.dart';
import '../../widgets/root_surface.dart';
import '../../widgets/announcement_banner.dart';

/// 首页:公告 + 话题流(web HomePage.vue 的移动端形态)。
class HomePage extends ConsumerStatefulWidget {
  const HomePage({super.key});

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

class _HomePageState extends ConsumerState<HomePage> {
  static const String _feedModeKey = 'goose:home-feed-mode';

  AsyncValue<HomeProps> _page = const AsyncValue.loading();
  String _sort = '';
  int _loadSequence = 0;
  int _interactionRevision = 0;
  final _pendingInteractions = <(int, bool)>{};
  final _interactionOverrides =
      <int, ({int revision, bool? liked, bool? bookmarked, int? likeCount})>{};

  // Only writes completed after a read started override that response. A later
  // refresh (including returning from topic detail) remains authoritative.
  List<TopicPayload> _mergeInteractions(
    List<TopicPayload> incoming,
    int readRevision,
  ) => [for (final topic in incoming) _mergeInteraction(topic, readRevision)];

  TopicPayload _mergeInteraction(TopicPayload topic, int readRevision) {
    final update = _interactionOverrides[topic.id];
    if (update == null) return topic;
    if (update.revision <= readRevision) {
      _interactionOverrides.remove(topic.id);
      return topic;
    }
    return topic.copyWith(
      liked: update.liked ?? topic.liked,
      bookmarked: update.bookmarked ?? topic.bookmarked,
      likeCount: update.likeCount ?? topic.likeCount,
    );
  }

  final List<TopicPayload> _topics = <TopicPayload>[];
  bool _loadingMore = false;
  GfTopicFeedMode _feedMode = GfTopicFeedMode.card;
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
        _topics.addAll(_mergeInteractions(props.topics, revision));
      });
    } catch (e, st) {
      if (!mounted ||
          sequence != _loadSequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() => _page = AsyncValue.error(e, st));
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
    setState(() => _loadingMore = true);
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
      if (next != null && next.topics.isNotEmpty) {
        setState(() {
          _topics.addAll(_mergeInteractions(next.topics, revision));
          _page = AsyncValue.data(next);
        });
      }
    } catch (_) {
      // 加载更多失败静默(用户可再次点击)。
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
    final key = (topic.id, bookmark);
    if (!_pendingInteractions.add(key)) return false;
    final epoch = ref.read(offlineCacheEpochProvider);
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
      if (!success ||
          !mounted ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return false;
      }
      final previous = _interactionOverrides[topic.id];
      // 增量基于列表当前值而非闭包捕获的旧 payload：连续快速点击时不丢步。
      final currentLikeCount = _topics
          .firstWhere((t) => t.id == topic.id, orElse: () => topic)
          .likeCount;
      final update = (
        revision: ++_interactionRevision,
        liked: bookmark ? previous?.liked : target,
        bookmarked: bookmark ? target : previous?.bookmarked,
        likeCount: bookmark
            ? previous?.likeCount
            : math.max(0, currentLikeCount + (target ? 1 : -1)),
      );
      setState(() {
        _interactionOverrides[topic.id] = update;
        for (var i = 0; i < _topics.length; i++) {
          if (_topics[i].id == topic.id) {
            _topics[i] = _topics[i].copyWith(
              liked: bookmark ? _topics[i].liked : target,
              bookmarked: bookmark ? target : _topics[i].bookmarked,
              likeCount: bookmark
                  ? _topics[i].likeCount
                  : math.max(0, _topics[i].likeCount + (target ? 1 : -1)),
            );
          }
        }
      });
      return true;
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
      return false;
    } finally {
      _pendingInteractions.remove(key);
    }
  }

  void _switchSort(String sort) {
    if (sort == _sort) return;
    _sort = sort;
    _load();
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return RootSurface(
      title: 'YourTJ',
      actions: [
        IconButton(
          tooltip: l10n.commonSearch,
          icon: const GfSymbol('search'),
          onPressed: () => context.push('/search'),
        ),
      ],
      toolbarHeight: 44,
      toolbar: _page.hasValue
          ? _HomeToolbar(
              props: _page.requireValue,
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
              controller: controller,
              padding: EdgeInsets.only(top: top, bottom: bottom),
              header: AnnouncementBanner(announcement: props.announcement),
              loading: _loadingMore,
              topics: _topics,
              feedMode: _feedMode,
              onLikeTopic: _toggleTopicInteraction,
              onBookmarkTopic: (topic, target) =>
                  _toggleTopicInteraction(topic, target, bookmark: true),
              onReturnFromTopic: () => _load(silent: true),
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
    required this.selected,
    required this.feedMode,
    required this.onSelected,
    required this.onFeedModeSelected,
  });

  final HomeProps props;
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
                    label: _sortTabLabel(context, tab.key, tab.label ?? ''),
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
