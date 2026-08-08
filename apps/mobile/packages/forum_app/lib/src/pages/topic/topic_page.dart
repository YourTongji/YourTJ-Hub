import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../asset_url.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/status_views.dart';

/// 话题详情页(web TopicPage.vue 的移动端形态):
/// 话题信息 + 帖子流(分页)+ markdown 渲染 + 图片查看器 + 互动(点赞/收藏/评论)。
class TopicPage extends ConsumerStatefulWidget {
  const TopicPage({super.key, required this.topicId});

  final int topicId;

  @override
  ConsumerState<TopicPage> createState() => _TopicPageState();
}

class _TopicPageState extends ConsumerState<TopicPage> {
  AsyncValue<TopicDetailProps> _page = const AsyncValue.loading();
  bool _loadingMore = false;
  final List<PostPayload> _posts = [];

  // 互动状态(乐观更新)。
  bool _liked = false;
  bool _bookmarked = false;
  int _likeCount = 0;

  // 评论输入。
  final TextEditingController _replyController = TextEditingController();
  bool _replying = false;
  int _replyToPostId = 0;

  final FocusNode _replyFocus = FocusNode();

  // 浮动层状态(web TopicFloatingControls / PostComposer 语义)。
  bool _composerOpen = false;
  bool _railOpen = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _replyController.dispose();
    _replyFocus.dispose();
    super.dispose();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .topicDetail(widget.topicId);
      final props = parsePageProps<TopicDetailProps>(payload);
      if (props == null) {
        setState(
          () => _page = AsyncValue.error(
            AppLocalizations.of(context).commonParseFailed,
            StackTrace.current,
          ),
        );
        return;
      }
      // 写入 drift 离线缓存(供断网时回读);缓存失败静默降级。
      await _cachePut(widget.topicId, payload.toJson());
      setState(() {
        _page = AsyncValue.data(props);
        _posts.clear();
        _posts.addAll(props.postStream.posts);
        _liked = props.topic.isLiked;
        _bookmarked = props.topic.isBookmarked;
        _likeCount = props.topic.likeCount;
      });
    } catch (e, st) {
      // 网络失败:回退 drift 离线缓存(已浏览话题离线可读)。
      final PagePayload? cached = await _cacheGet(widget.topicId);
      final props = cached == null
          ? null
          : parsePageProps<TopicDetailProps>(cached);
      if (props != null) {
        setState(() {
          _page = AsyncValue.data(props);
          _posts.clear();
          _posts.addAll(props.postStream.posts);
          _liked = props.topic.isLiked;
          _bookmarked = props.topic.isBookmarked;
          _likeCount = props.topic.likeCount;
        });
      } else {
        setState(() => _page = AsyncValue.error(e, st));
      }
    }
  }

  /// 写缓存;失败静默(缓存不可用不影响页面加载)。
  Future<void> _cachePut(int topicId, Map<String, dynamic> json) async {
    try {
      await ref.read(offlineTopicCacheProvider).put(topicId, json);
    } catch (_) {
      // 缓存不可用时忽略(页面加载不依赖缓存)。
    }
  }

  /// 读缓存;失败返回 null。
  Future<PagePayload?> _cacheGet(int topicId) async {
    try {
      return await ref.read(offlineTopicCacheProvider).get(topicId);
    } catch (_) {
      return null;
    }
  }

  Future<void> _loadMore() async {
    final props = _page.value;
    if (props == null || !props.postStream.hasAfter || _loadingMore) return;
    setState(() => _loadingMore = true);
    try {
      final window = await ref
          .read(topicRepositoryProvider)
          .getPostWindow(
            topicId: widget.topicId,
            afterPostNo: props.postStream.afterPostNo,
          );
      if (window.posts.isNotEmpty) {
        setState(() => _posts.addAll(window.posts));
      }
    } catch (_) {
      // 加载更多失败静默。
    } finally {
      setState(() => _loadingMore = false);
    }
  }

  Future<void> _toggleLike() async {
    final topic = _page.value?.topic;
    if (topic == null) return;
    final bool target = !_liked;
    setState(() {
      _liked = target;
      _likeCount += target ? 1 : -1;
    });
    try {
      await ref
          .read(topicRepositoryProvider)
          .likeTopic(topicId: topic.id, action: target ? 1 : 2);
    } catch (_) {
      // 回滚。
      setState(() {
        _liked = !target;
        _likeCount += target ? -1 : 1;
      });
    }
  }

  Future<void> _toggleBookmark() async {
    final topic = _page.value?.topic;
    if (topic == null) return;
    final bool target = !_bookmarked;
    setState(() => _bookmarked = target);
    try {
      await ref
          .read(topicRepositoryProvider)
          .bookmarkTopic(topicId: topic.id, action: target ? 1 : 2);
    } catch (_) {
      setState(() => _bookmarked = !target);
    }
  }

  Future<void> _submitReply() async {
    final String content = _replyController.text.trim();
    if (content.isEmpty) return;
    setState(() => _replying = true);
    try {
      await ref
          .read(postRepositoryProvider)
          .createPost(
            topicId: widget.topicId,
            content: content,
            replyToPostId: _replyToPostId,
          );
      _replyController.clear();
      _replyToPostId = 0;
      if (mounted) {
        showGfToast(context, AppLocalizations.of(context).topicReplySuccess);
      }
      // 局部刷新:回复成功后静默重载(不置 loading、不清空列表,
      // 保留滚动位置,对齐 web 乐观追加语义)。
      if (mounted) {
        await _load(silent: true);
      }
    } on ApiException catch (e) {
      if (mounted) {
        showGfToast(context, e.messageKey, error: true);
      }
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).topicReplyFailed('$e'),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _replying = false);
    }
  }

  Future<void> _reportPost(PostPayload post) async {
    if (!mounted) return;
    final AppLocalizations l10n = AppLocalizations.of(context);
    final reason = await showGfModal<String>(
      context,
      builder: (ctx) {
        final ctrl = TextEditingController();
        return Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            Text(l10n.topicReport, style: GfTheme.typographyOf(ctx).title3),
            const SizedBox(height: 16),
            GfInput(
              controller: ctrl,
              maxLines: 3,
              hintText: l10n.topicReportHint,
            ),
            const SizedBox(height: 20),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: <Widget>[
                GfButton(
                  label: l10n.commonCancel,
                  variant: GfButtonVariant.ghost,
                  onPressed: () => Navigator.pop(ctx),
                ),
                const SizedBox(width: 8),
                GfButton(
                  label: l10n.topicReportSubmit,
                  onPressed: () => Navigator.pop(ctx, ctrl.text.trim()),
                ),
              ],
            ),
          ],
        );
      },
    );
    if (reason == null || reason.isEmpty) return;
    try {
      await ref
          .read(postRepositoryProvider)
          .report(
            targetType: 'post',
            targetId: post.id,
            reason: reason,
            note: '',
          );
      if (mounted) {
        showGfToast(context, l10n.topicReportSubmitted);
      }
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.topicReportFailed('$e'), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.topicTitle)),
      body: _page.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (props) {
          return Stack(
            children: [
              Positioned.fill(
                child: RefreshIndicator(
                  onRefresh: () => _load(silent: true),
                  child: ListView.separated(
                    padding: const EdgeInsets.only(bottom: 12),
                    itemCount: _posts.length + 2,
                    separatorBuilder: (_, _) => const GfDivider(),
                    itemBuilder: (context, index) {
                      if (index == 0) {
                        return _TopicHeader(
                          topic: props.topic,
                          liked: _liked,
                          bookmarked: _bookmarked,
                          likeCount: _likeCount,
                          onLike: _toggleLike,
                          onBookmark: _toggleBookmark,
                        );
                      }
                      if (index == _posts.length + 1) {
                        return GfListFooter(
                          loading: _loadingMore,
                          hasMore: props.postStream.hasAfter,
                          onLoadMore: _loadMore,
                        );
                      }
                      return _PostCard(
                        post: _posts[index - 1],
                        onReply: () {
                          _replyToPostId = _posts[index - 1].id;
                          _replyController.text =
                              '@${_posts[index - 1].author.username} ';
                          _replyController.selection = TextSelection.collapsed(
                            offset: _replyController.text.length,
                          );
                          FocusScope.of(context).requestFocus(_replyFocus);
                        },
                        onReport: () => _reportPost(_posts[index - 1]),
                      );
                    },
                  ),
                ),
              ),
              // 底部浮动操作条 + 浮动回复编辑器(web TopicFloatingControls +
              // PostComposer 移动端形态)。
              Positioned(
                left: 0,
                right: 0,
                bottom: 16,
                child: SafeArea(
                  top: false,
                  child: Center(
                    child: _composerOpen
                        ? ConstrainedBox(
                            constraints: const BoxConstraints(maxWidth: 560),
                            child: GfPostComposer(
                              controller: _replyController,
                              targetName: _replyToPostId != 0
                                  ? l10n.topicReplying
                                  : null,
                              onCloseTarget: () {
                                _replyController.clear();
                                setState(() => _replyToPostId = 0);
                              },
                              publishing: _replying,
                              publishLabel: l10n.commonSend,
                              onPublish: _submitReply,
                              toolbar: null,
                            ),
                          )
                        : GfFloatingControls(
                            actions: [
                              GfTopicAction(
                                icon: Icons.favorite_border,
                                active: _liked,
                                activeColor: colors.error,
                                onTap: _toggleLike,
                              ),
                              GfTopicAction(
                                icon: Icons.bookmark_border,
                                active: _bookmarked,
                                activeColor: colors.primary,
                                onTap: _toggleBookmark,
                              ),
                            ],
                            onOpenReply: () {
                              if (_replyToPostId == 0) {
                                _replyController.clear();
                              }
                              FocusScope.of(context).requestFocus(_replyFocus);
                              setState(() => _composerOpen = true);
                            },
                            currentNo: props.postStream.posts.isEmpty
                                ? 1
                                : props.postStream.posts.first.postNo,
                            maxNo: props.postStream.maxPostNo,
                            onFloorTap: () =>
                                setState(() => _railOpen = !_railOpen),
                          ),
                  ),
                ),
              ),
              // 楼层滑轨浮动面板(web PostPositionRail 移动端形态)。
              if (_railOpen && !_composerOpen)
                Positioned(
                  left: 16,
                  right: 16,
                  bottom: 76,
                  child: SafeArea(
                    top: false,
                    child: GfFloatingSurface(
                      padding: const EdgeInsets.all(8),
                      child: GfPostPositionRail(
                        current: props.postStream.posts.isEmpty
                            ? 1
                            : props.postStream.posts.first.postNo,
                        max: props.postStream.maxPostNo,
                        onSelect: (floor) {
                          setState(() => _railOpen = false);
                          showGfToast(context, l10n.topicFloorSelected(floor));
                        },
                        onEarliest: () => _load(silent: true),
                        onLatest: () => _loadMore(),
                      ),
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

class _TopicHeader extends StatelessWidget {
  const _TopicHeader({
    required this.topic,
    required this.liked,
    required this.bookmarked,
    required this.likeCount,
    required this.onLike,
    required this.onBookmark,
  });

  final TopicDetailPayload topic;
  final bool liked;
  final bool bookmarked;
  final int likeCount;
  final VoidCallback onLike;
  final VoidCallback onBookmark;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(topic.title, style: GfTheme.typographyOf(context).title1),
          const SizedBox(height: 8),
          Row(
            children: [
              GfAvatar(
                src: resolveApiAssetUrl(topic.author.avatarUrl),
                size: 24,
              ),
              const SizedBox(width: 8),
              Text(
                topic.author.nickname ?? topic.author.username,
                style: GfTheme.typographyOf(context).bodyStrong,
              ),
              const SizedBox(width: 12),
              Text(
                timeAgo(topic.createdAt),
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
            ],
          ),
          Wrap(
            spacing: 6,
            children: [
              for (final cat in topic.categories)
                GfChip(label: cat.name, color: colorFromHex(cat.color)),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              _MetaItem(
                icon: Icons.visibility_outlined,
                value: formatNumber(topic.viewCount),
              ),
              const SizedBox(width: 16),
              _MetaItem(
                icon: Icons.chat_bubble_outline,
                value: formatNumber(topic.replyCount),
              ),
              const SizedBox(width: 16),
              InkWell(
                onTap: onLike,
                borderRadius: BorderRadius.circular(16),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  child: _MetaItem(
                    icon: liked ? Icons.favorite : Icons.favorite_border,
                    value: formatNumber(likeCount),
                    color: liked ? colors.error : null,
                  ),
                ),
              ),
              const SizedBox(width: 8),
              InkWell(
                onTap: onBookmark,
                borderRadius: BorderRadius.circular(16),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  child: Icon(
                    bookmarked ? Icons.bookmark : Icons.bookmark_border,
                    size: 18,
                    color: bookmarked
                        ? colors.primary
                        : GfTheme.colorsOf(context).iconMuted,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _MetaItem extends StatelessWidget {
  const _MetaItem({required this.icon, required this.value, this.color});

  final IconData icon;
  final String value;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(
          icon,
          size: 15,
          color: color ?? GfTheme.colorsOf(context).iconMuted,
        ),
        const SizedBox(width: 4),
        Text(
          value,
          style: GfTheme.typographyOf(context).caption.copyWith(
            color: color ?? GfTheme.colorsOf(context).iconMuted,
          ),
        ),
      ],
    );
  }
}

class _PostCard extends StatelessWidget {
  const _PostCard({
    required this.post,
    required this.onReply,
    required this.onReport,
  });

  final PostPayload post;
  final VoidCallback onReply;
  final VoidCallback onReport;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              GfAvatar(
                src: resolveApiAssetUrl(post.author.avatarUrl),
                size: 24,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  post.author.nickname ?? post.author.username,
                  style: GfTheme.typographyOf(context).bodyStrong,
                ),
              ),
              if (post.postNo > 0)
                Text(
                  '#${post.postNo}',
                  style: GfTheme.typographyOf(context).caption.copyWith(
                    color: GfTheme.colorsOf(context).iconMuted,
                  ),
                ),
            ],
          ),
          if (post.replyToUsername != null) ...[
            const SizedBox(height: 6),
            Text(
              '${l10n.topicReply} @${post.replyToUsername}',
              style: GfTheme.typographyOf(
                context,
              ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
            ),
          ],
          const SizedBox(height: 10),
          GfMarkdownView(data: post.content),
          const SizedBox(height: 10),
          Row(
            children: [
              Text(
                timeAgo(post.createdAt),
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
              const Spacer(),
              Icon(
                post.isLiked ? Icons.favorite : Icons.favorite_border,
                size: 16,
                color: post.isLiked
                    ? colors.error
                    : GfTheme.colorsOf(context).iconMuted,
              ),
              const SizedBox(width: 4),
              Text(
                '${post.likeCount}',
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
              const SizedBox(width: 16),
              InkWell(
                onTap: onReply,
                child: Icon(
                  Icons.reply_outlined,
                  size: 16,
                  color: GfTheme.colorsOf(context).iconMuted,
                ),
              ),
              const SizedBox(width: 16),
              InkWell(
                onTap: onReport,
                child: Icon(
                  Icons.flag_outlined,
                  size: 16,
                  color: GfTheme.colorsOf(context).iconMuted,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
