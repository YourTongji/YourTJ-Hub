import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';
import '../../widgets/topic_list.dart';

/// 首页:公告 + 话题流(web HomePage.vue 的移动端形态)。
class HomePage extends ConsumerStatefulWidget {
  const HomePage({super.key});

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

class _HomePageState extends ConsumerState<HomePage> {
  AsyncValue<HomeProps> _page = const AsyncValue.loading();
  String _sort = '';
  final List<dynamic> _topics = [];
  bool _loadingMore = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .home(sort: _sort);
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
        _topics.addAll(props.topics);
      });
    } catch (e, st) {
      setState(() => _page = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadMore() async {
    final HomeProps? props = _page.value;
    if (props == null || !props.pagination.hasNext || _loadingMore) return;
    final String nextUrl = props.pagination.nextUrl;
    if (nextUrl.isEmpty) return;
    setState(() => _loadingMore = true);
    try {
      // 真实分页:按后端 nextUrl 请求下一页(页面级数据通道)。
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch(nextUrl);
      final HomeProps? next = parsePageProps<HomeProps>(payload);
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

  void _switchSort(String sort) {
    if (sort == _sort) return;
    _sort = sort;
    _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('yourtj'),
        actions: [
          IconButton(
            icon: const Icon(Icons.settings_outlined),
            onPressed: () => context.go('/settings'),
          ),
        ],
      ),
      body: _page.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (props) {
          return Column(
            children: [
              _AnnouncementBanner(props: props),
              _SortTabs(props: props, selected: _sort, onSelected: _switchSort),
              Expanded(
                child: GfTopicList(
                  loading: _loadingMore,
                  topics: _topics,
                  hasMore: props.pagination.hasNext,
                  onLoadMore: _loadMore,
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

/// 公告横幅:多条公告自动轮播(PageView + Timer),单条静态展示。
class _AnnouncementBanner extends ConsumerStatefulWidget {
  const _AnnouncementBanner({required this.props});

  final HomeProps props;

  @override
  ConsumerState<_AnnouncementBanner> createState() =>
      _AnnouncementBannerState();
}

class _AnnouncementBannerState extends ConsumerState<_AnnouncementBanner> {
  static const Duration _interval = Duration(seconds: 5);

  final PageController _controller = PageController();
  Timer? _timer;
  int _current = 0;

  @override
  void initState() {
    super.initState();
    final items = widget.props.announcement.items ?? const [];
    if (items.length > 1) {
      // 自动轮播:每 5s 切到下一条,循环。
      _timer = Timer.periodic(_interval, (_) {
        if (!mounted || !_controller.hasClients) return;
        final int next = (_current + 1) % items.length;
        _controller.animateToPage(
          next,
          duration: GfMotion.content,
          curve: GfMotion.standardEase,
        );
        _current = next;
      });
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!widget.props.announcement.enabled) return const SizedBox.shrink();
    final items = widget.props.announcement.items ?? const [];
    if (items.isEmpty) return const SizedBox.shrink();

    return Container(
      width: double.infinity,
      color: const Color(0xFFFFF3CD),
      child: items.length == 1
          ? _bannerText(items.first.title)
          : SizedBox(
              height: 38,
              child: Stack(
                children: [
                  PageView.builder(
                    controller: _controller,
                    itemCount: items.length,
                    onPageChanged: (i) => setState(() => _current = i),
                    itemBuilder: (context, i) => _bannerText(items[i].title),
                  ),
                  // 轮播指示点。
                  Positioned(
                    right: 10,
                    bottom: 5,
                    child: Row(
                      children: [
                        for (int i = 0; i < items.length; i++)
                          Container(
                            width: i == _current ? 14 : 6,
                            height: 4,
                            margin: const EdgeInsets.only(left: 3),
                            decoration: BoxDecoration(
                              color: i == _current
                                  ? const Color(0xFF664D03)
                                  : const Color(
                                      0xFF664D03,
                                    ).withValues(alpha: 0.35),
                              borderRadius: BorderRadius.circular(2),
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

  Widget _bannerText(String title) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 11),
      child: Align(
        alignment: Alignment.centerLeft,
        child: Text(
          title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(color: Color(0xFF664D03), fontSize: 13),
        ),
      ),
    );
  }
}

class _SortTabs extends ConsumerWidget {
  const _SortTabs({
    required this.props,
    required this.selected,
    required this.onSelected,
  });

  final HomeProps props;
  final String selected;
  final ValueChanged<String> onSelected;

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
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      alignment: Alignment.centerLeft,
      child: GfTabBar(
        tabs: <GfTab>[
          for (final tab in props.tabs)
            GfTab(label: tab.label ?? tab.key, value: tab.key),
        ],
        selected: effective,
        onSelected: (Object value) => onSelected(value as String),
      ),
    );
  }
}
