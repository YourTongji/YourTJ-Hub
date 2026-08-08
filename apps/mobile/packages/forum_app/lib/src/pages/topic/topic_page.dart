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
  bool _watched = false;
  int _likeCount = 0;

  // 评论输入。
  final TextEditingController _replyController = TextEditingController();
  bool _replying = false;
  int _replyToPostId = 0;

  final FocusNode _replyFocus = FocusNode();

  // 浮动层状态(web TopicFloatingControls / PostComposer 语义)。
  bool _composerOpen = false;
  bool _railOpen = false;
  final Set<int> _expandedReplyGroups = <int>{};

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
        _watched = props.topic.isWatched;
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
          _watched = props.topic.isWatched;
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

  Future<void> _toggleWatch() async {
    final TopicDetailPayload? topic = _page.value?.topic;
    if (topic == null) return;
    final bool target = !_watched;
    setState(() => _watched = target);
    try {
      await ref
          .read(topicRepositoryProvider)
          .watchTopic(topicId: topic.id, action: target ? 1 : 2);
    } catch (_) {
      if (mounted) setState(() => _watched = !target);
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
    await _reportTarget(targetType: 'post', targetId: post.id);
  }

  Future<void> _reportTopic(TopicDetailPayload topic) async {
    await _reportTarget(targetType: 'topic', targetId: topic.id);
  }

  Future<void> _reportTarget({
    required String targetType,
    required int targetId,
  }) async {
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
            targetType: targetType,
            targetId: targetId,
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

  List<_PostGroup> _postGroups() {
    final Map<int, PostPayload> byId = <int, PostPayload>{
      for (final PostPayload post in _posts) post.id: post,
    };
    final List<PostPayload> roots = <PostPayload>[];
    final Map<int, List<PostPayload>> repliesByRoot =
        <int, List<PostPayload>>{};

    PostPayload resolveRoot(PostPayload post) {
      final Set<int> visited = <int>{};
      PostPayload cursor = post;
      while (cursor.replyToPostId != null && cursor.replyToPostId! > 0) {
        if (!visited.add(cursor.id)) break;
        final PostPayload? parent = byId[cursor.replyToPostId!];
        if (parent == null) break;
        cursor = parent;
      }
      return cursor;
    }

    for (final PostPayload post in _posts) {
      final PostPayload? parent = post.replyToPostId == null
          ? null
          : byId[post.replyToPostId!];
      if (parent == null) {
        roots.add(post);
        continue;
      }
      final PostPayload root = resolveRoot(post);
      repliesByRoot.putIfAbsent(root.id, () => <PostPayload>[]).add(post);
    }
    roots.sort((a, b) => a.postNo.compareTo(b.postNo));
    return <_PostGroup>[
      for (final PostPayload root in roots)
        _PostGroup(
          root: root,
          replies: (repliesByRoot[root.id] ?? <PostPayload>[])
            ..sort((a, b) => a.postNo.compareTo(b.postNo)),
        ),
    ];
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
          final List<_PostGroup> groups = _postGroups();
          return Stack(
            children: [
              Positioned.fill(
                child: RefreshIndicator(
                  onRefresh: () => _load(silent: true),
                  child: ListView(
                    padding: const EdgeInsets.only(bottom: 104),
                    children: <Widget>[
                      _TopicHeader(topic: props.topic, likeCount: _likeCount),
                      Padding(
                        padding: const EdgeInsets.fromLTRB(12, 12, 12, 0),
                        child: GfCard(
                          emphasized: true,
                          child: Column(
                            children: <Widget>[
                              for (
                                int index = 0;
                                index < groups.length;
                                index++
                              ) ...<Widget>[
                                if (index > 0) const GfDivider(),
                                _PostGroupView(
                                  group: groups[index],
                                  expanded: _expandedReplyGroups.contains(
                                    groups[index].root.id,
                                  ),
                                  liked: _liked,
                                  bookmarked: _bookmarked,
                                  watched: _watched,
                                  likeCount: _likeCount,
                                  onToggleLike: _toggleLike,
                                  onToggleBookmark: _toggleBookmark,
                                  onToggleWatch: _toggleWatch,
                                  canReportTopic: !props.permissions.isOwnTopic,
                                  onReportTopic: () =>
                                      _reportTopic(props.topic),
                                  onToggleReplies: () => setState(() {
                                    final int id = groups[index].root.id;
                                    if (!_expandedReplyGroups.add(id)) {
                                      _expandedReplyGroups.remove(id);
                                    }
                                  }),
                                  onReply: _replyTo,
                                  onReport: _reportPost,
                                ),
                              ],
                            ],
                          ),
                        ),
                      ),
                      if (_loadingMore || props.postStream.hasAfter)
                        GfListFooter(
                          loading: _loadingMore,
                          hasMore: props.postStream.hasAfter,
                          onLoadMore: _loadMore,
                        ),
                    ],
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

  void _replyTo(PostPayload post) {
    setState(() {
      _replyToPostId = post.id;
      _replyController.text = '@${post.author.username} ';
      _replyController.selection = TextSelection.collapsed(
        offset: _replyController.text.length,
      );
      _composerOpen = true;
    });
    FocusScope.of(context).requestFocus(_replyFocus);
  }
}

class _TopicHeader extends StatelessWidget {
  const _TopicHeader({required this.topic, required this.likeCount});

  final TopicDetailPayload topic;
  final int likeCount;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final TextStyle metaStyle = TextStyle(
      color: colors.baseContent.withValues(alpha: 0.55),
      fontSize: 13,
      height: 1.3,
    );

    return Container(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 15),
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(color: colors.line.withValues(alpha: 0.7)),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(
            topic.title,
            style: TextStyle(
              color: colors.baseContent,
              fontSize: 24,
              height: 1.22,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 16,
            runSpacing: 8,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: <Widget>[
              Row(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  GfAvatar(
                    src: resolveApiAssetUrl(topic.author.avatarUrl),
                    size: 20,
                  ),
                  const SizedBox(width: 7),
                  Text(
                    topic.author.nickname ?? topic.author.username,
                    style: metaStyle.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.75),
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
              ),
              _MetaItem(
                icon: Icons.schedule,
                value: formatDateTime(topic.createdAt),
              ),
              for (final CategoryBriefPayload category in topic.categories)
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: <Widget>[
                    Container(
                      width: 8,
                      height: 8,
                      decoration: BoxDecoration(
                        color: colorFromHex(category.color),
                        borderRadius: BorderRadius.circular(3),
                      ),
                    ),
                    const SizedBox(width: 6),
                    Text(category.name, style: metaStyle),
                  ],
                ),
              _MetaItem(
                icon: Icons.chat_bubble_outline,
                value: formatNumber(topic.replyCount),
              ),
              _MetaItem(
                icon: Icons.visibility_outlined,
                value: formatNumber(topic.viewCount),
              ),
              _MetaItem(
                icon: Icons.favorite_border,
                value: formatNumber(likeCount),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _MetaItem extends StatelessWidget {
  const _MetaItem({required this.icon, required this.value});

  final IconData icon;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 15, color: GfTheme.colorsOf(context).iconMuted),
        const SizedBox(width: 4),
        Text(
          value,
          style: GfTheme.typographyOf(
            context,
          ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
        ),
      ],
    );
  }
}

class _PostGroup {
  const _PostGroup({required this.root, required this.replies});

  final PostPayload root;
  final List<PostPayload> replies;
}

class _PostGroupView extends StatelessWidget {
  const _PostGroupView({
    required this.group,
    required this.expanded,
    required this.liked,
    required this.bookmarked,
    required this.watched,
    required this.likeCount,
    required this.onToggleLike,
    required this.onToggleBookmark,
    required this.onToggleWatch,
    required this.canReportTopic,
    required this.onReportTopic,
    required this.onToggleReplies,
    required this.onReply,
    required this.onReport,
  });

  static const int _previewCount = 3;

  final _PostGroup group;
  final bool expanded;
  final bool liked;
  final bool bookmarked;
  final bool watched;
  final int likeCount;
  final VoidCallback onToggleLike;
  final VoidCallback onToggleBookmark;
  final VoidCallback onToggleWatch;
  final bool canReportTopic;
  final VoidCallback onReportTopic;
  final VoidCallback onToggleReplies;
  final ValueChanged<PostPayload> onReply;
  final ValueChanged<PostPayload> onReport;

  @override
  Widget build(BuildContext context) {
    final List<PostPayload> replies = expanded
        ? group.replies
        : group.replies.take(_previewCount).toList();
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 16, 12, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          _PostCard(
            post: group.root,
            isOriginalPost: group.root.postNo == 1,
            liked: liked,
            bookmarked: bookmarked,
            watched: watched,
            topicLikeCount: likeCount,
            onTopicLike: onToggleLike,
            onTopicBookmark: onToggleBookmark,
            onTopicWatch: onToggleWatch,
            canReportTopic: canReportTopic,
            onTopicReport: onReportTopic,
            onReply: () => onReply(group.root),
            onReport: () => onReport(group.root),
          ),
          if (replies.isNotEmpty) ...<Widget>[
            const SizedBox(height: 12),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10),
              decoration: BoxDecoration(
                color: GfTheme.colorsOf(context).base200.withValues(alpha: 0.7),
                borderRadius: BorderRadius.circular(
                  GfTheme.radiiOf(context).field,
                ),
                border: Border.all(color: GfTheme.colorsOf(context).line),
              ),
              child: Column(
                children: <Widget>[
                  for (
                    int index = 0;
                    index < replies.length;
                    index++
                  ) ...<Widget>[
                    if (index > 0) const GfDivider(),
                    _NestedPostCard(
                      post: replies[index],
                      onReply: () => onReply(replies[index]),
                      onReport: () => onReport(replies[index]),
                    ),
                  ],
                  if (group.replies.length > _previewCount)
                    TextButton.icon(
                      onPressed: onToggleReplies,
                      icon: Icon(
                        expanded
                            ? Icons.expand_less
                            : Icons.keyboard_arrow_down,
                        size: 18,
                      ),
                      label: Text(
                        expanded
                            ? AppLocalizations.of(context).commonClose
                            : '+${group.replies.length - _previewCount}',
                      ),
                    ),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _PostCard extends StatelessWidget {
  const _PostCard({
    required this.post,
    required this.isOriginalPost,
    required this.liked,
    required this.bookmarked,
    required this.watched,
    required this.topicLikeCount,
    required this.onTopicLike,
    required this.onTopicBookmark,
    required this.onTopicWatch,
    required this.canReportTopic,
    required this.onTopicReport,
    required this.onReply,
    required this.onReport,
  });

  final PostPayload post;
  final bool isOriginalPost;
  final bool liked;
  final bool bookmarked;
  final bool watched;
  final int topicLikeCount;
  final VoidCallback onTopicLike;
  final VoidCallback onTopicBookmark;
  final VoidCallback onTopicWatch;
  final bool canReportTopic;
  final VoidCallback onTopicReport;
  final VoidCallback onReply;
  final VoidCallback onReport;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.only(top: 2),
          child: GfAvatar(
            src: resolveApiAssetUrl(post.author.avatarUrl),
            size: 38,
            ring: true,
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              _PostHeading(
                post: post,
                original: isOriginalPost,
                onReply: onReply,
                onReport: onReport,
              ),
              if (post.replyToUsername != null) ...<Widget>[
                const SizedBox(height: 7),
                _ReplyReference(username: post.replyToUsername!),
              ],
              const SizedBox(height: 10),
              GfMarkdownView(data: post.content),
              if (isOriginalPost) ...<Widget>[
                const SizedBox(height: 14),
                Divider(height: 1, color: colors.line),
                const SizedBox(height: 10),
                Wrap(
                  spacing: 8,
                  runSpacing: 6,
                  children: <Widget>[
                    _TopicActionButton(
                      icon: liked ? Icons.favorite : Icons.favorite_border,
                      label: formatNumber(topicLikeCount),
                      active: liked,
                      activeColor: colors.error,
                      onTap: onTopicLike,
                    ),
                    _TopicActionButton(
                      icon: bookmarked ? Icons.bookmark : Icons.bookmark_border,
                      label: '',
                      active: bookmarked,
                      activeColor: colors.primary,
                      onTap: onTopicBookmark,
                    ),
                    _TopicActionButton(
                      icon: watched
                          ? Icons.notifications
                          : Icons.notifications_none,
                      label: '',
                      active: watched,
                      activeColor: colors.success,
                      onTap: onTopicWatch,
                    ),
                    if (canReportTopic)
                      _TopicActionButton(
                        icon: Icons.flag_outlined,
                        label: '',
                        active: false,
                        activeColor: colors.warning,
                        onTap: onTopicReport,
                      ),
                  ],
                ),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

class _PostHeading extends StatelessWidget {
  const _PostHeading({
    required this.post,
    required this.original,
    required this.onReply,
    required this.onReport,
  });

  final PostPayload post;
  final bool original;
  final VoidCallback onReply;
  final VoidCallback onReport;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Text(
                post.author.nickname ?? post.author.username,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 2),
              Row(
                children: <Widget>[
                  if (post.postNo > 0) ...<Widget>[
                    Text(
                      '#${post.postNo}',
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.55),
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(width: 8),
                  ],
                  Flexible(
                    child: Text(
                      formatDateTime(post.createdAt),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: colors.baseContent.withValues(alpha: 0.55),
                        fontSize: 12,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
        if (original)
          Container(
            margin: const EdgeInsets.only(top: 2, right: 2),
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(
              color: colors.base200,
              borderRadius: BorderRadius.circular(4),
            ),
            child: Text(
              l10n.topicTitle,
              style: TextStyle(
                color: colors.baseContent.withValues(alpha: 0.55),
                fontSize: 11,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        GfIconButton(
          icon: Icons.reply_outlined,
          tooltip: l10n.topicReply,
          onPressed: onReply,
        ),
        if (!original)
          GfIconButton(
            icon: Icons.flag_outlined,
            tooltip: l10n.topicReport,
            onPressed: onReport,
          ),
      ],
    );
  }
}

class _ReplyReference extends StatelessWidget {
  const _ReplyReference({required this.username});

  final String username;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(
        color: colors.base200,
        border: Border(left: BorderSide(color: colors.primary, width: 2)),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        '${AppLocalizations.of(context).topicReply} @$username',
        style: TextStyle(
          color: colors.baseContent.withValues(alpha: 0.55),
          fontSize: 12,
        ),
      ),
    );
  }
}

class _NestedPostCard extends StatelessWidget {
  const _NestedPostCard({
    required this.post,
    required this.onReply,
    required this.onReport,
  });

  final PostPayload post;
  final VoidCallback onReply;
  final VoidCallback onReport;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 11),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          GfAvatar(
            src: resolveApiAssetUrl(post.author.avatarUrl),
            size: 28,
            ring: true,
          ),
          const SizedBox(width: 9),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                _PostHeading(
                  post: post,
                  original: false,
                  onReply: onReply,
                  onReport: onReport,
                ),
                if (post.replyToUsername != null) ...<Widget>[
                  const SizedBox(height: 6),
                  _ReplyReference(username: post.replyToUsername!),
                ],
                const SizedBox(height: 8),
                GfMarkdownView(data: post.content),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _TopicActionButton extends StatelessWidget {
  const _TopicActionButton({
    required this.icon,
    required this.label,
    required this.active,
    required this.activeColor,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final bool active;
  final Color activeColor;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Color foreground = active
        ? activeColor
        : colors.baseContent.withValues(alpha: 0.55);
    return Material(
      color: active ? activeColor.withValues(alpha: 0.1) : Colors.transparent,
      borderRadius: BorderRadius.circular(6),
      child: InkWell(
        borderRadius: BorderRadius.circular(6),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              Icon(icon, size: 17, color: foreground),
              if (label.isNotEmpty) ...<Widget>[
                const SizedBox(width: 5),
                Text(
                  label,
                  style: TextStyle(
                    color: foreground,
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
